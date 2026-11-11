package dev.herdrremote.watch

import android.app.KeyguardManager
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.ServiceInfo
import android.os.IBinder
import android.os.PowerManager
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import dev.herdrremote.HerdrRemoteApp
import dev.herdrremote.MainActivity
import dev.herdrremote.R
import dev.herdrremote.net.Agent
import dev.herdrremote.net.DecisionRequest
import dev.herdrremote.net.PromptView
import dev.herdrremote.net.Uuid7
import dev.herdrremote.ui.Texts
import dev.herdrremote.ui.theme.NoteKind
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.onSubscription
import kotlinx.coroutines.launch

/**
 * Background watching (ADR 0013, ADR 0019): an opt-in foreground service that holds /v1/events. It
 * alerts when an agent is blocked, with the prompt and Approve and Deny, and when an agent
 * finishes.
 */
class WatchService : Service() {
    // One thread: a cancelled prompt fetch can never post after the change that cancelled it.
    @OptIn(ExperimentalCoroutinesApi::class)
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default.limitedParallelism(1))
    private val notifier = Notifier()
    /** The prompt fetch in flight for each agent. */
    private val fetches = HashMap<String, Job>()
    /** How to build each agent's alert, for a locked or an unlocked phone. */
    private val shown = HashMap<String, (Boolean) -> NotificationCompat.Builder>()
    @Volatile private var locked = true
    private var watching = false

    private var unlock: Job? = null

    /**
     * SR-11: a locked phone shows only the generic text, whatever its lock screen setting. The
     * screen going off redacts each alert; an unlock shows it in full again.
     */
    private val lock =
        object : BroadcastReceiver() {
            override fun onReceive(context: Context, intent: Intent) {
                val off = intent.action == Intent.ACTION_SCREEN_OFF
                scope.launch {
                    unlock?.cancel()
                    if (off) {
                        locked = true
                        repost()
                        return@launch
                    }
                    // ponytail: USER_PRESENT never reached the app on Android 17 (the S26), so the
                    // screen coming on starts a 500 ms check of the keyguard until the unlock.
                    unlock = scope.launch {
                        val keyguard = getSystemService(KeyguardManager::class.java)
                        while (keyguard.isKeyguardLocked) delay(500)
                        if (locked) {
                            locked = false
                            repost()
                        }
                    }
                }
            }
        }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        channels(this)
        locked =
            getSystemService(KeyguardManager::class.java).isKeyguardLocked ||
                !getSystemService(PowerManager::class.java).isInteractive
        ContextCompat.registerReceiver(
            this,
            lock,
            IntentFilter(Intent.ACTION_SCREEN_OFF).apply {
                addAction(Intent.ACTION_SCREEN_ON)
                addAction(Intent.ACTION_USER_PRESENT)
            },
            ContextCompat.RECEIVER_NOT_EXPORTED,
        )
        // Posted once and never updated: an update brought a dismissed notification back.
        startForeground(ONGOING_ID, ongoing(), ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE)
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val app = application as HerdrRemoteApp
        if (!app.store.paired || !app.store.watch) {
            stopSelf()
            return START_NOT_STICKY
        }
        if (!watching) {
            watching = true
            watch(app)
        }
        if (intent?.action == ACTION_DECIDE) scope.launch { decide(app, intent) }
        return START_STICKY
    }

    override fun onDestroy() {
        unregisterReceiver(lock)
        scope.cancel()
        // Alerts exist only while watching: a stale Approve must not stay behind.
        manager().cancelAll()
        super.onDestroy()
    }

    private fun watch(app: HerdrRemoteApp) {
        app.repository.start()
        scope.launch {
            app.repository.events
                // The app may have opened the stream first: start from the agents known now.
                .onSubscription {
                    emit(SseEvent.Snapshot(app.repository.state.value.agents.values.toList()))
                }
                .collect { ev ->
                    val agents =
                        when (ev) {
                            is SseEvent.Snapshot -> ev.agents
                            is SseEvent.Changed -> listOf(ev.agent)
                            is SseEvent.Removed -> {
                                notifier.forget(ev.address)
                                hide(ev.address)
                                emptyList()
                            }
                        }
                    for ((a, alert) in notifier.alerts(agents)) {
                        fetch(a.address, null)
                        when (alert) {
                            Alert.Blocked,
                            Alert.Refresh ->
                                fetch(a.address) { postBlocked(app, a, alert == Alert.Blocked) }
                            Alert.Done -> show(a.address, true) { done(a, it) }
                            Alert.Clear -> hide(a.address)
                        }
                    }
                }
        }
    }

    /** Cancels the agent's fetch in flight, and starts [block] in its place. */
    private fun fetch(address: String, block: (suspend () -> Unit)?) {
        fetches.remove(address)?.cancel()
        if (block != null) fetches[address] = scope.launch { block() }
    }

    /** Posts an agent's alert. A [loud] one alerts; any other only updates what shows. */
    private fun show(
        address: String,
        loud: Boolean,
        make: (Boolean) -> NotificationCompat.Builder,
    ) {
        shown[address] = make
        manager().notify(id(address), make(locked).setOnlyAlertOnce(!loud).build())
    }

    private fun hide(address: String) {
        fetch(address, null)
        shown.remove(address)
        manager().cancel(id(address))
    }

    /** Each alert that still shows, again, redacted or in full. The user's dismissals stay. */
    private fun repost() {
        val active = manager().activeNotifications.map { it.id }.toSet()
        shown.keys.retainAll { id(it) in active }
        for ((address, make) in shown) manager()
            .notify(id(address), make(locked).setOnlyAlertOnce(true).build())
    }

    /** The prompt, for the preview and the buttons. Without it, the alert still comes. */
    private suspend fun postBlocked(
        app: HerdrRemoteApp,
        a: Agent,
        loud: Boolean,
        note: String? = null,
    ) {
        val p =
            try {
                app.api()?.prompt(a.address)
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                null
            }
        show(a.address, loud) { blocked(a, p, it, note) }
    }

    /**
     * Approve or Deny from the notification. Never retried, as in the app. The daemon still checks
     * the turn and the prompt hash of the prompt that the notification showed.
     */
    private suspend fun decide(app: HerdrRemoteApp, i: Intent) {
        val address = i.getStringExtra(EXTRA_ADDRESS) ?: return
        val name = i.getStringExtra(EXTRA_NAME).orEmpty()
        val approve = i.getBooleanExtra(EXTRA_APPROVE, false)
        val api = app.api() ?: return
        // A fetch in flight must not bring the buttons back.
        fetch(address, null)
        show(address, false) {
            result(address, name, if (approve) "Approving…" else "Denying…", it)
        }
        val text =
            try {
                Texts.result(
                    api.decide(
                        address,
                        DecisionRequest(
                            Uuid7.make(api.now()),
                            if (approve) "approve" else "deny",
                            i.getLongExtra(EXTRA_TURN, -1),
                            i.getStringExtra(EXTRA_HASH).orEmpty(),
                        ),
                    )
                )
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                Texts.actionError(e)
            }
        // Refused while the prompt still waits (the Mac in use, a stale prompt): the buttons come
        // back, under the reason.
        val a = app.repository.state.value.agents[address]
        if (Texts.kind(text) == NoteKind.Error && a?.blocked == true)
            fetch(address) { postBlocked(app, a, false, text) }
        else show(address, false) { result(address, name, text, it) }
    }

    private fun manager() = getSystemService(NotificationManager::class.java)

    private fun ongoing(): Notification =
        NotificationCompat.Builder(this, CHANNEL_WATCH)
            .setSmallIcon(R.drawable.ic_stat_herdr)
            .setContentTitle("Herdr Remote is watching")
            .setOngoing(true)
            .setSilent(true)
            .setShowWhen(false)
            .setContentIntent(open(null))
            .build()

    /**
     * An alert. A locked phone (and so the watch) gets only the [title], the agent's name and what
     * it needs: no prompt, no buttons. The public version says the same, for a lock screen that
     * hides sensitive content.
     */
    private fun alert(
        channel: String,
        address: String,
        locked: Boolean,
        title: String,
        text: String,
    ): NotificationCompat.Builder =
        NotificationCompat.Builder(this, channel)
            .setSmallIcon(R.drawable.ic_stat_herdr)
            .setContentTitle(title)
            .setContentText(if (locked) null else text)
            .setVisibility(NotificationCompat.VISIBILITY_PRIVATE)
            .setPublicVersion(
                NotificationCompat.Builder(this, channel)
                    .setSmallIcon(R.drawable.ic_stat_herdr)
                    .setContentTitle(title)
                    .build()
            )
            .setAutoCancel(true)
            .setContentIntent(open(address))

    private fun blocked(
        a: Agent,
        p: PromptView?,
        locked: Boolean,
        note: String? = null,
    ): NotificationCompat.Builder {
        val msg = a.msg?.ifEmpty { null } ?: "Waiting for a decision (${a.kind ?: "blocked"})"
        val b =
            alert(
                CHANNEL_ALERTS,
                a.address,
                locked,
                "${name(a)} needs ${if (p?.approvable == true) "approval" else "attention"}",
                note ?: msg,
            )
        if (locked) return b
        // Expanded: the prompt as the Mac shows it.
        val block = p?.block.orEmpty().take(1500)
        b.setStyle(
            NotificationCompat.BigTextStyle()
                .bigText(
                    listOfNotNull(note, msg, block).filter { it.isNotEmpty() }.joinToString("\n\n")
                )
        )
        if (p?.approvable == true) b.addAction(action(a, p, true)).addAction(action(a, p, false))
        return b
    }

    private fun done(a: Agent, locked: Boolean): NotificationCompat.Builder =
        alert(
            CHANNEL_DONE,
            a.address,
            locked,
            "${name(a)} ${if (a.state == "error") "stopped with an error" else "finished"}",
            a.title.ifEmpty { a.app },
        )

    /**
     * The outcome of an action from the notification: quiet, and gone after 10 s when it worked.
     */
    private fun result(
        address: String,
        name: String,
        text: String,
        locked: Boolean,
    ): NotificationCompat.Builder =
        alert(CHANNEL_ALERTS, address, locked, name, text).setSilent(true).apply {
            if (Texts.kind(text) == NoteKind.Ok) setTimeoutAfter(10_000)
        }

    /** Only on an unlocked phone's alert, and the phone must be unlocked to use it. */
    private fun action(a: Agent, p: PromptView, approve: Boolean): NotificationCompat.Action {
        val i =
            Intent(this, WatchService::class.java)
                .setAction(ACTION_DECIDE)
                .putExtra(EXTRA_ADDRESS, a.address)
                .putExtra(EXTRA_NAME, name(a))
                .putExtra(EXTRA_TURN, p.turn)
                .putExtra(EXTRA_HASH, p.promptHash)
                .putExtra(EXTRA_APPROVE, approve)
        val pi =
            PendingIntent.getService(
                this,
                id(a.address) * 2 + if (approve) 1 else 0,
                i,
                PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
            )
        return NotificationCompat.Action.Builder(0, if (approve) "Approve" else "Deny", pi)
            .setAuthenticationRequired(true)
            .build()
    }

    private fun open(address: String?): PendingIntent {
        val i =
            Intent(this, MainActivity::class.java)
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP)
        if (address != null) i.putExtra(MainActivity.EXTRA_ADDRESS, address)
        return PendingIntent.getActivity(
            this,
            address?.hashCode() ?: 0,
            i,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
    }

    companion object {
        const val CHANNEL_WATCH = "watch"
        const val CHANNEL_ALERTS = "alerts"
        const val CHANNEL_DONE = "done"
        private const val ONGOING_ID = 1
        private const val ACTION_DECIDE = "dev.herdrremote.DECIDE"
        private const val EXTRA_ADDRESS = "address"
        private const val EXTRA_NAME = "name"
        private const val EXTRA_TURN = "turn"
        private const val EXTRA_HASH = "hash"
        private const val EXTRA_APPROVE = "approve"

        /** One notification for each agent: a new alert replaces its last one. */
        private fun id(address: String) = address.hashCode()

        private fun name(a: Agent) = a.where.ifEmpty { a.label }

        fun channels(context: Context) {
            val m = context.getSystemService(NotificationManager::class.java)
            m.createNotificationChannel(
                NotificationChannel(
                    CHANNEL_WATCH,
                    "Background watching",
                    NotificationManager.IMPORTANCE_LOW,
                )
            )
            m.createNotificationChannel(
                NotificationChannel(
                        CHANNEL_ALERTS,
                        "Agent needs attention",
                        NotificationManager.IMPORTANCE_HIGH,
                    )
                    .apply { lockscreenVisibility = Notification.VISIBILITY_PRIVATE }
            )
            m.createNotificationChannel(
                NotificationChannel(
                        CHANNEL_DONE,
                        "Agent finished",
                        NotificationManager.IMPORTANCE_DEFAULT,
                    )
                    .apply { lockscreenVisibility = Notification.VISIBILITY_PRIVATE }
            )
        }

        fun start(context: Context) =
            ContextCompat.startForegroundService(context, Intent(context, WatchService::class.java))

        fun stop(context: Context) = context.stopService(Intent(context, WatchService::class.java))
    }
}

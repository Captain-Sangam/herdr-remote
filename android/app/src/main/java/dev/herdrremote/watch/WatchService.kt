package dev.herdrremote.watch

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.ServiceInfo
import android.os.IBinder
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import dev.herdrremote.HerdrRemoteApp
import dev.herdrremote.MainActivity
import dev.herdrremote.R
import dev.herdrremote.net.Agent
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancelAndJoin
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.job
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking

/**
 * Background watching (ADR 0013): an opt-in foreground service that holds /v1/events and shows one
 * private notification for each blocked episode.
 */
class WatchService : Service() {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private val notifier = BlockedNotifier()

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        channels(this)
        startForeground(ONGOING_ID, ongoing(0), ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE)
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val app = application as HerdrRemoteApp
        if (!app.store.paired || !app.store.watch) {
            stopSelf()
            return START_NOT_STICKY
        }
        val repo = app.repository
        repo.start()
        scope.launch {
            repo.state
                .map { it.agents.size to it.connected }
                .distinctUntilChanged()
                .collect { (n, connected) ->
                    manager().notify(ONGOING_ID, ongoing(n, connected))
                }
        }
        scope.launch {
            repo.events.collect { ev ->
                val agents =
                    when (ev) {
                        is SseEvent.Snapshot -> ev.agents
                        is SseEvent.Changed -> listOf(ev.agent)
                        is SseEvent.Removed -> emptyList()
                    }
                for (a in notifier.newEpisodes(agents)) manager()
                    .notify(a.address.hashCode() xor a.turn.toInt(), blocked(a))
            }
        }
        return START_STICKY
    }

    override fun onDestroy() {
        // Wait for the collectors: a state reset during stopService() can repost the
        // ongoing notification after the system removed it, and it would stay for good.
        runBlocking { scope.coroutineContext.job.cancelAndJoin() }
        manager().cancel(ONGOING_ID)
        super.onDestroy()
    }

    private fun manager() = getSystemService(NotificationManager::class.java)

    private fun ongoing(n: Int, connected: Boolean = false): Notification =
        NotificationCompat.Builder(this, CHANNEL_WATCH)
            .setSmallIcon(R.drawable.ic_stat_herdr)
            .setContentTitle(
                if (connected) "Watching $n agent${if (n == 1) "" else "s"}"
                else "Connecting to the Mac…"
            )
            .setOngoing(true)
            .setSilent(true)
            .setVisibility(NotificationCompat.VISIBILITY_PRIVATE)
            .setPublicVersion(
                NotificationCompat.Builder(this, CHANNEL_WATCH)
                    .setSmallIcon(R.drawable.ic_stat_herdr)
                    .setContentTitle("Herdr Remote is watching")
                    .build()
            )
            .setContentIntent(open(null))
            .build()

    /**
     * The same generic line in the private and the public version: the tab name shows only inside
     * the app.
     */
    private fun blocked(a: Agent): Notification {
        val public =
            NotificationCompat.Builder(this, CHANNEL_ALERTS)
                .setSmallIcon(R.drawable.ic_stat_herdr)
                .setContentTitle("An agent needs attention")
                .build()
        return NotificationCompat.Builder(this, CHANNEL_ALERTS)
            .setSmallIcon(R.drawable.ic_stat_herdr)
            .setContentTitle("An agent needs attention")
            .setVisibility(NotificationCompat.VISIBILITY_PRIVATE)
            .setPublicVersion(public)
            .setAutoCancel(true)
            .setContentIntent(open(a.address))
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
        private const val ONGOING_ID = 1

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
                    .apply {
                        lockscreenVisibility = Notification.VISIBILITY_PRIVATE
                    }
            )
        }

        fun start(context: Context) =
            ContextCompat.startForegroundService(context, Intent(context, WatchService::class.java))

        fun stop(context: Context) = context.stopService(Intent(context, WatchService::class.java))
    }
}

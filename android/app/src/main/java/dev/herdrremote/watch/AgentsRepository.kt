package dev.herdrremote.watch

import dev.herdrremote.HerdrRemoteApp
import dev.herdrremote.net.HerdrApi
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import okhttp3.Call
import okhttp3.Response
import java.io.IOException
import kotlin.coroutines.coroutineContext

/**
 * Holds /v1/events: one SSE connection with a 30 s read timeout (decision 5),
 * reconnecting with backoff. Screens read [state]; the service reads [events].
 * stop() cancels the stream's call: the keepalives would otherwise hold the
 * blocking read open long after the loop was cancelled.
 */
class AgentsRepository(private val app: HerdrRemoteApp) {
    private val _state = MutableStateFlow(AgentsState())
    val state: StateFlow<AgentsState> = _state.asStateFlow()

    /** Every event, for notifications. */
    val events = MutableSharedFlow<SseEvent>(extraBufferCapacity = 256)

    /** Fires on each new connection: live views reopen then. */
    val reconnected = MutableSharedFlow<Unit>(extraBufferCapacity = 8)

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private var job: Job? = null
    private var call: Call? = null

    @Synchronized
    fun start() {
        if (job?.isActive == true) return
        job = scope.launch { loop() }
    }

    @Synchronized
    fun stop() {
        job?.cancel()
        job = null
        call?.cancel()
        _state.value = AgentsState()
    }

    /** The stream call of the loop [me], or null when stop() came first. */
    @Synchronized
    private fun openCall(api: HerdrApi, host: String, me: Job?): Call? =
        if (me !== job) null else api.events(host).also { call = it }

    private suspend fun loop() {
        val me = coroutineContext[Job]
        var backoff = 1000L
        while (scope.isActive) {
            val api = app.api() ?: return
            var served = false
            for (host in api.hosts()) {
                val c = openCall(api, host, me) ?: return
                try {
                    api.open(c).use { resp ->
                        api.markGood(host)
                        served = true
                        backoff = 1000L
                        _state.update { it.copy(connected = true) }
                        reconnected.tryEmit(Unit)
                        // Refresh the clock offset and the endpoint list beside the stream, never before it.
                        scope.launch { runCatching { api.hello() } }
                        read(resp)
                    }
                } catch (e: IOException) {
                    if (c.isCanceled()) return // stop(): not the next endpoint
                    // next endpoint, or back off
                }
                _state.update { it.copy(connected = false) }
                if (served) break
            }
            delay(backoff)
            backoff = minOf(backoff * 2, 30_000L)
        }
    }

    private fun read(resp: Response) {
        val source = resp.body?.source() ?: return
        var event = ""
        val data = StringBuilder()
        while (true) {
            val line = source.readUtf8Line() ?: return
            when {
                line.isEmpty() -> {
                    if (event.isNotEmpty()) {
                        Reducer.parse(event, data.toString())?.let { ev ->
                            _state.update { Reducer.reduce(it, ev, System.currentTimeMillis()) }
                            events.tryEmit(ev)
                        }
                    }
                    event = ""
                    data.setLength(0)
                }
                line.startsWith(":") -> {}
                line.startsWith("event: ") -> event = line.substring(7)
                line.startsWith("data: ") -> { if (data.isNotEmpty()) data.append('\n'); data.append(line.substring(6)) }
            }
        }
    }
}

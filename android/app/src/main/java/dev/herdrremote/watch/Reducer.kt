package dev.herdrremote.watch

import dev.herdrremote.net.Agent
import dev.herdrremote.net.json
import kotlinx.serialization.builtins.ListSerializer

/** One Server-Sent Event from /v1/events. */
sealed class SseEvent {
    data class Snapshot(val agents: List<Agent>) : SseEvent()

    data class Changed(val agent: Agent) : SseEvent()

    data class Removed(val address: String) : SseEvent()
}

data class AgentsState(
    val agents: Map<String, Agent> = emptyMap(),
    /** Phone clock, ms, of the last event; 0 before the first snapshot. */
    val updatedAt: Long = 0,
    val connected: Boolean = false,
) {
    val sorted: List<Agent>
        get() =
            agents.values.sortedWith(
                compareBy({ !it.blocked }, { it.space }, { it.tab }, { it.address })
            )
}

object Reducer {
    fun parse(event: String, data: String): SseEvent? =
        when (event) {
            "snapshot" ->
                SseEvent.Snapshot(json.decodeFromString(ListSerializer(Agent.serializer()), data))
            "agent" -> SseEvent.Changed(json.decodeFromString(Agent.serializer(), data))
            "agent_removed" ->
                SseEvent.Removed(json.decodeFromString(Removed.serializer(), data).address)
            else -> null
        }

    @kotlinx.serialization.Serializable private data class Removed(val address: String)

    /**
     * A snapshot (first, or after `missed`) replaces everything; a change or removal edits one
     * agent.
     */
    fun reduce(state: AgentsState, ev: SseEvent, now: Long): AgentsState =
        when (ev) {
            is SseEvent.Snapshot ->
                state.copy(
                    agents = ev.agents.associateBy { it.address },
                    updatedAt = now,
                    connected = true,
                )
            is SseEvent.Changed ->
                state.copy(agents = state.agents + (ev.agent.address to ev.agent), updatedAt = now)
            is SseEvent.Removed -> state.copy(agents = state.agents - ev.address, updatedAt = now)
        }
}

/** What the watch service does with an agent's notification. */
enum class Alert {
    /** A new blocked episode: alert, with the prompt. */
    Blocked,
    /** The same episode with a new turn: update the prompt quietly (its hash holds the turn). */
    Refresh,
    /** It stopped working (done, idle or error): alert. */
    Done,
    /** It left blocked some other way: the old prompt's notification goes. */
    Clear,
}

/**
 * The alerts (ADR 0013, ADR 0019). A blocked episode starts when an agent becomes blocked, or shows
 * another prompt while blocked. Not the turn: hc sends the turn after the blocked status, so one
 * episode has two turns. A reconnect's snapshot with the same prompt notifies nothing. Done needs a
 * state the phone saw before, so the first snapshot never alerts for it. Herdr says `idle`, not
 * `done`, when someone was looking at the agent.
 */
class Notifier {
    private val last = HashMap<String, Agent>()

    fun alerts(agents: Collection<Agent>): List<Pair<Agent, Alert>> = agents.mapNotNull { a ->
        val prev = last.put(a.address, a)
        when {
            a.blocked ->
                when {
                    prev?.blocked != true || prev.msg.orEmpty() != a.msg.orEmpty() -> Alert.Blocked
                    prev.turn != a.turn -> Alert.Refresh
                    else -> null
                }
            a.state != "working" &&
                (prev?.state == "working" || (prev?.blocked == true && a.state == "done")) ->
                Alert.Done
            prev?.blocked == true -> Alert.Clear
            else -> null
        }?.let { a to it }
    }

    fun forget(address: String) {
        last.remove(address)
    }
}

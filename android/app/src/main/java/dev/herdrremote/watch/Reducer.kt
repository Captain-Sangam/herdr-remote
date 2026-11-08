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
    val sorted: List<Agent> get() = agents.values.sortedWith(compareBy({ !it.blocked }, { it.space }, { it.tab }, { it.address }))
}

object Reducer {
    fun parse(event: String, data: String): SseEvent? = when (event) {
        "snapshot" -> SseEvent.Snapshot(json.decodeFromString(ListSerializer(Agent.serializer()), data))
        "agent" -> SseEvent.Changed(json.decodeFromString(Agent.serializer(), data))
        "agent_removed" -> SseEvent.Removed(json.decodeFromString(Removed.serializer(), data).address)
        else -> null
    }

    @kotlinx.serialization.Serializable
    private data class Removed(val address: String)

    /** A snapshot (first, or after `missed`) replaces everything; a change or removal edits one agent. */
    fun reduce(state: AgentsState, ev: SseEvent, now: Long): AgentsState = when (ev) {
        is SseEvent.Snapshot -> state.copy(agents = ev.agents.associateBy { it.address }, updatedAt = now, connected = true)
        is SseEvent.Changed -> state.copy(agents = state.agents + (ev.agent.address to ev.agent), updatedAt = now)
        is SseEvent.Removed -> state.copy(agents = state.agents - ev.address, updatedAt = now)
    }
}

/**
 * One notification for each blocked episode: the same (terminal, turn, msg)
 * never notifies twice, also across a reconnect's snapshot.
 */
class BlockedNotifier {
    private val seen = LinkedHashSet<String>()

    private fun key(a: Agent) = "${a.address}|${a.turn}|${a.msg.orEmpty()}"

    /** The agents that just became blocked and were not notified yet. */
    fun newEpisodes(agents: Collection<Agent>): List<Agent> {
        val out = agents.filter { it.blocked && seen.add(key(it)) }
        if (seen.size > 500) seen.iterator().let { it.next(); it.remove() }
        return out
    }
}

package dev.herdrremote

import dev.herdrremote.net.Agent
import dev.herdrremote.watch.AgentsState
import dev.herdrremote.watch.BlockedNotifier
import dev.herdrremote.watch.Reducer
import dev.herdrremote.watch.SseEvent
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class ReducerTest {
    private val snapshot =
        """[{"address":"t1","space":"Niti","tab":"main","where":"Niti / main","app":"claude-code","state":"blocked","kind":"permission","msg":"approve Bash: touch x","turn":2,"title":"t"},
        {"address":"t3","space":"","tab":"","where":"Email / codex","app":"codex","state":"idle","turn":5,"title":""}]"""

    @Test
    fun snapshotChangeRemovedAndMissed() {
        var s = AgentsState()
        s = Reducer.reduce(s, Reducer.parse("snapshot", snapshot)!!, 1000)
        assertEquals(setOf("t1", "t3"), s.agents.keys)
        assertTrue(s.connected)
        assertEquals("permission", s.agents["t1"]!!.kind)
        assertEquals(1000, s.updatedAt)

        s =
            Reducer.reduce(
                s,
                Reducer.parse(
                    "agent",
                    """{"address":"t3","app":"codex","state":"blocked","kind":"permission","turn":6,"title":""}""",
                )!!,
                2000,
            )
        assertEquals("blocked", s.agents["t3"]!!.state)
        assertEquals(6, s.agents["t3"]!!.turn)
        assertEquals(2000, s.updatedAt)
        // Both blocked: the empty space name sorts first.
        assertEquals(listOf("t3", "t1"), s.sorted.map { it.address })

        s = Reducer.reduce(s, Reducer.parse("agent_removed", """{"address":"t1"}""")!!, 3000)
        assertEquals(setOf("t3"), s.agents.keys)

        // missed: a fresh snapshot replaces everything that was there.
        s =
            Reducer.reduce(
                s,
                Reducer.parse(
                    "snapshot",
                    """[{"address":"t9","app":"pi","state":"working","turn":1,"title":""}]""",
                )!!,
                4000,
            )
        assertEquals(setOf("t9"), s.agents.keys)
        assertNull(Reducer.parse("bogus", "{}"))
    }

    @Test
    fun blockedFirstInSortedList() {
        val s = Reducer.reduce(AgentsState(), Reducer.parse("snapshot", snapshot)!!, 1)
        val t3 = s.agents["t3"]!!.copy(state = "blocked", kind = "permission")
        val sorted =
            s.copy(
                    agents =
                        s.agents + ("t0" to Agent(address = "t0", state = "idle")) + ("t3" to t3)
                )
                .sorted
        assertEquals(listOf("t3", "t1", "t0"), sorted.map { it.address })
        assertTrue(sorted.takeWhile { it.blocked }.size == 2)
    }

    @Test
    fun oneNotificationPerEpisode() {
        val n = BlockedNotifier()
        val first = Reducer.parse("snapshot", snapshot) as SseEvent.Snapshot
        assertEquals(listOf("t1"), n.newEpisodes(first.agents).map { it.address })
        assertEquals(emptyList<Agent>(), n.newEpisodes(first.agents))
        val again = first.agents.first { it.address == "t1" }
        assertEquals(0, n.newEpisodes(listOf(again)).size)
        // A reconnect snapshot with the same blocked agent: nothing new.
        assertEquals(0, n.newEpisodes(first.agents).size)
        // hc sends the turn after the blocked status: the same episode.
        assertEquals(0, n.newEpisodes(listOf(again.copy(turn = 3))).size)
        // Another prompt while still blocked: another episode.
        assertEquals(1, n.newEpisodes(listOf(again.copy(turn = 3, msg = "approve Write: y"))).size)
        // Not blocked: never. The next block is a new episode, even with the same prompt.
        assertEquals(0, n.newEpisodes(listOf(again.copy(turn = 4, state = "working"))).size)
        assertEquals(1, n.newEpisodes(listOf(again.copy(turn = 5))).size)
    }
}

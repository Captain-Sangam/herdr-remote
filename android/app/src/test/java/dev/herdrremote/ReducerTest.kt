package dev.herdrremote

import dev.herdrremote.net.Agent
import dev.herdrremote.watch.AgentsState
import dev.herdrremote.watch.Alert
import dev.herdrremote.watch.Notifier
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
        val n = Notifier()
        fun alerts(vararg a: Agent) = n.alerts(a.toList()).map { (x, alert) -> x.address to alert }
        val first = Reducer.parse("snapshot", snapshot) as SseEvent.Snapshot
        // The first snapshot: the blocked agent alerts, the idle one does not.
        assertEquals(listOf("t1" to Alert.Blocked), alerts(*first.agents.toTypedArray()))
        assertEquals(emptyList<Any>(), n.alerts(first.agents))
        val again = first.agents.first { it.address == "t1" }
        // hc sends the turn after the blocked status: the same episode, a quiet refresh.
        assertEquals(listOf("t1" to Alert.Refresh), alerts(again.copy(turn = 3)))
        assertEquals(emptyList<Any>(), alerts(again.copy(turn = 3)))
        // Another prompt while still blocked: another episode.
        assertEquals(
            listOf("t1" to Alert.Blocked),
            alerts(again.copy(turn = 3, msg = "approve Write: y")),
        )
        // Approved: working clears the prompt; then done, or idle when someone looked, alerts.
        assertEquals(listOf("t1" to Alert.Clear), alerts(again.copy(turn = 4, state = "working")))
        assertEquals(listOf("t1" to Alert.Done), alerts(again.copy(turn = 5, state = "idle")))
        assertEquals(emptyList<Any>(), alerts(again.copy(turn = 5, state = "done")))
        // The next block is a new episode, even with the same prompt.
        assertEquals(listOf("t1" to Alert.Blocked), alerts(again.copy(turn = 5)))
        // Denied: idle, no finish alert.
        assertEquals(listOf("t1" to Alert.Clear), alerts(again.copy(turn = 5, state = "idle")))
        // Blocked straight to done: finished.
        alerts(again.copy(turn = 6))
        assertEquals(listOf("t1" to Alert.Done), alerts(again.copy(turn = 7, state = "done")))
        // An agent gone and back: like a first sight.
        n.forget("t1")
        assertEquals(emptyList<Any>(), alerts(again.copy(state = "done")))
    }
}

package dev.herdrremote

import dev.herdrremote.net.ApiException
import dev.herdrremote.net.TranscriptItem
import dev.herdrremote.net.UnreachableException
import dev.herdrremote.ui.Texts
import dev.herdrremote.ui.endpointKind
import dev.herdrremote.ui.mergeTail
import dev.herdrremote.ui.theme.NoteKind
import java.io.IOException
import java.net.SocketTimeoutException
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class TextsTest {
    @Test
    fun newestPageMergesOntoLoadedItems() {
        fun items(vararg t: String) = t.map { TranscriptItem("assistant", "text", it, 1) }
        // The window moved on by one item: the earlier pages stay, the new item comes after.
        assertEquals(
            items("a", "b", "c", "d", "e"),
            mergeTail(items("a", "b", "c", "d"), items("c", "d", "e")),
        )
        // The same page: nothing changes.
        assertEquals(items("a", "b"), mergeTail(items("a", "b"), items("a", "b")))
        // Repeated items: the longest overlap wins, so nothing is shown twice.
        assertEquals(
            items("x", "y", "x", "y", "z"),
            mergeTail(items("x", "y", "x", "y"), items("x", "y", "x", "y", "z")),
        )
        // Nothing loaded yet: the page.
        assertEquals(items("a"), mergeTail(emptyList(), items("a")))
        // No overlap (another file, or terminal text): null, so the caller replaces.
        assertNull(mergeTail(items("a", "b"), items("c")))
    }

    @Test
    fun everyResultAndReasonHasWords() {
        for (r in listOf("applied", "sent_unconfirmed", "uncertain", "outcome_unknown", "sent")) {
            assertTrue(r, Texts.result(r).length > 5 && Texts.result(r) != r)
        }
        assertTrue(
            Texts.result("outcome_unknown").contains("Check the transcript before you send again")
        )
        assertTrue(Texts.result("uncertain").contains("Check the transcript"))
        for (reason in
            listOf(
                "stale",
                "changed",
                "already_answered",
                "mac_in_use",
                "not_actionable",
                "blocked",
                "not_agent",
                "expired_request",
                "cursor",
            )) {
            val t = Texts.error(ApiException(409, "x", reason, 4200))
            assertTrue(reason, t.length > 10 && !t.contains("_"))
        }
        assertEquals(
            "Someone is using the Mac right now (idle 4 s). Try again when it is free.",
            Texts.error(ApiException(409, "x", "mac_in_use", 4200)),
        )
        assertTrue(Texts.error(ApiException(429, "too many actions")).contains("Too fast"))
        assertTrue(Texts.error(ApiException(503, "hc unavailable")).contains("Herdr"))
        assertTrue(Texts.error(UnreachableException("x")).contains("not reachable"))
        assertTrue(Texts.error(RuntimeException("boom")).contains("Outcome unknown"))
    }

    @Test
    fun anActionKeepsTheDaemonsReasonAndNothingElse() {
        assertEquals(
            "The prompt has moved on. Reload it.",
            Texts.actionError(ApiException(409, "x", "stale")),
        )
        assertEquals(
            "The Mac cannot reach Herdr right now.",
            Texts.actionError(ApiException(503, "hc unavailable")),
        )
        for (e in
            listOf(
                UnreachableException("x"),
                SocketTimeoutException("timeout"),
                IOException("reset"),
                RuntimeException("boom"),
            )) {
            assertEquals(
                e.toString(),
                "Outcome unknown. Check the transcript before you send again.",
                Texts.actionError(e),
            )
        }
    }

    @Test
    fun theRefreshAfterAnActionKeepsTheActionsNote() {
        val note = Texts.result("outcome_unknown")
        assertEquals(note, Texts.refreshError(note, UnreachableException("x"), manual = false))
        assertEquals(null, Texts.refreshError(null, UnreachableException("x"), manual = false))
        assertEquals(
            "The Mac is not reachable.",
            Texts.refreshError(note, UnreachableException("x"), manual = true),
        )
    }

    @Test
    fun aNoteLooksLikeItsOutcome() {
        assertEquals(NoteKind.Ok, Texts.kind(Texts.result("applied")))
        assertEquals(NoteKind.Ok, Texts.kind(Texts.result("sent")))
        for (r in listOf("sent_unconfirmed", "uncertain", "outcome_unknown")) assertEquals(
            r,
            NoteKind.Warn,
            Texts.kind(Texts.result(r)),
        )
        assertEquals(
            NoteKind.Error,
            Texts.kind(Texts.error(ApiException(409, "x", "mac_in_use", 4200))),
        )
        assertEquals(NoteKind.Error, Texts.kind("Unlock the app first."))
    }

    @Test
    fun endpointsAreWifiOrTailscale() {
        assertEquals("Wi-Fi", endpointKind("192.168.1.24"))
        assertEquals("Wi-Fi", endpointKind("10.0.0.5"))
        assertEquals("Tailscale", endpointKind("100.101.2.3"))
        assertEquals("Wi-Fi", endpointKind("100.128.2.3"))
        assertEquals("Tailscale", endpointKind("zeus-mbp.tail3c2e.ts.net"))
    }
}

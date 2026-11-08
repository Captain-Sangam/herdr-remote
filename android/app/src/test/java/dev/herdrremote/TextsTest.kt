package dev.herdrremote

import dev.herdrremote.net.ApiException
import dev.herdrremote.net.UnreachableException
import dev.herdrremote.ui.Texts
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class TextsTest {
    @Test
    fun everyResultAndReasonHasWords() {
        for (r in listOf("applied", "sent_unconfirmed", "uncertain", "outcome_unknown", "sent")) {
            assertTrue(r, Texts.result(r).length > 5 && Texts.result(r) != r)
        }
        assertTrue(Texts.result("outcome_unknown").contains("Check the transcript before you send again"))
        assertTrue(Texts.result("uncertain").contains("Check the transcript"))
        for (reason in listOf("stale", "changed", "already_answered", "mac_in_use", "not_actionable", "blocked", "not_agent", "expired_request", "cursor")) {
            val t = Texts.error(ApiException(409, "x", reason, 4200))
            assertTrue(reason, t.length > 10 && !t.contains("_"))
        }
        assertEquals("Someone is using the Mac right now (idle 4 s). Try again when it is free.", Texts.error(ApiException(409, "x", "mac_in_use", 4200)))
        assertTrue(Texts.error(ApiException(429, "too many actions")).contains("Too fast"))
        assertTrue(Texts.error(ApiException(503, "hc unavailable")).contains("Herdr"))
        assertTrue(Texts.error(UnreachableException("x")).contains("not reachable"))
        assertTrue(Texts.error(RuntimeException("boom")).contains("Outcome unknown"))
    }
}

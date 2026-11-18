package dev.herdrremote

import dev.herdrremote.net.UsageResponse
import dev.herdrremote.net.UsageSource
import dev.herdrremote.net.UsageWindow
import dev.herdrremote.net.json
import dev.herdrremote.ui.USAGE_STALE_MS
import dev.herdrremote.ui.UsageStatus
import dev.herdrremote.ui.usageHeaderStatus
import dev.herdrremote.ui.usageWindowStatus
import kotlinx.serialization.decodeFromString
import org.junit.Assert.assertEquals
import org.junit.Test

class UsageStateTest {
    private val now = 1_795_000_000_000L
    private val window = UsageWindow("5h", 72.0, now + 60_000)

    @Test
    fun thresholdUsesTheActualPercent() {
        for ((percent, expected) in
            listOf(
                89.99 to UsageStatus.Normal,
                90.0 to UsageStatus.Near,
                99.99 to UsageStatus.Near,
                100.0 to UsageStatus.Reached,
            )) {
            assertEquals(expected, usageWindowStatus(window.copy(usedPercent = percent), now, now))
        }
    }

    @Test
    fun ageAndResetTakePrecedenceOverOldPercent() {
        val high = window.copy(usedPercent = 100.0)
        val boundary = now - USAGE_STALE_MS
        assertEquals(UsageStatus.Reached, usageWindowStatus(high, boundary, now))
        assertEquals(UsageStatus.Stale, usageWindowStatus(high, boundary - 1, now))
        assertEquals(UsageStatus.Reset, usageWindowStatus(high.copy(resetsAt = now), boundary, now))
    }

    @Test
    fun iconUsesCurrentWarningsAndReportsMissingSources() {
        val normal = UsageSource(now, windows = listOf(window))
        val warning = normal.copy(windows = listOf(window.copy(usedPercent = 94.0)))
        val stale = warning.copy(asOf = now - USAGE_STALE_MS - 1)
        val expired = warning.copy(windows = listOf(window.copy(resetsAt = now)))
        assertEquals(UsageStatus.Missing, usageHeaderStatus(null, now))
        assertEquals(UsageStatus.Missing, usageHeaderStatus(UsageResponse(codex = normal), now))
        assertEquals(UsageStatus.Normal, usageHeaderStatus(UsageResponse(normal, normal), now))
        assertEquals(UsageStatus.Stale, usageHeaderStatus(UsageResponse(stale, normal), now))
        assertEquals(UsageStatus.Reset, usageHeaderStatus(UsageResponse(expired, normal), now))
        assertEquals(UsageStatus.Near, usageHeaderStatus(UsageResponse(stale, warning), now))
    }

    @Test
    fun decodesTheEndpointWithANullSourceAndOneWindow() {
        val reading =
            json.decodeFromString<UsageResponse>(
                """{"claude":null,"codex":{"as_of":$now,"plan":"plus","windows":[{"name":"7d","used_percent":2.5,"resets_at":${now + 60_000}}]}}"""
            )
        assertEquals(
            UsageResponse(
                codex = UsageSource(now, "plus", listOf(UsageWindow("7d", 2.5, now + 60_000)))
            ),
            reading,
        )
    }
}

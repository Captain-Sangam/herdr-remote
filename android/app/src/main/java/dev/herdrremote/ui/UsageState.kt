package dev.herdrremote.ui

import dev.herdrremote.net.UsageResponse
import dev.herdrremote.net.UsageSource
import dev.herdrremote.net.UsageWindow

internal const val USAGE_STALE_MS = 30 * 60_000L

internal enum class UsageStatus(val mark: String, val label: String) {
    Normal("", ""),
    Near("[!]", "Near limit"),
    Reached("[x]", "Limit reached"),
    Stale("[~]", "Stale"),
    Reset("[↺]", "Reset time passed"),
    Missing("[?]", "No data"),
}

internal fun usageStale(asOf: Long, now: Long): Boolean = now - asOf > USAGE_STALE_MS

internal fun usageWindowStatus(window: UsageWindow, asOf: Long, now: Long): UsageStatus =
    when {
        window.resetsAt <= now -> UsageStatus.Reset
        usageStale(asOf, now) -> UsageStatus.Stale
        window.usedPercent >= 100 -> UsageStatus.Reached
        window.usedPercent >= 90 -> UsageStatus.Near
        else -> UsageStatus.Normal
    }

internal fun UsageResponse?.usageSources(): List<UsageSource?> =
    listOf(this?.claude, this?.codex).map { it?.takeIf { source -> source.windows.isNotEmpty() } }

/** A current warning takes precedence over an older or missing source. */
internal fun usageHeaderStatus(response: UsageResponse?, now: Long): UsageStatus {
    val statuses =
        response.usageSources().flatMap { source ->
            source?.windows?.map { usageWindowStatus(it, source.asOf, now) }
                ?: listOf(UsageStatus.Missing)
        }
    return listOf(
            UsageStatus.Reached,
            UsageStatus.Near,
            UsageStatus.Reset,
            UsageStatus.Stale,
            UsageStatus.Missing,
        )
        .firstOrNull { it in statuses } ?: UsageStatus.Normal
}

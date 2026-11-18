package dev.herdrremote.ui

import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.ExitTransition
import androidx.compose.animation.core.tween
import androidx.compose.animation.slideInHorizontally
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.sizeIn
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.paneTitle
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.stateDescription
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import dev.herdrremote.HerdrRemoteApp
import dev.herdrremote.R
import dev.herdrremote.net.UsageResponse
import dev.herdrremote.net.UsageSource
import dev.herdrremote.net.UsageWindow
import dev.herdrremote.ui.theme.KeyButton
import dev.herdrremote.ui.theme.KeyKind
import dev.herdrremote.ui.theme.Pane
import dev.herdrremote.ui.theme.PaneBox
import dev.herdrremote.ui.theme.bottomRule
import dev.herdrremote.ui.theme.topRule
import java.math.RoundingMode
import java.text.NumberFormat
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.delay

/** One in-memory reading while Spaces is open. A request can finish after the panel closes. */
internal class SpacesUsage(initialTime: Long) {
    var response by mutableStateOf<UsageResponse?>(null)
        private set

    var open by mutableStateOf(false)
        private set

    var request by mutableIntStateOf(0)
        private set

    var loading by mutableStateOf(true)
        private set

    var failed by mutableStateOf(false)
        private set

    var now by mutableLongStateOf(initialTime)

    val status: UsageStatus
        get() =
            if (response == null && loading) UsageStatus.Normal
            else usageHeaderStatus(response, now)

    fun show() {
        open = true
        refresh()
    }

    fun dismiss() {
        open = false
    }

    fun refresh() {
        request++
        loading = true
        failed = false
    }

    fun complete(id: Int, reading: UsageResponse?) {
        if (id != request) return
        if (reading != null) response = reading
        failed = reading == null
        loading = false
    }
}

@Composable
internal fun rememberSpacesUsage(app: HerdrRemoteApp): SpacesUsage {
    val usage = remember(app) { SpacesUsage(System.currentTimeMillis() + app.store.clockOffsetMs) }
    val request = usage.request
    LaunchedEffect(app, request) {
        val reading =
            try {
                app.api()?.usage()
            } catch (e: CancellationException) {
                throw e
            } catch (_: Exception) {
                null
            }
        usage.complete(request, reading)
    }
    val sources = usage.response.usageSources().filterNotNull()
    val open = usage.open
    LaunchedEffect(sources, open) {
        while (true) {
            val now = System.currentTimeMillis() + app.store.clockOffsetMs
            usage.now = now
            // Let the icon age and clear expired windows without another API request.
            val deadlines = sources.flatMap { source ->
                val staleAt = source.asOf + USAGE_STALE_MS + 1
                val nextMinute = now + (60_000 - (now - source.asOf).coerceAtLeast(0) % 60_000)
                source.windows.map { it.resetsAt } +
                    staleAt +
                    if (open) listOf(nextMinute) else emptyList()
            }
            val next = deadlines.filter { it > now }.minOrNull() ?: break
            delay(next - now)
        }
    }
    return usage
}

@Composable
private fun UsageStatus.signal(): Color =
    when (this) {
        UsageStatus.Near,
        UsageStatus.Stale,
        UsageStatus.Reset -> Pane.colors.error
        UsageStatus.Reached -> Pane.colors.blocked
        UsageStatus.Missing -> Pane.colors.muted
        UsageStatus.Normal -> Pane.colors.accent
    }

@Composable
internal fun UsageKey(usage: SpacesUsage) {
    val status = usage.status
    Row(
        Modifier.sizeIn(minWidth = Pane.space.touch, minHeight = Pane.space.touch)
            .then(if (usage.open) Modifier.background(Pane.colors.surface2) else Modifier)
            .clickable(role = Role.Button, onClick = usage::show)
            .semantics {
                contentDescription = "Usage limits"
                if (status.label.isNotEmpty()) stateDescription = status.label
            }
            .padding(horizontal = Pane.space.xs),
        horizontalArrangement = Arrangement.spacedBy(Pane.space.xs, Alignment.CenterHorizontally),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(painterResource(R.drawable.ic_usage), null, Modifier.size(18.dp), status.signal())
        if (status.mark.isNotEmpty())
            Text(
                status.mark,
                Modifier.clearAndSetSemantics {},
                style = Pane.type.caption.copy(fontWeight = FontWeight.Bold),
                color = status.signal(),
            )
    }
}

@Composable
internal fun UsageOverlay(usage: SpacesUsage) {
    BackHandler(enabled = usage.open, onBack = usage::dismiss)
    val scrimInteraction = remember { MutableInteractionSource() }
    BoxWithConstraints(Modifier.fillMaxSize().clipToBounds()) {
        if (usage.open)
            Box(
                Modifier.matchParentSize()
                    .background(Pane.terminalBg.copy(alpha = 0.23f))
                    .clickable(
                        interactionSource = scrimInteraction,
                        indication = null,
                        onClickLabel = "Close usage limits",
                        role = Role.Button,
                        onClick = usage::dismiss,
                    )
            )
        AnimatedVisibility(
            visible = usage.open,
            modifier =
                Modifier.align(Alignment.TopEnd)
                    .padding(
                        top = Pane.space.touch + Pane.space.xl + Pane.space.l,
                        bottom = Pane.space.l,
                    )
                    .width(minOf(260.dp, maxWidth * 0.79f)),
            enter = slideInHorizontally(animationSpec = tween(210)) { it },
            exit = ExitTransition.None,
        ) {
            UsagePanel(usage)
        }
    }
}

@Composable
private fun UsagePanel(usage: SpacesUsage) {
    val c = Pane.colors
    val s = Pane.space
    val response = usage.response
    val latest = response.usageSources().filterNotNull().maxOfOrNull { it.asOf }
    Column(
        Modifier.fillMaxWidth()
            .background(c.surface)
            .border(s.line, c.line)
            // Blank space in the panel owns its taps as well.
            .pointerInput(Unit) { detectTapGestures(onTap = {}) }
            .semantics { paneTitle = "Usage limits" }
    ) {
        Row(
            Modifier.fillMaxWidth().bottomRule(c.line).padding(start = s.l),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text("usage limits", Modifier.weight(1f), style = Pane.type.title, color = c.text)
            Box(
                Modifier.size(s.touch)
                    .clickable(role = Role.Button, onClick = usage::dismiss)
                    .clearAndSetSemantics {
                        contentDescription = "Close usage limits"
                    },
                contentAlignment = Alignment.Center,
            ) {
                Text("[x]", style = Pane.type.label, color = c.accent)
            }
        }
        Column(Modifier.weight(1f, fill = false).verticalScroll(rememberScrollState())) {
            if (latest != null) {
                Column(Modifier.fillMaxWidth().bottomRule(c.line).padding(s.s)) {
                    if (usageStale(latest, usage.now)) UsageFlag(UsageStatus.Stale)
                    Text(readAge(latest, usage.now), style = Pane.type.caption, color = c.muted)
                }
            }
            if (response != null || !usage.loading)
                Column(
                    Modifier.fillMaxWidth().padding(s.s),
                    verticalArrangement = Arrangement.spacedBy(s.s),
                ) {
                    UsageSourcePane("claude code", response?.claude, latest, usage.now)
                    UsageSourcePane("codex", response?.codex, latest, usage.now)
                }
            if (usage.failed)
                Text(
                    "[!] Could not read usage.",
                    Modifier.padding(horizontal = s.s, vertical = s.xs),
                    style = Pane.type.caption,
                    color = c.error,
                )
            Text(
                "Claude Code updates only during a session.",
                Modifier.fillMaxWidth().topRule(c.line).padding(s.s),
                style = Pane.type.caption,
                color = c.muted,
            )
        }
        Box(Modifier.fillMaxWidth().topRule(c.line).padding(s.s)) {
            KeyButton(
                if (usage.loading) "Loading…" else "Refresh",
                usage::refresh,
                Modifier.fillMaxWidth(),
                key = 'r',
                kind = KeyKind.Quiet,
                enabled = !usage.loading,
            )
        }
    }
}

@Composable
private fun UsageSourcePane(title: String, source: UsageSource?, latest: Long?, now: Long) {
    val c = Pane.colors
    val s = Pane.space
    PaneBox(title, behind = c.surface) {
        Column(
            Modifier.padding(horizontal = s.s, vertical = s.xs),
            verticalArrangement = Arrangement.spacedBy(s.s),
        ) {
            if (source == null || source.windows.isEmpty()) {
                UsageFlag(UsageStatus.Missing)
            } else {
                if (!source.plan.isNullOrBlank()) {
                    val plan = source.plan.replaceFirstChar { it.titlecase(Locale.ROOT) }
                    Text("$plan plan", style = Pane.type.caption, color = c.muted)
                }
                if (source.asOf != latest) {
                    Column {
                        if (usageStale(source.asOf, now)) UsageFlag(UsageStatus.Stale)
                        Text(readAge(source.asOf, now), style = Pane.type.caption, color = c.muted)
                    }
                }
                source.windows.forEach { UsageMeter(it, source.asOf, now) }
            }
        }
    }
}

@Composable
private fun UsageMeter(window: UsageWindow, asOf: Long, now: Long) {
    val c = Pane.colors
    val s = Pane.space
    val status = usageWindowStatus(window, asOf, now)
    Column(verticalArrangement = Arrangement.spacedBy(s.xs)) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(s.xs),
            verticalAlignment = Alignment.Top,
        ) {
            Text(
                displayName(window.name),
                Modifier.weight(1f),
                style = Pane.type.label,
                color = c.text,
            )
            if (status == UsageStatus.Reset) {
                Text("Reset", style = Pane.type.label, color = c.muted)
            } else {
                val percent = formatPercent(window.usedPercent)
                val previous = if (status == UsageStatus.Stale) "Last read " else ""
                Text(
                    "$previous${percent}% used",
                    style = Pane.type.label.copy(fontWeight = FontWeight.Bold),
                    color = if (status == UsageStatus.Stale) c.muted else c.text,
                )
            }
        }
        if (status != UsageStatus.Reset) {
            val fill =
                when (status) {
                    UsageStatus.Stale -> c.working.copy(alpha = 0.45f)
                    UsageStatus.Normal -> c.working
                    else -> status.signal()
                }
            Box(Modifier.fillMaxWidth().height(s.xs).background(c.surface2)) {
                Box(
                    Modifier.fillMaxWidth((window.usedPercent / 100).coerceIn(0.0, 1.0).toFloat())
                        .height(s.xs)
                        .background(fill)
                )
            }
        }
        if (status in listOf(UsageStatus.Near, UsageStatus.Reached, UsageStatus.Reset))
            UsageFlag(status)
        val verb =
            if (status == UsageStatus.Reset || status == UsageStatus.Stale) "Scheduled reset ·"
            else "Resets"
        Text("$verb ${resetTime(window.resetsAt, now)}", style = Pane.type.caption, color = c.muted)
    }
}

@Composable
private fun UsageFlag(status: UsageStatus) {
    Row(horizontalArrangement = Arrangement.spacedBy(Pane.space.xs)) {
        val style = Pane.type.caption.copy(fontWeight = FontWeight.Bold)
        Text(status.mark, Modifier.clearAndSetSemantics {}, style = style, color = status.signal())
        Text(status.label, style = style, color = Pane.colors.text)
    }
}

private val clockFormat = DateTimeFormatter.ofPattern("HH:mm", Locale.ROOT)
private val dayFormat = DateTimeFormatter.ofPattern("d MMM", Locale.ENGLISH)

private fun readAge(asOf: Long, now: Long): String {
    val time = Instant.ofEpochMilli(asOf).atZone(ZoneId.systemDefault()).format(clockFormat)
    val minutes = (now - asOf).coerceAtLeast(0) / 60_000
    return "Mac read $time · $minutes min ago"
}

private fun resetTime(resetsAt: Long, now: Long): String {
    val zone = ZoneId.systemDefault()
    val reset = Instant.ofEpochMilli(resetsAt).atZone(zone)
    val today = Instant.ofEpochMilli(now).atZone(zone).toLocalDate()
    val day = if (reset.toLocalDate() == today) "Today" else reset.format(dayFormat)
    return "$day ${reset.format(clockFormat)}"
}

private fun displayName(name: String): String =
    if (name.matches(Regex("[0-9]+[mhd]"))) "${name.dropLast(1)} ${name.last()}" else name

private fun formatPercent(percent: Double): String =
    NumberFormat.getNumberInstance()
        .apply {
            maximumFractionDigits = 1
            roundingMode = RoundingMode.DOWN
            isGroupingUsed = false
        }
        .format(percent)

package dev.herdrremote.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import dev.herdrremote.HerdrRemoteApp
import dev.herdrremote.net.Agent
import dev.herdrremote.net.Space
import dev.herdrremote.ui.theme.NoteKind
import dev.herdrremote.ui.theme.Pane
import dev.herdrremote.ui.theme.PaneBanner
import dev.herdrremote.ui.theme.PaneBox
import dev.herdrremote.ui.theme.PaneHeader
import dev.herdrremote.ui.theme.PaneNote
import dev.herdrremote.ui.theme.PaneSubline
import dev.herdrremote.ui.theme.StatusMeter
import dev.herdrremote.ui.theme.StatusRow
import dev.herdrremote.ui.theme.StatusTag
import dev.herdrremote.watch.AgentsState
import kotlinx.coroutines.delay

val urgency = listOf("blocked", "error", "done", "working", "idle")

/** The place of a state in [urgency]; an unknown state sorts last. */
fun rank(state: String) = urgency.indexOf(state).let { if (it < 0) 99 else it }

fun mostUrgent(agents: List<Agent>): String = agents.map { it.state }.minByOrNull(::rank) ?: ""

/** "4 agents · 1 blocked · 1 error". */
fun counts(agents: List<Agent>): String {
    val blocked = agents.count { it.blocked }
    val errors = agents.count { it.state == "error" }
    return "${agents.size} agent${if (agents.size == 1) "" else "s"}" +
        (if (blocked > 0) " · $blocked blocked" else "") +
        (if (errors > 0) " · $errors error" else "")
}

/** Old data stays readable but dimmed while the events stream is down. */
fun Modifier.stale(state: AgentsState) = alpha(if (state.connected) 1f else 0.45f)

/**
 * "Last update 12 s ago" while the events stream is down (failure case: hc unavailable, endpoint
 * unreachable).
 */
@Composable
fun StaleBanner(state: AgentsState) {
    var now by remember { mutableStateOf(System.currentTimeMillis()) }
    LaunchedEffect(state.connected) {
        while (!state.connected) {
            now = System.currentTimeMillis()
            delay(1000)
        }
    }
    if (!state.connected) {
        val age =
            if (state.updatedAt == 0L) "no data yet"
            else "last update ${(now - state.updatedAt) / 1000} s ago"
        PaneBanner("Not connected to the Mac: $age")
    }
}

/** The first load: a short terminal log of the endpoints the phone tries, not a spinner. */
@Composable
fun ConnectLog(endpoints: List<String>, error: String? = null) {
    val c = Pane.colors
    Column(
        Modifier.padding(horizontal = Pane.space.l, vertical = 14.dp),
        verticalArrangement = Arrangement.spacedBy(Pane.space.xs),
    ) {
        Text(
            "$ herdr-remote connect",
            style = Pane.type.label.copy(fontWeight = FontWeight.Bold),
            color = c.text,
        )
        endpoints.forEach { Text("› trying $it …", style = Pane.type.label, color = c.muted) }
        if (error != null) Text("› $error", style = Pane.type.label, color = c.blocked)
        Box(Modifier.size(8.dp, 15.dp).background(c.text))
    }
}

/** A pane of rows, 8 dp in from the screen edges. */
@Composable
fun ListPane(
    title: String,
    color: Color = Pane.colors.line,
    content: @Composable ColumnScope.() -> Unit,
) = PaneBox(title, Modifier.padding(horizontal = Pane.space.s), color = color, content = content)

/** An agent row: name, "space · app · kind", and the prompt or the task title. */
@Composable
fun AgentRow(a: Agent, withSpace: Boolean, tag: Boolean = true, onClick: () -> Unit) {
    StatusRow(
        state = a.state,
        title = a.label,
        meta =
            listOfNotNull(if (withSpace) a.space else null, a.app, a.kind)
                .filter { it.isNotEmpty() }
                .joinToString(" · "),
        onClick = onClick,
        detail = (if (a.blocked) a.msg else a.title)?.ifEmpty { null },
        trailing = { if (tag) StatusTag(a.state) },
    )
}

@Composable
fun SpacesScreen(app: HerdrRemoteApp, onSpace: (Space) -> Unit, onAgent: (String) -> Unit) {
    val state by app.repository.state.collectAsStateWithLifecycle()
    var spaces by remember { mutableStateOf<List<Space>>(emptyList()) }
    var error by remember { mutableStateOf<String?>(null) }
    // /v1/spaces is fetched on each visit and when the agent set changes (a rename makes no event).
    LaunchedEffect(state.agents.keys) {
        runCatching { app.api()?.spaces() }
            .onSuccess {
                if (it != null) {
                    spaces = it
                    error = null
                }
            }
            .onFailure { error = it.message }
    }
    val loading = spaces.isEmpty() && state.updatedAt == 0L
    Column(Modifier.fillMaxSize()) {
        PaneHeader("~/spaces", online = state.connected)
        PaneSubline {
            Text(if (loading) "Connecting to the Mac…" else counts(state.agents.values.toList()))
        }
        if (loading) {
            ConnectLog(app.store.endpoints, error?.let { "Could not load spaces: $it" })
            return
        }
        StaleBanner(state)
        if (error != null && spaces.isEmpty())
            PaneNote("Could not load spaces: $error", NoteKind.Error)
        Column(
            Modifier.weight(1f)
                .verticalScroll(rememberScrollState())
                .stale(state)
                .padding(bottom = Pane.space.s),
            verticalArrangement = Arrangement.spacedBy(Pane.space.s),
        ) {
            val blocked = state.sorted.filter { it.blocked }
            if (blocked.isNotEmpty())
                ListPane("needs you (${blocked.size})", Pane.colors.blocked) {
                    Text(
                        buildAnnotatedString {
                            withStyle(
                                SpanStyle(color = Pane.colors.blocked, fontWeight = FontWeight.Bold)
                            ) {
                                append("${blocked.size}")
                            }
                            append(
                                if (blocked.size == 1) " agent needs you" else " agents need you"
                            )
                        },
                        Modifier.padding(horizontal = Pane.space.m).padding(bottom = Pane.space.xs),
                        style = Pane.type.label,
                        color = Pane.colors.muted,
                    )
                    blocked.forEach {
                        AgentRow(it, withSpace = true, tag = false) { onAgent(it.address) }
                    }
                }
            ListPane("${spaces.size} space${if (spaces.size == 1) "" else "s"}") {
                spaces.forEach { sp ->
                    val live = sp.agents.map { state.agents[it.address] ?: it }
                    StatusRow(
                        state = mostUrgent(live),
                        title = sp.name,
                        meta = counts(live),
                        onClick = { onSpace(sp) },
                        trailing = { StatusMeter(live.map { it.state }) },
                    )
                }
            }
        }
    }
}

@Composable
fun SpaceScreen(app: HerdrRemoteApp, space: Space, onAgent: (String) -> Unit, onBack: () -> Unit) {
    val state by app.repository.state.collectAsStateWithLifecycle()
    val agents =
        space.agents
            .map { state.agents[it.address] ?: it }
            .filter { state.agents.isEmpty() || it.address in state.agents }
    Column(Modifier.fillMaxSize()) {
        PaneHeader("~/spaces/${space.name}", onBack = onBack, online = state.connected)
        PaneSubline { Text(counts(agents)) }
        StaleBanner(state)
        Column(
            Modifier.weight(1f)
                .verticalScroll(rememberScrollState())
                .stale(state)
                .padding(bottom = Pane.space.s)
        ) {
            ListPane("agents") {
                agents.forEach { AgentRow(it, withSpace = false) { onAgent(it.address) } }
            }
        }
    }
}

@Composable
fun AllAgentsScreen(app: HerdrRemoteApp, onAgent: (String) -> Unit) {
    val state by app.repository.state.collectAsStateWithLifecycle()
    val loading = state.agents.isEmpty() && state.updatedAt == 0L
    val empty = state.agents.isEmpty() && !loading
    Column(Modifier.fillMaxSize()) {
        PaneHeader("~/agents", online = state.connected)
        PaneSubline {
            Text(
                when {
                    loading -> "Connecting to the Mac…"
                    empty -> "Nothing runs on the Mac"
                    else -> "${state.agents.size} agents · blocked first"
                }
            )
        }
        if (loading) {
            ConnectLog(app.store.endpoints)
            return
        }
        StaleBanner(state)
        if (empty) {
            EmptyAgents()
            return
        }
        // Blocked first, then by urgency; the stable sort keeps the space and tab order inside
        // each.
        val (blocked, others) = state.sorted.sortedBy { rank(it.state) }.partition { it.blocked }
        Column(
            Modifier.weight(1f)
                .verticalScroll(rememberScrollState())
                .stale(state)
                .padding(bottom = Pane.space.s),
            verticalArrangement = Arrangement.spacedBy(Pane.space.s),
        ) {
            if (blocked.isNotEmpty())
                ListPane("needs you · ${blocked.size}") {
                    blocked.forEach { AgentRow(it, withSpace = true) { onAgent(it.address) } }
                }
            if (others.isNotEmpty())
                ListPane("others · ${others.size}") {
                    others.forEach { AgentRow(it, withSpace = true) { onAgent(it.address) } }
                }
        }
    }
}

@Composable
fun EmptyAgents() {
    val c = Pane.colors
    Column(
        Modifier.fillMaxWidth().padding(horizontal = 32.dp, vertical = 56.dp),
        verticalArrangement = Arrangement.spacedBy(Pane.space.l),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            "┌─ hc agents ─────┐\n│                 │\n│   (0 agents)    │\n│                 │\n└─────────────────┘",
            style = Pane.type.label,
            color = c.muted,
            softWrap = false,
        )
        Text("No agents", style = Pane.type.head, color = c.text)
        Text(
            "Start an agent in Herdr on the Mac. It shows here within 2 s.",
            style = Pane.type.body,
            color = c.muted,
            textAlign = TextAlign.Center,
        )
    }
}

package dev.herdrremote.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import dev.herdrremote.HerdrRemoteApp
import dev.herdrremote.net.Agent
import dev.herdrremote.net.Space
import dev.herdrremote.watch.AgentsState
import kotlinx.coroutines.delay

fun statusColor(state: String): Color = when (state) {
    "blocked" -> Color(0xFFD94C4C)
    "working" -> Color(0xFF3B82F6)
    "done" -> Color(0xFF22A06B)
    "error" -> Color(0xFFB45309)
    else -> Color(0xFF8A8F98)
}

val urgency = listOf("blocked", "error", "done", "working", "idle")

fun mostUrgent(agents: List<Agent>): String = agents.map { it.state }.minByOrNull { urgency.indexOf(it).let { i -> if (i < 0) 99 else i } } ?: ""

/** "Last update 12 s ago" while the events stream is down (failure case: hc unavailable, endpoint unreachable). */
@Composable
fun StaleBanner(state: AgentsState) {
    var now by remember { mutableStateOf(System.currentTimeMillis()) }
    LaunchedEffect(state.connected) { while (!state.connected) { now = System.currentTimeMillis(); delay(1000) } }
    if (!state.connected) {
        val age = if (state.updatedAt == 0L) "no data yet" else "last update ${(now - state.updatedAt) / 1000} s ago"
        Surface(color = MaterialTheme.colorScheme.errorContainer, modifier = Modifier.fillMaxWidth()) {
            Text("Not connected to the Mac: $age", Modifier.padding(12.dp), color = MaterialTheme.colorScheme.onErrorContainer)
        }
    }
}

@Composable
fun AgentRow(a: Agent, onClick: () -> Unit) {
    Row(Modifier.fillMaxWidth().clickable(onClick = onClick).padding(horizontal = 16.dp, vertical = 12.dp), verticalAlignment = Alignment.CenterVertically) {
        Surface(color = statusColor(a.state), shape = MaterialTheme.shapes.small, modifier = Modifier.padding(end = 12.dp)) { Text("  ", Modifier.padding(4.dp)) }
        Column(Modifier.weight(1f)) {
            Text(a.label, style = MaterialTheme.typography.titleMedium, maxLines = 1)
            val detail = listOf(a.space, a.app, a.state + (a.kind?.let { " · $it" } ?: "")).filter { it.isNotEmpty() }.joinToString(" · ")
            Text(detail, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1)
            if (a.blocked && !a.msg.isNullOrEmpty()) Text(a.msg, style = MaterialTheme.typography.bodySmall, maxLines = 2)
        }
    }
    HorizontalDivider()
}

@Composable
fun SpacesScreen(app: HerdrRemoteApp, onSpace: (Space) -> Unit, onAllAgents: () -> Unit, onSettings: () -> Unit) {
    val state by app.repository.state.collectAsStateWithLifecycle()
    var spaces by remember { mutableStateOf<List<Space>>(emptyList()) }
    var error by remember { mutableStateOf<String?>(null) }
    // /v1/spaces is fetched on each visit and when the agent set changes (a rename makes no event).
    LaunchedEffect(state.agents.keys) {
        runCatching { app.api()?.spaces() }.onSuccess { if (it != null) { spaces = it; error = null } }.onFailure { error = it.message }
    }
    Column(Modifier.fillMaxSize()) {
        Header("Spaces", action = { TextButton(onClick = onSettings) { Text("Settings") } })
        StaleBanner(state)
        if (error != null && spaces.isEmpty()) Text("Could not load spaces: $error", Modifier.padding(16.dp), color = MaterialTheme.colorScheme.error)
        LazyColumn(Modifier.weight(1f)) {
            items(spaces, key = { it.id }) { sp ->
                val live = sp.agents.map { state.agents[it.address] ?: it }
                Row(Modifier.fillMaxWidth().clickable { onSpace(sp) }.padding(16.dp), verticalAlignment = Alignment.CenterVertically) {
                    Surface(color = statusColor(mostUrgent(live)), shape = MaterialTheme.shapes.small, modifier = Modifier.padding(end = 12.dp)) { Text("  ", Modifier.padding(4.dp)) }
                    Column(Modifier.weight(1f)) {
                        Text(sp.name, style = MaterialTheme.typography.titleMedium)
                        val blocked = live.count { it.blocked }
                        Text("${live.size} agent${if (live.size == 1) "" else "s"}" + if (blocked > 0) " · $blocked blocked" else "", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
                HorizontalDivider()
            }
        }
        TextButton(onClick = onAllAgents, Modifier.padding(8.dp)) { Text("All agents (${state.agents.size})") }
    }
}

@Composable
fun SpaceScreen(app: HerdrRemoteApp, space: Space, onAgent: (String) -> Unit, onBack: () -> Unit) {
    val state by app.repository.state.collectAsStateWithLifecycle()
    val agents = space.agents.map { state.agents[it.address] ?: it }.filter { state.agents.isEmpty() || it.address in state.agents }
    Column(Modifier.fillMaxSize()) {
        Header(space.name, onBack = onBack)
        StaleBanner(state)
        LazyColumn { items(agents, key = { it.address }) { AgentRow(it) { onAgent(it.address) } } }
    }
}

@Composable
fun AllAgentsScreen(app: HerdrRemoteApp, onAgent: (String) -> Unit, onBack: () -> Unit) {
    val state by app.repository.state.collectAsStateWithLifecycle()
    Column(Modifier.fillMaxSize()) {
        Header("All agents", onBack = onBack)
        StaleBanner(state)
        if (state.agents.isEmpty()) Text("No agents", Modifier.padding(16.dp), color = MaterialTheme.colorScheme.onSurfaceVariant)
        LazyColumn { items(state.sorted, key = { it.address }) { AgentRow(it) { onAgent(it.address) } } }
    }
}

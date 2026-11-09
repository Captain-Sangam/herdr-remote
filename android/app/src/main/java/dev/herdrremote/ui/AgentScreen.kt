package dev.herdrremote.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import dev.herdrremote.HerdrRemoteApp
import dev.herdrremote.net.TranscriptItem

@Composable
fun AgentScreen(app: HerdrRemoteApp, address: String, onBack: () -> Unit) {
    val vm: AgentViewModel = viewModel(key = address) { AgentViewModel(app, address) }
    val agent by vm.agent.collectAsStateWithLifecycle()
    val prompt by vm.prompt.collectAsStateWithLifecycle()
    val note by vm.note.collectAsStateWithLifecycle()
    val busy by vm.busy.collectAsStateWithLifecycle()
    val state by app.repository.state.collectAsStateWithLifecycle()
    var tab by remember { mutableIntStateOf(0) }
    var draft by remember { mutableStateOf("") }

    Column(Modifier.fillMaxSize()) {
        Header(agent?.label ?: address, onBack = onBack)
        StaleBanner(state)
        val a = agent
        if (a == null) {
            Text("This agent is gone.", Modifier.padding(16.dp), color = MaterialTheme.colorScheme.onSurfaceVariant)
            return
        }
        Text(listOf(a.space, a.app, a.state + (a.kind?.let { " · $it" } ?: "")).filter { it.isNotEmpty() }.joinToString(" · "), Modifier.padding(horizontal = 16.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)

        if (a.blocked) {
            Card(Modifier.fillMaxWidth().padding(12.dp)) {
                Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(a.msg?.ifEmpty { null } ?: "Waiting for a decision (${a.kind ?: "blocked"})", style = MaterialTheme.typography.titleSmall)
                    val p = prompt
                    if (p != null && p.block.isNotEmpty()) {
                        // At most a third of the screen, so Approve and Deny stay in reach under a long command.
                        val maxBlock = (LocalConfiguration.current.screenHeightDp / 3).dp
                        Text(p.block, Modifier.heightIn(max = maxBlock).verticalScroll(rememberScrollState()).horizontalScroll(rememberScrollState()), fontFamily = FontFamily.Monospace, style = MaterialTheme.typography.bodySmall)
                    }
                    if (p?.actionable == true) {
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            Button(onClick = { vm.decide(true) }, enabled = !busy) { Text("Approve") }
                            OutlinedButton(onClick = { vm.decide(false) }, enabled = !busy) { Text("Deny") }
                            TextButton(onClick = { vm.refreshPrompt() }) { Text("Reload") }
                        }
                    } else {
                        Text(if (p == null) "Loading the prompt…" else "Answer this on the Mac.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            }
        }
        if (note != null) {
            Text(note!!, Modifier.padding(horizontal = 16.dp, vertical = 4.dp), color = MaterialTheme.colorScheme.primary)
        }

        TabRow(selectedTabIndex = tab) {
            Tab(selected = tab == 0, onClick = { tab = 0 }, text = { Text("Transcript") })
            Tab(selected = tab == 1, onClick = { tab = 1 }, text = { Text("Live") })
        }
        if (tab == 0) {
            val tr by vm.transcript.collectAsStateWithLifecycle()
            LazyColumn(Modifier.weight(1f).fillMaxWidth()) {
                if (tr.nextCursor.isNotEmpty()) {
                    item { TextButton(onClick = { vm.loadOlder() }, enabled = !tr.loading) { Text(if (tr.loading) "Loading…" else "Load earlier") } }
                }
                if (tr.source == "terminal") {
                    item { Text("terminal text" + (tr.hint?.let { " · $it" } ?: ""), Modifier.padding(horizontal = 16.dp, vertical = 4.dp), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant) }
                }
                if (tr.error != null) item { Text(tr.error!!, Modifier.padding(16.dp), color = MaterialTheme.colorScheme.error) }
                itemsIndexed(tr.items) { _, it -> TranscriptRow(it) }
            }
        } else {
            LiveView(app, address, Modifier.weight(1f).fillMaxWidth().background(androidx.compose.ui.graphics.Color.Black))
        }

        Row(Modifier.fillMaxWidth().padding(8.dp), verticalAlignment = androidx.compose.ui.Alignment.CenterVertically) {
            OutlinedTextField(value = draft, onValueChange = { draft = it.take(4000) }, modifier = Modifier.weight(1f), placeholder = { Text(if (a.blocked) "Answer the prompt first" else "Message the agent") }, maxLines = 4, enabled = !a.blocked)
            Button(onClick = { vm.send(draft) { draft = "" } }, enabled = !busy && !a.blocked && draft.isNotBlank(), modifier = Modifier.padding(start = 8.dp)) { Text("Send") }
        }
    }
}

@Composable
fun TranscriptRow(it: TranscriptItem) {
    val label = when (it.kind) { "tool_call" -> "${it.role} · tool call"; "tool_result" -> "tool result"; else -> it.role }
    Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 6.dp)) {
        Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(it.text, fontFamily = if (it.kind == "text" && it.role != "terminal") FontFamily.Default else FontFamily.Monospace, style = MaterialTheme.typography.bodyMedium)
    }
}

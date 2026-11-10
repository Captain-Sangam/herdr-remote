package dev.herdrremote.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.repeatOnLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import dev.herdrremote.HerdrRemoteApp
import dev.herdrremote.net.Agent
import dev.herdrremote.net.PromptView
import dev.herdrremote.net.TranscriptItem
import dev.herdrremote.ui.theme.CodeBlock
import dev.herdrremote.ui.theme.KeyButton
import dev.herdrremote.ui.theme.KeyKind
import dev.herdrremote.ui.theme.NoteKind
import dev.herdrremote.ui.theme.Pane
import dev.herdrremote.ui.theme.PaneBox
import dev.herdrremote.ui.theme.PaneHeader
import dev.herdrremote.ui.theme.PaneNote
import dev.herdrremote.ui.theme.PaneSubline
import dev.herdrremote.ui.theme.PaneTabs
import dev.herdrremote.ui.theme.PromptField
import dev.herdrremote.ui.theme.StatusTag
import dev.herdrremote.ui.theme.topRule
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

@Composable
fun AgentScreen(app: HerdrRemoteApp, address: String, onBack: () -> Unit) {
    val vm: AgentViewModel = viewModel(key = address) { AgentViewModel(app, address) }
    val agent by vm.agent.collectAsStateWithLifecycle()
    val prompt by vm.prompt.collectAsStateWithLifecycle()
    val note by vm.note.collectAsStateWithLifecycle()
    val busy by vm.busy.collectAsStateWithLifecycle()
    val model = vm.transcript.collectAsStateWithLifecycle().value.model
    val state by app.repository.state.collectAsStateWithLifecycle()
    var tab by remember { mutableIntStateOf(0) }
    var draft by remember { mutableStateOf("") }

    Column(Modifier.fillMaxSize()) {
        val a = agent
        PaneHeader(
            listOf("~", a?.space.orEmpty(), a?.label ?: address)
                .filter { it.isNotEmpty() }
                .joinToString("/"),
            onBack = onBack,
        )
        if (a != null)
            PaneSubline {
                StatusTag(a.state)
                Text(
                    listOf(a.app, model, a.space, a.kind.orEmpty())
                        .filter { it.isNotEmpty() }
                        .joinToString(" · "),
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        StaleBanner(state)
        if (a == null) {
            Text(
                "This agent is gone.",
                Modifier.padding(Pane.space.l),
                style = Pane.type.body,
                color = Pane.colors.muted,
            )
            return
        }

        val n = note
        val kind = n?.let(Texts::kind)
        // A refusal shows in the approval pane, above the keys; every other note under it.
        val refusal =
            if (a.blocked && prompt?.actionable == true && kind == NoteKind.Error) n else null
        if (a.blocked)
            Approval(
                a,
                prompt,
                busy,
                refusal,
                onDecide = { vm.decide(it) },
                onReload = { vm.refreshPrompt() },
            )
        if (n != null && kind != null && refusal == null)
            PaneNote(n, kind, Modifier.padding(Pane.space.s))

        PaneTabs(listOf("transcript", "live"), tab) { tab = it }
        if (tab == 0) Transcript(vm, a.app, a.state == "working", Modifier.weight(1f))
        else
            Column(Modifier.weight(1f).fillMaxWidth().background(Pane.terminalBg)) {
                LiveView(app, address, Modifier.weight(1f).fillMaxWidth())
                Text(
                    listOfNotNull(
                            "Live",
                            if (a.cols > 0 && a.rows > 0) "${a.cols} × ${a.rows}" else null,
                            "pinch to zoom",
                        )
                        .joinToString(" · "),
                    Modifier.fillMaxWidth()
                        .topRule(Pane.terminalFg.copy(alpha = 0.1f))
                        .padding(horizontal = Pane.space.l, vertical = Pane.space.s),
                    style = Pane.type.caption,
                    color = Pane.terminalFg.copy(alpha = 0.75f),
                )
            }

        PromptField(
            value = draft,
            onValueChange = { draft = it },
            onSend = { if (!busy) vm.send(draft) { draft = "" } },
            placeholder = if (a.blocked) "Answer the prompt first" else "Message the agent",
            enabled = !a.blocked,
        )
    }
}

/**
 * The prompt of a blocked agent: a red "permission" pane with the command and the keys, or a blue
 * "question" pane when only the Mac can answer it.
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun Approval(
    a: Agent,
    p: PromptView?,
    busy: Boolean,
    refusal: String?,
    onDecide: (Boolean) -> Unit,
    onReload: () -> Unit,
) {
    val c = Pane.colors
    val ask = p != null && !p.actionable
    val color = if (ask) c.working else c.blocked
    PaneBox(
        if (ask) "question" else "permission",
        Modifier.padding(horizontal = Pane.space.s, vertical = Pane.space.xs),
        color = color,
    ) {
        Column(
            Modifier.padding(horizontal = Pane.space.m, vertical = Pane.space.xs),
            verticalArrangement = Arrangement.spacedBy(Pane.space.s),
        ) {
            Column {
                Text(
                    (if (ask) "? " else "! ") +
                        (a.msg?.ifEmpty { null }
                            ?: "Waiting for a decision (${a.kind ?: "blocked"})"),
                    style = Pane.type.body.copy(fontWeight = FontWeight.Bold),
                    color = color,
                )
                if (p != null)
                    Text(
                        listOfNotNull(
                                p.kind ?: a.kind,
                                "turn ${p.turn}",
                                if (ask) "view only" else null,
                            )
                            .joinToString(" · "),
                        style = Pane.type.caption,
                        color = c.muted,
                    )
            }
            if (p == null) {
                Text("Loading the prompt…", style = Pane.type.label, color = c.muted)
                return@Column
            }
            // At most a third of the screen, so Approve and Deny stay in reach under a long
            // command.
            if (p.block.isNotEmpty())
                CodeBlock(p.block, maxHeight = (LocalConfiguration.current.screenHeightDp / 3).dp)
            if (ask) {
                Text(
                    "» Answer this on the Mac.",
                    style = Pane.type.label.copy(fontWeight = FontWeight.Bold),
                    color = c.error,
                )
                return@Column
            }
            if (refusal != null) PaneNote(refusal, NoteKind.Error)
            // Reload moves to a second line when the three keys do not fit (large font scale).
            FlowRow(
                horizontalArrangement = Arrangement.spacedBy(6.dp),
                verticalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                KeyButton(
                    "Approve",
                    { onDecide(true) },
                    Modifier.weight(1f),
                    key = 'y',
                    kind = KeyKind.Go,
                    enabled = !busy,
                )
                KeyButton(
                    "Deny",
                    { onDecide(false) },
                    Modifier.weight(1f),
                    key = 'n',
                    kind = KeyKind.Stop,
                    enabled = !busy,
                )
                KeyButton("Reload", onReload, key = 'r', kind = KeyKind.Quiet)
            }
        }
    }
}

@Composable
fun Transcript(vm: AgentViewModel, app: String, working: Boolean, modifier: Modifier = Modifier) {
    val tr by vm.transcript.collectAsStateWithLifecycle()
    val c = Pane.colors
    val list = rememberLazyListState()
    // Live: the newest page every 2 s while the agent works and the screen is in view.
    // ponytail: polls the whole last page (about 60 KiB); add an "after" cursor if that costs too
    // much.
    val owner = LocalLifecycleOwner.current
    LaunchedEffect(working) {
        if (working)
            owner.repeatOnLifecycle(Lifecycle.State.STARTED) {
                while (true) {
                    delay(2_000)
                    vm.refresh()
                }
            }
    }
    // The end marker's index: the head items before tr.items, plus tr.items.
    val end =
        tr.items.size +
            listOf(tr.nextCursor.isNotEmpty(), tr.source == "terminal", tr.error != null).count {
                it
            }
    // Like a terminal: new items scroll into view while the end is in view. Read before the new
    // items are laid out, so it is the old layout's answer.
    val atEnd = remember(tr.items) { !list.canScrollForward }
    LaunchedEffect(tr.items) { if (atEnd) list.scrollToItem(end) }
    val scope = rememberCoroutineScope()

    Box(modifier.fillMaxWidth()) {
        LazyColumn(
            Modifier.fillMaxSize(),
            state = list,
            contentPadding = PaddingValues(horizontal = Pane.space.m, vertical = Pane.space.s),
            verticalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            if (tr.nextCursor.isNotEmpty())
                item {
                    Box(
                        Modifier.heightIn(min = Pane.space.touch).clickable(
                            enabled = !tr.loading,
                            role = Role.Button,
                        ) {
                            vm.loadOlder()
                        },
                        contentAlignment = Alignment.CenterStart,
                    ) {
                        Text(
                            if (tr.loading) "Loading…" else "Load earlier",
                            style = Pane.type.label,
                            color = c.muted,
                        )
                    }
                }
            if (tr.source == "terminal")
                item {
                    Text(
                        "Terminal text" + (tr.hint?.let { " · $it" } ?: ""),
                        style = Pane.type.caption,
                        color = c.error,
                    )
                }
            tr.error?.let { item { PaneNote(it, NoteKind.Error) } }
            items(tr.items) { TranscriptRow(it, app) }
            item { Spacer(Modifier.height(1.dp)) }
        }
        if (list.canScrollForward)
            KeyButton(
                "Bottom ↓",
                { scope.launch { list.animateScrollToItem(end) } },
                Modifier.align(Alignment.BottomEnd).padding(Pane.space.m).background(c.bg),
            )
    }
}

/** One transcript item: "you ›", "claude ›", "tool $" or "⎿" in a left column, then the text. */
@Composable
fun TranscriptRow(it: TranscriptItem, app: String) {
    val c = Pane.colors
    val (label, labelColor, textColor) =
        when (if (it.kind == "text") it.role else it.kind) {
            "terminal" -> Triple(null, c.text, c.text)
            "user" -> Triple("you ›", c.accent, c.text)
            "tool_call" -> Triple("tool $", c.working, c.working)
            "tool_result" -> Triple("  ⎿", c.muted, c.muted)
            else -> Triple("${app.ifEmpty { it.role }} ›", c.text, c.text)
        }
    Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        if (label != null)
            Text(
                label,
                // Nine cells at the label size; sp, so it grows with the font scale.
                Modifier.width(with(LocalDensity.current) { 66.sp.toDp() }),
                style = Pane.type.label.copy(fontWeight = FontWeight.Bold),
                color = labelColor,
                maxLines = 1,
            )
        Text(it.text, style = Pane.type.label, color = textColor)
    }
}

package dev.herdrremote.ui

import android.content.ClipDescription
import android.os.PersistableBundle
import androidx.compose.animation.core.animate
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.FlingBehavior
import androidx.compose.foundation.gestures.ScrollScope
import androidx.compose.foundation.gestures.ScrollableDefaults
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
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.MutableState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.input.nestedscroll.NestedScrollConnection
import androidx.compose.ui.input.nestedscroll.NestedScrollSource
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.platform.ClipEntry
import androidx.compose.ui.platform.Clipboard
import androidx.compose.ui.platform.LocalClipboard
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.text.withStyle
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
import dev.herdrremote.ui.theme.spinnerFrame
import dev.herdrremote.ui.theme.topRule
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.drop

@Composable
fun AgentScreen(app: HerdrRemoteApp, address: String, onBack: () -> Unit) {
    val vm: AgentViewModel = viewModel(key = address) { AgentViewModel(app, address) }
    val agent by vm.agent.collectAsStateWithLifecycle()
    val prompt by vm.prompt.collectAsStateWithLifecycle()
    val note by vm.note.collectAsStateWithLifecycle()
    val busy by vm.busy.collectAsStateWithLifecycle()
    val skills by vm.skills.collectAsStateWithLifecycle()
    val model = vm.transcript.collectAsStateWithLifecycle().value.model
    val state by app.repository.state.collectAsStateWithLifecycle()
    var tab by remember { mutableIntStateOf(0) }
    var draft by remember { mutableStateOf("") }
    val follow = remember { mutableStateOf(true) }

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
                onAnswer = { vm.answer(it) },
                onReload = { vm.refreshPrompt() },
            )
        if (n != null && kind != null && refusal == null)
            PaneNote(n, kind, Modifier.padding(Pane.space.s))

        PaneTabs(listOf("transcript", "live"), tab) { tab = it }
        if (tab == 0)
            SensitiveClipboard {
                Transcript(vm, a.app, a.state == "working", follow, Modifier.weight(1f))
            }
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

        // A draft that starts with "/" lists the agent's skills that match, like Claude Code.
        val query = draft.takeIf { it.startsWith("/") && ' ' !in it }?.drop(1)
        if (query != null && !a.blocked) {
            LaunchedEffect(Unit) { vm.loadSkills() }
            SkillList(skills, query) { draft = "/$it " }
        }
        PromptField(
            value = draft,
            onValueChange = { draft = it },
            onSkills = { if (draft.isEmpty()) draft = "/" },
            onSend = {
                if (!busy) {
                    follow.value = true
                    vm.send(draft) { draft = "" }
                }
            },
            placeholder = if (a.blocked) "Answer the prompt first" else "Message the agent",
            enabled = !a.blocked,
        )
    }
}

/**
 * The prompt of a blocked agent: a red "permission" pane with the command and the keys, or a blue
 * "question" pane with a key for each option. A prompt that only the Mac can answer shows no keys.
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun Approval(
    a: Agent,
    p: PromptView?,
    busy: Boolean,
    refusal: String?,
    onDecide: (Boolean) -> Unit,
    onAnswer: (String) -> Unit,
    onReload: () -> Unit,
) {
    val c = Pane.colors
    val ask = p != null && !p.actionable
    val question = if (p != null) !p.approvable else a.kind == "question"
    val color = if (question) c.working else c.blocked
    PaneBox(
        if (question) "question" else "permission",
        Modifier.padding(horizontal = Pane.space.s, vertical = Pane.space.xs),
        color = color,
    ) {
        Column(
            Modifier.padding(horizontal = Pane.space.m, vertical = Pane.space.xs),
            verticalArrangement = Arrangement.spacedBy(Pane.space.s),
        ) {
            Column {
                // A question with keys shows its own text below. The status names only its first
                // question, so the title would repeat it.
                if (p?.options.isNullOrEmpty())
                    Text(
                        (if (question) "? " else "! ") +
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
            val text = if (p.options.isEmpty()) p.block else p.question
            if (text.isNotEmpty())
                CodeBlock(text, maxHeight = (LocalConfiguration.current.screenHeightDp / 3).dp)
            if (ask) {
                Text(
                    "» Answer this on the Mac.",
                    style = Pane.type.label.copy(fontWeight = FontWeight.Bold),
                    color = c.error,
                )
                return@Column
            }
            if (refusal != null) PaneNote(refusal, NoteKind.Error)
            if (p.options.isNotEmpty()) {
                // No number chips: the phone presses each option's own key, and the numbers have a
                // gap where "Type something" is. Next (tab) ends a multi-select.
                for (o in p.options) KeyButton(
                    o.label + if (o.key == "tab") " ⇥" else "",
                    { onAnswer(o.key) },
                    Modifier.fillMaxWidth(),
                    enabled = !busy,
                    detail = o.detail.ifEmpty { null },
                    align = Alignment.Start,
                )
                if (p.options.any { it.label == "Chat about this" })
                    Text(
                        "» To type your own answer, pick Chat about this, then send a message.",
                        style = Pane.type.caption,
                        color = c.muted,
                    )
                KeyButton("Reload", onReload, key = 'r', kind = KeyKind.Quiet)
                return@Column
            }
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

/** A transcript list entry: one message, or a run of tool steps in one fold. */
data class Entry(val items: List<TranscriptItem>, val steps: Boolean)

/**
 * Folds a transcript: every text shows, and each run of tool calls and results between two texts
 * goes into one fold.
 */
fun fold(items: List<TranscriptItem>): List<Entry> {
    val out = mutableListOf<Entry>()
    var run = mutableListOf<TranscriptItem>()
    for (it in items) if (it.kind == "text") {
        if (run.isNotEmpty()) out += Entry(run, steps = true)
        run = mutableListOf()
        out += Entry(listOf(it), steps = false)
    } else run += it
    if (run.isNotEmpty()) out += Entry(run, steps = true)
    return out
}

private val heading = Regex("^#{1,6} ")
// A code span opens and closes with the same one or two backticks; three start a fence.
private val inline = Regex("""\*\*(.+?)\*\*|(?<!`)(`{1,2})(?!`)(.+?)(?<!`)\2(?!`)""")

/**
 * Light markdown for a message: `# heading` and `**bold**` show bold, `` `code` `` shows in the
 * [code] colour, and their marks go. The rest (lists, tables, fences) stays as written.
 */
fun markdown(text: String, code: Color): AnnotatedString = buildAnnotatedString {
    text.split('\n').forEachIndexed { i, raw ->
        if (i > 0) append('\n')
        val h = heading.find(raw)
        val line = if (h == null) raw else raw.substring(h.range.last + 1)
        val start = length
        var at = 0
        for (m in inline.findAll(line)) {
            append(line, at, m.range.first)
            val (bold, _, mono) = m.destructured
            if (m.groups[1] != null)
                withStyle(SpanStyle(fontWeight = FontWeight.Bold)) { append(bold) }
            // "`` `x` ``" shows "`x`": the spaces only keep the inner backticks apart.
            else withStyle(SpanStyle(color = code)) { append(mono.removeSurrounding(" ")) }
            at = m.range.last + 1
        }
        append(line, at, line.length)
        if (h != null) addStyle(SpanStyle(fontWeight = FontWeight.Bold), start, length)
    }
}

@Composable
fun Transcript(
    vm: AgentViewModel,
    app: String,
    working: Boolean,
    follow: MutableState<Boolean>,
    modifier: Modifier = Modifier,
) {
    val tr by vm.transcript.collectAsStateWithLifecycle()
    val c = Pane.colors
    val list = rememberLazyListState()
    // Live: the newest page at once and then each second while the agent works and the screen is
    // in view.
    // ponytail: polls the whole last page (about 60 KiB); add an "after" cursor if that costs too
    // much.
    val owner = LocalLifecycleOwner.current
    LaunchedEffect(working) {
        if (working)
            owner.repeatOnLifecycle(Lifecycle.State.STARTED) {
                while (true) {
                    vm.refresh()
                    delay(1_000)
                }
            }
    }
    val entries = remember(tr.items) { fold(tr.items) }
    // The open folds, by their first item.
    val open = remember { mutableStateListOf<TranscriptItem>() }

    // Follow, like a terminal: the view stays at the end while items come, the keyboard opens or
    // the last fold grows. A scroll up turns it off. The end, a send or "Bottom ↓" turn it on.
    val scrollUp = remember {
        object : NestedScrollConnection {
            override fun onPreScroll(available: Offset, source: NestedScrollSource): Offset {
                if (
                    source == NestedScrollSource.UserInput &&
                        available.y > 0 &&
                        list.canScrollBackward
                )
                    follow.value = false
                return Offset.Zero
            }
        }
    }
    LaunchedEffect(list) {
        // The transcript opens at the end.
        follow.value = true
        var more = false
        snapshotFlow { follow.value to list.canScrollForward }
            .collect { (on, now) ->
                // Only a scroll that reaches the end turns follow on. A fold that opens at the
                // end turns it off before the list grows.
                if (more && !now) follow.value = true
                else if (on && now) list.scrollToItem(list.layoutInfo.totalItemsCount - 1)
                more = now
            }
    }
    // A tick each time a message start passes the top, like the detents of a dial.
    val haptic = LocalHapticFeedback.current
    LaunchedEffect(list) {
        snapshotFlow { list.firstVisibleItemIndex }
            .drop(1)
            .collect {
                if (!follow.value)
                    haptic.performHapticFeedback(HapticFeedbackType.SegmentFrequentTick)
            }
    }
    val scrolling = list.isScrollInProgress && !follow.value
    val bar by
        animateFloatAsState(
            if (scrolling) 1f else 0f,
            tween(if (scrolling) 100 else 400, delayMillis = if (scrolling) 0 else 600),
            label = "scroll bar",
        )

    Box(modifier.fillMaxWidth().nestedScroll(scrollUp)) {
        LazyColumn(
            Modifier.fillMaxSize().scrollBar(list, c.muted) { bar },
            state = list,
            contentPadding = PaddingValues(horizontal = Pane.space.m, vertical = Pane.space.s),
            verticalArrangement = Arrangement.spacedBy(6.dp),
            flingBehavior = rememberMessageFling(list),
        ) {
            if (tr.nextCursor.isNotEmpty())
                item {
                    Box(
                        Modifier.heightIn(min = Pane.space.touch).clickable(
                            enabled = !tr.loading,
                            role = Role.Button,
                        ) {
                            follow.value = false
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
            for (e in entries) if (!e.steps) item { TranscriptRow(e.items[0], app) }
            else {
                val key = e.items[0]
                val shown = key in open
                item {
                    StepsRow(e.items, shown) {
                        if (shown) open -= key
                        else {
                            open += key
                            // Show the steps from the top, not the end of the list.
                            follow.value = false
                        }
                    }
                }
                if (shown) items(e.items) { TranscriptRow(it, app) }
            }
            if (working) item { WorkingRow(tr.items.lastOrNull()?.ts ?: 0, vm::now) }
            item { Spacer(Modifier.height(1.dp)) }
        }
        if (!follow.value)
            KeyButton(
                "Bottom ↓",
                { follow.value = true },
                Modifier.align(Alignment.BottomEnd).padding(Pane.space.m).background(c.bg),
            )
    }
}

/**
 * A fling that comes to rest on the nearest message start, so a scroll moves from message to
 * message. A message taller than the screen scrolls freely, so all of it can be read.
 */
@Composable
fun rememberMessageFling(list: LazyListState): FlingBehavior {
    val decay = ScrollableDefaults.flingBehavior()
    return remember(list, decay) {
        object : FlingBehavior {
            override suspend fun ScrollScope.performFling(initialVelocity: Float): Float {
                val left = with(decay) { performFling(initialVelocity) }
                val info = list.layoutInfo
                val shown = info.visibleItemsInfo
                val i = shown.indexOfFirst { it.offset + it.size > 0 }
                val top = shown.getOrNull(i) ?: return left
                // At the end the view stays there, so that follow comes back on.
                if (
                    top.offset >= 0 || top.size > info.viewportSize.height || !list.canScrollForward
                )
                    return left
                val next = shown.getOrNull(i + 1)?.offset
                val by = if (next == null || -top.offset < next) top.offset else next
                var done = 0f
                animate(0f, by.toFloat()) { v, _ -> done += scrollBy(v - done) }
                return left
            }
        }
    }
}

/** A thin bar on the right that shows where the view is in the loaded transcript. */
fun Modifier.scrollBar(list: LazyListState, color: Color, alpha: () -> Float) = drawWithContent {
    drawContent()
    val a = alpha()
    val info = list.layoutInfo
    val shown = info.visibleItemsInfo
    val n = info.totalItemsCount
    if (a == 0f || shown.isEmpty() || !(list.canScrollForward || list.canScrollBackward))
        return@drawWithContent
    // Item counts stand in for heights: rows are mostly short now that the steps fold.
    val first = shown.first()
    val pos = first.index + (-first.offset).coerceAtLeast(0).toFloat() / first.size.coerceAtLeast(1)
    val h = maxOf(size.height * shown.size / n, 24.dp.toPx())
    val f =
        if (!list.canScrollForward) 1f
        else (pos / (n - shown.size).coerceAtLeast(1)).coerceIn(0f, 1f)
    val w = 3.dp.toPx()
    drawRoundRect(
        color.copy(alpha = a),
        Offset(size.width - w - 2.dp.toPx(), (size.height - h) * f),
        Size(w, h),
        CornerRadius(w / 2),
    )
}

/** A turn's folded steps: "▸ 5 steps · Bash git status". A tap opens or closes them. */
@Composable
fun StepsRow(steps: List<TranscriptItem>, open: Boolean, onToggle: () -> Unit) {
    val c = Pane.colors
    val calls = steps.filter { it.kind == "tool_call" }
    val n = calls.size.takeIf { it > 0 } ?: steps.size
    // The last call: in a turn that runs, what the agent does now.
    val last = (calls.lastOrNull() ?: steps.last()).text.lineSequence().first()
    Row(
        Modifier.fillMaxWidth()
            .heightIn(min = Pane.space.touch)
            .clickable(role = Role.Button, onClick = onToggle),
        horizontalArrangement = Arrangement.spacedBy(6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            if (open) "  ▾" else "  ▸",
            Modifier.width(labelWidth()),
            style = Pane.type.label.copy(fontWeight = FontWeight.Bold),
            color = c.muted,
        )
        Text(
            "$n ${if (n == 1) "step" else "steps"} · $last",
            style = Pane.type.label,
            color = c.muted,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

/**
 * The end of the transcript while the agent works: a spinner and the time since the last item. The
 * transcript gets a message only when it is complete, so this shows that the agent has not stopped.
 */
@Composable
fun WorkingRow(since: Long, now: () -> Long) {
    val c = Pane.colors
    // The spinner recomposes the row each frame, so the seconds count too.
    val frame = spinnerFrame()
    Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        Text(
            "  $frame",
            Modifier.width(labelWidth()),
            style = Pane.type.label.copy(fontWeight = FontWeight.Bold),
            color = c.working,
        )
        Text(
            if (since > 0) "working · ${((now() - since) / 1000).coerceAtLeast(0)} s"
            else "working",
            style = Pane.type.label,
            color = c.muted,
        )
    }
}

/** Nine cells at the label size; sp, so it grows with the font scale. */
@Composable private fun labelWidth() = with(LocalDensity.current) { 66.sp.toDp() }

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
                Modifier.width(labelWidth()),
                style = Pane.type.label.copy(fontWeight = FontWeight.Bold),
                color = labelColor,
                maxLines = 1,
            )
        // A long press selects the text, for Copy.
        SelectionContainer {
            // Tool steps and terminal text stay as written: their marks are part of the output.
            if (it.kind == "text" && it.role != "terminal")
                Text(markdown(it.text, c.accent), style = Pane.type.label, color = textColor)
            else Text(it.text, style = Pane.type.label, color = textColor)
        }
    }
}

/** The agent's skills that match the [query] after the "/". A tap puts "/name " in the draft. */
@Composable
fun SkillList(skills: List<String>?, query: String, onPick: (String) -> Unit) {
    val c = Pane.colors
    val shown = skills.orEmpty().filter { it.contains(query, ignoreCase = true) }
    PaneBox("skills", Modifier.padding(horizontal = Pane.space.s, vertical = Pane.space.xs)) {
        Column(
            Modifier.heightIn(max = (LocalConfiguration.current.screenHeightDp / 3).dp)
                .verticalScroll(rememberScrollState())
        ) {
            if (shown.isEmpty())
                Text(
                    if (skills == null) "Loading…" else "No skills match.",
                    Modifier.padding(Pane.space.m),
                    style = Pane.type.label,
                    color = c.muted,
                )
            for (name in shown) Box(
                Modifier.fillMaxWidth()
                    .heightIn(min = Pane.space.touch)
                    .clickable(role = Role.Button) { onPick(name) }
                    .padding(horizontal = Pane.space.m),
                contentAlignment = Alignment.CenterStart,
            ) {
                Text("/$name", style = Pane.type.label, color = c.accent)
            }
        }
    }
}

/** The clipboard, with each copy marked sensitive: Android then hides its preview. */
@Composable
fun SensitiveClipboard(content: @Composable () -> Unit) {
    val base = LocalClipboard.current
    val clip =
        remember(base) {
            object : Clipboard by base {
                override suspend fun setClipEntry(clipEntry: ClipEntry?) {
                    clipEntry?.clipData?.description?.extras =
                        PersistableBundle().apply {
                            putBoolean(ClipDescription.EXTRA_IS_SENSITIVE, true)
                        }
                    base.setClipEntry(clipEntry)
                }
            }
        }
    CompositionLocalProvider(LocalClipboard provides clip, content = content)
}

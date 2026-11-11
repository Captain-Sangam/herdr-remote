package dev.herdrremote.ui.theme

import android.provider.Settings
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.sizeIn
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.toggleable
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.LocalContentColor
import androidx.compose.material3.ProvideTextStyle
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.delay

/*
 * The Pane building blocks (docs/design.md, section Components). Screens compose these; the
 * mockups in design/viewer.html show each one in use.
 */

private val TAG =
    mapOf(
        "idle" to "IDLE",
        "working" to "WORK",
        "done" to "DONE",
        "blocked" to "BLOCKED",
        "error" to "ERROR",
    )
private val GLYPH = mapOf("blocked" to '!', "error" to 'x', "done" to '+', "working" to '>')

/** A 1 dp rule under the element, solid or dashed. */
fun Modifier.bottomRule(color: Color, dashed: Boolean = false) = rule(color, bottom = true, dashed)

/** A 1 dp rule over the element. */
fun Modifier.topRule(color: Color) = rule(color, bottom = false, dashed = false)

private fun Modifier.rule(color: Color, bottom: Boolean, dashed: Boolean) = drawBehind {
    val w = 1.dp.toPx()
    val y = if (bottom) size.height - w / 2 else w / 2
    val dash =
        if (dashed) PathEffect.dashPathEffect(floatArrayOf(4.dp.toPx(), 3.dp.toPx())) else null
    drawLine(color, Offset(0f, y), Offset(size.width, y), w, pathEffect = dash)
}

/** Inverse-video status tag: BLOCKED in the background colour on the status colour. */
@Composable
fun StatusTag(state: String, modifier: Modifier = Modifier) {
    Text(
        TAG[state] ?: "IDLE",
        modifier
            .alpha(statusAlpha(state))
            .background(Pane.colors.status(state))
            .padding(horizontal = 6.dp),
        style = Pane.type.caption.copy(fontWeight = FontWeight.Bold),
        color = Pane.colors.bg,
        maxLines = 1,
    )
}

/** The row lead: [!] blocked, [x] error, [+] done, [>] working, [ ] idle. Never colour alone. */
@Composable
fun StatusGlyph(state: String, modifier: Modifier = Modifier) {
    Text(
        "[${GLYPH[state] ?: ' '}]",
        modifier.alpha(statusAlpha(state)).clearAndSetSemantics { contentDescription = state },
        style = Pane.type.body.copy(fontWeight = FontWeight.Bold),
        color = Pane.colors.status(state),
    )
}

/**
 * A working status blinks like a terminal cursor: 530 ms bright, 530 ms dim. The wall clock keeps
 * every blink on the screen in step.
 */
@Composable
fun statusAlpha(state: String): Float {
    if (state != "working" || animationsOff()) return 1f
    var on by remember { mutableStateOf(true) }
    LaunchedEffect(Unit) {
        while (true) {
            val t = System.currentTimeMillis()
            on = t / BLINK_MS % 2 == 0L
            delay(BLINK_MS - t % BLINK_MS)
        }
    }
    return if (on) 1f else 0.35f
}

private const val BLINK_MS = 530L

/**
 * The phone's "Remove animations": the animator duration scale is 0. Nothing then blinks or spins.
 */
@Composable
private fun animationsOff(): Boolean {
    val cr = LocalContext.current.contentResolver
    return remember(cr) {
        Settings.Global.getFloat(cr, Settings.Global.ANIMATOR_DURATION_SCALE, 1f) == 0f
    }
}

/** A braille spinner: the frame changes each 80 ms. */
@Composable
fun spinnerFrame(): Char {
    if (animationsOff()) return SPINNER[0]
    var frame by remember { mutableIntStateOf(0) }
    LaunchedEffect(Unit) {
        while (true) {
            delay(80)
            frame++
        }
    }
    return SPINNER[frame % SPINNER.length]
}

private const val SPINNER = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"

/** One bar for each agent of a space, in the colour of its state. */
@Composable
fun StatusMeter(states: List<String>, modifier: Modifier = Modifier) {
    Row(
        modifier.clearAndSetSemantics { contentDescription = states.joinToString() },
        horizontalArrangement = Arrangement.spacedBy(2.dp),
    ) {
        states.forEach {
            Box(
                Modifier.size(7.dp, 14.dp).alpha(statusAlpha(it)).background(Pane.colors.status(it))
            )
        }
    }
}

/**
 * The pane: a 1 dp box with its title in the top border, like a tmux pane. Pane titles are
 * lowercase ("4 spaces", "needs you (2)", "permission"). [color] draws the border and the title:
 * the line colour for lists, a status colour for an alert. [behind] is the colour behind the box,
 * so that the title cuts the border.
 */
@Composable
fun PaneBox(
    title: String?,
    modifier: Modifier = Modifier,
    color: Color = Pane.colors.line,
    behind: Color = Pane.colors.bg,
    content: @Composable ColumnScope.() -> Unit,
) {
    val alert = color != Pane.colors.line
    Box(modifier.padding(top = 9.dp)) {
        Column(
            Modifier.fillMaxWidth()
                .border(Pane.space.line, color)
                .padding(top = if (title == null) 4.dp else 10.dp, bottom = 4.dp),
            content = content,
        )
        if (title != null)
            Text(
                "┤ $title ├",
                Modifier.padding(start = 8.dp)
                    .offset(y = (-9).dp)
                    .background(behind)
                    .padding(horizontal = 6.dp),
                style =
                    Pane.type.label.copy(
                        fontWeight = if (alert) FontWeight.Bold else FontWeight.Normal
                    ),
                color = if (alert) color else Pane.colors.muted,
            )
    }
}

/**
 * A list row: status glyph, name, a meta line, an optional detail line and a trailing slot (the
 * status tag by default). A blocked row has a faint red background.
 */
@Composable
fun StatusRow(
    state: String,
    title: String,
    meta: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    detail: String? = null,
    trailing: @Composable RowScope.() -> Unit = { StatusTag(state) },
) {
    val c = Pane.colors
    Row(
        modifier
            .fillMaxWidth()
            .then(
                if (state == "blocked") Modifier.background(c.blocked.copy(alpha = 0.12f))
                else Modifier
            )
            .clickable(onClick = onClick)
            .heightIn(min = Pane.space.touch)
            .padding(horizontal = Pane.space.m, vertical = Pane.space.xs),
        horizontalArrangement = Arrangement.spacedBy(Pane.space.m),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        StatusGlyph(state)
        Column(Modifier.weight(1f)) {
            Text(
                title,
                style = Pane.type.title.copy(fontSize = Pane.type.body.fontSize),
                maxLines = 1,
            )
            Text(meta, style = Pane.type.caption, color = c.muted, maxLines = 1)
            if (detail != null)
                Text(
                    detail,
                    style = Pane.type.caption,
                    color = if (state == "blocked") c.blocked else c.text,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
        }
        trailing()
    }
}

/**
 * The top line of a screen: a back key, the path ("~/spaces/herdr-remote", last part bold) and the
 * link state when [online] is not null.
 */
@Composable
fun PaneHeader(
    path: String,
    onBack: (() -> Unit)? = null,
    online: Boolean? = null,
    actions: @Composable RowScope.() -> Unit = {},
) {
    val c = Pane.colors
    Row(
        Modifier.fillMaxWidth()
            .background(c.surface)
            .bottomRule(c.line)
            .heightIn(min = Pane.space.touch)
            .padding(end = Pane.space.m),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (onBack != null)
            Box(
                Modifier.size(Pane.space.touch)
                    .clickable(onClickLabel = "Back", role = Role.Button, onClick = onBack)
                    .semantics { contentDescription = "Back" },
                contentAlignment = Alignment.Center,
            ) {
                Text("←", style = Pane.type.title, color = c.accent)
            }
        else Box(Modifier.width(Pane.space.l))
        val cut = path.lastIndexOf('/') + 1
        Text(
            buildAnnotatedString {
                withStyle(SpanStyle(color = c.muted, fontWeight = FontWeight.Normal)) {
                    append(path.take(cut))
                }
                append(path.drop(cut))
            },
            Modifier.weight(1f),
            style = Pane.type.title,
            color = c.text,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
        actions()
        if (online != null)
            Text(
                if (online) "● online" else "○ offline",
                style = Pane.type.caption,
                color = if (online) c.done else c.blocked,
            )
    }
}

/** The line under the header: counts or the agent's state, with a dashed rule. */
@Composable
fun PaneSubline(content: @Composable RowScope.() -> Unit) {
    CompositionLocalProvider(LocalContentColor provides Pane.colors.muted) {
        ProvideTextStyle(Pane.type.label) {
            Row(
                Modifier.fillMaxWidth()
                    .bottomRule(Pane.colors.line, dashed = true)
                    .padding(horizontal = Pane.space.l, vertical = 6.dp),
                horizontalArrangement = Arrangement.spacedBy(Pane.space.s),
                verticalAlignment = Alignment.CenterVertically,
                content = content,
            )
        }
    }
}

/** Tabs as terminal labels: the selected tab is inverse on the accent colour. */
@Composable
fun PaneTabs(tabs: List<String>, selected: Int, onSelect: (Int) -> Unit) {
    val c = Pane.colors
    Row(
        Modifier.fillMaxWidth()
            .background(c.surface)
            .bottomRule(c.line)
            .height(Pane.space.touch)
            .padding(horizontal = Pane.space.s)
    ) {
        tabs.forEachIndexed { i, tab ->
            val on = i == selected
            Box(
                Modifier.fillMaxHeight()
                    .then(if (on) Modifier.background(c.accent) else Modifier)
                    .selectable(selected = on, role = Role.Tab, onClick = { onSelect(i) })
                    .padding(horizontal = Pane.space.l),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    tab.lowercase(),
                    style =
                        Pane.type.label.copy(
                            fontWeight = if (on) FontWeight.Bold else FontWeight.Normal
                        ),
                    color = if (on) c.onAccent else c.muted,
                )
            }
        }
    }
}

/**
 * The bottom navigation, a tmux status line: "[herdr] 0:spaces* 1:agents(2!) 2:settings". [alerts]
 * maps an item index to a count of blocked agents.
 */
@Composable
fun TmuxBar(
    items: List<String>,
    selected: Int,
    onSelect: (Int) -> Unit,
    alerts: Map<Int, Int> = emptyMap(),
) {
    val c = Pane.colors
    // Scrolls sideways when it does not fit (font scale 1.3 on a narrow phone).
    Row(
        Modifier.fillMaxWidth()
            .height(Pane.space.touch)
            .background(c.surface2)
            .topRule(c.line)
            .horizontalScroll(rememberScrollState())
    ) {
        Box(
            Modifier.fillMaxHeight().background(c.done).padding(horizontal = Pane.space.m),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                "[herdr]",
                style = Pane.type.label.copy(fontWeight = FontWeight.Bold),
                color = c.bg,
            )
        }
        items.forEachIndexed { i, name ->
            val on = i == selected
            Box(
                Modifier.fillMaxHeight()
                    .then(if (on) Modifier.background(c.accent.copy(alpha = 0.18f)) else Modifier)
                    .selectable(selected = on, role = Role.Tab, onClick = { onSelect(i) })
                    .padding(horizontal = 9.dp),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    buildAnnotatedString {
                        append(name.lowercase())
                        if (on) append("*")
                        val n = alerts[i] ?: 0
                        if (n > 0)
                            withStyle(SpanStyle(color = c.blocked, fontWeight = FontWeight.Bold)) {
                                append("($n!)")
                            }
                    },
                    style =
                        Pane.type.label.copy(
                            fontWeight = if (on) FontWeight.Bold else FontWeight.Normal
                        ),
                    color = if (on) c.text else c.muted,
                )
            }
        }
    }
}

/** What a [KeyButton] does: Go is the main action, Stop denies or destroys, Quiet is minor. */
enum class KeyKind {
    Go,
    Plain,
    Stop,
    Quiet,
}

/**
 * A square key: 1 dp border, at least 48 dp high, an optional key hint in an inverse chip ("[y]
 * Approve"). Go is filled (Approve, Scan the QR code, Unlock). Plain is a normal action. Stop is
 * Deny or Forget this Mac. Quiet is a minor action (Reload).
 */
@Composable
fun KeyButton(
    label: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    key: Char? = null,
    kind: KeyKind = KeyKind.Plain,
    enabled: Boolean = true,
    detail: String? = null,
    align: Alignment.Horizontal = Alignment.CenterHorizontally,
) {
    val c = Pane.colors
    val fg =
        when (kind) {
            KeyKind.Go -> c.bg
            KeyKind.Plain -> c.accent
            KeyKind.Stop -> c.blocked
            KeyKind.Quiet -> c.muted
        }
    val edge =
        when (kind) {
            KeyKind.Go -> c.done
            KeyKind.Quiet -> c.line
            else -> fg
        }
    Row(
        modifier
            .alpha(if (enabled) 1f else 0.45f)
            .then(if (kind == KeyKind.Go) Modifier.background(c.done) else Modifier)
            .border(Pane.space.line, edge)
            .clickable(enabled = enabled, role = Role.Button, onClick = onClick)
            .heightIn(min = Pane.space.touch)
            .padding(horizontal = Pane.space.l),
        horizontalArrangement = Arrangement.spacedBy(Pane.space.s, align),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (key != null)
            Box(
                Modifier.size(20.dp)
                    .background(if (kind == KeyKind.Go) c.bg else fg)
                    .clearAndSetSemantics {},
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    key.toString(),
                    style = Pane.type.label.copy(fontWeight = FontWeight.Bold),
                    color = if (kind == KeyKind.Go) c.done else c.bg,
                )
            }
        val bold = Pane.type.label.copy(fontWeight = FontWeight.Bold)
        if (detail == null) Text(label, style = bold, color = fg)
        else
            Column(Modifier.padding(vertical = Pane.space.xs)) {
                Text(label, style = bold, color = fg)
                Text(detail, style = Pane.type.caption, color = c.muted)
            }
    }
}

/**
 * The message composer: a "›" prompt, a monospace field, the character count and the send key.
 * While the agent is blocked it is off and the placeholder says why ("Answer the prompt first").
 */
@Composable
fun PromptField(
    value: String,
    onValueChange: (String) -> Unit,
    onSend: () -> Unit,
    placeholder: String,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    maxLength: Int = 4000,
    onSkills: (() -> Unit)? = null,
) {
    val c = Pane.colors
    var field by remember { mutableStateOf(TextFieldValue(value)) }
    val focus = remember { FocusRequester() }
    // A value set from outside (a skill, the clear after a send) puts the cursor at its end.
    if (field.text != value) field = TextFieldValue(value, TextRange(value.length))
    Row(
        modifier
            .fillMaxWidth()
            .background(c.surface)
            .topRule(c.line)
            .padding(horizontal = Pane.space.s, vertical = 6.dp),
        horizontalArrangement = Arrangement.spacedBy(6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            "›",
            style = Pane.type.body.copy(fontWeight = FontWeight.Bold),
            color = if (enabled) c.accent else c.muted,
        )
        BasicTextField(
            value = field,
            onValueChange = {
                val t = it.text.take(maxLength)
                field = if (t == it.text) it else TextFieldValue(t, TextRange(t.length))
                onValueChange(t)
            },
            modifier = Modifier.weight(1f).focusRequester(focus),
            enabled = enabled,
            textStyle = Pane.type.body.copy(color = c.text),
            cursorBrush = SolidColor(c.accent),
            maxLines = 4,
            decorationBox = { field ->
                Box(
                    Modifier.heightIn(min = Pane.space.touch),
                    contentAlignment = Alignment.CenterStart,
                ) {
                    if (value.isEmpty()) Text(placeholder, style = Pane.type.body, color = c.muted)
                    field()
                }
            },
        )
        if (value.isNotEmpty())
            Text("${value.length}/$maxLength", style = Pane.type.caption, color = c.muted)
        if (onSkills != null)
            KeyButton(
                "/",
                {
                    onSkills()
                    focus.requestFocus() // and the keyboard opens, to filter the list
                },
                Modifier.semantics { contentDescription = "Skills" },
                kind = KeyKind.Quiet,
                enabled = enabled,
            )
        KeyButton("⏎ send", onSend, enabled = enabled && value.isNotBlank())
    }
}

/** What a [PaneNote] reports. */
enum class NoteKind {
    Ok,
    Warn,
    Error,
}

/**
 * The result of an action, in words ("Done. The agent continues."): a 3 dp bar in the colour of the
 * result on the surface colour. Use the texts in AgentViewModel.Texts.
 */
@Composable
fun PaneNote(text: String, kind: NoteKind, modifier: Modifier = Modifier) {
    val c = Pane.colors
    val bar =
        when (kind) {
            NoteKind.Ok -> c.done
            NoteKind.Warn -> c.error
            NoteKind.Error -> c.blocked
        }
    Row(
        modifier
            .fillMaxWidth()
            .background(c.surface)
            .drawBehind { drawRect(bar, size = Size(3.dp.toPx(), size.height)) }
            .padding(start = 13.dp, end = Pane.space.m, top = 6.dp, bottom = 6.dp),
        horizontalArrangement = Arrangement.spacedBy(Pane.space.s),
    ) {
        Text(
            when (kind) {
                NoteKind.Ok -> "+"
                NoteKind.Warn -> "!"
                NoteKind.Error -> "x"
            },
            Modifier.clearAndSetSemantics {},
            style = Pane.type.label.copy(fontWeight = FontWeight.Bold),
            color = bar,
        )
        Text(text, style = Pane.type.label, color = c.text)
    }
}

/**
 * A full-width warning strip under the header ("Not connected to the Mac: last update 48 s ago").
 */
@Composable
fun PaneBanner(text: String, modifier: Modifier = Modifier) {
    Text(
        text,
        modifier
            .fillMaxWidth()
            .background(Pane.colors.error)
            .padding(horizontal = Pane.space.l, vertical = 6.dp),
        style = Pane.type.label.copy(fontWeight = FontWeight.Bold),
        color = Pane.colors.bg,
    )
}

/** An on/off switch drawn as "[ on ]" and "[ off ]". */
@Composable
fun PaneSwitch(
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
    modifier: Modifier = Modifier,
) {
    Box(
        modifier
            .sizeIn(minWidth = Pane.space.touch, minHeight = Pane.space.touch)
            .toggleable(value = checked, role = Role.Switch, onValueChange = onCheckedChange),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            if (checked) "[ on ]" else "[ off ]",
            style = Pane.type.body.copy(fontWeight = FontWeight.Bold),
            color = if (checked) Pane.colors.done else Pane.colors.muted,
        )
    }
}

/** A text field in a pane, with its label in the top border ("This phone's name"). */
@Composable
fun PaneField(
    label: String,
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    placeholder: String = "",
    singleLine: Boolean = true,
    minLines: Int = 1,
) {
    PaneBox(title = label, modifier = modifier) {
        BasicTextField(
            value = value,
            onValueChange = onValueChange,
            modifier = Modifier.fillMaxWidth().padding(horizontal = Pane.space.m, vertical = 6.dp),
            textStyle = Pane.type.body.copy(color = Pane.colors.text),
            cursorBrush = SolidColor(Pane.colors.accent),
            singleLine = singleLine,
            minLines = minLines,
            decorationBox = { field ->
                Box {
                    if (value.isEmpty())
                        Text(placeholder, style = Pane.type.body, color = Pane.colors.muted)
                    field()
                }
            },
        )
    }
}

/**
 * A command, a diff or prompt text, exactly as the agent shows it: no wrapping, scrolls both ways,
 * at most [maxHeight] high so that the keys under it stay in reach.
 */
@Composable
fun CodeBlock(text: String, modifier: Modifier = Modifier, maxHeight: Dp = Dp.Infinity) {
    val c = Pane.colors
    Text(
        text,
        modifier
            .fillMaxWidth()
            .heightIn(max = maxHeight)
            .background(c.surface)
            .drawBehind { drawRect(c.line, size = Size(2.dp.toPx(), size.height)) }
            .verticalScroll(rememberScrollState())
            .horizontalScroll(rememberScrollState())
            .padding(
                start = Pane.space.l,
                end = Pane.space.m,
                top = Pane.space.s,
                bottom = Pane.space.s,
            ),
        style = Pane.type.label,
        color = c.text,
        softWrap = false,
    )
}

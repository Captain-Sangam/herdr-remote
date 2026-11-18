# herdr-remote design system: Pane

- **Status:** Chosen on 9 November 2026.
- **Owner:** Ajay.
- **Code:** `android/app/src/main/java/dev/herdrremote/ui/theme/`: `PaneTheme.kt` (tokens) and `PaneComponents.kt` (components).
- **iOS code:** `ios/App/Pane.swift` (tokens) and `ios/App/PaneComponents.swift` (components), with the same names. Launch a debug build with `-panePreview` to see every component on one screen.
- **Mockups:** [`design/viewer.html`](../design/viewer.html) shows one screen at a time. [`design/options.html`](../design/options.html) shows all screens, the tokens and the components. Both use [`design/mockups.js`](../design/mockups.js) and [`design/mockups.css`](../design/mockups.css).

## 1. Summary

Pane makes the app look like the terminal that it controls.

- All text is monospace: JetBrains Mono.
- Content sits in square panes with 1 dp lines and a title in the top border, like tmux.
- Status uses ANSI colours, and always a glyph or a tag too.
- Dark is the main mode. Light mode follows the system setting.

## 2. Rules

1. Read every colour, text style and size from `Pane.colors`, `Pane.type` and `Pane.space`. Do not hard-code a value in a screen.
2. Use a Pane component first. Use a Material 3 component only when Pane has none. `PaneTheme` maps the tokens onto Material 3. So a Material component still matches.
3. Use square corners and 1 dp lines. Do not use rounded corners, shadows, elevation, gradients or blur.
4. Show a status with its colour and a glyph or a tag. Never use colour alone.
5. Make every tap target 48 dp or more, also in a dense list.
6. Keep content between the status bar and the navigation bar. The app root applies `safeDrawingPadding()`. Do not draw content under the system bars.
7. Write pane titles, tabs and navigation items in lowercase: "4 spaces", "transcript". Write content, buttons and messages in sentence case: "Forget this Mac".
8. Show commands, diffs and prompt text exactly as the agent shows them. Use `CodeBlock`. The type scale turns ligatures off for this reason.
9. Show no commands or prompts on the lock screen ([SR-11](product.md#8-security-and-privacy-requirements)). The lock screen can show the agent name. An alert on an unlocked phone shows the prompt too ([ADR 0019](adr/0019-alert-preview-and-actions.md)).
10. Use the result and error texts in `AgentViewModel.Texts`. Write new texts in the same plain style.

## 3. Colour tokens

| Token      | Light     | Dark      | Use                                                                   |
| ---------- | --------- | --------- | --------------------------------------------------------------------- |
| `bg`       | `#EEF1F6` | `#191C26` | Screen background. Also the text on a status tag and on the Go key.   |
| `surface`  | `#FFFFFF` | `#1F2330` | Header, tab bar, composer, code block, note.                          |
| `surface2` | `#E2E7EF` | `#272C3B` | Tmux bar.                                                             |
| `line`     | `#B7C0D1` | `#3B4258` | Every border and rule.                                                |
| `text`     | `#222836` | `#D8DEE9` | Main text.                                                            |
| `muted`    | `#5B667D` | `#7D879E` | Meta lines, pane titles, placeholders, inactive tabs.                 |
| `accent`   | `#2E7D8C` | `#88C0D0` | Back key, selected tab, cursor, Plain keys.                           |
| `onAccent` | `#FFFFFF` | `#191C26` | Text on the accent colour.                                            |
| `idle`     | `#7A8499` | `#6C7891` | Idle agents.                                                          |
| `working`  | `#3A6EA5` | `#81A1C1` | Working agents. The border of a view-only prompt.                     |
| `done`     | `#4F7F2E` | `#A3BE8C` | Done agents. The Go key, the `[herdr]` block, "online", Ok notes.     |
| `blocked`  | `#B23A48` | `#BF616A` | Blocked agents. The approval pane, Stop keys, "offline", Error notes. |
| `error`    | `#B4561E` | `#D08770` | Agents in error. The offline banner, Warn notes, "Answer on the Mac". |

### Status

| State     | Glyph | Tag       | Token     |
| --------- | ----- | --------- | --------- |
| `blocked` | `[!]` | `BLOCKED` | `blocked` |
| `error`   | `[x]` | `ERROR`   | `error`   |
| `done`    | `[+]` | `DONE`    | `done`    |
| `working` | `[>]` | `WORK`    | `working` |
| `idle`    | `[ ]` | `IDLE`    | `idle`    |

- Lists sort by this order: blocked, error, done, working, idle.
- `Pane.colors.status(state)` gives the colour. An unknown state is idle.

### Terminal palette

The live view uses this palette in both modes. The values are in `assets/xterm/live.html`. Kotlin has the background and the foreground as `Pane.terminalBg` and `Pane.terminalFg`.

| Role       | Colour    |
| ---------- | --------- |
| Background | `#14161E` |
| Foreground | `#D8DEE9` |
| Black      | `#3B4252` |
| Red        | `#BF616A` |
| Green      | `#A3BE8C` |
| Yellow     | `#EBCB8B` |
| Blue       | `#81A1C1` |
| Magenta    | `#B48EAD` |
| Cyan       | `#88C0D0` |
| White      | `#E5E9F0` |

## 4. Type

- **Family:** JetBrains Mono 2.304, Regular and Bold, in `res/font/`.
- **License:** SIL Open Font License 1.1, in `assets/licenses/JetBrainsMono-OFL.txt`. The APK contains it.

| Style     | Kotlin              | Size / line (sp) | Weight  | Use                                        |
| --------- | ------------------- | ---------------- | ------- | ------------------------------------------ |
| `code`    | `Pane.type.code`    | 44 / 52          | Bold    | Pairing code.                              |
| `display` | `Pane.type.display` | 18 / 26          | Bold    | Title of a full-screen flow.               |
| `head`    | `Pane.type.head`    | 16 / 24          | Bold    | Empty-state title.                         |
| `title`   | `Pane.type.title`   | 14 / 20          | Bold    | Header path. Row names use it at 13 sp.    |
| `body`    | `Pane.type.body`    | 13 / 20          | Regular | Body text, field input.                    |
| `label`   | `Pane.type.label`   | 12 / 18          | Regular | Keys, tabs, pane titles, transcript, code. |
| `caption` | `Pane.type.caption` | 11 / 16          | Regular | Meta lines, status tags, hints.            |

## 5. Spacing and shape

| Token              | Value | Use                                   |
| ------------------ | ----- | ------------------------------------- |
| `Pane.space.xs`    | 4 dp  | Row padding, top and bottom.          |
| `Pane.space.s`     | 8 dp  | Gaps between items, pane side margin. |
| `Pane.space.m`     | 10 dp | Row padding, left and right.          |
| `Pane.space.l`     | 12 dp | Header and code block padding.        |
| `Pane.space.xl`    | 16 dp | Space around a full-screen flow.      |
| `Pane.space.touch` | 48 dp | Minimum tap target.                   |
| `Pane.space.line`  | 1 dp  | Every border and rule.                |

- All corners are 0 dp. `PaneShapes` gives Material 3 square shapes.
- Nothing has elevation or a shadow.

## 6. Components

| Component                   | Use                                                                                               |
| --------------------------- | ------------------------------------------------------------------------------------------------- |
| `PaneHeader`                | Top line of a screen: back key, path (`~/spaces/herdr-remote`), link state.                       |
| `PaneSubline`               | Line under the header: counts or the agent state, dashed rule.                                    |
| `PaneBox`                   | A pane: 1 dp box, title in the top border. A status colour makes it an alert.                     |
| `StatusRow`                 | List row: glyph, name, meta line, detail line, status tag. Blocked rows have a red tint.          |
| `StatusGlyph`               | `[!]` lead of a row.                                                                              |
| `StatusTag`                 | `BLOCKED` inverse tag.                                                                            |
| `StatusMeter`               | One bar for each agent of a space.                                                                |
| `PaneTabs`                  | `transcript` and `live` tabs. The selected tab is inverse on the accent colour.                   |
| `TmuxBar`                   | Bottom navigation: `[herdr] spaces* agents(2!) settings`. No window numbers: they read as counts. |
| `KeyButton`, `KeyKind`      | Square key with an optional key hint. Go, Plain, Stop or Quiet.                                   |
| `PromptField`               | Message composer: `›` prompt, field, character count, send key.                                   |
| `PaneNote`, `NoteKind`      | Result of an action: Ok, Warn or Error bar.                                                       |
| `PaneBanner`                | Full-width warning strip, for example "Not connected to the Mac".                                 |
| `PaneSwitch`                | `[ on ]` and `[ off ]` switch.                                                                    |
| `PaneField`                 | Text field in a pane, label in the top border.                                                    |
| `CodeBlock`                 | Command, diff or prompt text. No wrap, scrolls both ways, optional maximum height.                |
| `bottomRule()`, `topRule()` | 1 dp rule under or over an element.                                                               |

## 7. Screen anatomy

A screen has these parts, from top to bottom:

1. `PaneHeader` with the path and, on list screens, the link state.
2. `PaneSubline` with the counts, or with the status tag, app and space of an agent.
3. `PaneBanner`, only while the phone is not connected.
4. The content: `PaneBox` panes with `StatusRow` rows, or `PaneTabs` with the transcript or the live view.
5. `TmuxBar` on the top-level screens (spaces, agents, settings), or `PromptField` on the agent screen.

The paths are `~/spaces`, `~/spaces/<space>`, `~/agents`, `~/<space>/<tab>` and `~/settings`.

### Patterns

- **Approval:**
  - Use `PaneBox("permission", color = blocked)`.
  - Inside it, show the prompt title in bold blocked text with a "! " prefix, then the kind line ("Command · turn 41").
  - Show the command or diff in a `CodeBlock`, at most one third of the screen high.
  - Show the keys under it: Approve (`y`, Go), Deny (`n`, Stop), Reload (`r`, Quiet).
  - Show a refusal as a `PaneNote` of kind Error, above the keys.
- **Question:** use `PaneBox("question", color = working)` and a "? " prefix. Show the question text (not the options) in a `CodeBlock`, then one full-width Plain key for each option, with the option's number as the key hint and its description on a second line. Show Next (`⇥`) after a multi-select, then Reload. A landed answer shows no note: the next screen of the question is the feedback.
- **View-only prompt:** use the question pane, with the line "» Answer this on the Mac." in the error colour. Show no keys.
- **Skills:** when the draft starts with "/" and has no space, show `PaneBox("skills")` above the message box. List `/name` for each skill that matches, at most one third of the screen high. A tap puts `/name ` in the draft. The Quiet "/" key beside Send starts the list.
- **Transcript text:** show every message. Fold each run of tool steps into one "▸ 3 steps · Bash git status" row. In a message, show `# heading` and `**bold**` bold and `` `code` `` in the accent colour, without their marks ([ADR 0026](adr/0026-show-agent-text.md)).
- **Copy:** a long press on a transcript message selects its text. Each copy is marked sensitive, so Android hides its preview.
- **Result of an action:** show a `PaneNote` under the header. Use Ok for applied, Warn for sent but not confirmed, uncertain or outcome unknown, and Error for a refusal.
- **Not connected:** show a `PaneBanner` with the age of the data. Show "○ offline" in the header. Dim the old content to 45%.
- **Usage limits:** on Android, the gauge beside "● online" on `~/spaces` opens a narrow pane from the right. Tap outside it, press Back, or tap `[x]` to close it. See the [Side meter mockup](../design/usage.html).
  - Show a bar, the percentage used, and the reset time in each source pane. Show the Codex plan if available. The "Mac read" line gives the reading time and age. If the sources have different times, show the older time in its source pane.
  - Use `[!]` and the warning colour at 90% or more. Use `[x]` and the blocked colour at 100% or more. After 30 min, mark the reading as stale and dim its values. When a reset time passes, show "Reset" and the scheduled time. Show "No data" for a missing source.
  - Read usage when Spaces opens, when the pane opens, and when the user taps the Quiet `[r]` Refresh key. The key shows "Loading…" while the phone waits. The readings stay in memory.
- **Loading:** show a short terminal log, not a spinner. Show `$ herdr-remote connect`, one line for each endpoint that the phone tries, and a block cursor.
- **Working:** a working status (the glyph, the tag and the meter bar) blinks like a terminal cursor, 530 ms bright and 530 ms dim, in step on the whole screen. Nothing blinks or spins when "Remove animations" is on. While an agent works, end its transcript with a braille spinner in the working colour and "working · 14 s", the seconds since the last item. The transcript gets a message only when it is complete, so the row shows that the agent has not stopped.
- **Empty:** show a small box drawing, the title in `head` and one line that says what to do.
- **Pairing code:** use `Pane.type.code` in the done colour. Show two groups of three digits, each group in a 1 dp box.
- **Key hints:** the chips (`y`, `n`, `r`) are visual only today.

## 8. App icon

- **Idea:** a split terminal pane. The red pane is an agent that waits.
- **Launcher:** `res/mipmap-anydpi/ic_launcher.xml`, an adaptive icon with `ic_launcher_foreground`, the background colour `ic_launcher_background` (`#191C26`) and a themed layer `ic_launcher_monochrome`.
- **Notifications:** `res/drawable/ic_stat_herdr.xml`, the same shapes in white.
- **UI icons:** `res/drawable/ic_lock`, `ic_mac`, `ic_tail`, `ic_key` and `ic_bell`. They are the stroke icons from `design/mockups.js` with square ends. The screen tints them.

## 9. Add a screen

1. Draw the screen in `design/mockups.js` if it has a new pattern. Check it in `design/viewer.html` in light and dark mode.
2. Start the screen with `PaneHeader` and its path.
3. Build the content from the components in section 6.
4. Check the screen in light and dark mode, at font scale 1.3, with 3-button navigation.
5. Check that no tap target is smaller than 48 dp.

## 10. Current state

- All screens use the Pane components and match `design/viewer.html`:
  - lock
  - pair (start, code, failed)
  - spaces (loading, not connected), space, all agents (empty)
  - agent (approval, refusal, view-only question, result notes, transcript, live)
  - settings.
- `TmuxBar` is the navigation between spaces, agents and settings. It scrolls sideways when it does not fit, for example at font scale 1.3.
- Status colours come only from `Pane.colors.status()`, through `StatusRow`, `StatusGlyph`, `StatusTag` and `StatusMeter`. `statusColor()` is gone.
- The live view has the terminal background `#14161E` behind it and in the WebView before the page paints.
- `Texts.kind()` gives the `NoteKind` of a result or error text.

Known differences from the mockups:

- The API has no agent age. Meta lines show "space · app · kind" and not "app · 1 min".
- The API has no tool name. The "needs you" rows show the prompt text and not "Bash · herdr-remote".
- The loading log lists the stored endpoints. The phone does not report a result for each endpoint, so no line shows "timeout".
- The header shows the link state on the list screens only (section 7). The mockup also shows it on the agent screen.
- The mockups show window numbers in the tmux bar (`0:spaces*`). The app has none, because "0:spaces" read as "0 spaces" (`907cdfc`).
- At font scale 1.3, a pane title sits about 3 dp below the border line. The cause: `PaneBox` offsets the title in dp, but the title grows in sp.

Checks:

- The JVM tests pass, and `Texts.kind()` and `endpointKind()` have tests.
- The pair screen was checked on the emulator with `uiautomator dump` at font scale 1.0 and 1.3, in dark mode. Keys and fields are 48 dp high.
- The other screens need a paired phone. Check them on the phone in light and dark mode, at font scale 1.3, with 3-button navigation. `adb exec-out screencap -p` takes a screenshot.

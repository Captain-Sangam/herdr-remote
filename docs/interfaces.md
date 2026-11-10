# Herdr and agent interfaces

herdr-remote uses Herdr.app as shipped. This page lists the facts about Herdr, Claude Code, Codex and macOS that the code depends on. Check them again when one of these tools updates.

- **Verified on:** 2026-11-08, with probe terminals. The implementer verified them again before the first build step. Nothing differed.
- **Related:** [architecture](architecture.md), [testing: probe agents](testing.md#probe-agents).

## Versions

| Tool              | Version verified |
| ----------------- | ---------------- |
| Herdr.app         | 0.1.13           |
| herdr-core (`hc`) | 0.1.18           |
| Claude Code       | 2.1.295          |
| codex-cli         | 0.167.0          |

A prompt that the app does not know is view-only. So a new version fails safe.

## herdr-core API

- **Protocol:** JSON-RPC 2.0, one message on each line, on a local socket.
  - `hc connect` speaks the protocol over stdin and stdout (`hc connect --help`).
  - `hello` takes `{}`.
  - `hc schema` prints the full OpenRPC document.
- **Read methods:**
  - `terminal.list` (with a `labels` filter and `all`).
  - `terminal.get`.
  - `terminal.read {terminal, lines}`: the screen plus `lines` of history, as text. `lines: 0` gives the visible screen only.
- **History limit:** while the status is `working`, `blocked` or `error`, `terminal.read` returns only the visible screen, marked incomplete (`hc --skill`, "History reads").
- **Events:** `events.subscribe` returns `{seq, terminals[]}`. After that, the connection carries only notifications:
  - fact changes: created, lost, restarted, finished, removed, status, turn, title, labels, size, doc
  - `heartbeat {seq}` every 15 s
  - `missed`, then a fresh snapshot, for a subscriber that falls behind.
- **Action methods:**
  - `terminal.send {terminal, text, force, if, wait, timeout}` pastes text and presses Enter. It refuses a blocked program unless `force` is true. `if` accepts only `idle` or `working`.
  - `terminal.keys {terminal, keys[]}` presses named keys and returns `KeysResult {status, turn}`. It has no blocked refusal and no status condition.
  - Both methods wait until nobody types at the Mac for about 3 s. hc holds keys that a person presses in that time.
  - A `terminal.send` from a process outside any terminal, with no `caller`, arrives as a plain prompt without an `<hc from=…>` tag. One sent from a terminal arrives tagged.
- **Methods that herdr-remote never uses:** `terminal.run`, `terminal.kill`, `terminal.rm`, `terminal.put`, `terminal.label`, `terminal.attach` (framed mode, wire format not documented), `machine.exec`, `file.read`, `file.write`, `file.edit`, `doc.get`, `doc.set`, `server.*`, `machine.*`.

### Event shapes

The schema does not describe the payloads of fact changes. These shapes come from the probes, in `daemon/testdata/events-probe.jsonl`:

- **Snapshot:** `{"channel":null,"seq":N,"terminals":[Terminal, …]}`.
- **Notifications:**
  - `{"jsonrpc":"2.0","method":"event","params":{"type":T,"terminal":"t2c","seq":N,"data":D}}`
  - `{"jsonrpc":"2.0","method":"heartbeat","params":{"seq":N}}`.
- **`data` by type:**

  | Type         | `data`                                                     |
  | ------------ | ---------------------------------------------------------- |
  | `created`    | The full Terminal.                                         |
  | `status`     | `{records:{}, reported, status:{app, state, kind?, msg?}}` |
  | `turn`       | `{state, turn}`                                            |
  | `title`      | `{title}`                                                  |
  | `labels`     | `{labels}`                                                 |
  | `size`       | `{cols, rows}`                                             |
  | `foreground` | `{pid, name, argv, cwd}`                                   |
  | `finished`   | `{exit_code, finished_ms, reason, signal}`                 |
  | `removed`    | `{}`                                                       |

- `missed` and the snapshot after it were not seen. The daemon treats any notification whose params carry a `terminals` list as a snapshot.
- hc sends the status and the turn as two events. The turn goes up after the blocked status arrives.
- The fixture keeps only three `title` events for each terminal.

## Terminal facts

`hc get <t> --json` returns:

- `address`, `cols`, `rows`, `created_ms`, `reported`, `turn`, `title`, `state` (`live` or `finished`).
- `foreground {pid, name, argv, cwd}`. Names differ: `claude`, `claude.exe`, and `node` for npm Codex.
- `labels`. Herdr's own terminals have `herdr.space`, `herdr.tab` and `where`, for example `"where": "Niti / Niti Phase 0 PR"`.
- `status {state, app, kind, msg, title, progress}`:
  - `state` is `idle`, `working`, `done`, `blocked` or `error`.
  - `kind` is `permission`, `question` or `auth`. It is set only when the state is `blocked`.
  - `turn` goes up each time the status leaves `working`.
- A program reports its status over OSC 7501. A plain terminal shows as an agent if it prints `ESC ] 7501 ; state=idle BEL`.

## Live stream

- `hc attach --observe [--history N] <t>` writes the first paint and then the live bytes to a plain pipe, with colors.
- It needs no TTY. It never types and never resizes.
- A probe gave 12 KiB of colored bytes in 3 s with `--history 20`, with the SGR colors intact.

## Window model

- `herdr-app ls` returns `{spaces: [{id, name, folder, machine, active, tabs: [{id, name, active, terminals: [{terminal, pane, status, app, name, sub_agents}]}]}], recent}`.
- `sub_agents` is a list of terminal addresses (strings), not objects.
- `hc ls` and `herdr-app ls` both work with `env -i HOME=$HOME PATH=/usr/bin:/bin`. So a LaunchAgent can run them.
- A space rename does not cause an hc event.

## Herdr mods

- A mod is trusted JavaScript in `~/.config/herdr-app/mods/<name>/` (`main.js`, `mod.json`). It can run programs with `exec` and use the network (`~/.config/herdr-app/AGENTS.md:971, 996`).
- `exec` has a limit of 30 s and 8 MiB of output (`herdr.d.ts:10`).
- `image(path)` and `svg(path)` load files from the app directory only. `div()` is a plain layout element (`gpui-kit.d.ts:20271` to `20290` in each shipped mod folder).
- A save of any file in a mod folder reloads the mod. Hidden files, `.d.ts`, `jsconfig.json` and `node_modules` are exceptions (`AGENTS.md:789`).
- Each mod redraw redraws the full window. So a mod must not redraw on a timer (`AGENTS.md:893`).
- `mod.open {name}` opens or closes a mod as the overlay that its `mod.json` describes (`herdr-app commands`).
- `herdr-app check <dir>` validates a mod. `herdr-app shot <file>` captures the window.
- The shipped mods belong to Herdr. herdr-remote does not change them.

## Claude Code permission prompts

Verified with probe agents in `--permission-mode default`:

- **Command prompt:**
  - Status: `{app: "claude-code", kind: "permission", msg: "approve Bash: touch <path>", state: "blocked"}`.
  - Screen: `Do you want to proceed?`, `1. Yes` (with the cursor on it), `2. Yes, and always allow …`, `3. Yes, and switch to auto mode …`, `4. No`, then `Esc to cancel · Tab to amend`.
- **Write-tool prompt:**
  - Status `msg`: `approve Write: <path>`.
  - Screen: `Do you want to create <file>?`, `1. Yes` (with the cursor on it), `2. Yes, and switch to accept edits …`, `3. No`, then `Esc to cancel · Tab to amend`.
- **Keys:**
  - `1` approves. The file appeared, the status went to `done`, and the turn went from 1 to 2.
  - `esc` denies. No file appeared, the status went to `idle`, and the turn did not change.
  - "No" was option 4 in one prompt and option 3 in the other. So a digit is not a safe way to deny.
- **Markers that the daemon checks:** `1. Yes` and `Esc to cancel`.
- **Auto mode:** Claude in `--permission-mode auto` ran `touch` with no prompt. So auto-mode prompts were not seen.

## Codex approval prompts

Verified with `codex --no-daemon -s read-only -a on-request`. This version rejects `-a untrusted`.

- **Command prompt:**
  - Status: `{app: "codex", kind: "permission", state: "blocked"}`, with no `msg`.
  - Screen: `Would you like to run the following command?`, `Reason: …`, `$ <command>`, `1. Yes, proceed (y)`, `2. Yes, and don't ask again … (p)`, `3. No, and tell Codex what to do differently (esc)`, `Press enter to confirm or esc to cancel`.
- **File-edit prompt:**
  - Screen: `Would you like to make the following edits?`, `Description: …`, `Destination: …`, `1. Yes, proceed (y)`, `2. Yes, and don't ask again for these files (a)`, `3. No, … (esc)`, `Press enter to confirm or esc to cancel`.
- **Keys:** `y` approves and `esc` denies, for both prompts. After `esc`, the Codex turn goes up by one. The Claude turn does not.
- **Markers that the daemon checks:** `Yes, proceed (y)` and `(esc)`.

## Transcript files

### Claude Code

- `foreground.pid` leads to `~/.claude/sessions/<pid>.json`. Its keys include `sessionId`, `cwd` and `updatedAt`.
- `sessionId` leads to `~/.claude/projects/<slug>/<sessionId>.jsonl`.
- Records with `type` `user` or `assistant` have `message.content`. It is a string, or a list of `text`, `thinking`, `tool_use` and `tool_result` blocks. Other types (`attachment`, `mode`, `ai-title` and more) are not messages.
- The sessions file appears after the first prompt.

### Codex with `--no-daemon`

- The `codex` process under the foreground process keeps its rollout file open: `~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<thread id>.jsonl` (`lsof -p`).
- Records are `session_meta`, `response_item` (payload `message`, `custom_tool_call`, `custom_tool_call_output`), `event_msg`, `turn_context` and others.
- Guardian sub-agent threads have their own rollout files, with `session_meta.parent_thread_id` set. The main thread has no `parent_thread_id`.

### Codex in daemon mode (plain `codex`)

- A plain `codex` runs its threads in a shared `codex app-server` daemon, whose parent is pid 1.
- No process of the terminal holds the rollout file. The daemon closes rollouts that are idle.
- Each live thread has a lock file `~/.codex/thread-writer-locks/<thread id>.lock`. The process that serves the thread holds the lock:
  - the app-server, for daemon-mode threads
  - the `codex` process itself, for `--no-daemon` threads.
- A Codex that has had no prompt has a lock but no rollout yet.
- The first line of each rollout is `session_meta`, with `cwd`, `source` (`cli`, `vscode`, or an object for sub-agents), `timestamp`, and `parent_thread_id` for sub-agents.
- No shipped interface ties a daemon-mode terminal to its thread. The files in `~/.codex/tui-thread-reference-capabilities/` are empty markers.
- A match by folder can pick the session of another Codex client: one outside Herdr, one that exited, or one that was resumed. So herdr-remote shows terminal text for daemon-mode Codex ([ADR 0009](adr/0009-transcripts.md)).
- Herdr's own launcher for Codex runs plain `codex`: `~/.config/herdr-app/shared.json`, `new-menu.agents[1].run = ["codex"]`. `hc --skill` says that Codex under hc must run with `--no-daemon`.

## macOS

- `ioreg -c IOHIDSystem -d 4` prints `HIDIdleTime`: the nanoseconds since the last keyboard or mouse input on the Mac.
- A sleeping display is fine for the idle guard, because the idle time only grows.

## Assumptions

These facts are not documented by their tools. Each one has a safe fallback:

- `~/.claude/sessions/<pid>.json`, the Claude projects layout and the Codex rollout layout can change. Then the terminal-text fallback applies.
- The prompt markers match Claude Code 2.1.x and Codex 0.167.x. A change makes the prompt view-only, which is safe. Then update the markers in `daemon/actions.go`.
- Claude auto-mode prompts were not seen. They probably use the same dialog. If not, they are view-only.

## Check after a tool update

1. Print the versions: `hc --version`, `herdr-app --version`, `claude --version`, `codex --version`.
2. Run the [probe agents](testing.md#probe-agents) and compare the prompts and keys with this page.
3. Run `hc attach --observe` with the output to a file. Check for colored bytes.
4. Run `ioreg -c IOHIDSystem -d 4 | grep HIDIdleTime`.
5. Check a transcript for a Claude agent and for a `codex --no-daemon` agent on the phone.

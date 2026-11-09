# Plan: herdr-remote

## 1. Outcome

Build an Android app that controls Herdr.app sessions on this Mac, and the Mac-side daemon it talks to, in `/Users/zeus/code/herdr-remote`. There are three parts:

- **`herdr-remoted`**, a Go daemon that runs as a LaunchAgent. It is the only thing the phone can reach. It serves a small HTTPS API over mutual TLS and talks to Herdr only through the shipped `hc` and `herdr-app` commands. A fixed set of operations is built into the code.
- **A Herdr mod, `herdr-remote`**, opened as an overlay in the Herdr window. It shows the pairing QR code, asks you to confirm a new phone, and lists paired phones with a Revoke button.
- **The Android app.** You scan the QR code, check that the 6-digit code on the phone matches the Mac, and click Allow on the Mac. The app then shows:
  - spaces, the agents in each space, and all agents;
  - per agent: the transcript, a live colored view of its terminal, Approve or Deny for a permission prompt, and a box to send a message;
  - an optional background service that raises a notification when an agent becomes blocked.

The daemon keeps an audit log of every pairing, revocation, decision and message. It refuses an approval from the phone while someone is using the Mac.

What you get:
- You can watch and unblock Claude Code and Codex agents from the phone, over Wi-Fi or over Tailscale when you are away.
- A phone that is not paired, or has been revoked, never gets past the TLS handshake.
- Nothing on the phone can run commands, read files or reach any `hc` method outside the fixed set.

## 2. Current behaviour

The repo `/Users/zeus/code/herdr-remote` is new: branch `main`, no commits. Herdr is closed source. We inspected Herdr.app 0.1.13, herdr-core `hc` 0.1.18, Claude Code 2.1.295 and codex-cli 0.167.0 on 2026-11-08.

- **herdr-core API.** JSON-RPC 2.0, one message per line, on a local socket. `hc connect` speaks it over stdin and stdout (`hc connect --help`). `hc schema` prints the full OpenRPC document.
  - **Read:**
    - `terminal.list` (`labels` filter, `all`)
    - `terminal.get`
    - `terminal.read {terminal, lines}`: the screen plus `lines` of history, as text. `lines: 0` is the visible screen only.
  - **Events:** `events.subscribe` returns `{seq, terminals[]}`. The connection then carries only notifications:
    - fact changes (created, lost, restarted, finished, removed, status, turn, title, labels, size, doc);
    - `heartbeat {seq}` every 15 s;
    - `missed` plus a fresh snapshot for a subscriber that falls behind.

    The schema does not describe the payloads of the change notifications. A 20 s sample showed only the snapshot and `{"method":"heartbeat","params":{"seq":7637}}`.
  - **Act:**
    - `terminal.send {terminal, text, force, if, wait, timeout}` pastes text and presses Enter. It refuses a blocked program unless `force` is set, and `if` accepts only `idle` or `working`.
    - `terminal.keys {terminal, keys[]}` presses named keys and returns `KeysResult {status, turn}`. It has no blocked refusal and no status condition.
    - Both wait until nobody has typed at the Mac for about 3 s; keys a person presses meanwhile are held.
    - A `terminal.send` from a process outside any terminal, with no `caller`, arrives as a plain prompt with no `<hc from=…>` tag. Verified: a probe Claude agent answered "PONG" to an untagged message sent with `env -i … hc send`.
  - **History while busy:** `terminal.read` returns only the visible screen, marked incomplete, while the status is `working`, `blocked` or `error` (`hc --skill`, "History reads").
  - **Never exposed:** `terminal.run`, `terminal.kill`, `terminal.rm`, `terminal.put`, `terminal.label`, `terminal.attach` (framed mode; its wire format is not documented in the schema), `machine.exec`, `file.read/write/edit`, `doc.get/set`, `server.*` and `machine.*`.
- **Terminal facts** (`hc get <t> --json`):
  - `address`, `cols`, `rows`, `created_ms`, `reported`, `turn`, `title`, `state` (`live` or `finished`);
  - `foreground {pid, name, argv, cwd}`;
  - `labels`: Herdr's own terminals carry `herdr.space`, `herdr.tab` and `where`, for example `"where": "Niti / Niti Phase 0 PR"`;
  - `status {state, app, kind, msg, title, progress}`. `state` is `idle`, `working`, `done`, `blocked` or `error`. `kind` is `permission`, `question` or `auth`, and is set only when blocked. `turn` goes up each time the status leaves `working`.
  - Foreground names vary: `claude`, `claude.exe`, and `node` for npm Codex.
- **Live stream.** `hc attach --observe [--history N] <t>` writes the first paint and then the live bytes to a plain pipe, with colors. It needs no TTY, and it never types or resizes. We verified this with a probe terminal: 507 bytes in 2 s, SGR colors intact.
- **Window model.**
  - `herdr-app ls` returns `{spaces: [{id, name, folder, machine, active, tabs: [{id, name, active, terminals: [{terminal, pane, status, app, name, sub_agents}]}]}], recent}`.
  - `hc ls` and `herdr-app ls` both work under `env -i HOME=$HOME PATH=/usr/bin:/bin`, so a LaunchAgent can run them.
  - Renaming a space produces no hc event.
- **Mods.**
  - Trusted JavaScript in `~/.config/herdr-app/mods/<name>/` (`main.js`, `mod.json`). Mods can `exec` programs and use the network (`~/.config/herdr-app/AGENTS.md:971, 996`).
  - `image(path)` and `svg(path)` load files from the app directory only. `div()` is a plain layout element (`gpui-kit.d.ts:20271-20290` in each shipped mod folder).
  - `mod.open {name}` toggles a mod as the overlay its `mod.json` describes (`herdr-app commands`).
  - `herdr-app check <dir>` validates a mod, and `herdr-app shot <file>` captures the window.
  - Shipped mods belong to Herdr and must not be edited.
  - `exec` has a 30 s limit and 8 MiB of output (`herdr.d.ts:10`).
  - Saving any file in a mod folder reloads the mod, except hidden files, `.d.ts`, `jsconfig.json` and `node_modules` (`AGENTS.md:789`).
  - Every mod redraw redraws the whole window, so mods must not redraw on a timer (`AGENTS.md:893`).
- **Claude Code permission prompt** (verified with a probe agent in `--permission-mode default`):
  - Status: `{app: "claude-code", kind: "permission", msg: "approve Bash: touch <path>", state: "blocked"}`.
  - Screen: `Do you want to proceed?`, then `❯ 1. Yes`, `2. Yes, and always allow …`, `3. Yes, and switch to auto mode …`, `4. No`, and `Esc to cancel · Tab to amend`.
  - `hc keys <t> 1` approved: the file was created, and the status went to `done` with the turn moving from 1 to 2.
  - `esc` denied: no file, status `idle`, turn unchanged.
  - "No" was option 4, so a digit is not a safe deny.
  - A Write-tool prompt has the same markers: `Do you want to create <file>?`, `❯ 1. Yes`, `2. Yes, and switch to accept edits …`, `3. No`, `Esc to cancel · Tab to amend`, with `msg: "approve Write: <path>"`.
- **Codex approval prompt** (verified with `-s read-only -a on-request`; this version rejects `-a untrusted`):
  - Status: `{app: "codex", kind: "permission", state: "blocked"}`, with no `msg`.
  - Screen: `Would you like to run the following command?`, `Reason: …`, `$ <command>`, `1. Yes, proceed (y)`, `2. Yes, and don't ask again … (p)`, `3. No, and tell Codex what to do differently (esc)`, `Press enter to confirm or esc to cancel`.
  - `y` approved and `esc` denied.
  - A file-edit approval has the same markers: `Would you like to make the following edits?`, `Description: …`, `Destination: …`, `1. Yes, proceed (y)`, `2. Yes, and don't ask again for these files (a)`, `3. No, … (esc)`, `Press enter to confirm or esc to cancel`.
- **Transcripts.**
  - **Claude Code:** `foreground.pid` leads to `~/.claude/sessions/<pid>.json` (keys include `sessionId`, `cwd`, `updatedAt`). That leads to `~/.claude/projects/<slug>/<sessionId>.jsonl`.
    - Records have `type` `user` or `assistant`, whose `message.content` is a string or blocks of `text`, `thinking`, `tool_use` and `tool_result`. Other types (`attachment`, `mode`, `ai-title`, …) are skipped.
    - The sessions file appears after the first prompt.
  - **Codex:** the `codex` process under the terminal's foreground process keeps its rollout file open, `~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<id>.jsonl` (`lsof -p`).
    - Records are `session_meta`, `response_item` (payload `message`, `custom_tool_call`, `custom_tool_call_output`), `event_msg`, `turn_context` and others.
    - Guardian subagent threads have their own rollout files with `session_meta.parent_thread_id` set; the main thread has none.
    - This holds only for Codex started with `--no-daemon`. A plain `codex` (for example t1v, `node /opt/homebrew/bin/codex`) runs its threads in a shared `codex app-server` daemon (parent pid 1). No process of the terminal holds the rollout file, and the daemon holds lock files under `~/.codex/thread-writer-locks/` for every thread it serves. A Codex that has not been prompted has no rollout yet.
    - The first line of every rollout is `session_meta`, with `cwd`, `source` (`cli`, `vscode`, or a subagent object), `timestamp`, and `parent_thread_id` for subagents.
- **Mac idle time.** `ioreg -c IOHIDSystem -d 4` prints `HIDIdleTime` in nanoseconds since the last keyboard or mouse input on the Mac.
- **Toolchain.** Java 17 (Homebrew openjdk@17), Go and Swift are installed. The Android SDK, `adb`, Gradle and Tailscale are not. Homebrew has the `android-commandlinetools` cask. The Mac's LAN address was 192.168.1.3.

## 3. Decisions

| # | Decision | Why | Strongest rejected alternative |
|---|---|---|---|
| 1 | One Go daemon, `herdr-remoted`, runs as a per-user LaunchAgent. It is the only network listener. | One static binary with a good TLS stack and simple subprocess control. Go is installed. | Swift: the same capability with more macOS-only code. A mod as the server: mods are trusted code with no isolation and a 250 ms quit budget. |
| 2 | The daemon reaches hc through three long-lived `hc connect` subprocesses: one for reads, one for `events.subscribe`, one for actions. It runs `herdr-app ls`, `hc attach --observe`, `lsof`, `ps`, `ioreg` and, if present, `tailscale status --json` through one helper that only accepts fixed program paths and argument shapes. There is no generic pass-through. | The raw API includes `machine.exec` and `file.write`. Actions wait for local typing to stop and must not stall reads, and a subscribed connection carries only notifications. | Proxying JSON-RPC with a method filter: one filter bug exposes everything. |
| 3 | Allowed hc use:<br>- `hello`, `terminal.list`, `terminal.get`, `terminal.read`, `events.subscribe`;<br>- `terminal.send` with `force: false`;<br>- `terminal.keys` with exactly one of `["1"]`, `["y"]`, `["esc"]`, and only from the approval code;<br>- `hc attach --observe`.<br>Targets must match `^t[0-9a-z]+$` (no `machine/`), be `live` and have `reported: true`. | It is the minimum for the requested screens, and it fails toward refusing. | `terminal.interrupt` or a keyboard: out of scope by user decision. |
| 4 | Two TLS listeners share one P-256 self-signed server key, which the phone pins by SPKI SHA-256.<br>- The **paired listener** on port 7766 is always on.<br>- The **enrollment listener** on port 7767 exists only while a pairing window is open, takes any client certificate, and serves only `POST /v1/enroll`.<br>Go settings on both listeners:<br>- `MinVersion: tls.VersionTLS13`;<br>- `ClientAuth: tls.RequireAnyClientCert`, since there is no CA pool;<br>- `SessionTicketsDisabled: true`;<br>- checks in `VerifyConnection`, which runs on every handshake including resumptions, never in `VerifyPeerCertificate`, which is skipped on resumed connections.<br>What `VerifyConnection` checks differs by listener:<br>- On **7766**, the client SPKI hash must be in the device store, and every request looks the device up again.<br>- On **7767**, only that the client presented one well-formed P-256 certificate. Authorization there is the token, bound to that SPKI, and the device store is never consulted. | The handshake cannot see the request path, so one listener would have to accept unknown keys. Two listeners keep "the paired listener never accepts an unknown key" true at all times. Disabling tickets and checking in `VerifyConnection` stops a revoked phone from resuming an old session. Pinning by key makes LAN and Tailscale addresses work alike. | One listener with path middleware: unknown keys reach HTTP parsing whenever pairing is open. A relay: ruled out by the user. |
| 5 | Plain HTTP on top of the TLS listeners:<br>- JSON for reads and actions;<br>- Server-Sent Events for `/v1/events`, with a comment line every 10 s;<br>- a chunked `application/octet-stream` body for the live view.<br>Read timeouts on the phone:<br>- SSE: 30 s, then reconnect, so a half-open socket after a network change or Mac sleep is detected.<br>- Live stream: no read timeout, because an idle terminal sends nothing. Every live view is reopened when the SSE connection reconnects.<br>- `/v1/enroll`: 130 s, longer than the 120 s window it long-polls.<br>Other settings:<br>- The daemon's listeners use TCP keepalive (Go's default 15 s).<br>- Action calls use `retryOnConnectionFailure(false)`.<br>Go uses the standard library only, plus a QR encoder. | The live view is watch-only, so no two-way socket is needed. The SSE keepalive gives a reliable liveness signal for the whole connection. | WebSocket: needs a Go dependency and adds nothing for one-way streams. No read timeout on SSE: a dead socket is never noticed. |
| 6 | Pairing:<br>1. You open the overlay (`herdr-app do mod.open --name herdr-remote`; opening it again closes it). It shows the paired devices with Revoke, and a **Pair a phone** button. Only that button runs `herdr-remoted pair --json`, which opens a 120 s pairing window and returns `{matrix, expires_at}`. Closing the overlay cancels an open window.<br>2. The overlay draws the QR matrix as run-length `div()` segments per row and shows the expiry as a clock time. Nothing is written to disk, and there is no countdown redraw.<br>3. The QR holds `herdr-remote://pair?v=1&pin=<b64url SPKI sha256>&ep=<addr>,<addr>&p=7766&ep_port=7767&t=<token>&exp=<unix>`. The token is 32 random bytes, kept in memory only.<br>4. The phone makes a non-exportable P-256 key in Android Keystore and calls `/v1/enroll {token, name}` on 7767. `name` must be 1 to 64 printable characters. The token binds to the first client SPKI that uses it, and any other SPKI is refused.<br>5. The overlay polls `herdr-remoted pair status --json` every second (each call returns at once, well under the 30 s `exec` limit). It redraws only when the state changes, to show the device name and a 6-digit code from SHA-256(server SPKI ‖ client SPKI). The phone shows the same code.<br>6. Allow (`pair allow`) stores the device; Deny (`pair deny`) drops it. `/v1/enroll` long-polls until Allow, Deny or expiry. Then the window and the 7767 listener close.<br>The mod calls the daemon by the absolute path that `install.sh` writes into it. | A QR can leak, for example over a screen share or in a photo. Requiring Allow on the Mac means a leaked QR alone enrolls nothing. `image()` only loads files, so drawing with `div()` keeps the token off disk. A blocking `pair wait` would hit the 30 s `exec` limit. | Token-only enrollment: a QR seen over a screen share is enough to get in. A hidden SVG file in the mod folder: avoids the reload, but puts the token on disk. |
| 7 | Revocation: `herdr-remoted devices rm <id>`, or Revoke in the overlay, deletes the device, closes its connections and ends its live-view subprocesses. Every request looks the device up again. | A lost phone loses access at once, including open streams. | Certificate expiry only: too slow. |
| 8 | Live view:<br>- `GET /v1/terminals/{t}/live` runs `hc attach --observe --history 500 <t>` and copies its stdout to the response, flushing each read.<br>- The phone renders it with xterm.js, bundled in a locked-down WebView, at the terminal's own `cols` and `rows`, scaled to fit the width with pinch zoom.<br>- Bytes reach xterm.js in base64 chunks of at most 64 KiB, each acknowledged by xterm's `write` callback. Kotlin keeps at most 1 MiB pending; above that it drops the queue and reopens the stream for a fresh paint.<br>- When a `size` event arrives, the daemon ends that terminal's live streams, and the phone reopens one for a fresh paint at the new size.<br>- At most 2 live views per device. The subprocess ends when the response ends. | Verified to stream colored, incremental bytes with no PTY and no protocol decoding. A fresh paint from `attach` makes dropping a backlog safe, so overflow recovery is a reopen. | `terminal.read` polling at 4 Hz: plain text, no color, 250 ms steps. Unbounded writes: xterm.js discards data past its write buffer limit. |
| 9 | Transcripts, tried in order:<br>1. **Claude:** `foreground.pid` → `~/.claude/sessions/<pid>.json` → `sessionId` → the single match of `~/.claude/projects/*/<sessionId>.jsonl`.<br>2. **Codex started with `--no-daemon`:** walk the descendants of `foreground.pid` (with `ps`) to the process named `codex`. Run `lsof -Fn -p <pid>` and keep the open `~/.codex/sessions/**/rollout-*.jsonl`. Choose the one whose `session_meta` has no `parent_thread_id`.<br>3. **Codex in daemon mode (plain `codex`):** bind by elimination, and only when it is unambiguous:<br>- *Live daemon threads:* the thread IDs whose lock files under `~/.codex/thread-writer-locks/` are held by a `codex app-server` process (`lsof +D`). The rollout file name ends in the thread ID.<br>- *Candidates:* rollouts of live daemon threads whose first-line `session_meta` has `cwd` equal to the terminal's foreground cwd, `source` `cli`, no `parent_thread_id`, and `timestamp` at or after the terminal's `created_ms`, and that are not already bound to another terminal by sources 1 or 2.<br>- *Peers:* live terminals with `app` `codex` and the same foreground cwd that sources 1 and 2 could not bind.<br>- The binding holds only if there is exactly one candidate and exactly one peer, this terminal. It is labelled "matched by folder". Anything else means the next item.<br>4. **Terminal text:** `terminal.read` with `lines: 2000`, labelled "terminal text". While the agent is working or blocked, this is the visible screen only.<br>Rules for every source:<br>- Real paths (after following symlinks) must lie under the roots.<br>- A page is a byte range read backwards from the end, at most 2 MiB. A page that starts inside a file drops everything up to the first newline. A record longer than a page shows as `[record too large]`.<br>- The cursor is `{file, inode, offset}`. A cursor for another file returns 409, and the phone restarts from the end. | Explicit identities where they exist, verified on live agents. Folder matching is only used when it is unambiguous. The `setup` docs recommend `codex --no-daemon`, which `hc --skill` already requires for hc. | A cwd-and-time guess without the exactly-one rule: shows the wrong session when two agents share a folder. Reading Codex's `state_5.sqlite`: an undocumented database and a SQLite dependency for the same answer. |
| 10 | Approvals apply only when `state = blocked`, `kind = permission` and `app` is `claude-code` or `codex`.<br>1. `GET /prompt` returns `{msg, block, turn, prompt_hash}`. `block` is the screen region from the line that starts the dialog (`Do you want` or `Would you like`) to the line with `Esc to cancel` or `Press enter to confirm`. `prompt_hash` is SHA-256(`app` ‖ `turn` ‖ `msg` ‖ `block`).<br>2. `POST /decision {request_id, decision, turn, prompt_hash}` first looks up `request_id`. A seen ID returns its recorded result, and an ID that was dispatching when the daemon stopped returns `outcome_unknown`.<br>3. Under a per-terminal lock it re-reads `terminal.get` and `terminal.read(lines 0)`. It requires the same state, kind, app and turn; the same hash; the option markers (Claude: `1. Yes` and `Esc to cancel`; Codex: `Yes, proceed (y)` and `(esc)`); and that the terminal's consumed marker is not already `{turn, prompt_hash}`.<br>4. **Mac-in-use guard**, a heuristic, run last, immediately before dispatch:<br>&nbsp;&nbsp;- If `HIDIdleTime` is below the `mac_idle_seconds` setting (default 10; 0 turns the guard off), it returns 409 `mac_in_use` with the measured idle time, and audits the refusal.<br>&nbsp;&nbsp;- If `ioreg` fails or its output cannot be parsed, it refuses (fail closed).<br>&nbsp;&nbsp;- It sees local keyboard and mouse input. It does not see SSH sessions, `hc send` or `hc keys` from other terminals or agents (hc's own 3 s quiet rule covers those), and remote-desktop input is unverified.<br>5. It appends a `dispatching` audit line `{request_id, terminal, turn, prompt_hash, decision}` and fsyncs it. If that write fails: 503 and no key. Then it sets the marker and presses one key: Claude `1` or `esc`, Codex `y` or `esc`.<br>6. Afterwards it reads `HIDIdleTime` again. If the Mac saw input during the call, the result is `uncertain`. Otherwise it reads the status for up to 3 s and returns `applied` (the status or turn moved) or `sent_unconfirmed`. A result audit line follows. It never presses again.<br>Any mismatch returns 409 and presses nothing. On start, the daemon rebuilds request IDs and consumed markers from the last 10 minutes of `audit.log` and `audit.log.1`. | The keys and markers were verified on command and file-edit prompts for both agents. `1` selects Yes whatever the cursor; `esc` is the only deny that does not depend on option order. The consumed marker and request IDs stop retries and second phones. The audit log makes that survive a restart at no extra cost. The idle guard closes the main path to the residual race: someone answering at the Mac while the phone answers too. | A separate durable action store: the audit log already records every action. Accepting the race unmitigated: it could approve an action nobody saw. Pressing `enter` when Yes is selected: depends on the cursor. |
| 11 | Questions (`kind: question`), `auth`, Pi and unknown apps are view-only. The phone says "answer this on the Mac". | Control is scoped to approve, deny and message. Question menus vary, so a digit could pick the wrong answer. | Digits 1 to 9: a wrong parse answers for you. |
| 12 | Messages: `POST /message {request_id, text}` → `terminal.send` with `force: false` and no `caller`, so the text arrives untagged, like typing. It goes only to a live terminal with `reported: true`.<br>- C0 and C1 control characters are removed except `\n` and `\t`.<br>- The text is capped at 4000 characters.<br>- `request_id` is handled as in decision 10: a fsynced `dispatching` line `{request_id, terminal, payload_sha256}` comes before the send, and a write failure gives 503 with nothing sent. Delivery is at most once per `request_id`.<br>- A blocked agent refuses the message, and the phone shows why. | It matches the scope. Removing escapes keeps input from breaking out of the paste. Request IDs stop a retried POST from sending the prompt twice. Verified untagged delivery from a process outside any terminal. | Raw keys: out of scope. Passing a `caller`: tags the text as a report from another agent. |
| 13 | Background alerts:<br>- An opt-in Android foreground service holds `/v1/events`. It posts a notification when a terminal becomes blocked, de-duplicated by terminal, turn and `msg`. On reconnect, the snapshot replays agents that are still blocked.<br>- Manifest: `FOREGROUND_SERVICE`, `FOREGROUND_SERVICE_SPECIAL_USE`, `POST_NOTIFICATIONS`, `RECEIVE_BOOT_COMPLETED`, `REQUEST_IGNORE_BATTERY_OPTIMIZATIONS`, `INTERNET` and `USE_BIOMETRIC`. The service has `android:foregroundServiceType="specialUse"` with the `PROPERTY_SPECIAL_USE_FGS_SUBTYPE` property.<br>- `BOOT_COMPLETED` and `MY_PACKAGE_REPLACED` receivers restart it when the toggle is on.<br>- The service is started from the app in the foreground. It shows an ongoing notification, "Watching N agents".<br>- With notifications denied, the toggle explains why and stays off.<br>- There is no push server. Alerts arrive only while the service runs and is connected. | No relay or push service of ours, as decided. `specialUse` is not one of the types Android 15 blocks from `BOOT_COMPLETED`. | FCM: needs a server and Google credentials. `dataSync`: limited to 6 hours a day on Android 15. |
| 14 | Phone key and app privacy:<br>- The Keystore key is P-256, `PURPOSE_SIGN`, digests `SHA256` and `NONE`, `setUserAuthenticationRequired(false)`, and not unlock-bound. Its chain is the self-signed certificate Keystore generates.<br>- Opening the app after 5 minutes in the background asks for BiometricPrompt (strong biometric or the device credential).<br>- The activity sets `FLAG_SECURE`, so recents previews and screenshots show nothing.<br>- Notifications use `VISIBILITY_PRIVATE`. The public version says "An agent needs attention"; the private one shows the tab name and "permission", never the command.<br>- Transcripts and terminal bytes are held in memory only.<br>- No build, debug or release, logs transcript text, terminal bytes, message text, tokens or codes.<br>- Approve, deny and message work only in the unlocked, authenticated app, and notifications carry no action buttons. | An unlock-bound key makes every handshake fail while the phone is locked, which stops alerts. The gate, `FLAG_SECURE` and private notifications keep project data off the lock screen and out of recents. | `setUnlockedDeviceRequired`: kills background alerts. Full detail in notifications: shows commands on the lock screen. |
| 15 | Android stack: Kotlin and Jetpack Compose, minSdk 31, targetSdk 35. OkHttp for TLS, SSE and streams; kotlinx.serialization; the Google Play services code scanner (no camera permission); androidx.biometric; xterm.js 5.x as a bundled asset. One activity, ViewModels, no DI framework. | It is the smallest standard stack. | Termux terminal-view: native rendering, but a JitPack dependency and more glue. React Native or Flutter: more toolchain. |
| 16 | Spaces and agents:<br>- `/v1/spaces` runs `herdr-app ls` on each request, and also on created, removed and labels events. It drops spaces whose `machine` is set.<br>- An agent is a live local terminal with `reported: true`. Its space comes from `herdr-app ls`, else from its `herdr.space` label.<br>- If Herdr.app is not running, agents are grouped by their `where` label under "Herdr window closed". | Renames emit no hc event. Remote machines are a non-goal. | A periodic poll timer: a request-time refresh is simpler and always current. |
| 17 | Logging and audit:<br>- `audit.log` (JSON lines, 0600) sits in the state directory. Each line is `{ts, device_id, action, terminal, turn, request_id, decision, prompt_hash, payload_sha256, idle_ms, result}`, with the fields that apply. Lines are written for enroll, allow, deny, revoke, every decision (`dispatching`, then the result), every refusal, and every message (`dispatching` with the SHA-256 of the text, then the result).<br>- `dispatching` lines are fsynced before any key or send.<br>- Request IDs are valid for 10 minutes, and the phone abandons an unacknowledged request after 5 minutes ("outcome unknown").<br>- Never logged: tokens, pairing codes, keys, transcript text, terminal bytes, message text.<br>- The LaunchAgent's stdout and stderr go to `daemon.log` (0600) in the same directory, under the same rule.<br>- Both files rotate at 10 MiB, keeping one old file (`.1`). Start-up reads both, so a rotation never drops the last 10 minutes. | You need a trail of what the phone did. The audit log is also the restart-safe store for request IDs (decision 10). | No audit trail: no way to see what a lost phone did. |
| 18 | Threat model boundary: processes running as the same macOS user are trusted. They can already drive hc directly (`machine.exec`, `file.write`), read `server.key` and `devices.json`, or call `herdr-remoted pair allow`, and file modes cannot stop them. The design protects against the network (LAN and tailnet), leaked QR codes, lost phones, second phones, replays and hostile terminal or transcript bytes on the phone. | File permissions cannot separate processes of the same user. Saying so prevents false assurance. | A separate macOS user or a Keychain-protected key: real isolation, but much more setup, and agents can still drive hc themselves. |

## 4. Scope and acceptance criteria

**Delivered:** `daemon/`, `mod/herdr-remote/`, `android/`, and `scripts/setup-android.sh` and `scripts/install.sh` in the repo.

**Excluded:**
- keyboard input into terminals, answering questions, interrupting agents;
- starting or stopping agents, spaces or tabs;
- remote `hc` machines;
- iOS;
- a relay or push service;
- Play Store release;
- changes to Herdr or its shipped mods.

**Acceptance criteria:**
1. `herdr-remoted` runs under a LaunchAgent and comes back after a reboot. A TLS client without a certificate, or with an unpaired one, fails the handshake on 7766. Port 7767 is closed outside a pairing window.
2. Opening the overlay shows the paired devices. **Pair a phone** shows a QR code. Scanning it shows a 6-digit code on the phone that matches the Mac. After Allow, the phone is paired. With Deny, no Allow, or after 120 s, nothing is stored. A second phone using the same token is refused.
3. The app lists spaces with their Herdr names. A space lists its agents with status. The all-agents view lists every local agent. Status changes on the Mac show on the phone within 2 s.
4. A Claude Code agent's transcript, and a Codex agent's when Codex runs with `--no-daemon` or is the only Codex session in its folder, shows user and assistant messages and tool calls, and pages back across at least three pages. Other agents show labelled terminal text.
5. The live view shows the agent's terminal in color and follows new output within about 1 s on the LAN. It recovers after the pane is resized on the Mac, after sustained heavy output, and after a network change.
6. A blocked Claude Code or Codex permission prompt, for a command or a file edit, can be approved or denied from the phone, and the agent continues accordingly. No key is pressed, and the phone says why, when:
   - the prompt changed;
   - it was answered on the Mac;
   - it was already answered from another phone or by a retry;
   - the Mac had keyboard or mouse input within the `mac_idle_seconds` setting (a heuristic that does not see SSH or other agents' input);
   - the daemon restarted mid-action (the result is `outcome_unknown`).
7. A message from the phone reaches an idle or working agent as a new, untagged prompt, at most once per request, even if the request is retried or the daemon restarts. A blocked agent refuses it, with a reason.
8. With background watching on and the app closed, a newly blocked agent raises one notification within 5 s, while the service runs and is connected. Watching resumes after a phone reboot and an app update.
9. Revoking the phone on the Mac ends its open streams within 2 s, and later handshakes fail, including attempts to resume a TLS session.
10. The same pairing works over Tailscale from a cellular connection.
11. After 5 minutes in the background, the app asks for biometric or device credential before showing anything.
12. The recents screen shows no app content. The lock screen shows only "An agent needs attention".
13. `audit.log` has a line for every pairing event, revocation, decision and message, and contains no token, code, transcript text, terminal bytes or message text.

**Failure cases to handle (no others):**
- hc is unavailable: the daemon returns 503 and reconnects, and the phone shows the last data with its age.
- `missed`: take a fresh snapshot.
- Herdr.app is closed: group agents by `where`.
- The transcript is missing or unparseable: show terminal text.
- An endpoint is unreachable: try the next endpoint, and remember the last one that worked.
- A size change: restart the live stream.
- The device is revoked mid-stream: close the stream.
- The approval guard fails, or the Mac is in use: 409 with a reason.
- An SSE read timeout of 30 s: reconnect, and reopen any live view.
- Live-view backlog over 1 MiB: reopen the stream.
- A transcript cursor for a different file: 409, and restart from the end.
- Notifications denied: background watching stays off, with an explanation.

## 5. Implementation plan

Paths are proposed unless stated otherwise; the repo is empty. Each step: **Change**, **Implements**, **Invariant**, **Verified today**, **Depends on**, **Check**.

**Step 1. Toolchain and skeleton.**
- **Change:**
  - `scripts/setup-android.sh`:
    - `brew install --cask android-commandlinetools` and `brew install gradle`;
    - `sdkmanager --licenses`;
    - `sdkmanager "platform-tools" "platforms;android-35" "build-tools;35.0.0" "emulator" "system-images;android-35;google_apis_playstore;arm64-v8a"`;
    - `avdmanager create avd -n herdr-remote -k "system-images;android-35;google_apis_playstore;arm64-v8a"`.
  - `daemon/go.mod` (module `herdr-remote/daemon`).
  - The `android/` Gradle project with a wrapper, made with `gradle wrapper`.
  - A `.gitignore` covering build output and `local.properties`.
- **Implements:** User decision 3, Decision 15.
- **Invariant:** Only Homebrew packages and the SDK directory change outside the repo.
- **Verified today:** No SDK, `adb` or Gradle; Java 17 present; the cask exists.
- **Depends on:** Nothing.
- **Check:** `go version`; `adb version`; `cd android && ./gradlew assembleDebug` builds an empty Compose app.

**Step 2. Subprocess helper, hc client, event fixtures.**
- **Change:**
  - `daemon/run.go`: the only way the daemon starts a process. It accepts only `hc`, `herdr-app`, `lsof` (including `lsof +D` on the Codex lock folder), `ps`, `ioreg` and `tailscale`, at absolute paths found at start, with fixed argument shapes, no shell and a timeout.
  - `daemon/hc.go`: three `hc connect` connections (reads, events, actions) speaking JSON-RPC lines, with typed functions only:
    - `List`, `Get`, `Read(t, lines)`, `Subscribe`;
    - `Send(t, text)`, which always sends `force: false` and never a `caller`;
    - `Keys(t, k)`, where `k` is one of `1`, `y`, `esc`.

    It validates targets as in Decision 3 and reconnects with backoff.
  - `daemon/idle.go`: `MacIdle()` parses `HIDIdleTime` from `ioreg -c IOHIDSystem -d 4`.
  - Capture real notifications with `hc events --json` while a probe agent goes working → blocked → idle, and save them as `daemon/testdata/events-*.jsonl`. Write the event structs from these.
- **Implements:** Decisions 2, 3 and 10 (idle).
- **Invariant:** No code path sends a method outside the list, sets `force`, sets `caller`, or passes another key. Each fails toward refusing.
- **Verified today:** Methods and parameters in `hc schema`; `hc connect` exists; bare-environment access; `HIDIdleTime` printed.
- **Depends on:** Step 1.
- **Check:** `cd daemon && go test ./...` with a fake `hc`: a test binary that records every request. Tests fail on any method outside the list, `force: true`, a `caller`, any other key, a `machine/` target, or a program outside the helper's list. A parser test covers `MacIdle`.

**Step 3. Identity, device store, pairing, admin socket, audit log.**
- **Change:**
  - `daemon/tls.go`, `daemon/devices.go`, `daemon/pair.go`, `daemon/admin.go`, `daemon/audit.go`.
  - State lives in `~/Library/Application Support/herdr-remote/` (directory 0700):
    - `server.key` (0600) and `server.crt` (self-signed P-256, ten-year validity, subject `CN=herdr-remoted`);
    - `devices.json` (0600, `[{id, name, spki_sha256, added}]`), written atomically;
    - `admin.sock` (0600);
    - `audit.log` and `daemon.log` (0600), per Decision 17.
  - TLS settings exactly as in Decision 4.
  - CLI over the admin socket. Each call returns within 1 s:
    - `pair --json` opens a window and returns `{matrix, expires_at}`;
    - `pair status --json` returns `{state: open|pending|allowed|denied|expired, device_name?, code?}`;
    - `pair allow`, `pair deny`, `pair cancel`;
    - `devices ls --json`, `devices rm <id>`.
  - Endpoints are the private IPv4 addresses of active interfaces plus any 100.64.0.0/10 address. If `tailscale` exists, also the MagicDNS name from `tailscale status --json`, added to the helper's allowed list.
  - The QR matrix comes from a small Go QR library, chosen at implementation; record it in As built.
- **Implements:** Decisions 4, 6, 7, 17 and 18.
- **Invariant:**
  - An unknown key fails the handshake on 7766.
  - A token is single-use, binds to its first SPKI, and lasts 120 s.
  - No device is stored without Allow.
  - Nothing secret is logged.

  Each fails toward refusing.
- **Verified today:** Nothing exists.
- **Depends on:** Step 2.
- **Check:** Go tests:
  - a fresh key, never seen before, enrolls through the real 7767 listener and succeeds after Allow;
  - an Allow given 60 s after the scan still completes the long-poll;
  - Deny, cancel and expiry store nothing;
  - a second SPKI with the same token is refused;
  - an over-long or non-printable `name` is refused;
  - 7767 is closed after the window;
  - an unpaired certificate fails on 7766;
  - after `devices rm`, a client that tries to resume its earlier session fails the handshake (use `tls.Config.ClientSessionCache` in the test client);
  - a scan of `audit.log` after a full pairing finds neither the token nor the code.

  Also, `openssl s_client -connect 127.0.0.1:7766` with no certificate fails.

**Step 4. Read API and events.**
- **Change:**
  - `daemon/api.go`, `daemon/state.go`.
  - Endpoints, all on 7766:
    - `GET /v1/hello` → `{daemon_version, endpoints[]}`;
    - `GET /v1/spaces` → `[{id, name, agents[]}]`;
    - `GET /v1/agents`;
    - `GET /v1/terminals/{t}` → `{address, space, tab, where, app, status, turn, cols, rows, title}`;
    - `GET /v1/events` (SSE). Each event is a terminal summary; there is a `: ka` comment every 10 s, and a `snapshot` event on connect and after `missed`.
  - State follows Decision 16. Errors are JSON `{error, reason}` with 400, 401 (unknown device, after revocation), 404, 409 and 503.
- **Implements:** Decisions 5 and 16.
- **Invariant:**
  - On hc loss: 503, and reconnect.
  - On `missed`: a fresh snapshot.
  - A revoked device's requests and streams end.
- **Verified today:** The `herdr-app ls` shape; snapshot and heartbeat shapes.
- **Depends on:** Step 3.
- **Check:** Go tests with the fake hc and the Step 2 fixtures (snapshot, status change, missed); a test that a renamed space shows on the next `/v1/spaces`; a test that SSE emits `: ka` within 11 s on an idle system. Manual: `curl --cert … --key … -k https://127.0.0.1:7766/v1/agents`.

**Step 5. Live stream.**
- **Change:**
  - `daemon/live.go`: `GET /v1/terminals/{t}/live` runs `hc attach --observe --history 500 <t>` through the helper and copies stdout, flushing each read.
  - Two streams per device.
  - Ends on client disconnect, on a `size` event for `t`, or on revocation.
- **Implements:** Decisions 7 and 8.
- **Invariant:** The phone never writes to the terminal. Every ended stream kills its subprocess.
- **Verified today:** The probe streamed colored bytes to a pipe.
- **Depends on:** Step 4.
- **Check:** Go tests: the subprocess count returns to zero after disconnect, after a size event, and after `devices rm` (the stream closes within 2 s). Manual: `curl -N` shows bytes.

**Step 6. Transcripts.**
- **Change:**
  - `daemon/transcript.go`, with the four sources in Decision 9, in order.
  - `GET /v1/terminals/{t}/transcript?cursor=<opaque>` → `{source: claude|codex|codex_folder|terminal, items: [{role, kind: text|tool_call|tool_result, text, ts}], next_cursor}`.
    - Tool results are cut to 2000 characters, and thinking blocks are skipped.
    - Pages are aligned to records as in Decision 9.
- **Implements:** Decision 9.
- **Invariant:** Only files whose real path is under the two roots are read. A source that cannot resolve falls through to the next one, ending at terminal text. The only error is 409 for a cursor that belongs to a different file.
- **Verified today:** The mapping chains, record types, `parent_thread_id`, and daemon-mode Codex behaviour (section 2).
- **Depends on:** Step 4.
- **Check:** Go tests on fixture JSONL written by the test:
  - Claude and Codex parsing;
  - a partial last line;
  - a page boundary inside a record;
  - a record over 2 MiB;
  - three pages back, with no record repeated or lost;
  - a stale cursor after the file is replaced;
  - a symlink pointing outside the roots, which is refused;
  - two open rollouts where the guardian one is ignored;
  - daemon-mode matching: one candidate and one peer binds; two peers in one folder fall back; a candidate already bound to a `--no-daemon` terminal is excluded; a rollout whose lock was released (session ended) is ignored;
  - a missing sessions file, which falls back to terminal text.

**Step 7. Approvals and messages.**
- **Change:**
  - `daemon/actions.go`.
  - `GET /v1/terminals/{t}/prompt` → `{app, kind, msg, block, turn, prompt_hash, actionable, reason?}`.
  - `POST /v1/terminals/{t}/decision {request_id, decision: approve|deny, turn, prompt_hash}` → `{result: applied|sent_unconfirmed|uncertain|outcome_unknown}`, or 409 `{reason: stale|changed|already_answered|mac_in_use|not_actionable, current}`.
  - `POST /v1/terminals/{t}/message {request_id, text}` → `{result: sent|outcome_unknown}`, or 409 `{reason: blocked|not_agent}`.
  - Decisions 10, 11 and 12 in full, including the per-terminal lock, the consumed marker, the idle guard, request IDs and the audit-log reload on start.
- **Implements:** Decisions 10, 11, 12 and 17.
- **Invariant:** A key is pressed only after every guard passes, at most once per `{turn, prompt_hash}` and per `request_id`, including across a daemon restart. Otherwise 409 and no key, failing toward refusing.
- **Verified today:** The keys and markers for command and file-edit prompts (section 2). The narrow race through `terminal.keys` is known.
- **Depends on:** Steps 3 and 4.
- **Check:** Go tests with the fake hc and a fake `ioreg`:
  - a stale turn, a changed hash, missing option markers, `kind: question`, `app: pi` and a `machine/` target each press nothing;
  - two concurrent decisions press exactly one key;
  - the same `request_id` twice returns the first result, with one key;
  - a restart between `dispatching` and the result returns `outcome_unknown` for that ID, and no key;
  - after a restart, the consumed marker is rebuilt from the audit log, including across a rotation;
  - a failed audit write returns 503 and presses nothing;
  - an `ioreg` failure refuses;
  - a retry after a dropped response, with the fake status held unchanged, presses exactly one key;
  - idle under 10 s returns `mac_in_use`;
  - input during the call returns `uncertain`;
  - a message with ESC and C1 bytes arrives cleaned, and the same message `request_id` sends once;
  - a blocked target refuses a message.

  Manual: the section 6 probes.

**Step 8. Herdr mod and install.**
- **Change:**
  - `mod/herdr-remote/mod.json` (overlay) and `main.js`. The overlay:
    - calls the daemon by the absolute path filled in by `install.sh`;
    - opens to the device list; the **Pair a phone** button calls `pair --json`, and closing the overlay calls `pair cancel`;
    - draws the matrix as run-length `div()` segments and shows the expiry time;
    - polls `pair status --json` every second, calling `cx.notify()` only on a change;
    - shows the device name, the code, Allow and Deny;
    - lists `devices ls --json` with Revoke.
  - `scripts/install.sh`:
    - builds the daemon to `~/.local/bin/herdr-remoted`;
    - writes `~/Library/LaunchAgents/dev.herdr-remote.plist` (RunAtLoad, KeepAlive, StandardOutPath and StandardErrorPath to `daemon.log`) and runs `launchctl bootstrap gui/$(id -u)`;
    - copies the mod to `~/.config/herdr-app/mods/herdr-remote/` without overwriting an existing folder;
    - runs `herdr-app check`;
    - prints the firewall note and the `codex --no-daemon` tip.
- **Implements:** Decisions 1, 6, 7 and 17.
- **Invariant:** No shipped mod or setting is edited. The mod writes no file and holds the token only while open, and no single `exec` call can reach 30 s.
- **Verified today:** The mod APIs, the `exec` limit, the reload rule, `mod.open` and `herdr-app check`.
- **Depends on:** Step 3.
- **Check:** `herdr-app check ~/.config/herdr-app/mods/herdr-remote` passes. `herdr-app do mod.open --name herdr-remote`, then `herdr-app shot /tmp/hr.png`, shows the QR code. Opening the overlay alone leaves port 7767 closed (`nc -z 127.0.0.1 7767` fails). A pairing that takes 60 s still shows Allow. `launchctl print gui/$(id -u)/dev.herdr-remote` shows it running.

**Step 9. Android pairing, connection and app gate.**
- **Change:**
  - `android/app/src/main/...`:
    - the Keystore key per Decision 14;
    - an `X509KeyManager` that returns the Keystore alias, its private key and the chain;
    - an `X509TrustManager` and a `HostnameVerifier` that both accept only a leaf whose SPKI SHA-256 equals the pin, so certificate names and IP addresses play no part;
    - an OkHttp `ConnectionSpec` with TLS 1.3 only;
    - endpoint fallback that remembers the last good endpoint and refreshes endpoints from `/v1/hello`;
    - the Pair screen, with the code scanner, the 6-digit code and the long-poll;
    - the BiometricPrompt gate on start and after 5 minutes in the background;
    - `FLAG_SECURE`.
- **Implements:** Decisions 4, 6, 14 and 15.
- **Invariant:**
  - A pin mismatch never connects.
  - The private key never leaves the Keystore.
  - Nothing is shown before the gate passes.
- **Depends on:** Steps 3 and 4.
- **Check:**
  - JVM tests: QR payload parsing (bad version, expired, missing fields), the pin check in both the trust manager and the hostname verifier, and the 6-digit code matching the Go code on shared test vectors in `testdata/pairing-vectors.json`.
  - Instrumented tests on the emulator: pairing against a dev daemon at `10.0.2.2`; a connection to an endpoint whose certificate has the right SPKI but no matching name succeeds; a wrong SPKI fails.
  - Gate test: the clock is moved past 5 minutes and the gate shows; an action before unlock is refused.

**Step 10. Android lists, events and background service.**
- **Change:**
  - Spaces, Space and All agents screens; the SSE client (30 s read timeout) and reducer; the stale-age banner.
  - The foreground service and manifest per Decision 13, with the settings toggle and the boot and update receivers.
  - Private notifications per Decision 14, de-duplicated by terminal, turn and `msg`.
- **Implements:** Decisions 5, 13, 14 and 16.
- **Invariant:** A stale view shows its age. One notification per blocked episode.
- **Depends on:** Step 9.
- **Check:**
  - JVM tests for the reducer (snapshot, change, missed) and the de-duplication.
  - Manual:
    - swipe the app away, block a probe agent, and a notification arrives;
    - `adb reboot`, then block a probe, and a notification arrives;
    - deny notifications, and the toggle explains;
    - lock the phone, and the lock-screen text is generic;
    - switch Wi-Fi off and on, and the view recovers within 40 s.

**Step 11. Android agent screen.**
- **Change:**
  - Transcript tab, with paging back.
  - Live tab:
    - a WebView with JavaScript on;
    - assets served by `WebViewAssetLoader`, with every other request blocked;
    - no file or content access;
    - xterm.js sized to `cols` and `rows`;
    - the flow control in Decision 8;
    - reopened on a `size` event and on an SSE reconnect.
  - Approve and Deny, showing `msg` and `block`, with each 409 reason and result explained.
  - A Message box. A send whose outcome is unknown says "check the transcript before sending again" and does not retry by itself.
- **Implements:** Decisions 8 to 12.
- **Invariant:** The WebView loads only bundled assets and never sends input. No action is retried automatically.
- **Depends on:** Step 10.
- **Check:**
  - An instrumented test that the WebView refuses a network URL.
  - Manual: approve and deny on probe agents; resize the Mac pane during output, and the view recovers; run `yes | head -c 50000000` in a watched terminal, and the view keeps up or reopens without freezing.

**Step 12. End-to-end acceptance.**
- **Change:** None. Run criteria 1 to 13 on the emulator and a real phone, including Tailscale over cellular.
- **Depends on:** Steps 1 to 11, plus Tailscale installed on the Mac and the phone.
- **Check:** Section 6.

## 6. Testing strategy

| Criterion | Check |
|---|---|
| 1 | Step 3 Go tests. `openssl s_client -connect 127.0.0.1:7766` with no certificate fails. `nc -z 127.0.0.1 7767` fails outside a window. `launchctl print` after a reboot. |
| 2 | Step 3 Go tests. Manual pairing with Allow, with Deny, with a timeout, with a 60 s delay before scanning, and with a 60 s delay before Allow. |
| 3 | Step 4 Go tests. Manual: change an agent's status and time it on the phone. |
| 4 | Step 6 Go tests. Manual on a Claude agent, a `codex --no-daemon` agent, a plain `codex` agent alone in its folder, two plain `codex` agents in one folder (both show terminal text), and an agent before its first prompt (terminal text). |
| 5 | Step 5 Go tests. Step 11 manual: resize, heavy output, network change. |
| 6 | Step 7 Go tests. Manual with the probes below: approve and deny each prompt kind; answer one on the Mac before tapping on the phone; tap while typing on the Mac (`mac_in_use`). |
| 7 | Step 7 Go tests. Manual message to an idle probe; check the prompt has no `<hc` tag. |
| 8 | Step 10 manual, including the reboot. |
| 9 | Step 3 resumption test, Step 5 Go test. Manual revoke during a live view. |
| 10 | Manual over Tailscale on cellular. |
| 11 | Step 9 gate test. |
| 12 | Step 10 manual lock-screen check; manual recents check. |
| 13 | Step 3 audit scan; manual `grep` of `audit.log` and `daemon.log` after the run. |

**Probe agents.** All in a scratch folder, removed afterwards with `hc kill` and `hc rm`:
- Claude command: `hc run --bg --name hr-probe-claude --cwd <repo> -- claude --permission-mode default "Use Bash to run: touch <scratch>/ok"`.
- Claude edit: the same, with "Use the Write tool to create <scratch>/edit.txt containing hi".
- Codex command: `hc run --bg --name hr-probe-codex --cwd <repo> -- codex --no-daemon -s read-only -a on-request "Run: touch <scratch>/ok"`.
- Codex edit: the same, with "Use apply_patch to create <scratch>/edit.txt containing hi".
- If a prompt can be produced in Claude auto mode, run the same checks on it and record the result in As built.

**Commands:**
- `cd daemon && go test ./...`
- `cd android && ./gradlew testDebugUnitTest assembleDebug`
- `emulator -avd herdr-remote -no-window &`, then `./gradlew connectedDebugAndroidTest`
- `adb install -r app/build/outputs/apk/debug/app-debug.apk`

All the tests above are proposed. Integration tests that need a real core use their own `HC_SESSION` and scratch XDG paths, never the user's core (`hc --skill`).

## 7. Constraints and open questions

**Settled constraints:**
- Herdr is used unchanged.
- No relay.
- The user installs Tailscale on the Mac and the phone for away access.
- The phone needs Google Play services, for the code scanner.
- The macOS firewall may ask once to allow incoming connections to `herdr-remoted`.
- Processes running as the same macOS user are trusted (Decision 18).

**Implementation discretion:**
- the Go QR library;
- the exact Compose layout;
- backoff timings;
- how the overlay is laid out.

**Non-blocking assumptions:**
- `~/.claude/sessions/<pid>.json`, the Claude projects layout and the Codex rollout layout are undocumented. If they change, the terminal-text fallback covers it.
- The prompt markers are tied to Claude Code 2.1.x and Codex 0.167.x. A change fails toward view-only, which is safe, and needs a marker update.
- Claude auto-mode prompts were not observed. They are expected to use the same dialog. If not, they are view-only.
- Residual risk, accepted: `terminal.keys` has no status condition. The Mac-in-use guard is a heuristic. It does not see input over SSH, or from other terminals and agents through hc, and remote-desktop input is unverified. If such input, or local input starting in the milliseconds between the idle check and the key, changes the screen first, the key can land on a different prompt. Local input during the call is reported as `uncertain` and audited. No shipped hc API can remove the race.
- `mac_idle_seconds` can be set to 0 when a device such as a mouse jiggler keeps the Mac permanently "in use".
- Event change payloads are captured in Step 2 before the reducer is written.

All steps are ready. Nothing blocks.

## 8. For the implementer

1. Re-verify section 2 before starting:
   - `hc --version`, `herdr-app --version`, `claude --version`, `codex --version`;
   - the probe prompts and keys in section 6;
   - `hc attach --observe` piped to a file;
   - `ioreg -c IOHIDSystem -d 4 | grep HIDIdleTime`.
2. Work in step order. End each step with its Check and a commit. Never push.
3. Record as-built notes in a dated "As built" section at the end of this document: the QR library chosen, the captured event shapes, and any marker changes.
4. Stop and report if a verified fact fails: a key or marker, the attach behaviour, a transcript path, the untagged send, or the bare-environment access. Routine details (naming, layout, timings) are yours to settle.

## 9. Verification status

- Inspected on 2026-11-08: Herdr.app 0.1.13, hc 0.1.18, Claude Code 2.1.295, codex-cli 0.167.0, the repo at no commits.
- Verified with probe terminals, all removed afterwards:
  - the approval keys for both agents;
  - the command and file-edit prompt markers for both agents;
  - the attach stream to a pipe;
  - the transcript mappings, including daemon-mode Codex;
  - untagged `send` from outside a terminal;
  - bare-environment access;
  - `HIDIdleTime`.
- Checked in the installed guides and schema: the mod `exec` limit, the reload and redraw rules, the `hc` methods, and the event snapshot and heartbeat shapes.
- Event change payloads and Claude auto-mode prompts were not observed.

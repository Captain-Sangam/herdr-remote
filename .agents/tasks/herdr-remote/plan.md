# Plan: herdr-remote

## 1. Outcome

Build an Android app that controls Herdr.app sessions on this Mac, and the Mac-side daemon it talks to, in `/Users/zeus/code/herdr-remote`. There are three parts:

- **`herdr-remoted`**, a Go daemon that runs as a LaunchAgent. It is the only thing the phone can reach. It serves a small HTTPS API over mutual TLS and talks to Herdr only through the shipped `hc` and `herdr-app` commands. A fixed set of operations is built into the code.
- **A Herdr mod, `herdr-remote`**, opened as an overlay in the Herdr window. It shows the pairing QR code, asks you to confirm a new phone, and lists paired phones with a Revoke button.
- **The Android app.** You scan the QR code, check that the 6-digit code on the phone matches the Mac, and click Allow on the Mac. The app then shows:
  - spaces, the agents in each space, and all agents;
  - per agent: the transcript, a live colored view of its terminal, Approve or Deny for a permission prompt, and a box to send a message;
  - an optional background service that raises a notification when an agent becomes blocked.

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
- **Claude Code permission prompt** (verified with a probe agent in `--permission-mode default`):
  - Status: `{app: "claude-code", kind: "permission", msg: "approve Bash: touch <path>", state: "blocked"}`.
  - Screen: `Do you want to proceed?`, then `❯ 1. Yes`, `2. Yes, and always allow …`, `3. Yes, and switch to auto mode …`, `4. No`, and `Esc to cancel · Tab to amend`.
  - `hc keys <t> 1` approved: the file was created, and the status went to `done` with the turn moving from 1 to 2.
  - `esc` denied: no file, status `idle`, turn unchanged.
  - "No" was option 4, so a digit is not a safe deny.
- **Codex approval prompt** (verified with `-s read-only -a on-request`; this version rejects `-a untrusted`):
  - Status: `{app: "codex", kind: "permission", state: "blocked"}`, with no `msg`.
  - Screen: `Would you like to run the following command?`, `Reason: …`, `$ <command>`, `1. Yes, proceed (y)`, `2. Yes, and don't ask again … (p)`, `3. No, and tell Codex what to do differently (esc)`, `Press enter to confirm or esc to cancel`.
  - `y` approved and `esc` denied.
- **Transcripts.**
  - **Claude Code:** `foreground.pid` leads to `~/.claude/sessions/<pid>.json` (keys include `sessionId`, `cwd`, `updatedAt`). That leads to `~/.claude/projects/<slug>/<sessionId>.jsonl`.
    - Records have `type` `user` or `assistant`, whose `message.content` is a string or blocks of `text`, `thinking`, `tool_use` and `tool_result`. Other types (`attachment`, `mode`, `ai-title`, …) are skipped.
    - The sessions file appears after the first prompt.
  - **Codex:** the `codex` process under the terminal's foreground process keeps its rollout file open, `~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<id>.jsonl` (`lsof -p`).
    - Records are `session_meta`, `response_item` (payload `message`, `custom_tool_call`, `custom_tool_call_output`), `event_msg`, `turn_context` and others.
    - Guardian subagent threads have their own rollout files with `session_meta.parent_thread_id` set; the main thread has none.
- **Toolchain.** Java 17 (Homebrew openjdk@17), Go and Swift are installed. The Android SDK, `adb`, Gradle and Tailscale are not. Homebrew has the `android-commandlinetools` cask. The Mac's LAN address was 192.168.1.3.

## 3. Decisions

| # | Decision | Why | Strongest rejected alternative |
|---|---|---|---|
| 1 | One Go daemon, `herdr-remoted`, runs as a per-user LaunchAgent. It is the only network listener. | One static binary with a good TLS stack and simple subprocess control. Go is installed. | Swift: the same capability with more macOS-only code. A mod as the server: mods are trusted code with no isolation and a 250 ms quit budget. |
| 2 | The daemon reaches hc through three long-lived `hc connect` subprocesses: one for reads, one for `events.subscribe`, one for actions. It runs `herdr-app ls`, `hc attach --observe`, `lsof` and `ps` through one helper that only accepts fixed program paths and argument shapes. There is no generic pass-through. | The raw API includes `machine.exec` and `file.write`. Actions wait for local typing to stop and must not stall reads, and a subscribed connection carries only notifications. | Proxying JSON-RPC with a method filter: one filter bug exposes everything. |
| 3 | Allowed hc use:<br>- `hello`, `terminal.list`, `terminal.get`, `terminal.read`, `events.subscribe`;<br>- `terminal.send` with `force: false`;<br>- `terminal.keys` with exactly one of `["1"]`, `["y"]`, `["esc"]`, and only from the approval code;<br>- `hc attach --observe`.<br>Targets must match `^t[0-9a-z]+$` (no `machine/`), be `live` and have `reported: true`. | It is the minimum for the requested screens, and it fails toward refusing. | `terminal.interrupt` or a keyboard: out of scope by user decision. |
| 4 | Two TLS 1.3 listeners share one P-256 server key, which the phone pins by SPKI SHA-256.<br>- The **paired listener** on port 7766 is always on. It requires a client certificate whose SPKI hash is in the device store, so an unknown key fails in the handshake.<br>- The **enrollment listener** on port 7767 exists only while a pairing window is open. It takes any client certificate and serves only `POST /v1/enroll`. | The handshake cannot see the request path, so one listener would have to accept unknown keys. Two listeners keep "the paired listener never accepts an unknown key" true at all times. Pinning by key makes LAN and Tailscale addresses work alike. | One listener with path middleware: unknown keys reach HTTP parsing whenever pairing is open. A relay: ruled out by the user. |
| 5 | Plain HTTP on top of the TLS listeners:<br>- JSON for reads and actions;<br>- Server-Sent Events for `/v1/events`, with a comment line every 10 s;<br>- a chunked `application/octet-stream` body for the live view.<br>The phone turns off OkHttp's read timeout on streaming calls. Go uses the standard library only, plus a QR encoder. | The live view is watch-only, so no two-way socket is needed. An idle terminal sends no bytes, and hc heartbeats only every 15 s, against OkHttp's 10 s default read timeout. | WebSocket: needs a Go dependency and adds nothing for one-way streams. |
| 6 | Pairing:<br>1. You open the overlay (`herdr-app do mod.open --name herdr-remote`). It runs `herdr-remoted pair --json`, which opens a pairing window and returns `{matrix, code_url}`. The overlay draws the QR matrix with `div()` cells, and nothing is written to disk.<br>2. The QR holds `herdr-remote://pair?v=1&pin=<b64url SPKI sha256>&ep=<addr>,<addr>&p=7766&ep_port=7767&t=<token>&exp=<unix>`. The token is 32 random bytes, kept in memory only, and the window lasts 120 s.<br>3. The phone makes a non-exportable P-256 key in Android Keystore and calls `/v1/enroll {token, name}` on 7767. The token binds to the first client SPKI that uses it, so the same phone can retry. Any other SPKI is refused.<br>4. The overlay shows the device name and a 6-digit code from SHA-256(server SPKI ‖ client SPKI). The phone shows the same code.<br>5. Allow stores the device and Deny drops it. `/v1/enroll` long-polls until Allow, Deny or expiry. Then the window and the 7767 listener close. | A QR code can leak, for example over a screen share or in a photo. Requiring Allow on the Mac means a leaked QR alone enrolls nothing. `image()` only loads files, so drawing the QR with `div()` keeps the token off disk. | Token-only enrollment: a QR seen over a screen share is enough to get in. An SVG file in the mod folder: puts the token on disk. |
| 7 | Revocation: `herdr-remoted devices rm <id>`, or Revoke in the overlay, deletes the device, closes its connections and ends its live-view subprocesses. Every request looks the device up again. | A lost phone loses access at once, including open streams. | Certificate expiry only: too slow. |
| 8 | Live view:<br>- `GET /v1/terminals/{t}/live` runs `hc attach --observe --history 500 <t>` and copies its stdout to the response, flushing each read.<br>- The phone renders it with xterm.js, bundled in a locked-down WebView, at the terminal's own `cols` and `rows`, scaled to fit the width with pinch zoom.<br>- When a `size` event arrives, the daemon ends that terminal's live streams, and the phone reopens one for a fresh paint at the new size.<br>- At most 2 live views per device. The subprocess ends when the response ends. | Verified to stream colored, incremental bytes with no PTY and no protocol decoding. That is closest to screen sharing. Restarting on resize avoids depending on resize ordering, which the CLI does not promise. | `terminal.read` polling at 4 Hz: plain text, no color, 250 ms steps. `herdr-app shot`: the whole window as PNGs. |
| 9 | Transcripts:<br>- **Claude:** `foreground.pid` → `~/.claude/sessions/<pid>.json` → `sessionId` → the single match of `~/.claude/projects/*/<sessionId>.jsonl`.<br>- **Codex:** walk the descendants of `foreground.pid` (with `ps`) to the process named `codex`. Run `lsof -Fn -p <pid>` and keep the open `~/.codex/sessions/**/rollout-*.jsonl` files. Choose the one whose `session_meta` has no `parent_thread_id`.<br>- **Anything else, or a lookup that fails:** `terminal.read` with `lines: 2000`, shown as "terminal text".<br>Real paths (after following symlinks) must lie under those roots. Reads take the last 2 MB, and the phone pages backwards by byte offset. | Both are explicit identities, verified on live agents. Guardian threads would otherwise be picked by mistake. | A cwd-and-time heuristic: picks the wrong session when two agents share a folder. A manual binding chosen on the Mac: extra UI for a case the explicit lookups already cover. |
| 10 | Approvals apply only when `state = blocked`, `kind = permission` and `app` is `claude-code` or `codex`.<br>1. `GET /prompt` returns `{msg, block, turn, prompt_hash}`. `block` is the screen region from the first prompt line to the last option line, and `prompt_hash` is SHA-256(`app` ‖ `turn` ‖ `msg` ‖ `block`).<br>2. `POST /decision {decision, turn, prompt_hash}` takes a per-terminal lock, then re-reads `terminal.get` and `terminal.read(lines 0)`.<br>3. It requires the same state, kind, app and turn; the same hash; the expected option text (Claude: `1. Yes` and `Esc to cancel`; Codex: `Yes, proceed (y)` and `(esc)`); and that this terminal's **consumed marker** is not already `{turn, prompt_hash}`.<br>4. It sets the marker, then presses one key: Claude `1` or `esc`, Codex `y` or `esc`.<br>5. Then it reads the status for up to 3 s and returns `applied` (the status or turn moved) or `sent_unconfirmed`. It never presses again.<br>Any mismatch returns 409 and presses nothing. | The keys were verified on live prompts. `1` selects Yes whatever the cursor; `esc` is the only deny that does not depend on option order. The marker covers the gap between the key and the agent's next status report, which is when a retry or a second phone would otherwise pass the guard. | A durable action-ID store: the marker only has to outlive that gap, and a restarted daemon sees the new state. Pressing `enter` when Yes is selected: depends on the cursor. |
| 11 | Questions (`kind: question`), `auth`, Pi and unknown apps are view-only. The phone says "answer this on the Mac". | Control is scoped to approve, deny and message. Question menus vary, so a digit could pick the wrong answer. | Digits 1 to 9: a wrong parse answers for you. |
| 12 | Messages: `terminal.send` with `force: false` to a live terminal with `reported: true`. C0 and C1 control characters are removed except `\n` and `\t`, and the text is capped at 4000 characters. A blocked agent refuses the message, and the phone shows why. | It matches the scope. Removing escapes keeps input from breaking out of the paste. | Raw keys: out of scope. |
| 13 | Background alerts:<br>- An opt-in Android foreground service (type `specialUse`) holds `/v1/events`.<br>- It posts a notification when a terminal becomes blocked, de-duplicated by terminal, turn and `msg`. On reconnect, the snapshot replays agents that are still blocked.<br>- It needs `POST_NOTIFICATIONS`, and the app asks to be exempt from battery optimization.<br>- There is no push server. Alerts arrive only while the service runs and is connected. | No relay or push service of ours, as decided. | FCM: needs a server and Google credentials. |
| 14 | Phone key and app gate:<br>- The Keystore key has no unlock requirement, so the service can reconnect while the phone is locked.<br>- Opening the app after 5 minutes in the background asks for BiometricPrompt (strong biometric or the device credential).<br>- Approve, deny and message work only in the unlocked, authenticated app. Notifications carry no action buttons. | An unlock-required key makes every handshake fail while the phone is locked, which stops alerts. The gate still stops someone casually picking up an unlocked phone. | `setUnlockedDeviceRequired`: kills background alerts. |
| 15 | Android stack: Kotlin and Jetpack Compose, minSdk 31, targetSdk 35. OkHttp for TLS, SSE and streams; kotlinx.serialization; the Google Play services code scanner (no camera permission); androidx.biometric; xterm.js 5.x as a bundled asset. One activity, ViewModels, no DI framework. | It is the smallest standard stack. | Termux terminal-view: native rendering, but a JitPack dependency and more glue. React Native or Flutter: more toolchain. |
| 16 | Spaces and agents:<br>- `/v1/spaces` runs `herdr-app ls` on each request, and also on created, removed and labels events. It drops spaces whose `machine` is set.<br>- An agent is a live local terminal with `reported: true`. Its space comes from `herdr-app ls`, else from its `herdr.space` label.<br>- If Herdr.app is not running, agents are grouped by their `where` label under "Herdr window closed". | Renames emit no hc event. Remote machines are a non-goal. | A periodic poll timer: a request-time refresh is simpler and always current. |

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
2. Opening the overlay shows a QR code. Scanning it shows a 6-digit code on the phone that matches the Mac. After Allow, the phone is paired. With Deny, no Allow, or after 120 s, nothing is stored. A second phone using the same token is refused.
3. The app lists spaces with their Herdr names. A space lists its agents with status. The all-agents view lists every local agent. Status changes on the Mac show on the phone within 2 s.
4. A Claude Code or Codex agent's transcript shows user and assistant messages and tool calls, and pages back. Other agents show labelled terminal text.
5. The live view shows the agent's terminal in color, follows new output within about 1 s on the LAN, and recovers after the pane is resized on the Mac.
6. A blocked Claude Code or Codex permission prompt can be approved or denied from the phone, and the agent continues accordingly. A stale request presses no key and says so: the prompt changed, was answered on the Mac, or was already answered from another phone or a retry.
7. A message from the phone reaches an idle or working agent as a new prompt. A blocked agent refuses it, with a reason.
8. With background watching on and the app closed, a newly blocked agent raises one notification within 5 s, while the service runs and is connected.
9. Revoking the phone on the Mac ends its open streams within 2 s, and later handshakes fail.
10. The same pairing works over Tailscale from a cellular connection.
11. After 5 minutes in the background, the app asks for biometric or device credential before showing anything.

**Failure cases to handle (no others):**
- hc is unavailable: the daemon returns 503 and reconnects, and the phone shows the last data with its age.
- `missed`: take a fresh snapshot.
- Herdr.app is closed: group agents by `where`.
- The transcript is missing or unparseable: show terminal text.
- An endpoint is unreachable: try the next endpoint, and remember the last one that worked.
- A size change: restart the live stream.
- The device is revoked mid-stream: close the stream.
- The approval guard fails: 409.

## 5. Implementation plan

Paths are proposed unless stated otherwise; the repo is empty. Each step: **Change**, **Implements**, **Invariant**, **Verified today**, **Depends on**, **Check**.

**Step 1. Toolchain and skeleton.**
- **Change:**
  - `scripts/setup-android.sh`:
    - `brew install --cask android-commandlinetools` and `brew install gradle`;
    - `sdkmanager --licenses`;
    - `sdkmanager "platform-tools" "platforms;android-35" "build-tools;35.0.0" "emulator" "system-images;android-35;google_apis_playstore;arm64-v8a"`. Use the Play Store image so the code scanner works.
    - `avdmanager create avd -n herdr-remote -k <that image>`.
  - `daemon/go.mod` (module `herdr-remote/daemon`).
  - The `android/` Gradle project with a wrapper, made with `gradle wrapper`.
  - A `.gitignore` covering build output and `local.properties`.
- **Implements:** User decision 3, Decision 15.
- **Invariant:** Only Homebrew packages and `~/Library/Android/sdk` change outside the repo.
- **Verified today:** No SDK, `adb` or Gradle; Java 17 present; the cask exists.
- **Depends on:** Nothing.
- **Check:** `go version`; `adb version`; `cd android && ./gradlew assembleDebug` builds an empty Compose app.

**Step 2. Subprocess helper and hc client.**
- **Change:**
  - `daemon/run.go`: one helper that accepts only the fixed programs `hc`, `herdr-app`, `lsof` and `ps`, at absolute paths found at start, with fixed argument shapes, and no shell.
  - `daemon/hc.go`: three `hc connect` connections (reads, events, actions) speaking JSON-RPC lines, with typed functions only:
    - `List`, `Get`, `Read(t, lines)`, `Subscribe`;
    - `Send(t, text)` with force always false;
    - `Keys(t, k)` where `k` is one of `1`, `y`, `esc`.

    It validates targets as in Decision 3 and reconnects with backoff.
  - Before writing the event reducer, capture real notifications with `hc events --json` while a probe agent changes status, and save them as test fixtures.
- **Implements:** Decisions 2 and 3.
- **Invariant:** No code path sends a method outside the list, sets `force`, or passes a key outside the set. Each fails toward refusing.
- **Verified today:** Methods and parameters in `hc schema`; `hc connect` exists; bare-environment access works.
- **Depends on:** Step 1.
- **Check:** `cd daemon && go test ./...` with a fake `hc` built as a test binary that records every request. Tests fail on:
  - any method outside the list;
  - `force: true`;
  - any other key;
  - a `machine/` target;
  - a program outside the helper's list.

**Step 3. Identity, device store, pairing, admin socket.**
- **Change:**
  - `daemon/tls.go`, `daemon/devices.go`, `daemon/pair.go`, `daemon/admin.go`.
  - State lives in `~/Library/Application Support/herdr-remote/` (directory 0700): `server.key` (0600), `server.crt`, `devices.json` (0600, `[{id, name, spki_sha256, added}]`) and `admin.sock` (0600).
  - CLI commands over the admin socket: `pair --json` (opens the window; returns the QR matrix and the payload), `pair wait` (blocks until a pending enrollment or expiry; returns `{device_name, code}`), `pair allow|deny`, `devices ls --json`, `devices rm <id>`.
  - The 7766 listener verifies the client SPKI against `devices.json`. The 7767 listener exists only during a window.
  - Endpoints are the private IPv4 addresses of active interfaces plus any 100.64.0.0/10 address. If `tailscale` is on the PATH, also the MagicDNS name from `tailscale status --json`.
  - The QR matrix comes from a small Go QR library, chosen at implementation; record it in As built.
- **Implements:** Decisions 4, 6 and 7.
- **Invariant:**
  - An unknown key fails the handshake on 7766.
  - A token is single-use, binds to its first SPKI, and lasts 120 s.
  - No device is stored without Allow.

  Each fails toward refusing.
- **Verified today:** Nothing exists.
- **Depends on:** Step 2.
- **Check:** Go tests:
  - a fresh enrollment succeeds after Allow;
  - Deny and expiry store nothing;
  - a second SPKI with the same token is refused;
  - 7767 is closed after the window;
  - an unpaired certificate fails on 7766.

  Also, `openssl s_client -connect 127.0.0.1:7766` with no certificate fails the handshake.

**Step 4. Read API and events.**
- **Change:**
  - `daemon/api.go`, `daemon/state.go`.
  - Endpoints, all on 7766:
    - `GET /v1/hello` → `{daemon_version, endpoints[]}`;
    - `GET /v1/spaces` → `[{id, name, agents[]}]`;
    - `GET /v1/agents`;
    - `GET /v1/terminals/{t}` → `{address, space, tab, where, app, status, turn, cols, rows, title}`;
    - `GET /v1/events` (SSE). Each event is a terminal summary; there is a `: ka` comment every 10 s, and a full `snapshot` event on connect and after `missed`.
  - State follows Decision 16.
- **Implements:** Decisions 5 and 16.
- **Invariant:**
  - On hc loss: 503, and reconnect.
  - On `missed`: a fresh snapshot.
  - Requests from a revoked device are refused.
- **Verified today:** The `herdr-app ls` shape; snapshot and heartbeat shapes.
- **Depends on:** Step 3.
- **Check:** Go tests with the fake hc and recorded event fixtures (snapshot, status change, missed); a test that a renamed space shows on the next `/v1/spaces`. Manual: `curl --cert … --key … https://127.0.0.1:7766/v1/agents`.

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
  - `daemon/transcript.go`, with resolvers and parsers per Decision 9.
  - `GET /v1/terminals/{t}/transcript?before=<offset>` → `{source: claude|codex|terminal, items: [{role, kind: text|tool_call|tool_result, text, ts}], next_before}`.
    - Tool results are cut to 2000 characters.
    - Thinking blocks are skipped.
    - A partial last line is ignored.
- **Implements:** Decision 9.
- **Invariant:** Only files whose real path is under the two roots are read. Any failure returns terminal text, not an error.
- **Verified today:** The mapping chains, record types and `parent_thread_id`, as in section 2.
- **Depends on:** Step 4.
- **Check:** Go tests on fixture JSONL written by the test:
  - Claude and Codex parsing;
  - a partial last line;
  - a symlink pointing outside the roots, which is refused;
  - two open rollouts where the guardian one is ignored;
  - a missing sessions file, which falls back to terminal text.

**Step 7. Approvals and messages.**
- **Change:**
  - `daemon/actions.go`.
  - `GET /v1/terminals/{t}/prompt` → `{app, kind, msg, block, turn, prompt_hash, actionable}`.
  - `POST /v1/terminals/{t}/decision {decision: approve|deny, turn, prompt_hash}` → `{result: applied|sent_unconfirmed}` or 409 `{reason, current}`.
  - `POST /v1/terminals/{t}/message {text}` → `{sent: true}` or 409 when blocked.
  - Decisions 10, 11 and 12 in full, including the per-terminal lock and the consumed marker.
- **Implements:** Decisions 10, 11 and 12.
- **Invariant:** A key is pressed only after every guard passes, at most once per `{turn, prompt_hash}`. Otherwise 409 and no key, failing toward refusing.
- **Verified today:** The keys and screens in section 2. The narrow race through `terminal.keys` is known.
- **Depends on:** Step 4.
- **Check:** Go tests with the fake hc:
  - a stale turn, a changed hash, missing option text, `kind: question`, `app: pi` and a `machine/` target each press nothing;
  - two concurrent decisions press exactly one key;
  - a retry after a dropped response, with the fake status held unchanged, presses exactly one key;
  - a message with ESC and C1 bytes arrives cleaned;
  - a blocked target refuses a message.

  Manual: probe agents as in section 6.

**Step 8. Herdr mod and install.**
- **Change:**
  - `mod/herdr-remote/mod.json` (overlay) and `main.js`. The overlay:
    - runs `herdr-remoted pair --json`;
    - draws the matrix with `div()` cells and shows a 120 s countdown;
    - shows the pending device and code from `pair wait`, with Allow and Deny;
    - lists devices from `devices ls --json`, each with Revoke.
  - `scripts/install.sh`:
    - builds the daemon to `~/.local/bin/herdr-remoted`;
    - writes `~/Library/LaunchAgents/dev.herdr-remote.plist` (RunAtLoad, KeepAlive) and runs `launchctl bootstrap gui/$(id -u)`;
    - copies the mod to `~/.config/herdr-app/mods/herdr-remote/` without overwriting an existing folder;
    - runs `herdr-app check`.
- **Implements:** Decisions 1, 6 and 7.
- **Invariant:** No shipped mod or setting is edited. The mod writes no file and holds the token only while open.
- **Verified today:** The mod APIs, `mod.open` and `herdr-app check`.
- **Depends on:** Step 3.
- **Check:** `herdr-app check ~/.config/herdr-app/mods/herdr-remote` passes; `herdr-app do mod.open --name herdr-remote` then `herdr-app shot /tmp/hr.png` shows the QR code; `launchctl print gui/$(id -u)/dev.herdr-remote` shows it running.

**Step 9. Android pairing, connection and app gate.**
- **Change:**
  - `android/app/src/main/...`:
    - Keystore P-256 key and a self-signed client certificate;
    - an X509KeyManager over the Keystore key, and a TrustManager that accepts only the pinned SPKI;
    - endpoint fallback that remembers the last good endpoint and refreshes endpoints from `/v1/hello`;
    - the Pair screen, with the code scanner, the 6-digit code and the long-poll;
    - the BiometricPrompt gate on start and after 5 minutes in the background.
- **Implements:** Decisions 4, 6, 14 and 15.
- **Invariant:**
  - A pin mismatch never connects.
  - The private key never leaves the Keystore.
  - Nothing is shown before the gate passes.
- **Depends on:** Steps 3 and 4.
- **Check:**
  - JVM tests: QR payload parsing (bad version, expired, missing fields), the pin check, and the 6-digit code matching the Go code on shared test vectors.
  - Instrumented test: pairing against a dev daemon at `10.0.2.2`.
  - Gate test: the clock is moved past 5 minutes and the gate shows; an action before unlock is refused.

**Step 10. Android lists, events and background service.**
- **Change:**
  - Spaces, Space and All agents screens; an SSE client and reducer; the stale-age banner.
  - The foreground service (`specialUse`), with a settings toggle.
  - Notifications de-duplicated by terminal, turn and `msg`.
  - The `POST_NOTIFICATIONS` request and the battery-optimization prompt.
- **Implements:** Decisions 13 and 16.
- **Invariant:** A stale view shows its age. One notification per blocked episode.
- **Depends on:** Step 9.
- **Check:** JVM tests for the reducer (snapshot, change, missed) and for de-duplication. Manual: swipe the app away, block a probe agent, and a notification arrives.

**Step 11. Android agent screen.**
- **Change:**
  - Transcript tab, with paging back.
  - Live tab: a WebView with JavaScript on; assets served by WebViewAssetLoader; every other request blocked; no file or content access; xterm.js sized to `cols` and `rows`; bytes passed in base64 chunks; reopened on a `size` event.
  - Approve and Deny, showing `msg` and `block`, with the 409 reasons explained.
  - A Message box.
- **Implements:** Decisions 8 to 12.
- **Invariant:** The WebView loads only bundled assets and never sends input.
- **Depends on:** Step 10.
- **Check:** An instrumented test that the WebView refuses a network URL. A manual approve and deny on probe agents. A manual resize of the Mac pane during output, after which the view recovers.

**Step 12. End-to-end acceptance.**
- **Change:** None. Run criteria 1 to 11 on the emulator and a real phone, including Tailscale over cellular.
- **Depends on:** Steps 1 to 11, plus Tailscale installed on the Mac and the phone.
- **Check:** Section 6.

## 6. Testing strategy

| Criterion | Check |
|---|---|
| 1 | Step 3 Go tests. `openssl s_client -connect 127.0.0.1:7766` with no certificate fails. `nc -z 127.0.0.1 7767` fails outside a window. `launchctl print`. |
| 2 | Step 3 Go tests. Manual pairing with Allow, with Deny, and with a timeout. |
| 3 | Step 4 Go tests. Manual: change an agent's status and time it on the phone. |
| 4 | Step 6 Go tests. Manual on a Claude agent, a Codex agent and a plain shell. |
| 5 | Step 5 Go tests. Manual live view with a pane resize. |
| 6 | Step 7 Go tests. Manual with probe agents: `hc run --bg --name hr-probe-claude -- claude --permission-mode default "Use Bash to run: touch <scratch>/ok"` and `hc run --bg --name hr-probe-codex -- codex --no-daemon -s read-only -a on-request "Run: touch <scratch>/ok"`. Approve one, deny one, and answer one on the Mac before tapping on the phone. |
| 7 | Step 7 Go tests. Manual message to an idle probe. |
| 8 | Step 10 manual. |
| 9 | Step 5 Go test. Manual revoke during a live view. |
| 10 | Manual over Tailscale on cellular. |
| 11 | Step 9 gate test. |

Commands:
- `cd daemon && go test ./...`
- `cd android && ./gradlew testDebugUnitTest assembleDebug`
- `emulator -avd herdr-remote -no-window &`, then `./gradlew connectedDebugAndroidTest`
- `adb install -r app/build/outputs/apk/debug/app-debug.apk`

All the tests above are proposed. Integration tests that need a real core must use their own `HC_SESSION` and scratch XDG paths, never the user's core (`hc --skill`).

## 7. Constraints and open questions

**Settled constraints:**
- Herdr is used unchanged.
- No relay.
- The user installs Tailscale on the Mac and the phone for away access.
- The phone needs Google Play services, for the code scanner.
- The macOS firewall may ask once to allow incoming connections to `herdr-remoted`.

**Implementation discretion:**
- the Go QR library;
- the exact Compose layout;
- backoff timings;
- how the overlay is laid out.

**Non-blocking assumptions:**
- `~/.claude/sessions/<pid>.json`, the Claude projects layout and the Codex rollout layout are undocumented. If they change, the terminal-text fallback covers it.
- The prompt text checks are tied to Claude Code 2.1.x and Codex 0.167.x. A change in that text fails toward view-only, which is safe, and needs a profile update.
- Residual risk: `terminal.keys` has no status condition and waits while someone types at the Mac. If you type at the Mac while approving from the phone, the key can land on a different screen. No shipped hc API can prevent this.
- Event notification payloads are not described in the schema. Step 2 captures them before the reducer is written.

All steps are ready. Nothing blocks.

## 8. For the implementer

1. Re-verify section 2 before starting:
   - `hc --version` and `herdr-app --version`;
   - `claude --version` and `codex --version`;
   - the probe prompts and keys in section 6;
   - `hc attach --observe` piped to a file.
2. Work in step order. End each step with its Check and a commit. Never push.
3. Record as-built notes in a dated "As built" section at the end of this document, including the QR library chosen and the captured event shapes.
4. Stop and report if a verified fact fails: a key or screen string, the attach behaviour, a transcript path, or the bare-environment access. Routine details (naming, layout, timings) are yours to settle.

## 9. Verification status

- Inspected on 2026-11-08: Herdr.app 0.1.13, hc 0.1.18, Claude Code 2.1.295, codex-cli 0.167.0, the repo at no commits.
- Verified with probe terminals, all removed afterwards: the approval keys and screens for both agents, the attach stream to a pipe, the transcript mappings, and bare-environment access.
- Checked in the installed guides and schema: the mod APIs, the `hc` methods, the event snapshot and heartbeat shapes.
- Event change payloads were not observed.
- This revision has not yet been checked by the separate review stage that follows.

## Review (2026-11-08)

This plan was reviewed for three rounds by fresh agents and then superseded. The final, corrected design is `/Users/zeus/code/herdr-remote/TDD.md`, and the product requirements are `/Users/zeus/code/herdr-remote/PRD.md`. Build from `TDD.md`, not from this file. The main amendments were:
- two TLS listeners with per-listener checks in `VerifyConnection`, and session tickets off;
- the overlay polls `pair status` (the mod `exec` limit is 30 s), and only **Pair a phone** opens port 7767;
- an audit log that doubles as the restart-safe store for request IDs (UUIDv7, a 5-minute window, a rate limit);
- a Mac-in-use guard (`HIDIdleTime`) before any approval key;
- no folder matching for daemon-mode Codex: terminal text, plus a setup step for `codex --no-daemon`;
- record-aligned transcript pages, xterm.js flow control, and Android foreground-service and privacy details.

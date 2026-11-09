# Draft plan: herdr-remote (after round 1)

## 1. Outcome

Build two things in `/Users/zeus/code/herdr-remote`: a Go daemon on the Mac (`herdr-remoted`) and an Android app. A Herdr mod gives the daemon a QR code in the Herdr window.

- The daemon is the only thing the phone can reach. It serves a small HTTPS API with mutual TLS. It talks to Herdr only through `hc` and `herdr-app` as shipped, with a fixed list of allowed operations.
- The phone pairs by scanning the QR code, then you confirm the phone on the Mac.
- The app shows spaces, the agents in each space, all agents, and per agent: transcript, a live colored terminal view, approve or deny a permission prompt, and send a message.
- A foreground service on the phone keeps the event stream open and raises a notification when an agent becomes blocked.

The practical result: you can watch and unblock Claude Code and Codex agents from the phone over Wi-Fi or Tailscale, and a phone that is not paired, or has been revoked, gets nothing past the TLS handshake.

## 2. Current behaviour

The repo is new and empty (branch `main`, no commits). Herdr is closed source: Herdr.app 0.1.13 and herdr-core `hc` 0.1.18, inspected on 2026-11-08.

- **herdr-core API.** JSON-RPC 2.0 over a local socket, also over stdio with `hc connect` (`hc connect --help`). The schema comes from `hc schema`.
  - Read: `terminal.list`, `terminal.get`, `terminal.read`.
  - Stream: `events.subscribe` sends a snapshot, then status, turn, title, size, created and finished events, with a heartbeat every 15 s and `missed` plus a fresh snapshot for a subscriber that falls behind.
  - Act: `terminal.send` (refuses a blocked program unless `force`; `if` accepts only idle or working) and `terminal.keys` (no status condition; both wait for ~3 s of local typing quiet).
  - Dangerous, never exposed: `terminal.run`, `machine.exec`, `file.read/write/edit`, `doc.set`, `server.stop`, `machine.*`.
- **Terminal facts** (`hc get <t> --json`): `address`, `foreground {pid, name, argv, cwd}`, `labels` (Herdr's own terminals carry `herdr.space`, `herdr.tab` and `where`), `status {state, app, kind, msg}`, `turn`, `cols`, `rows`, `reported`.
  - `status.kind` is `permission`, `question` or `auth`, set only when blocked.
  - `turn` goes up each time the status leaves `working`.
- **Live stream.** `hc attach --observe [--history N] <t>` writes the first paint, then the live bytes, with colors, to a plain pipe. It needs no TTY and never types or resizes.
  - Verified with a probe terminal.
  - The framed socket wire format is not documented in the schema, so we don't use it.
- **Window model.** `herdr-app ls` returns `spaces[{id, name, folder, machine, tabs[{id, name, terminals[{terminal, pane, status, app}]}]}]`.
  - `hc` and `herdr-app` both work from a bare environment (`env -i HOME PATH`), so a LaunchAgent can run them.
- **Mods.** Trusted JavaScript in `~/.config/herdr-app/mods/<name>/` with `main.js` and `mod.json`.
  - A mod can run programs (`exec`) and show images from its folder (`~/.config/herdr-app/AGENTS.md:801-804, 971, 996`).
  - `mod.open {name}` toggles a mod as an overlay (`herdr-app commands`).
  - `herdr-app check <dir>` validates a mod.
- **Claude Code 2.1.295 approvals** (verified with a probe agent):
  - Status: `{app: claude-code, kind: permission, msg: "approve Bash: touch …", state: blocked}`.
  - Screen: "Do you want to proceed?", then "1. Yes", other options that vary, "Esc to cancel".
  - Key `1` approves, `esc` denies. "No" was option 4, so a digit is not a safe deny.
- **Codex 0.167.0 approvals** (verified):
  - Status: `{app: codex, kind: permission, state: blocked}`, with no `msg`.
  - Screen: "Would you like to run the following command?", "1. Yes, proceed (y)", "… (p)", "3. No, and tell Codex what to do differently (esc)".
  - Key `y` approves, `esc` denies.
- **Transcripts.**
  - Claude: `foreground.pid` leads to `~/.claude/sessions/<pid>.json`, which has `sessionId` and `cwd`. That leads to `~/.claude/projects/<slug>/<sessionId>.jsonl`, whose records are `user` and `assistant` with content blocks `text`, `thinking`, `tool_use` and `tool_result`. The sessions file appears after the first prompt.
  - Codex: the `codex` child of the foreground process group keeps `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` open (lsof). Records are `session_meta`, `response_item` (payload `message`, `custom_tool_call`, `custom_tool_call_output`) and `event_msg`.
- **Toolchain.** Java 17, Go and Swift are installed. The Android SDK, `adb`, Gradle and Tailscale are not. Homebrew has the `android-commandlinetools` cask.

## 3. Decisions

| # | Decision | Why | Strongest rejected alternative |
|---|---|---|---|
| 1 | One Go daemon, `herdr-remoted`, runs as a LaunchAgent for the user. It is the only network listener. | One static binary, a good TLS stack, simple subprocess control, and Go is installed. | Swift: the same capability with more macOS-only code. A Herdr mod as the server: mods are trusted code with no isolation, and a quit budget of 250 ms. |
| 2 | The daemon reaches hc through two long-lived `hc connect` subprocesses: one for requests and one for `events.subscribe`. It runs `herdr-app ls` and `hc attach --observe` as subprocesses. There is no generic pass-through: the code has one function per allowed operation. | The raw API includes `machine.exec` and `file.write`. An allow-list built into the code cannot be widened by a request. | Proxying JSON-RPC with a method filter: one filter bug exposes everything. |
| 3 | The allowed operations are: `hello`, `terminal.list`, `terminal.get`, `terminal.read`, `events.subscribe`; `terminal.send` with `force: false`; `terminal.keys` with exactly `["1"]`, `["y"]` or `["esc"]` after the approval guard; and `hc attach --observe`. Targets must match `^t[0-9a-z]+$`, be live and have `reported: true`. Remote `machine/t` addresses are refused. | It is the minimum for the requested screens. | Allowing `terminal.interrupt` or a keyboard: out of scope by user decision. |
| 4 | Transport is HTTPS over TLS 1.3 with a client certificate required, on port 7766 on all interfaces. The server key is a P-256 self-signed certificate. The phone pins its SPKI SHA-256, so any address (LAN or Tailscale) works. A client certificate is accepted only if its SPKI hash is in the device store, or it is presented to `/v1/enroll` with a live pairing token. | An unknown key is rejected in the handshake before any HTTP is parsed. Pinning by key avoids certificate names and IP changes. | A relay: ruled out by the user. Noise or custom crypto: more code than TLS for no gain. |
| 5 | The API is plain HTTP. JSON for reads and actions, Server-Sent Events for `/v1/events`, and a chunked `application/octet-stream` response for the live terminal. The Go side uses the standard library only, except for a QR encoder. | The live view is watch-only, so nothing needs a two-way socket. SSE and chunked bodies are in the Go standard library and in OkHttp. | WebSocket: needs a Go dependency and buys nothing for one-way streams. |
| 6 | Pairing works as follows.<br>1. The mod overlay runs `herdr-remoted pair`, which shows a QR code with `{v, pin, endpoints[], port, token, exp}`. The token is 32 random bytes, is held in memory only, is single-use and expires after 120 s.<br>2. The phone makes a non-exportable P-256 key in Android Keystore and calls `/v1/enroll` with the token and a device name.<br>3. The overlay then shows the device name and a 6-digit code computed from SHA-256(server SPKI ‖ client SPKI). The phone shows the same code.<br>4. You click Allow on the Mac. Only then is the device stored. | A QR code can leak (a screen share, a photo). Confirming on the Mac means a leaked code alone enrolls nothing. | Token-only enrollment: one less click, but a QR seen over a screen share is enough to get in. |
| 7 | Revocation: `herdr-remoted devices rm <id>`, or a Revoke button in the overlay, deletes the device and closes its open connections. Every request looks the device up again. | A lost phone must lose access at once, including live streams. | Certificate expiry only: too slow. |
| 8 | The live view runs `hc attach --observe --history 500 <t>` per viewer and copies its stdout to the HTTP response. xterm.js in a locked-down WebView renders it at the terminal's own `cols` and `rows`. Size changes come from `size` events. Each device can have at most 2 live views; the subprocess ends when the request ends. | It is verified to stream colored, incremental bytes with no PTY or protocol decoding. It is as simple as polling and much closer to screen sharing. | `terminal.read` at 4 Hz: plain text, no color, 250 ms steps. `herdr-app shot`: the whole window, as PNGs. |
| 9 | Transcripts.<br>- Claude: `foreground.pid` → `~/.claude/sessions/<pid>.json` (`sessionId`) → the single file matching `~/.claude/projects/*/<sessionId>.jsonl`.<br>- Codex: lsof on the `codex` process in the foreground process group → the open `~/.codex/sessions/**/rollout-*.jsonl`.<br>- Anything else, or a failed lookup: `terminal.read` with 2000 lines of history as plain text, labelled as terminal text.<br>Real paths must resolve under those roots. Reads are bounded: the last 2 MB, then paging backwards by byte offset. | Both lookups are explicit identities, verified on live agents. No guessing from cwd or time. | A cwd and time heuristic: picks the wrong session when two share a folder. Asking on the Mac which session to bind: extra UI for a case the explicit lookups already cover. |
| 10 | Approvals cover only `state: blocked`, `kind: permission` and `app` of `claude-code` or `codex`.<br>The phone shows `status.msg` and the prompt block from the visible screen. It sends `{decision, turn, prompt_hash}`.<br>Under a per-terminal lock, the daemon re-reads `terminal.get` and `terminal.read(lines 0)`. It requires the same state, kind, app and turn, the same hash of `msg` plus the prompt block, and the expected option text on screen (Claude: `1. Yes` and `Esc to cancel`; Codex: `Yes, proceed (y)` and `(esc)`).<br>Then it presses one key: Claude `1` or `esc`, Codex `y` or `esc`. A mismatch returns 409 and presses nothing. It reports the observed status afterwards. | Keys verified on live prompts. `1` selects Yes whatever the cursor; `esc` is the only deny that does not depend on option order. The guard makes a retry harmless, because the turn and hash no longer match once the dialog is answered. | Pressing `enter` when Yes is shown as selected: depends on where the cursor is. Durable request-ID tables: the state guard already prevents a double press. |
| 11 | Questions (`kind: question`), `auth`, Pi and unknown apps are view-only on the phone, with an "answer on the Mac" note. | You scoped control to approve, deny and message. Question menus vary by version, so a digit could pick the wrong answer. | Digits 1 to 9 for questions: one wrong parse answers for you. |
| 12 | Messages: `terminal.send` with `force: false` to a live terminal with `reported: true`. C0 and C1 control characters are removed except `\n` and `\t`, and text is capped at 4000 characters. A blocked agent refuses the message, and the phone says why. | It matches your scope. Stripping escapes stops input from breaking out of the paste. | Sending raw keys: out of scope. |
| 13 | Background alerts: an opt-in Android foreground service holds `/v1/events`. It posts a notification when a terminal becomes blocked, de-duplicated by terminal and turn. On reconnect, the snapshot replays agents that are still blocked. There is no push server. | No relay or push service of ours, as you decided. | FCM: needs a server and Google credentials. |
| 14 | Phone key and app gate:<br>- The Keystore key does not require an unlocked device, so the service can reconnect while the phone is locked.<br>- After 5 minutes in the background, opening the app asks for BiometricPrompt (strong biometric or device credential).<br>- Approve, deny and message are only possible inside the unlocked app. Notifications have no action buttons. | `setUnlockedDeviceRequired` would make every handshake fail while the phone is locked, which breaks alerts. The app gate still stops a casual user of an unlocked phone. | An unlock-required key: kills background alerts. |
| 15 | Android: Kotlin, Jetpack Compose, minSdk 31, targetSdk 35. OkHttp for TLS, SSE and streams. kotlinx.serialization. The Google code scanner (no camera permission). xterm.js bundled as an asset. One activity, ViewModels, no DI framework. | It is the smallest standard stack. | Termux terminal-view: native, but a JitPack dependency and more glue. React Native or Flutter: more toolchain. |

## 4. Scope and acceptance criteria

Delivered: `daemon/`, `mod/herdr-remote/`, `android/` and `scripts/`, in the repo.

Excluded: keyboard input into terminals, answering questions, starting or stopping agents, spaces or tabs, remote `hc` machines, iOS, a relay or push service, and Play Store release.

Acceptance criteria:
1. `herdr-remoted` runs under a LaunchAgent and survives logout and login. A TLS client without a certificate, or with an unknown one, fails the handshake.
2. Opening the mod overlay in Herdr shows a QR code. Scanning it on the phone shows a 6-digit code that matches the Mac's. After Allow, the phone is paired. With no Allow, or after 120 s, nothing is stored. A second use of the token fails.
3. The app lists spaces with names from Herdr. Opening a space lists its agents, with status. The all-agents view lists every agent terminal. Both update within 2 s of a status change on the Mac.
4. For a Claude Code or Codex agent, the transcript shows user and assistant messages and tool calls. For other agents, the terminal text is shown, labelled.
5. The live view shows the agent's terminal in color and follows new output within about 1 s on the LAN.
6. A blocked Claude Code or Codex permission prompt can be approved or denied from the phone, and the agent continues accordingly. A stale request (the prompt changed, or was answered on the Mac) presses no key and says so.
7. A message sent from the phone reaches an idle or working agent as a new prompt. A blocked agent refuses it.
8. With background watching on and the app closed, a newly blocked agent raises a notification within 5 s.
9. Revoking the phone on the Mac ends its open streams within 2 s, and later requests fail at the handshake.
10. Remote access over Tailscale works with the same pairing.

Failure cases to handle:
- hc is down: 503, and the phone shows the last data with its age.
- `missed` events: take a fresh snapshot.
- Herdr.app is closed: spaces are unavailable; agents are listed by terminal with `where` labels.
- A transcript is missing or unparseable: fall back to terminal text.
- The endpoint is unreachable: try the next endpoint.

## 5. Implementation plan

Each step: Change, Implements, Invariant, Verified today, Depends on, Check.

**Step 1. Toolchain and skeleton.**
- **Change:**
  - Proposed `scripts/setup-android.sh`:
    - `brew install --cask android-commandlinetools`, then `brew install gradle` (once, for the wrapper).
    - `sdkmanager "platform-tools" "platforms;android-35" "build-tools;35.0.0" "emulator" "system-images;android-35;google_apis;arm64-v8a"` and `avdmanager create avd`.
  - Proposed `daemon/go.mod`.
  - Proposed `android/` Gradle project with a wrapper.
  - Proposed `.gitignore`.
- **Implements:** User decision 3.
- **Invariant:** Nothing global changes beyond Homebrew packages.
- **Verified today:** No SDK, `adb` or Gradle; Java 17 present.
- **Depends on:** Nothing.
- **Check:** `go version` succeeds; `cd android && ./gradlew assembleDebug` builds an empty app; `adb version` succeeds.

**Step 2. hc client with the allow-list.**
- **Change:**
  - Proposed `daemon/hc.go`. It runs two `hc connect` subprocesses and sends JSON-RPC lines.
  - It exposes typed functions only: `List`, `Get`, `Read`, `Subscribe`, `Send(t, text)`, `Keys(t, oneOf{1,y,esc})`.
  - It validates targets.
- **Implements:** Decisions 2 and 3.
- **Invariant:** No code path sends a method outside the list. A bad target fails toward deny.
- **Verified today:** Methods and parameters in `hc schema`.
- **Depends on:** Step 1.
- **Check:** `go test ./...` with a fake `hc` (a test binary that records requests). The test fails if any other method appears, or if `force: true` is ever sent.

**Step 3. Identity, device store, admin socket, pairing.**
- **Change:**
  - Proposed `daemon/tls.go`, `daemon/devices.go`, `daemon/pair.go`, `daemon/admin.go`.
  - State lives in `~/Library/Application Support/herdr-remote/` (directory 0700): `server.key` (0600), `devices.json` (0600), and `admin.sock` (0600) for the CLI commands `pair`, `pair wait`, `pair approve`, `devices ls`, `devices rm`.
  - The QR code is SVG, from a small Go QR library.
  - Endpoints are the private IPv4 addresses of active interfaces, the 100.64.0.0/10 address, and the MagicDNS name if `tailscale` exists.
- **Implements:** Decisions 4, 6 and 7.
- **Invariant:** An unknown key fails the handshake. A token is single-use with a 120 s life. No device is stored without Allow. Each fails toward deny.
- **Verified today:** Nothing exists.
- **Depends on:** Step 2.
- **Check:**
  - Go tests: a token is reused or expired; a client cert is not enrolled; enroll without approve; revoke closes a live stream.
  - `openssl s_client -connect 127.0.0.1:7766` without a certificate fails.

**Step 4. Read API and events.**
- **Change:**
  - Proposed `daemon/api.go` and `daemon/state.go`.
  - Endpoints: `GET /v1/hello` (returns the current endpoints), `/v1/spaces`, `/v1/agents`, `/v1/terminals/{t}`, and `GET /v1/events` (SSE).
  - State merges `herdr-app ls`, refreshed on `created`, `finished`, `removed` and `labels` events, with hc terminal facts.
- **Implements:** Decisions 2 and 5.
- **Invariant:** On hc loss, 503 and reconnect. On `missed`, a fresh snapshot.
- **Verified today:** `herdr-app ls` shape; event kinds in `events.subscribe`.
- **Depends on:** Step 3.
- **Check:** Go tests with the fake hc; `curl --cert …` against a dev instance.

**Step 5. Live stream.**
- **Change:**
  - Proposed `daemon/live.go`: `GET /v1/terminals/{t}/live` runs `hc attach --observe --history 500 <t>` and copies its stdout, flushing as it goes.
  - It allows 2 per device, and kills the process on disconnect or revocation.
- **Implements:** Decision 8.
- **Invariant:** The phone never writes to the stream. When the viewer leaves, the subprocess ends; no orphan stays.
- **Verified today:** The probe streamed colored bytes to a pipe.
- **Depends on:** Step 4.
- **Check:** A Go test that counts subprocesses; a manual `curl` shows bytes.

**Step 6. Transcripts.**
- **Change:**
  - Proposed `daemon/transcript.go` with the Claude and Codex resolvers and parsers.
  - Endpoint: `GET /v1/terminals/{t}/transcript?before=<offset>`, returning normalized items `{role, kind: text|tool_call|tool_result, text, ts}`, with tool results truncated to 2000 characters. Thinking blocks are skipped.
- **Implements:** Decision 9.
- **Invariant:** Only files whose real path is under the two roots are read. A failure returns terminal text, never an error page.
- **Verified today:** The mapping chains and record types above.
- **Depends on:** Step 4.
- **Check:** Go tests on fixture JSONL files built in the test, including a partial last line and a symlink pointing outside the roots.

**Step 7. Approvals and messages.**
- **Change:**
  - Proposed `daemon/actions.go`.
  - `GET /v1/terminals/{t}/prompt` returns `{msg, block, turn, prompt_hash}`.
  - `POST /v1/terminals/{t}/decision` takes `{decision, turn, prompt_hash}`.
  - `POST /v1/terminals/{t}/message` takes `{text}`.
- **Implements:** Decisions 10, 11 and 12.
- **Invariant:** A key is pressed only after every guard passes, under the per-terminal lock. Otherwise 409 and no key; this fails toward deny.
- **Verified today:** The keys and screens above. `terminal.keys` has no condition, so a narrow race remains.
- **Depends on:** Step 4.
- **Check:**
  - Go tests: stale turn, changed hash, missing option text, a question kind, and an unknown app each press no key.
  - Manual: probe agents as in the verification above.

**Step 8. Herdr mod and install.**
- **Change:**
  - Proposed `mod/herdr-remote/main.js` and `mod.json` (an overlay). It shows the QR code, the pending device with its code and Allow/Deny, and the paired devices with Revoke, all through `exec("herdr-remoted", …)`.
  - Proposed `scripts/install.sh`: builds the daemon to `~/.local/bin/herdr-remoted`, writes `~/Library/LaunchAgents/dev.herdr-remote.plist`, copies the mod, and runs `herdr-app check`.
- **Implements:** Decisions 1 and 6.
- **Invariant:** No shipped mod or setting is edited. The mod has no secrets: it only shows the QR code while open.
- **Verified today:** Mod APIs and `mod.open`.
- **Depends on:** Step 3.
- **Check:** `herdr-app check ~/.config/herdr-app/mods/herdr-remote`; `herdr-app do mod.open --name herdr-remote`; `herdr-app shot` shows the QR code.

**Step 9. Android pairing and connection.**
- **Change:**
  - Proposed `android/app/...`: Keystore key, client certificate, pinned TrustManager, endpoint fallback, Pair screen with the code scanner and the 6-digit code.
- **Implements:** Decisions 4, 6, 14 and 15.
- **Invariant:** A pin mismatch never connects, and the private key never leaves the Keystore.
- **Depends on:** Steps 3 and 4.
- **Check:** JVM tests for QR payload parsing and the pin check. An emulator test pairs against a dev daemon at `10.0.2.2`.

**Step 10. Android lists, events, background service.**
- **Change:** Spaces, Space, and All agents screens; an SSE reducer; the foreground service; notifications.
- **Implements:** Decision 13.
- **Invariant:** A stale view shows its age; one notification per terminal and turn.
- **Depends on:** Step 9.
- **Check:** A JVM test for the reducer and the de-duplication; a manual test with the app closed.

**Step 11. Android agent screen.**
- **Change:** Transcript tab, Live tab (a WebView with xterm.js, assets only, no network or file access), Approve and Deny with the prompt block, and a Message box.
- **Implements:** Decisions 8 to 12.
- **Invariant:** The WebView loads only bundled assets.
- **Depends on:** Step 10.
- **Check:** Instrumented test for the agent screen; manual approve and deny on probe agents.

**Step 12. End-to-end acceptance.**
- **Change:** None. Run criteria 1 to 10 on the emulator and a real phone, including Tailscale.
- **Depends on:** All steps.
- **Check:** The list in section 6.

## 6. Testing strategy

- **Criteria 1, 2, 9:**
  - Go TLS, pairing and revocation tests (proposed).
  - `openssl s_client` without a certificate.
  - Manual pairing.
- **Criteria 3, 4, 5:** Go state, transcript and live tests with the fake hc (proposed); manual.
- **Criteria 6, 7:**
  - Go guard tests (proposed).
  - Manual with probe agents started as `claude --permission-mode default` and `codex -s read-only -a on-request`, asked to `touch` a scratch file.
- **Criterion 8:** Manual, with the app swiped away.
- **Criterion 10:** Manual over Tailscale on cellular.
- **Integration:**
  - Optionally against a separate core (`HC_SESSION=herdr-remote-test` with scratch XDG paths), never the user's own core.
  - Android: `./gradlew testDebugUnitTest`, `./gradlew connectedDebugAndroidTest` on the emulator.

## 7. Constraints and open questions

Constraints:
- Herdr unchanged.
- No relay.
- The user installs Tailscale on the Mac and the phone for away access.
- The phone needs Google Play services for the code scanner.
- The macOS firewall may ask once to allow incoming connections for `herdr-remoted`.

Non-blocking assumptions:
- `~/.claude/sessions/<pid>.json` and the Codex rollout layout are undocumented; the fallback covers a change.
- Prompt text checks are tied to Claude Code 2.1.x and Codex 0.167.x. A version change fails toward view-only.

All steps are ready.

## 8. For the implementer

1. Re-verify the facts in section 2 with the commands named there.
2. Work in step order, ending each step with its Check and a commit.
3. Record as-built notes in a dated "As built" section at the end of this document.
4. Stop if a verified fact no longer holds: a key, a screen string, the `hc attach` behaviour, or a transcript path.

## 9. Verification status

Herdr 0.1.13, hc 0.1.18, Claude Code 2.1.295 and Codex 0.167.0 were inspected on 2026-11-08. The approval keys, the attach stream, the transcript mapping and bare-environment access were verified with probe terminals. The round-2 check is pending.

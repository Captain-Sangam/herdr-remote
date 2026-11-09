# Design debate: herdr-remote

This file is the full record of how the design in `PRD.md` and `TDD.md` was reached. You don't need it to build anything; it explains why each decision is what it is, who argued what, and what was checked against the real tools.

Two stages:

1. **Planning debate.** Two agents worked from the approved brief, in two rounds:
   - Claude, model Opus 5.5 with xhigh effort, called the *Claude planner* below;
   - Codex, model GPT-6.1-Sol with xhigh effort, called the *Codex planner*.
2. **Plan review.** Two fresh agents reviewed and corrected the resulting plan, in three rounds:
   - Claude, model Fable 5.1, called the *Fable reviewer*;
   - Codex, model GPT-6-Astra, called the *Astra reviewer*.

All agents ran read-only in Herdr.app terminals: they could not edit files, and the orchestrator checked the repo after every round. The orchestrator wrote the brief, verified claims against the installed tools, settled the disputes, and wrote the plan and these documents.

## 0. The brief and the user's decisions

**The request:**
- An Android app to control Herdr sessions on the Mac.
- Pairing by scanning a QR code that Herdr shows.
- Screens: spaces, the agents in a space, an agents view, and per agent the chat transcript, approve actions, and a live view of the terminal "like screen sharing".
- It must be safe and secure.

**What the orchestrator found before the debate:**
- Herdr.app (0.1.13) and herdr-core `hc` (0.1.18) are closed source, installed from a DMG. The design can only use their public interfaces:
  - the `hc` JSON-RPC API (`hc schema`), which also includes dangerous methods such as `machine.exec` and `file.write`;
  - the `herdr-app ls` window model;
  - Herdr mods, which are trusted JavaScript that can run programs and show images.
- No Android SDK, Gradle or Tailscale on the Mac. Java 17 and Go are present.

**The user's decisions, before any agent started:**
1. **Reach:** a direct connection over Wi-Fi, or over Tailscale when away. No relay server.
2. **Control:** watch, approve or deny, and send a message. No raw keyboard into the terminal.
3. **Toolchain:** install the Android command-line tools with Homebrew, without Android Studio.

**Criteria, in order:** security; works with Herdr as shipped; smallest system; a responsive live view.

## 1. Planning debate, round 1 (blind)

Both agents got the same prompt and could not see each other's answer.

### Claude planner (confidence 72)

- **Shape:** one Go daemon, `herdr-remoted`, started by a LaunchAgent, as the only thing the phone can reach. It talks to `hc` through `hc connect` subprocesses and to the window through `herdr-app ls`. It serves HTTPS and WebSocket on port 7766 behind mutual TLS.
- **Allow-list:** `hello`, `terminal.list`, `terminal.get`, `terminal.read`, `events.subscribe`, `terminal.send` (never `force`), and `terminal.keys` (fixed keys only).
- **Pairing:**
  - A Herdr mod opened with `mod.open` shows an SVG QR code holding the daemon's SPKI SHA-256, its LAN and Tailscale addresses, the port, and a single-use token valid for 120 s.
  - The phone makes a P-256 key in Android Keystore, connects with the daemon's hash pinned, presents a self-signed client certificate and redeems the token. The daemon stores the phone's SPKI hash.
- **Revocation:** `herdr-remoted devices rm`, or a button in the mod, deletes the device and closes its connections.
- **Live view:** run `hc attach --observe` inside a PTY (the `creack/pty` library), stream the bytes over a WebSocket to xterm.js in a WebView, and discard anything the phone sends on that socket.
- **Transcripts:**
  - Claude Code: process ID → `~/.claude/sessions/<pid>.json` → `~/.claude/projects/<slug>/<sessionId>.jsonl`. It checked this chain itself on a live terminal.
  - Codex: the newest rollout file whose `cwd` matches and whose timestamp is after the terminal's creation, with terminal text as the fallback.
- **Approvals:** the phone sends `{turn, msg_hash, req_id}`. The daemon re-checks that the agent is still blocked with the same turn and message and that the expected option text is on screen, then presses one fixed key:
  - Claude: `1` to approve, `esc` to deny.
  - Codex: `y` to approve, `esc` to deny.
  - Questions: digits 1 to 9.
  - Auth prompts and unknown apps: view-only.
- **Security details:**
  - Message text has C0 and C1 control characters removed, so a paste cannot be escaped.
  - Terminal addresses must match `^t[0-9a-z]+$`, and remote `machine/` addresses are refused.
  - A retried request ID returns the cached result for 10 minutes.
  - The Keystore key requires an unlocked device.
- **Background alerts:** an opt-in foreground service keeps the event stream open and posts local notifications.
- **Tests:** Go tests with a fake `hc`; integration on a separate `hc` session; `openssl s_client` without a certificate must fail; the Android SDK through Homebrew; an emulator for instrumented tests.

### Codex planner (confidence 83)

- **Shape:** the same overall shape: a Go broker, a small pairing mod, and a Kotlin and Compose app, with the same use of `hc connect`, the same `herdr-app ls` approach and the same allow-list.
- **Differences:**
  - **Pairing:** a Mac-side confirmation of the phone's key fingerprint before the phone is stored, so a token alone is not enough. Revocation is checked on every request.
  - **Live view:** start with `terminal.read` snapshots of the visible screen at 4 Hz, one request at a time, skipping unchanged screens. Measure that before taking on decoding of the undocumented framed stream.
  - **Transcripts:** use only explicit session identifiers; folder and time only suggest candidates. When unsure, ask the user on the Mac to choose. Store the binding against core identity, terminal, process and session, and drop it when any of those change.
  - **Approvals:** only "certified" prompt profiles. Claude approves with `enter`, but only when a test fixture proves "Yes" is selected. Codex approves with `y`, and both deny with `esc`. Questions, auth prompts, Pi and unknown versions are view-only. Decisions are bound to device, terminal, turn and a screen hash. Phones are serialized, and durable request IDs are consumed before a key is sent; after a crash the result is "outcome unknown" with no retry.
  - It named the remaining race: `terminal.keys` has no condition, so a check followed by a key press can never be fully atomic.
  - It proposed an OpenAPI file for the protocol.
- Its sandbox blocked `hc ls` and `herdr-app ls`, so it worked from the schema and the guides only.

### Where they agreed

- One Go daemon as the only network-facing piece.
- `hc` reached through `hc connect`, layout through `herdr-app ls`, and the same method allow-list.
- A QR code holding a key pin, LAN and Tailscale addresses and a short single-use secret.
- A Keystore key on the phone, with TLS client authentication.
- Revocation that closes open streams.
- A turn check before any key press.
- `esc` to deny, and `y` to approve Codex.
- Message cleaning.
- A foreground service with local notifications.
- The Homebrew SDK and an emulator for tests.

### Where they disagreed

1. Live view: stream (Claude) or 4 Hz snapshots (Codex).
2. Codex transcript: folder-and-time heuristic (Claude) or explicit IDs with a manual fallback (Codex).
3. Claude approve key: `1` (Claude) or `enter` when proven selected (Codex).
4. Questions: digits (Claude) or Mac-only (Codex).
5. A Mac-side confirmation at pairing: Codex only.
6. Idempotency: a 10-minute request-ID cache (Claude) or durable request IDs (Codex).

### What the orchestrator verified after round 1

The orchestrator settled most of the disputes with throwaway probe terminals, all removed afterwards.

- **Live view.** `hc attach --observe` writes the first paint and then live bytes, with colors, to a plain pipe, without any TTY. Under a PTY it only adds terminal queries.
  - So the stream needs neither a PTY nor any decoding of the framed protocol, and the main cost of streaming that the Codex planner worried about disappears.
  - This settled dispute 1 in favour of streaming and removed the Claude planner's PTY dependency.
- **Approvals.** Real permission prompts were created on purpose:
  - Claude Code 2.1.295, in default permission mode, reported `{app: claude-code, kind: permission, msg: "approve Bash: touch …", state: blocked}`. The options were 1 Yes, 2 always allow, 3 switch to auto mode, 4 No, then "Esc to cancel".
  - Key `1` approved: the file was created and the turn moved on. Key `esc` denied: no file was created.
  - Because "No" was option 4, a digit is not a safe way to deny. `1` selects Yes wherever the cursor is, which is safer than `enter`. This settled dispute 3 for `1`.
  - Codex 0.167.0 (this version rejects `-a untrusted`; `-a on-request` was used) reported `kind: permission` with no message. The screen showed "1. Yes, proceed (y)" and "3. No … (esc)". `y` approved and `esc` denied.
- **Transcripts.** The Codex process keeps its rollout file open, so `lsof` on it gives an explicit mapping. That meets the Codex planner's requirement for explicit identity without the manual fallback, and beats the Claude planner's heuristic. This settled dispute 2.
- **Environment.** `hc` and `herdr-app` both work from a bare environment, so a LaunchAgent can run them.
- **Labels and overlays.** Herdr's own terminals carry `herdr.space` and `herdr.tab` labels, and `mod.open` exists to show a mod as an overlay.
- **Schema.**
  - `Status.kind` is `permission`, `question` or `auth`.
  - `turn` goes up each time the status leaves `working`.
  - The `if` condition of `terminal.send` accepts only idle or working.
  - `terminal.keys` has no condition at all. The Codex planner was right that the race is real, though narrow.

The orchestrator then wrote a full draft plan and sent it to both agents for round 2.

## 2. Planning debate, round 2

Each agent got the orchestrator's synthesis, the verified facts, the draft, and challenges aimed at its own position.

### Claude planner (confidence 78)

**Conceded:**
- The PTY: a pipe is enough.
- The `lsof` mapping, with two refinements:
  - npm-installed Codex shows `node` as the foreground process, so walk its child processes to find `codex`.
  - If several rollout files are open, pick the one whose `session_meta` has no `parent_thread_id`.
- Dropping the unlock-required key in favour of a biometric gate in the app.
- Server-Sent Events instead of WebSocket, if keepalives are added.
- View-only questions, and the Mac-side confirmation at pairing.

**Objected** to relying on the state check alone for repeated approvals:
- Two phones tap Approve, or one phone retries. The first request passes the check and presses `1`.
- The second gets the lock before Claude has reported "working", so the state, turn and hash still match, and it presses `1` again.
- That key lands in the input box, or in a new dialog if one opens during the ~3 s quiet wait.
- Its fix: an in-memory "pressed" marker per terminal that the check refuses, plus comparing the result of the key press.

**Plan check:**
- (a) Step 3's test revokes a live stream that does not exist until Step 5.
- (b) A mod's `image()` only loads a file. An SVG QR code would therefore write the token to disk. Draw the QR with `div()` cells instead.
- (c) The TLS handshake cannot see the request path, so unknown keys must be accepted only while pairing is open, limited to the enroll call. Enroll long-polls until Allow, and the token binds to the first key that uses it.
- (d) An idle terminal sends no bytes, and `hc` heartbeats only every 15 s, while OkHttp's default read timeout is 10 s. Send keepalive comments and turn the timeout off on streams.
- (e) Actions wait for local typing to stop and may block a shared `hc connect`. Give actions their own connection.
- (f) Reopen the live stream on each size change.
- (g) Run every subprocess through one helper with fixed arguments, and test it.
- (h) De-duplicate notifications by turn and prompt.
- (i) A space rename sends no `hc` event, so re-run `herdr-app ls` on each spaces request, and drop remote spaces.
- (j) Add the `POST_NOTIFICATIONS` permission and the `specialUse` foreground-service type.

### Codex planner (confidence 92)

**Conceded:** streaming, the Claude key `1`, explicit transcript lookups with terminal-text fallback, and dropping the OpenAPI file.

**Objected** to the state-only check, with the same kind of sequence: the key is injected, the agent has not redrawn yet, the response is lost, and the retry passes the check. It proposed a durable action ID consumed before the key is sent, and "outcome unknown" without a second key, with a test that holds the status unchanged after the key, drops the response, retries, and asserts that only one key was pressed.

**Plan check:**
- The pairing design cannot work as written: a strict handshake also rejects new phones. Add a temporary listener used only for enrollment and advertised in the QR, and test that a fresh enrollment succeeds, not only that bad ones fail.
- Define and test recovery after a resize: restart the observer for a fresh paint.
- The biometric gate needs its own step and tests.
- Spaces and tabs belong to the window, so add `herdr-app ls` reconciliation and a rename test.
- The background-alert criterion must say "while the service runs and is connected".

### What the orchestrator verified after round 2

- A mod's `image(path)` and `svg(path)` load only files, and `div()` exists (`gpui-kit.d.ts` lines 20271 to 20290). Point (b) holds.
- Codex "guardian" sub-agent threads write their own rollout files with `parent_thread_id` set, and the main thread has none. In 23 of the 40 most recent rollouts it was set. The refinement is necessary, not cosmetic.
- `KeysResult` returns `{status, turn}`.
- Event change payloads are not described in the schema. A 20-second sample showed only the snapshot and a heartbeat. The implementer must capture real events before writing the reducer.
- A user terminal's process was named `claude.exe`, so process names vary. The process-ID lookup does not depend on the name.

### How each dispute was resolved

- **Repeated approvals.** Both objections were correct about the sequence. The plan takes the in-memory consumed marker (set before the key is pressed; any later decision for the same turn and prompt returns "already answered"), plus reporting of "applied" or "sent but not yet confirmed" without a second key.
  - The durable store was rejected: the marker only has to outlive the gap between the key and the agent's next status report, and a restarted daemon sees the new state anyway.
  - The remaining risk is written down: `hc` holds keys while someone types at the Mac, so a key can land on a different screen, and no shipped API can prevent this.
- **Enrollment.** A separate enrollment-only listener (the Codex planner's design) was chosen over path middleware, because it keeps "the paired listener never accepts an unknown key" true at all times. The Claude planner's long-poll and token binding were kept.
- **Adopted from the plan checks:**
  - the QR drawn with `div()`;
  - three `hc connect` connections;
  - one subprocess helper;
  - SSE keepalives and no read timeout;
  - a restart on resize;
  - spaces refreshed per request;
  - notification de-duplication;
  - the Android permission and service type;
  - a step for the biometric gate;
  - the Step 3 test moved to Step 5.
- **No third planning round.** Every point had an evidence-backed decision, and the separate three-round review was due to check the result anyway.

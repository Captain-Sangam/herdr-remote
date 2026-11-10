# Design debate: herdr-remote

This file is the full record of how the design in `PRD.md` and `TDD.md` was reached. `PRD.md` is now [`docs/product.md`](../product.md). `TDD.md` is now [`docs/history/technical-design.md`](technical-design.md). You don't need it to build anything. It explains why each decision is what it is, who argued what, and what was checked against the real tools.

Two stages:

1. **Planning debate.** Two agents worked from the approved brief, in two rounds:
   - Claude, model Opus 5.5 with xhigh effort, called the _Claude planner_ below
   - Codex, model GPT-6.1-Sol with xhigh effort, called the _Codex planner_.
2. **Plan review.** Two fresh agents reviewed and corrected the resulting plan, in three rounds:
   - Claude, model Fable 5.1, called the _Fable reviewer_
   - Codex, model GPT-6-Astra, called the _Astra reviewer_.

All agents ran read-only in Herdr.app terminals: they could not edit files, and the orchestrator checked the repo after every round. The orchestrator wrote the brief, verified claims against the installed tools, settled the disputes, and wrote the plan and these documents.

## 0. The brief and the user's decisions

**The request:**

- An Android app to control Herdr sessions on the Mac.
- Pairing by scanning a QR code that Herdr shows.
- Screens: spaces, the agents in a space, an agents view, and per agent the chat transcript, approve actions, and a live view of the terminal "like screen sharing".
- It must be safe and secure.

**What the orchestrator found before the debate:**

- Herdr.app (0.1.13) and herdr-core `hc` (0.1.18) are closed source, installed from a DMG. The design can only use their public interfaces:
  - the `hc` JSON-RPC API (`hc schema`), which also includes dangerous methods such as `machine.exec` and `file.write`
  - the `herdr-app ls` window model
  - Herdr mods, which are trusted JavaScript that can run programs and show images.
- No Android SDK, Gradle or Tailscale on the Mac. Java 17 and Go are present.

**The user's decisions, before any agent started:**

1. **Reach:** a direct connection over Wi-Fi, or over Tailscale when away. No relay server.
2. **Control:** watch, approve or deny, and send a message. No raw keyboard into the terminal.
3. **Toolchain:** install the Android command-line tools with Homebrew, without Android Studio.

**Criteria, in order:**

1. Security.
2. Works with Herdr as shipped.
3. The smallest system.
4. A responsive live view.

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
- **Tests:**
  - Go tests with a fake `hc`.
  - Integration on a separate `hc` session.
  - `openssl s_client` without a certificate must fail.
  - The Android SDK through Homebrew, and an emulator for instrumented tests.

### Codex planner (confidence 83)

- **Shape:** the same overall shape: a Go broker, a small pairing mod, and a Kotlin and Compose app, with the same use of `hc connect`, the same `herdr-app ls` approach and the same allow-list.
- **Differences:**
  - **Pairing:** a Mac-side confirmation of the phone's key fingerprint before the phone is stored, so a token alone is not enough. Revocation is checked on every request.
  - **Live view:** start with `terminal.read` snapshots of the visible screen at 4 Hz, one request at a time, skipping unchanged screens. Measure that before taking on decoding of the undocumented framed stream.
  - **Transcripts:** use only explicit session identifiers. Folder and time only suggest candidates. When unsure, ask the user on the Mac to choose. Store the binding against core identity, terminal, process and session, and drop it when any of those change.
  - **Approvals:** only "certified" prompt profiles. Claude approves with `enter`, but only when a test fixture proves "Yes" is selected. Codex approves with `y`. Both deny with `esc`. Questions, auth prompts, Pi and unknown versions are view-only. Decisions are bound to device, terminal, turn and a screen hash. Phones are serialized, and durable request IDs are consumed before a key is sent. after a crash the result is "outcome unknown" with no retry.
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
  - Codex 0.167.0 (this version rejects `-a untrusted`, so `-a on-request` was used) reported `kind: permission` with no message. The screen showed "1. Yes, proceed (y)" and "3. No … (esc)". `y` approved and `esc` denied.
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

- **Repeated approvals.** Both objections were correct about the sequence. The plan takes the in-memory consumed marker (set before the key is pressed, so any later decision for the same turn and prompt returns "already answered"), plus reporting of "applied" or "sent but not yet confirmed" without a second key.
  - The durable store was rejected: the marker only has to outlive the gap between the key and the agent's next status report, and a restarted daemon sees the new state anyway.
  - The remaining risk is written down: `hc` holds keys while someone types at the Mac, so a key can land on a different screen, and no shipped API can prevent this.
- **Enrollment.** A separate enrollment-only listener (the Codex planner's design) was chosen over path middleware, because it keeps "the paired listener never accepts an unknown key" true at all times. The Claude planner's long-poll and token binding were kept.
- **Adopted from the plan checks:**
  - the QR drawn with `div()`
  - three `hc connect` connections
  - one subprocess helper
  - SSE keepalives and no read timeout
  - a restart on resize
  - spaces refreshed per request
  - notification de-duplication
  - the Android permission and service type
  - a step for the biometric gate
  - the Step 3 test moved to Step 5.
- **No third planning round.** Every point had an evidence-backed decision, and the separate three-round review was due to check the result anyway.

## 3. Plan review, round 1 (blind)

Two fresh agents, which had not seen the planning debate, reviewed the plan against the brief. Each was asked for at most ten findings, each with evidence, likelihood (1 to 10), severity (1 to 10) and a proposed correction. They could not see each other's findings.

### Fable reviewer: 10 findings

1. **Codex in daemon mode breaks the transcript lookup** (likelihood 8, severity 6).
   - A Codex started as plain `codex`, which is how Herdr's agent menu and the user's own terminals launch it, runs its threads in a shared `codex app-server` daemon. No process of the terminal holds the rollout file.
   - The fallback is also thinner than the plan said: `terminal.read` returns only the visible screen while an agent is working or blocked.
2. **A revoked phone could resume a TLS session** (6, 7). Go skips `VerifyPeerCertificate` on resumed connections, and both Go and Android cache sessions by default. A revoked phone could resume past the key check, and the criterion "later handshakes fail" would be false.
3. **The overlay cannot wait 120 s** (9, 5). A mod's `exec` has a 30-second limit (`herdr.d.ts:10`), so a blocking `pair wait` breaks the overlay on any slow scan. `exec` also resolves the binary through `PATH`.
4. **Dead connections go unnoticed** (7, 5). With the read timeout off, a socket left half-open by a network change or Mac sleep is never detected, so background alerts stop silently.
5. **The guard was only tested on default-mode command prompts** (5, 6). Auto-mode prompts and Codex file-edit approvals might look different, and the Check would not notice.
6. **TLS and key settings are unstated** (7, 5). Each one, if guessed wrong, means every handshake fails: `RequireAnyClientCert`, TLS 1.3, the Keystore key's purpose and digests, an OkHttp hostname verifier that accepts IP addresses and MagicDNS names.
7. **No logging policy** (9, 4). Nothing says what is logged, or that tokens, codes and transcript text must never be. The device name sent during enrollment is attacker-controlled and unbounded.
8. **Foreground-service details are missing** (8, 4): the permissions, the service type property, and a restart after reboot or app update.
9. **`hc` might tag the phone's message** (5, 5). `hc` may wrap sent text as `<hc from=…>`, which the `hc` guide tells agents to treat as a report rather than a user instruction.
10. **The QR drawing is over-built** (5, 2). Drawing the QR with thousands of `div()` cells and a one-second countdown redraws the whole Herdr window every second.

Its completeness verdict: transcript, approve or deny, and security were partial. Everything else was delivered.

### Astra reviewer: 10 findings

1. **A held key can approve an unseen action** (6, 9).
   - Someone typing at the Mac makes `hc` hold the phone's key. By the time it lands, the prompt may have changed, and the key approves an action nobody saw.
   - The per-terminal lock cannot exclude local input. It asked for this to be treated as unresolved.
2. **Restarts and retries can repeat actions** (6, 8). A daemon restart loses the in-memory marker, so a replay could press again. Messages have no idempotency at all. It wanted durable action IDs.
3. **Same-user processes are not separated** (5, 9). Any process running as the same user can read the server key or run `pair allow`, and file permissions cannot stop it. Either declare such processes trusted, or redesign the boundary.
4. **Project data leaks through phone surfaces** (7, 7): the recents screenshot after the biometric timeout, notification text on the lock screen, and logs.
5. **TLS resumption** (5, 5): the same issue as the Fable reviewer's second finding.
6. **Hostname verification** (8, 7). A certificate with the right key but without the current address still fails OkHttp's hostname check.
7. **The 30-second `exec` limit**: the same as the Fable reviewer's third finding.
8. **Foreground-service permissions**: the same as the Fable reviewer's eighth.
9. **xterm.js can overflow** (7, 6). Unbounded base64 delivery can outpace xterm.js, which drops data past its write buffer.
10. **Transcript pages can start mid-record** (9, 6). Reading the last 2 MB usually starts inside a JSON record, which would turn a valid transcript into terminal text. Pagination needs record alignment and cursors.

Its completeness verdict: almost every capability was partial.

### What the orchestrator verified

- **Codex daemon mode: confirmed.**
  - The user's terminal t1v runs `node /opt/homebrew/bin/codex`. Its processes hold only Unix sockets. The shared `codex app-server` holds a lock file per thread under `~/.codex/thread-writer-locks/`.
  - A Codex that has not been prompted has no rollout file yet.
  - Every rollout's first line records `cwd`, `source` and `parent_thread_id`, which is enough for a strict folder match.
  - A `threads` table in `~/.codex/state_5.sqlite` was also found. It was not used, because it is undocumented and would need SQLite.
- **TLS resumption: confirmed** from Go's documentation. `VerifyPeerCertificate` is not called on resumption, while `VerifyConnection` runs on every connection.
- **The 30-second `exec` limit: confirmed** (`herdr.d.ts:10`). Two more facts came out of the guide:
  - saving any non-hidden file in a mod folder reloads the mod (`AGENTS.md:789`)
  - every mod redraw redraws the whole window (`AGENTS.md:893`).
- **Other prompt kinds: verified with new probes.**
  - Claude Code's Write-tool prompt ("Do you want to create …?", "1. Yes", "Esc to cancel") and Codex's apply_patch prompt ("Would you like to make the following edits?", "1. Yes, proceed (y)", "3. No … (esc)") use the same markers as the command prompts.
  - A Claude auto-mode prompt cannot be produced on demand, so it remains an assumption that fails toward view-only.
- **Message tagging: refuted for this design.** A message sent with `hc send` from a process outside any terminal arrived as a plain prompt with no `<hc` tag, and the probe agent answered it ("PONG").
- **The held-key race: confirmed as residual, and a mitigation was found.** `ioreg -c IOHIDSystem` reports `HIDIdleTime`, the time since the last keyboard or mouse input on the Mac. It can serve as a "Mac in use" guard before and after the key press.
- **Everything else** was accepted on the reasoning given, with no contrary evidence.

### Corrections made for round 2

- **TLS:** TLS 1.3 only, `RequireAnyClientCert`, session tickets off, and the key check in `VerifyConnection`. Plus a test that a revoked phone cannot resume.
- **Pairing:** the overlay polls a `pair status` call that returns at once. It calls the daemon by an absolute path. The device name is limited to 64 printable characters. The QR is drawn as run-length rows with an expiry time instead of a countdown.
- **Connections:** the SSE read timeout is 30 s with automatic reconnect. Live views reopen after a reconnect, and the phone never retries an action by itself.
- **Live view flow control:** chunks of at most 64 KiB, each acknowledged by xterm.js. Above 1 MiB pending, the stream is reopened for a fresh paint.
- **Transcripts:** a third source for daemon-mode Codex, which matches by folder only when exactly one session fits. Pages are aligned to records, and a cursor is tied to the file's inode.
- **Approvals and messages:**
  - request IDs on both
  - the audit log writes a `dispatching` line before the key and a result line after, and is reloaded on start, so a restart returns `outcome_unknown` and never presses twice
  - the Mac-in-use guard (`mac_in_use` if the Mac had input in the last 10 s, `uncertain` if input arrives during the press)
  - messages never carry a `caller`.
- **Android:** the full list of foreground-service permissions, with restart on boot and on app update. The Keystore key parameters are spelled out. `FLAG_SECURE` is set, notifications are private, and transcript and terminal data stay in memory only.
- **Two new decisions:**
  - an audit log that never contains tokens, codes, transcript text, terminal bytes or message text
  - a stated threat-model boundary: processes running as the same macOS user are trusted, because they can already drive `hc` directly.
- **Rejected or narrowed:**
  - a separate durable action store, because the audit log already is one
  - leaving the race unresolved, because it is now mitigated and the remaining part is documented
  - dropping the `div()` QR, because a file would put the token on disk
  - reading Codex's SQLite index.

## 4. Plan review, round 2

Each reviewer got the orchestrator's verdict on every finding from both reviewers, the corrections, and the revised plan. The task was to accept, reject or amend each correction with evidence, and to check the revised plan for anything the corrections broke.

### Fable reviewer

**Accepted** the corrections for TLS resumption, the `exec` limit, half-open sockets, unverified prompt types, the TLS and key settings, logging, the foreground service and the QR drawing. **Withdrew** the message-tagging finding, because the orchestrator's probe refuted it.

**Amended the Codex daemon-mode fix:**

- As written, it still fails on this Mac. t1v (daemon mode) has two candidate rollouts after its start time. One of them belongs to t1x, which runs with `--no-daemon` in the same folder. "Several candidates" sends t1v to terminal text.
- Fix: remove rollouts that the explicit lookups already bind to other terminals.
- Also: the daemon closes idle rollouts, so a scan by modification time is right, and `lsof` on the daemon is not.

**Amended the Mac-in-use guard:**

- `HIDIdleTime` works here, but it sees only local keyboard and mouse input, and perhaps Screen Sharing (not verified). It does not see SSH, or `hc keys` from agents.
- A mouse jiggler or any device with periodic reports would keep the Mac "in use" forever, so the phone could never approve.
- Fix: make the threshold a setting, return the measured idle time in the refusal, and audit refusals.
- A sleeping display is fine, because idle time only grows.

**New findings:**

- The audit line has no prompt hash or decision, so the consumed marker cannot be rebuilt after a restart as the plan claims. Nothing says the `dispatching` line is fsynced before the key, that a failed write refuses the action, or that the reload reads the rotated file.
- `/v1/enroll` long-polls up to 120 s, but OkHttp's default 10 s read timeout would abort it before Allow. The planned check delayed the scan, not the Allow, so it would not catch this.
- Opening the overlay always opened a pairing window, so revoking a lost phone also opened port 7767 for 120 s.
- The testing table asked for a manual check on "a plain shell", which is not an agent under the plan's own rules, so that check could not run.

Its conclusion: nothing else material remained.

### Astra reviewer

**Accepted** most corrections, and **rejected** its own reading of the tagging finding in light of the probe.

**Amended the Codex daemon-mode fix:** a unique candidate is not an identity. Terminal A has had no prompt yet, so it has no rollout. Terminal B starts later in the same folder and makes one. A would show B's transcript. It required an exact binding or the terminal-text fallback.

**Amended the Mac-in-use guard:**

- The check ran before the lock, so a wait for the lock could make it arbitrarily stale.
- Move it to just before the key, fail closed when the idle time cannot be read, and call it a heuristic.
- Criterion 6 promised more than the guard delivers.

**Amended the idempotency fix:**

- Persist the request ID, a digest of the payload and the prompt hash before the action, and fail closed if that write fails.
- Make sure rotation and the 10-minute window cannot forget IDs.
- The message criterion must promise "at most once", not "exactly once".

**Other amendments:**

- Content-free logging must also cover the debug build that the testing section installs.
- The transcript step said "never an error", but the plan also returns 409 for a stale cursor.

**New finding, the most severe of the round:** the TLS decision applied the device-store check to both listeners. Built literally, the enrollment listener would reject every new phone, because a new phone is not in the store yet. First pairing would always fail.

### What the orchestrator verified

- **Codex lock files.** `lsof +D ~/.codex/thread-writer-locks` showed:
  - the daemon-mode app-server holds the lock of t1v's thread (t1v had no prompt yet)
  - each `codex --no-daemon` process holds its own lock
  - the ChatGPT desktop app's own app-server holds its own.
- **t1x.** It is a `codex --no-daemon` session in the same folder as t1v, started later. This confirmed the Fable reviewer's counter-example.
- **The other items** were checked against the plan text. Each was a real defect in the revision.

### Resolution

- **Codex daemon mode.** The two amendments were combined into one rule: bind by elimination.
  - Candidates: rollouts of live daemon threads (locks held by an app-server) that match the folder, source and start time, minus rollouts that other terminals already own.
  - The binding holds only if there is exactly one candidate and exactly one unbound Codex terminal in that folder.
  - This handles the Fable case (t1x's rollout is excluded), the Astra case (two daemon-mode terminals in one folder fall back to terminal text), and a session that ended (its lock is gone).
- **Approval guard order.** The guard now runs in this order:
  1. request ID
  2. lock
  3. state, hash and markers
  4. consumed marker
  5. the idle check, last
  6. a fsynced `dispatching` audit line with the prompt hash and decision
  7. one key.

  Other rules:
  - A failed audit write gives 503 and no key.
  - The threshold is a setting, and 0 turns the guard off.
  - An `ioreg` failure refuses.
  - The daemon rebuilds IDs and markers from both the current and the rotated audit log.

- **Messages.** A fsynced line with the request ID and a SHA-256 of the text, sent at most once.
- **Pairing and the overlay.**
  - Enrollment gets a 130 s client timeout, and a test with Allow given 60 s late.
  - The overlay opens to the device list, and only **Pair a phone** opens port 7767.
- **Listeners.** The checks are written separately for each listener: the device store on 7766, and only a well-formed certificate plus the token on 7767. A test enrolls a never-seen key through the real 7767 listener.
- **Other fixes:** no content logs in any build, the stale-cursor wording, and the testing-table row.
- **The documents.** The orchestrator then wrote the corrected plan as the two final documents, `PRD.md` and `TDD.md`, for the reviewers to check in round 3.

## 5. Plan review, round 3 (the drafts of PRD.md and TDD.md)

The orchestrator wrote the corrected plan as drafts of the two final documents. Both reviewers checked them against the earlier plan, the brief and the installed tools, and gave a verdict.

### Fable reviewer: `VERDICT: READY`

- **All round-two items: fixed.**
  - It verified the Codex lock evidence again. The daemon holds the lock of t1v's thread, and the thread's time (UUIDv7, 19:31:21Z) matches t1v's status time (19:31:25Z).
  - t1v had no prompt, so it falls to terminal text as designed.
  - The daemon keeps the locks of idle threads for over an hour, so "live" was the right test.
- **Small new findings:**
  - The SSE stream had no event for an agent that finished or was removed, so the phone could not drop a dead agent until the next snapshot (likelihood 8, severity 3).
  - The SHA-256 of a short message such as "yes" or "continue" can be guessed, against the no-content logging rule (5, 2).
  - "Do not overwrite a folder that exists" meant a reinstall never updates the mod, and the check would pass on the old copy (6, 2).
  - Three details were still guessed: how "input during the call" is computed, the status code of `/v1/enroll` after a cancel, and the shape of `status` in `GET /v1/terminals/{t}` (7, 2).
  - The PRD listed three approval results where the TDD had four.
- **Facts.** The tool facts in the TDD match what it could see: the lock holders, rollout names, `HIDIdleTime`, `hc events`, and the `exec` limit.

### Astra reviewer: `VERDICT: NOT READY`

- **Fixed:** the enrollment timeout, the overlay button, the testing row, the idle-check order, logging in every build, the cursor wording, and the listener-specific checks.
- **Not fixed: Codex attribution.**
  1. Start a Herdr Codex terminal A, and give it no prompt.
  2. Start a daemon-mode Codex B outside Herdr, in the same folder, and prompt it.
  3. B's rollout is the only candidate, and A is the only Herdr terminal there, so the rule gives B's transcript to A.

  Neither `source: cli` nor an app-server lock ties a thread to a Herdr terminal. This contradicts the PRD's promise of transcripts "identified for certain".

- **Not fixed: replay retention.** Request IDs were arbitrary UUIDs with no creation time that the daemon could check. After the daemon forgot an ID, it could not tell a replay from a new request. Two rotations could also drop IDs that were still valid. The PRD still promised that a message arrives "once".
- **New, severity 7:** the PRD said every unpaired TLS connection is rejected, but the TDD needs an unpaired phone to complete TLS on the enrollment port. Built literally from the PRD, pairing is impossible.

### What the orchestrator verified

- **No exact binding.** The files in `~/.codex/tui-thread-reference-capabilities/` are empty markers, so no shipped interface ties a daemon-mode terminal to its thread.
- **More holes in the folder match**, beyond the reviewer's example:
  - a daemon-mode session that exited keeps its lock
  - a resumed thread has an old timestamp.

  Each can make it show another session's transcript.

- **Herdr's launcher.** Herdr's own launcher for Codex runs plain `codex` (`shared.json`, `new-menu.agents[1].run = ["codex"]`), so daemon mode is the default for the user's agents. `hc --skill` already says Codex under hc must run with `--no-daemon`.
- **The other findings** were checked against the drafts and are correct.

### Final resolution (made after round 3, at the round cap)

- **Codex transcripts.** The folder match for daemon-mode Codex is removed.
  - A Codex in daemon mode shows terminal text, with the hint "Start Codex with `--no-daemon` for the full transcript".
  - The setup tells the user to set Herdr's Codex launcher to `["codex", "--no-daemon"]` once. `install.sh` prints this, but does not change Herdr settings.
  - **Reason:** a wrong transcript is worse than terminal text, because the user reads it before an approval. Three rounds kept finding new ways the match could be wrong. This sides with the Astra reviewer's position and overrides the Fable reviewer's "ready" on this point.
- **Replay.** Request IDs are now UUIDv7, so each one carries its time.
  - The daemon refuses an ID more than 5 minutes from its clock, and remembers each ID for 10 minutes.
  - The phone corrects its clock with `server_time_ms` from `/v1/hello`, so clock skew does not matter.
  - A rate limit of 2 actions a second for each device keeps 10 minutes of audit lines far below the 10 MiB rotation size, so a rotation cannot drop a valid ID.
  - Consumed markers live until the terminal's turn changes.
  - The PRD now says "at most once", with "outcome unknown" as a possible result.
- **Handshake guarantee.** The PRD and the TDD now limit "only paired phones complete the handshake" to the API port 7766, and describe the enrollment port as the exception. That port opens only in a window that the user starts, and it is authorized by the one-time token.
- **Fable's small findings, all adopted:**
  - an `agent_removed` event, with a reducer test
  - the message length in the audit log, in place of a hash
  - a hidden marker file, so `install.sh` updates its own mod folder and stops on a folder it does not own
  - exact definitions:
    - `uncertain` means the idle time after the key is less than the time since the idle check.
    - `/v1/enroll` returns 403 for Deny or a wrong token, and 410 for expiry or cancel.
    - `GET /v1/terminals/{t}` returns the agent object plus `cols` and `rows`.
  - four results in the PRD's approval flow (five, with "refused").

**Not re-reviewed:** these final fixes were made after the third and last round, so the reviewers did not check them again. The TDD's verification status says so.

## 6. Summary of positions

| Topic                  | Claude planner       | Codex planner                       | Fable reviewer           | Astra reviewer                | Final                                       |
| ---------------------- | -------------------- | ----------------------------------- | ------------------------ | ----------------------------- | ------------------------------------------- |
| Live view              | Stream through a PTY | 4 Hz text snapshots                 | Accepted the stream      | Flow control needed           | Stream through a pipe, with flow control    |
| Claude approve key     | `1`                  | `enter` when Yes is selected        | Accepted                 | Accepted                      | `1`, verified                               |
| Enrollment             | Path middleware      | Mac confirmation, separate listener | Timeouts, overlay button | Per-listener checks           | Separate listener, Mac Allow, 6-digit code  |
| Duplicate actions      | 10-minute cache      | Durable IDs                         | Rebuild from audit       | Enforceable expiry            | Audit log, UUIDv7 IDs, rate limit           |
| Codex transcript       | Folder heuristic     | Explicit ID or manual pick          | Elimination rule: ready  | Not an identity               | Exact IDs only, with `--no-daemon` in setup |
| Race with the Mac user | Not raised           | Named the race                      | Guard is a heuristic     | Guard runs last, fails closed | Idle guard, `uncertain`, accepted residual  |
| Questions              | Digits 1 to 9        | Mac only                            | Not raised               | Not raised                    | Mac only                                    |

## 7. Timing

All times are on 2026-11-08, IST.

| Stage                                                           | Start | End   | Duration     |
| --------------------------------------------------------------- | ----- | ----- | ------------ |
| Brief written and approved                                      | 01:05 | 01:13 | about 8 min  |
| Planning round 1                                                | 01:15 | 01:19 | 4 min 20 s   |
| Orchestrator verification (attach, approval probes)             | 01:19 | 01:26 | about 7 min  |
| Planning round 2                                                | 01:27 | 01:30 | 3 min 10 s   |
| Plan written                                                    | 01:30 | 01:34 | about 4 min  |
| Review round 1                                                  | 01:35 | 01:45 | 9 min 52 s   |
| Orchestrator verification (Codex daemon, edit prompts, tagging) | 01:45 | 01:52 | about 7 min  |
| Review round 2                                                  | 01:53 | 01:57 | 4 min 23 s   |
| Drafts of PRD and TDD                                           | 01:58 | 02:03 | about 5 min  |
| Review round 3                                                  | 02:04 | 02:08 | 4 min 36 s   |
| Final fixes and publication                                     | 02:08 | 02:20 | about 12 min |

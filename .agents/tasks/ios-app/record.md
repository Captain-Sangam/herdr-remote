# Council record: ios-app

## Roster

| Name | Role | Kind | Model | Terminal | Tab |
| --- | --- | --- | --- | --- | --- |
| ios-app-a1 | debater | codex | gpt-6-astra (xhigh) | t88 | tb7fcff196a7e0dd7dc5715f63584d461c |
| ios-app-a2 | debater | claude | fable (Fable 5.2) | t89 | tb23291db20d6d9fdb5ad7e52cbb95eaa8 |
| ios-app-impl | implementer | claude | configured (Opus 5.5, xhigh) | t8b | tbdb18d1e2b52ed31f6260768df4a0f4bf |

Space: spd47055cfff76e4ce1dcf22c70f3d3efa (herdr-remote).

## Invocation

```
/council run usnig codex astra and claude fable

Now i have an androind, want an iOS app with 100% feature parity
```

Interpreted: subcommand `run`; roster (debaters and reviewers) `codex with gpt-6-astra` and `claude with fable`; implementer `claude` with its configured model (no `--implementer`); no `--autofix`, no `--worktree`, no `--repo`, default rounds.

## Paths

- Task folder (D): /Users/zeus/code/herdr-remote/.agents/tasks/ios-app
- Scratch folder (S): /private/var/folders/t4/c_98fx1d1z3_z4nxc5xnf3t40000gn/T/council.hNIzcjXqt5
- Primary repo: /Users/zeus/code/herdr-remote (branch main, HEAD 866f9534d34856528b7ec3b7eb0c98b014c7e168)
- Plan file: /Users/zeus/code/herdr-remote/.agents/tasks/ios-app/plan.md
- Report file: (not yet written)

## Timeline

- 2026-11-15 13:40 IST: run started. Preflight passed (HC_TERMINAL_SOCKET set, hc on PATH, roster valid).
- 2026-11-15 13:40 IST: brief stage started.
- 2026-11-15 13:45 IST: user said mid-brief: no paid Apple Developer account. Brief updated: no APNs, no Local Push Connectivity, background alerts are a named gap.
- 2026-11-15: user did not approve the brief. Edits: test device is an iPad; 100% parity is required; if parity needs a paid account, the plan states the cost. Brief updated.
- 2026-11-15: user asked whether a paid account helps their other projects (website). Answered in chat; membership price added to the brief as a known fact.
- 2026-11-15 13:47 IST: brief approved by the user ("plan approved - proceed"). Brief stage ended.
- 2026-11-15 13:47 IST: plan stage started.
- 2026-11-15 13:48 IST: debate round 1 started (both debaters).

## Debate round 1 (2026-11-15T08:18:43Z to 08:23:40Z, 5 min)

Both answered; tree unchanged (only `.agents/` untracked).

**ios-app-a1 (Codex):** Swift 6, SwiftUI, iOS 17+, `ios/project.yml` with xcodegen, Core/App/Extensions/Tests. Secure Enclave key, self-signed DER cert, keychain identity ("Apple documents this identity workflow", no source given). Network.framework for TLS (unclear for which calls). WKWebView with bundled xterm. APNs for item 9 (paid), NSE fetches metadata over mTLS, prompt never in the notification body, an authenticated expanded UI. Proposes a daemon change: asynchronous action acceptance with result polling, because actions can take 150 s. Avoid App Groups; Keychain Sharing is free. Gaps: iPad haptics, sensitive clipboard flag, APNs best effort. macOS CI. Confidence 80.

**ios-app-a2 (Claude):** same stack, zero packages, `ios/Core` as a local Swift package tested with `swift test`, xcodegen project not committed, three targets (app, NSE, content extension). URLSession with delegate challenges, TLS 1.3 minimum. SE key `AfterFirstUnlockThisDeviceOnly`, shared keychain access group; SPKI DER = fixed 26-byte header + raw point, tested against the shared vectors. Content-free APNs push, NSE fetches agent name, content extension fetches the prompt and adds Approve/Deny only when protected data is readable. Daemon: PUT/DELETE /v1/push per device, Go port of Notifier, ES256 JWT, stdlib HTTP/2, .p8 key 0600. Simulator lacks SE: software key under `#if targetEnvironment(simulator)`. Store in UserDefaults. DataScannerViewController for QR. Gaps: iPad haptics, sensitive copy, alert removal (passive collapse-id replacement), APNs best effort, no ongoing notification. All iOS claims from memory. Confidence 72.

**Agreements:** native Swift 6/SwiftUI, iOS 17 minimum, xcodegen, no third-party packages, SE key + hand-built self-signed cert + keychain identity, WKWebView reusing the Android xterm assets, APNs as the only full-parity path for item 9 (paid), content-free push and mTLS fetch by an extension, prompt never in the notification body, free builds alert only while the app runs, daemon push additive (Android unaffected), 7-day free install vs 1-year paid. Gaps no API closes: iPad haptics; Android's sensitive-clip flag; APNs 5 s is typical not guaranteed.

**Disputes and my verification:**
- Transport: a1 Network.framework, a2 URLSession. URLSession gives HTTP, SSE (`bytes`), chunked streams, client-cert and server-trust challenges; Network.framework would need a hand-written HTTP client. ATS applies to URLSession only, and my search found conflicting reports on whether custom trust for a self-signed cert needs an `NSExceptionDomains` entry on iOS 17+ (forums 731217, 53314, 100113). Draft: URLSession, ATS keys settled by a first spike step. Challenge a1.
- Async action/result polling (a1): the daemon already continues a decision on a detached context and returns the recorded result for a repeated request_id (`docs/architecture.md` 3.5 "More rules"); Android shows "Outcome unknown" on any non-daemon failure (`docs/architecture.md` 4.4, `ui/AgentViewModel.kt:59,105`). Draft: no daemon API change; the content extension handles its own action while viewed, the app path uses a background task. Challenge a1.
- Store: a2 UserDefaults cannot be read by extensions without App Groups; a1 says avoid App Groups. Draft: one keychain item in the shared access group. Challenge a2.
- QR: a2 DataScanner (needs A12+, from memory); iPadOS 17 also runs on A10 iPads. Draft: AVFoundation metadata output. Challenge a2.
- Notification body: Android's collapsed text is `msg` (`watch/WatchService.kt:298-301`), shown on an unlocked phone only; on iOS a body can show on the lock screen under "Show Previews: Always". Draft: body never holds msg or prompt.
- New from my side: the notification category gets no static actions, and the content extension adds Approve/Deny dynamically after a fresh prompt fetch on an unlocked device, so the Apple Watch and the collapsed banner never show buttons (ADR 0019 rejected buttons on a redacted alert). To be verified by both.
- Simulator SE: Apple DTS says the simulator has no SE; one report says `SecureEnclave.isAvailable` is true on an iPad iOS 17 simulator. Draft: compile-time simulator fallback only.

**Verified references:** a2's `DeviceKey.kt:31` is `:29`; `WatchService.kt:331-338` is the result builder, the action builder is `:339-358` (claim holds). `Api.kt:91-95`, `LiveView.kt:36-37`, `AgentScreen.kt:404-406`, `:658-663`, `Gate.kt:12`, `Reducer.kt:79`, `daemon/tls.go:85,98-115`, `daemon/devices.go:12-17`, `docs/api.md:92` all match.

**Decisions so far:** stack, key, URLSession (pending a1), WKWebView reuse with an injected `hr` shim, APNs paid path, content extension with dynamic actions, keychain store, AVFoundation QR, no daemon action API change (pending a1). Draft plan written to `debate/draft.md`.
- 2026-11-15 14:01 IST: debate round 2 started; draft at debate/draft.md.

## Debate round 2 (2026-11-15T08:31:54Z to 08:34:15Z, 2 min)

Both answered; tree unchanged.

**ios-app-a1 (Codex):** no consensus "pending the retry and privacy fixes". Conceded URLSession for HTTP, SSE, live and enroll; keeps Network.framework only for the step 5 handshake. Withdrew the async action API (cites `daemon/actions.go:553`, `docs/architecture.md:344`). Identity source: Apple DTS forum thread 797509 (EC key, including Secure Enclave, plus a certificate gives a keychain identity); self-signing is its inference. Personal Team access groups: confirmed from a DTS experiment. Plan check: `countOfBytesSent` excludes headers, so it cannot prove "never sent" (Android marks `requestHeadersStart`, `net/Api.kt:43`); a single audit line cannot detect a retransmission because a known ID returns without redispatch (`daemon/actions.go:444`), so count HTTP requests. The unlock probe races a lock during the fetch; SR-11 needs evidence under "Always". A custom scheme handler does not intercept built-in schemes: add CSP and resource blocking with tests. `simctl push` does not run a service extension (Xcode 14 notes distinguish real APNs delivery); unit-test the handler, verify through APNs sandbox. Move hardware-identity, sharing and notification-privacy checks earlier. Confidence 88.

**ios-app-a2 (Claude):** CONSENSUS with fixes. Conceded the keychain store and AVFoundation (DataScanner needs A12+). Confirmed `msg` stays out. Dynamic `notificationActions` at run time from iOS 12, not certain with an empty category; the extension receives the tap and `.doNotDismiss` keeps it open; unsure about `simctl push` and service extensions. Plan check: add a device spike after step 3; step 5 contradicts decision 4 (URLSession cannot handshake without a request: use the enroll request's trust challenge, or one NWConnection); the store item needs `AfterFirstUnlockThisDeviceOnly`; the push payload needs the category and the generic title; `PUT /v1/push` must fail when the Mac has no APNs key so local alerts continue; no step adds `willPresent`; fallback to in-view buttons; the probe reads as unlocked for about 10 s after a lock; step 7 and 8 checks miss criterion 5 and 6 items; step 11 should check the content extension over a LAN endpoint. Confidence 78.

**Verified:** `daemon/actions.go:440-448` (a known request ID waits for and returns the recorded result), `:549-556` (the key runs on a background context after the marker), `docs/architecture.md:342-346`, `net/Api.kt:41-46` (`requestHeadersStart`), `assets/xterm/live.html:69-73` (`hr.written`). All match.

**Resolution (decisions taken for the final plan):**
- Transport: URLSession for every call, including enroll. The 6-digit code comes from the server certificate in the enroll request's server-trust challenge, before the body goes out, and shows while the long-poll waits. No Network.framework. Both agents' step 5 objection is resolved without a second transport.
- Action rule: "never sent" = no `requestStartDate` in the task's transaction metrics, plus a connect-phase error. The check counts requests at a fake TLS server (a1's point accepted).
- ATS: the app builds only `https` URLs and refuses every redirect; step 1 records the keys (a1's correction to the step 1 invariant accepted).
- Store item `AfterFirstUnlockThisDeviceOnly` (a2 accepted).
- Lock-screen privacy: the content extension shows the prompt and buttons only when the app's `showPreviewsSetting` is `whenAuthenticated` and the probe item is readable. Under "Always" or "Never" it shows a hint to set Show Previews to "When Unlocked". Device checks lock the iPad during the fetch and after rendering, under each setting (a1 and a2 accepted, gate is the orchestrator's resolution of the 10 s probe window).
- Live view: CSP header from the scheme handler plus a `WKContentRuleList` that blocks all but the bundle scheme; tests try fetch, image and WebSocket loads (a1 accepted).
- Push payload: generic title, category, `mutable-content`, address and kind (a2 accepted). `PUT /v1/push` returns 409 `push_off` without a Mac key, and the app keeps local alerts (a2 accepted). `willPresent` added (a2 accepted). In-view buttons as the fallback for dynamic actions, under the same gate (a2 accepted).
- Service extension: logic in Core with unit tests; end-to-end only with APNs sandbox delivery (paid) (a1 accepted).
- Order: a device spike on the iPad right after pairing (both agents). It needs the user, so the pre-implementation gate asks whether the device is ready now.
- No round 3: the open items are platform behaviours that the spike and the device checks settle, not arguments. The final plan's revisions were not re-checked by the agents; the plan says so.

## Final resolution of the plan stage

- ios-app-a1 conceded URLSession for HTTP, SSE, live and enroll, and withdrew the asynchronous action API. Its handshake-only Network.framework use is replaced by reading the server certificate in the enroll request's trust challenge (decision 5), which also answers ios-app-a2's step 5 objection. It held consensus pending the retry and privacy fixes; both are in the plan (decisions 6 and 11, steps 3 and 12). Not re-checked by the agents.
- ios-app-a2 conceded the keychain store and the AVFoundation scanner, and gave CONSENSUS with fixes. All its plan-check points are in the plan: device spike (step 6), store accessibility (decision 7), payload category and generic title (decision 14), `push_off` (decision 15), `willPresent` (decision 13), in-view buttons fallback (decision 12), stronger transcript and live checks (steps 8 and 9), LAN check (step 12).
- Rejected or deferred: Network.framework (fallback only), App Groups, DataScanner, an action API change, Local Push Connectivity, background keep-alive modes.
- The plan has two user blockers: the membership choice (steps 13 to 15) and the iPad setup (steps 6 and 12).
- 2026-11-15 14:08 IST: plan stage ended. Plan saved at `plan.md`.
- 2026-11-15 14:13 IST: user answered the plan blockers: build steps 13-14 now (step 15 waits for the membership); iPad ready early (planned pause at step 6). Written to plan.md. Implement stage started.
- 2026-11-15 15:22 IST: implementer turn 1 ended (run 08:44:29Z, ended 09:46:11Z, 62 min): steps 1-5 done, blocked at step 6 as planned. Orchestrator verification: all checks pass; one test-isolation fault (shared single-use pairing link). See implement/verification.md.
- 2026-11-16: the user asked to save and checkpoint, and to close the agents that are no longer needed. The run is paused at step 6 (the user's iPad check). The debaters and the implementer were closed; a resume restarts the implementer from the plan and its As built note.

## Paused state (2026-11-16)

- **Where:** implement stage, blocked at step 6 by plan (a planned pause). Steps 1 to 5 are built, committed and verified (`implement/verification.md`).
- **The user must do before resuming:** sign in to Xcode with an Apple ID, connect and trust the iPad, turn on Developer Mode, and check iPadOS 17 or later.
- **Resume:** `/council resume ios-app`. The orchestrator then runs the step 6 checks with the user (the list is in the As built note of `plan.md`), restarts `ios-app-impl` with the plan, and sends steps 7 to 14 and 16. Step 15 waits for the paid membership.
- **Carry into the next implementer turn:** `PairingTests` and `PairUITests` share one single-use pairing link (`TEST_RUNNER_HR_PAIR_LINK`); give each its own link or document it.
- **Scratch folder:** the path in the Paths block may be cleaned by the OS. Recreate it with `mkdir -p` and write `claude-permissions.json` again before a restart (see SKILL.md, Resume).

## Timing

| Stage or round | Start (UTC) | End (UTC) | Duration |
| --- | --- | --- | --- |
| Brief (to approval) | 2026-11-15 08:10 | 08:17 | about 7 min, plus the user's edits |
| Debate round 1 | 08:18:43 | 08:23:40 | 5 min |
| Debate round 2 | 08:31:54 | 08:34:15 | 2 min |
| Plan written, blockers answered | 08:34 | 08:43 | about 9 min |
| Implementer turn 1 (steps 1 to 6) | 08:44:29 | 09:46:11 | 62 min |
| Orchestrator verification | 09:46 | 09:55 | about 9 min |

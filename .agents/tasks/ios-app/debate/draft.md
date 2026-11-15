# Plan: the iOS app (draft for round 2)

## 1. Outcome

- **Task:** add a native iOS app to this repo with full feature parity with the Android app. It runs on iPhone and iPad. The user tests on an iPad.
- **Decision:** a Swift 6 and SwiftUI app for iOS and iPadOS 17 or later, with no third-party packages. It is a second client of the same daemon API and the same pairing flow. A free Apple ID can sign everything except push.
- **What changes:**
  - a new `ios/` folder: an xcodegen spec, a local Swift package of pure logic, the app and a notification content extension
  - for background alerts only: a notification service extension, and an additive push sender in the daemon
  - `scripts/format.sh`, CI, the docs and two new ADRs.
- **Practical consequence:**
  - On a free Apple ID, 12 of the 13 parity items work fully.
  - The thirteenth, background watching (item 9), works only while the app is open or for a short time after the user leaves it.
  - Alerts while the app is closed and after a reboot need Apple's push service (APNs), and APNs needs a paid Apple Developer Program membership.
  - A free install also stops after 7 days. The user must then install it again from Xcode.
- **Cost of full parity:** the membership costs USD 99 a year, in local currency at enrollment. Indian developers report ₹8,700 a year. In India, enrollment is only through the Apple Developer app. The membership unlocks two things for this app:
  - push alerts while the app is closed (item 9)
  - installs that last one year, not 7 days.
- **Gaps that no iOS API closes, free or paid:**
  - The haptic tick in the transcript (iPads have no haptic engine).
  - Android's "sensitive" clipboard flag (iOS has no such flag).
  - A guaranteed 5 s alert (APNs is best effort, and 5 s is typical).
  - The removal of a delivered alert when the agent stops waiting (a quiet replacement alert takes its place).

## 2. Current behaviour

Revision inspected: `herdr-remote` at `866f953` on `main`.

- **Daemon** (`daemon/`):
  - It accepts any P-256 client certificate on both ports. `mutualTLS` checks only `isP256` and then the per-port `verify` (`daemon/tls.go:85-115`).
  - No daemon code is Android-specific.
  - The API is in `docs/api.md`. A decision continues on a detached context after the phone drops. A repeated `request_id` returns the recorded result (`docs/architecture.md` section 3.5, "More rules").
  - The device store is `daemon/devices.go:12-17` (`id`, `name`, `spki_sha256`, `added`).
  - There is no outbound network client and no push. ADR 0013 records that the user ruled out a push service.
- **Android app** (`android/app/src/main/java/dev/herdrremote/`), the parity reference:
  - The key is P-256 in Android Keystore, with `setUserAuthenticationRequired(false)` (`net/DeviceKey.kt:29`).
  - Pin and identity: `PinnedTrust` and `DeviceKeyManager` (`net/Tls.kt`).
  - The link parse, the 6-digit code and the SPKI hash are in `net/Pairing.kt`. `testdata/pairing-vectors.json` holds the shared code vectors.
  - The API client (`net/Api.kt`):
    - the clients and timeouts at `:54-96`
    - endpoint failover with the last good first at `:103-127`
    - one action POST, with no retry after the request starts, at `:195-225`
    - the enroll handshake and long-poll at `:254-326`.
  - The SSE loop has a backoff from 1 s to 30 s (`watch/AgentsRepository.kt:60-90`). The parser and reducer are at `watch/Reducer.kt:30-58`. The alert rules (`Notifier`) are at `watch/Reducer.kt:60-102`.
  - The lists: `ui/ListScreens.kt`. The tab bar with the blocked count: `ui/App.kt`.
  - The agent screen, transcript, prompt card, skill picker and copy are in `ui/AgentScreen.kt`. The texts and actions are in `ui/AgentViewModel.kt`:
    - the result texts at `:51-128`
    - "Outcome unknown" at `:59` and `:105`.
  - The haptic tick is at `ui/AgentScreen.kt:404-406`. The sensitive clip is at `:651-665`.
  - The live view (`ui/LiveView.kt`):
    - 64 KiB chunks and a 1 MiB backlog (`:36-37`)
    - the bridge name `hr` (`:204`)
    - xterm.js and `live.html` in `assets/xterm/`. The page calls `hr.written(n)`.
  - The watch service (`watch/WatchService.kt`):
    - a blocked alert's title is "`<name>` needs approval" or "needs attention", and its collapsed text is `msg` (`:288-314`)
    - the expanded text holds the prompt block, cut to 1500 characters (`:305`)
    - the Approve and Deny buttons carry the turn and the prompt hash (`:339-358`)
    - `decide` shows the result in the alert, and the buttons come back after a refusal while the agent still waits (`:214-248`)
    - a locked phone gets a redacted alert (`:57-84`).
  - The app lock: `Gate.kt`, `MainActivity.kt`. The blank recents card: `MainActivity.kt:30-38`.
  - Settings and Forget: `ui/SettingsScreen.kt`. The endpoint kind is at `:50`.
  - The Pane theme is in `ui/theme/`. The JetBrains Mono fonts are in `res/font/`.
- **Tooling:**
  - `scripts/format.sh` covers md, json, yaml, js, html, css, go, kt and sh. It has no Swift.
  - The CI format job runs on `ubuntu-24.04` (`.github/workflows/ci.yaml`). `format.sh` pins shfmt for `Darwin_arm64` and `Linux_x86_64`.
  - Installed on the Mac: Xcode 27.0, Swift 6.4, iOS 27 simulators (iPhone and iPad) and `xcodegen`. There is no signing identity and no provisioning profile.
- **Manual harness:** `TestManualServe` (`daemon/manual_test.go`) runs a scratch daemon on 17766 and 17767. `HR_MANUAL_AUTOALLOW=1` allows each pending phone (`docs/testing.md`, "Manual harness").

## 3. Decisions

| # | Decision | Why | Strongest alternative, and why it lost |
| --- | --- | --- | --- |
| 1 | Swift 6, SwiftUI, iOS and iPadOS 17 or later, no third-party packages. | It is the smallest native stack, as ADR 0015 chose for Android. iOS 17 gives `scrollPosition` and `ScrollTargetBehavior` for the transcript. | A shared cross-platform codebase (KMP, Flutter): a non-goal in the brief, and "more toolchain" in ADR 0015. |
| 2 | `ios/project.yml` (xcodegen) is committed. The generated `.xcodeproj` is gitignored. Pure logic lives in a local package `ios/Core`, tested with `swift test` on the Mac. | It gives fast tests without a simulator, and no project-file merge noise. `xcodegen` is installed. | A committed `.xcodeproj`: large, noisy diffs, and nothing gained. |
| 3 | The device key is a Secure Enclave P-256 key: `AfterFirstUnlockThisDeviceOnly`, no user presence, in a shared keychain access group. A self-signed X.509 certificate, built in DER by the app and signed by that key, joins it in the keychain to form an identity. Simulator builds only use a software key, under `#if targetEnvironment(simulator)`. | It mirrors `net/DeviceKey.kt:29`: no unlock is needed, so alerts work while locked. The key cannot leave the hardware (SR-4). Apple DTS says the simulator has no Secure Enclave. | A CryptoKit `SecureEnclave.P256` key: it cannot form the `SecIdentity` that TLS needs. |
| 4 | Use URLSession for every HTTP call: the API, SSE, the live stream and enroll. Use an ephemeral configuration and TLS 1.3 minimum. The delegate answers the client-certificate challenge with the identity, and the server-trust challenge with the SPKI pin only. Step 1 finds the ATS keys this needs. | URLSession gives HTTP, SSE through `bytes(for:)`, chunked streams and both TLS challenges. Network.framework would need a hand-written HTTP client. | Network.framework: ATS does not apply to it, but it needs its own HTTP. Use it only if step 1 shows that URLSession cannot pin a self-signed key. |
| 5 | Keep Android's action rule. An action tries the next endpoint only if the request was never sent: a connect-phase error, and `countOfBytesSent == 0`. In every other case it shows "Outcome unknown", and the app never posts it again. | It matches `net/Api.kt:195-225` and `docs/architecture.md` 4.4. The daemon's `request_id` check stops a second key press in any case (SR-6). | A new asynchronous action API with result polling: the daemon already finishes a decision after a disconnect. Android's rule already covers an unknown outcome. |
| 6 | Keep all phone state in one keychain item (a generic password, JSON) in the shared access group: the pin, endpoints, last good, port, device ID, Mac name, the watching toggle and the clock offset. | The extensions must read it. A keychain access group is free for a Personal Team. | `UserDefaults`: extensions cannot read it without App Groups. |
| 7 | Scan the QR code with an `AVCaptureMetadataOutput`, beside a paste field. Add `NSCameraUsageDescription` and `NSLocalNetworkUsageDescription` to Info.plist. | It works on every iPad with a camera. iPadOS 17 also runs on A10 iPads. | `DataScannerViewController`: less code, but it needs an A12 chip or later (from memory). |
| 8 | Show the live view in a `WKWebView` with a non-persistent data store. A `WKURLSchemeHandler` serves the existing `android/app/src/main/assets/xterm/` folder from the bundle. Every other load is cancelled. A user script at document start defines `window.hr.written(n)` over a message handler. | It keeps one copy of xterm.js and `live.html`, with no change to the Android asset. The page still cannot send input (SR-13). | A native terminal renderer: much more code. |
| 9 | Never put the prompt or `msg` in a notification title or body. The title is "`<name>` needs approval", "needs attention" or "finished", and the body is empty. The category has `hiddenPreviewsShowTitle` and no static actions. A content extension shows the prompt and adds Approve and Deny. | iOS has no event for the screen lock, so it cannot redact a delivered alert as `watch/WatchService.kt:57-84` does. The "Show Previews: Always" setting shows the body on the lock screen. With no static actions, the Apple Watch and the collapsed banner never show buttons (ADR 0019 rejected buttons on a redacted alert). | `msg` in the body: under "Always" it shows on the lock screen and breaks SR-11. |
| 10 | When the user expands an alert, the content extension fetches `/v1/terminals/{t}/prompt` over mTLS. It does this only if a probe keychain item (`WhenUnlockedThisDeviceOnly`) is readable, which proves the device is unlocked. It shows `msg` and the block, cut to 1500 characters. It sets Approve and Deny with the turn and hash shown, with `authenticationRequired`. It handles the tap itself and shows the result text. After a refusal while the agent still waits, the buttons come back under the reason. A stale alert shows "No longer waiting" and no buttons. | It gives the parity of `watch/WatchService.kt:192-248` without a background time limit, because the extension lives while the user looks at it. | The app handles the tap in the background: iOS gives it limited time (about 30 s, from memory), and the result has nowhere to show. |
| 11 | Free path for item 9: while the app runs, the SSE stream feeds a Swift port of `Notifier`. The app posts local alerts, one for each agent, with the address as the identifier. A `Clear` removes the agent's delivered alert. When the app goes to the background, a background task keeps the stream for the time iOS gives, then closes it. | It is the closest equivalent that a free account allows. | A background audio or location keep-alive: it misuses those modes and drains the battery, and iOS still ends it. Background app refresh: minutes to hours, not 5 s. |
| 12 | **Paid path for item 9.** The daemon sends APNs alerts:<br>- `PUT /v1/push {token, env}` and `DELETE /v1/push` on 7766, stored as optional fields of the device.<br>- A Go port of `Notifier` for each device with a token.<br>- An ES256 JWT from a `.p8` key (mode 0600), and HTTP/2 from the Go standard library.<br>- A payload with only the address and the kind, `mutable-content` and the address as `apns-collapse-id`.<br>A notification service extension fetches `/v1/terminals/{t}` over mTLS and sets the title with the agent name. On a failure it keeps "An agent needs attention". | APNs is the only path that alerts with the app closed and after a reboot. The payload carries no name and no content (SR-9, SR-11). Android never registers a token, so Android does not change. | Local Push Connectivity: Wi-Fi only (no Tailscale), and it needs an entitlement that Apple grants on request. |
| 13 | With push on, the daemon is the only alert source: the app posts no local alerts. With push off, only the app posts alerts. | One alert for each episode. | Both sources: two alerts for each block. |
| 14 | When an agent stops waiting by another path, the daemon sends a quiet replacement with the same collapse ID ("`<name>` no longer waits", passive level). The `Refresh` rule sends nothing, because the content extension fetches the prompt fresh. | A server cannot remove a delivered alert. The content extension's fresh fetch already makes a stale Approve impossible. | A background push that wakes the app to remove the alert: iOS throttles these pushes. |
| 15 | `.swift` files go into `scripts/format.sh`, with `xcrun swift-format` (default config, `lint --strict` for `--check`). The CI format job moves to a macOS arm64 runner. iOS tests run locally, as the Android tests do. | One format command and one hook. The repo is public, so macOS runner minutes are free. `format.sh` already pins shfmt for `Darwin_arm64`. | A second CI job only for Swift: two jobs to keep in step. |
| 16 | Copy uses `UIPasteboard` with `.localOnly`, so Universal Clipboard does not send the text to other devices. The haptic tick uses `UISelectionFeedbackGenerator`, which does nothing on an iPad. | These are the closest equivalents of `ui/AgentScreen.kt:651-665` and `:404-406`. | An expiry on the copy: Android has none, so it would be a new behaviour. |
| 17 | Hide the app in the app switcher with an opaque cover. The cover goes on at `sceneWillResignActive` and comes off at active. Screenshots stay allowed. | The snapshot comes after resign-active, so the cover is in place in time (ADR 0024 parity). | `scenePhase` in SwiftUI only: it can come too late for the snapshot. |
| 18 | Two new ADRs: 0025 for the iOS app, with decisions 1 to 11 and 13 to 17, and 0026 for push through APNs, which changes ADR 0013. `docs/product.md` puts iOS in scope. | `AGENTS.md`: a new decision gets a new ADR, in the same commit as the code. | One ADR: push is a separate, paid, reversible decision. |

## 4. Scope and acceptance criteria

**Delivered:** `ios/` with the app and the extensions, the daemon push sender (if the user chooses the paid steps), the format and CI change, and the docs.

**Excluded:**
- changes to the Android app, Herdr.app or the mod
- an App Store or TestFlight release
- Mac layouts
- Local Push Connectivity
- features that Android does not have.

**Acceptance criteria:**
1. `swift test --package-path ios/Core` passes. It covers the link parse, the shared pairing vectors, the SPKI DER, UUIDv7, the SSE parser, the reducer, `Notifier`, the texts, the transcript merge and fold, the gate, the action-retry rule and the certificate DER.
2. `xcodebuild test` passes on an iPhone simulator and on an iPad simulator.
3. `cd daemon && go test ./...`, the Android tests and build, and `scripts/format.sh --check` all pass. The Android app does not change.
4. In the simulator, the app pairs with the manual harness by a pasted link and shows the code that the Mac shows. A status change shows within 2 s.
5. The transcript has every Android behaviour: the folds, follow and "Bottom ↓", older pages, a restart after a cursor 409, a reload each 1 s while the agent works, the working row, the model, the scroll bar, the stop at a message start and copy.
6. The live view shows color, follows a resize, recovers after a backlog over 1 MiB and after a reconnect, and zooms with two fingers. It loads nothing outside the bundle.
7. Approve, Deny, a question key, Next and a message give the same results and refusal texts as Android. A dropped connection after the send shows "Outcome unknown" and posts nothing again.
8. The app locks at start and after 5 minutes away. The app switcher shows a blank card.
9. On a free build, alerts come while the app runs. There is one for each episode, a finish alert, and no prompt or `msg` in the title or body. The content extension shows the prompt and the buttons only on an unlocked device.
10. On the iPad with free signing:
    - QR pairing works.
    - The key is in the Secure Enclave.
    - The lock and the blank card work.
    - The content extension works.
    - The layout is usable in both orientations and in Split View.
11. **Paid:** with the app closed, and after a reboot and the first unlock, an alert comes within about 5 s. Approve works from the expanded alert. A Go test shows that the push payload has no name, prompt or message.
12. ADR 0025, ADR 0026 (if paid), and the API, architecture, deployment, testing, operations and product docs describe the result.

**Failure cases to handle:**
- a pin mismatch, which refuses the connection
- enroll 403 or 410
- no reachable endpoint
- every 409 reason
- "Outcome unknown"
- a silent SSE stream (30 s)
- a live backlog over 1 MiB
- a cursor 409
- notifications denied
- a keychain that is not readable before the first unlock (the extensions fall back to the generic alert)
- an APNs 400 or 410 for a token (the daemon drops the token)
- APNs not reachable (no alert, no retry storm).

## 5. Implementation plan

Steps 1 to 11 need no paid membership. Steps 12 to 14 need it for the real push delivery. The code and simulator checks of steps 12 and 13 do not need it.

### Step 1. Skeleton, format, CI and a transport spike
- **Change:**
  - Add `ios/project.yml` (proposed) with the app target `HerdrRemote` (bundle `dev.herdrremote`, iOS 17, iPhone and iPad), `ios/Core/Package.swift` (proposed) and a minimal `ios/App/` (proposed).
  - Add the `xterm` folder reference to `android/app/src/main/assets/xterm` and the two fonts from `android/app/src/main/res/font/`.
  - Add `ios/*.xcodeproj`, `ios/build/` and `ios/Core/.build/` to `.gitignore`.
  - Add `.swift` to `scripts/format.sh`, and move the CI format job to a macOS arm64 runner.
  - Prove in the simulator, with a software key, that URLSession can do all of this against the manual harness: present the identity, pin the SPKI, refuse TLS 1.2 and call `/v1/hello`. Do it once for an IP endpoint and once for a MagicDNS name.
  - Record the ATS keys that this needs in Info.plist.
  - Write ADR 0025 and change `docs/product.md` section 5.
- **Implements:** decisions 1, 2, 4, 15 and 18.
- **Invariant:** a pin mismatch fails toward refusing the connection. No ATS key allows plain HTTP.
- **Verified today:** no `ios/` folder. `format.sh` has no Swift. CI format runs on Ubuntu.
- **Depends on:** nothing.
- **Check:**
  - `cd ios && xcodegen generate && xcodebuild build -scheme HerdrRemote -destination 'platform=iOS Simulator,name=iPhone 17' CODE_SIGNING_ALLOWED=NO` passes.
  - The spike calls `/v1/hello` against `HR_MANUAL_DIR=... go test -run TestManualServe`.
  - A wrong pin fails.
  - `scripts/format.sh --check` passes.
  - If URLSession cannot pin or present the identity, stop and report: decision 4 falls back to Network.framework.

### Step 2. Core logic
- **Change:** add these to `ios/Core` (proposed):
  - the link parse and the 6-digit code
  - the SPKI DER (the fixed 26-byte P-256 header plus the 65-byte point)
  - UUIDv7, the SSE parser and the reducer
  - `Notifier`, with the exact rules of `watch/Reducer.kt:79-97`
  - the texts (`ui/AgentViewModel.kt:51-128`), the transcript merge (`mergeTail`) and the fold
  - the gate (`Gate.kt`), the endpoint kind (`ui/SettingsScreen.kt:50`)
  - the action-retry rule as a pure function of the error and the bytes sent
  - the certificate DER builder, which signs through a closure.
- **Implements:** decisions 2, 5 and 11.
- **Invariant:** the texts and the rules match the Android sources word for word. An unknown error classifies as "sent".
- **Verified today:** the JVM tests `PairingTest`, `ReducerTest`, `TextsTest`, `GateTest`, `SentMarkTest` and `PinTest` define the behaviour.
- **Depends on:** 1.
- **Check:** `swift test --package-path ios/Core`. It reads `testdata/pairing-vectors.json` and has a test for each JVM test case.

### Step 3. Key, identity, store and API client
- **Change:**
  - Add the Secure Enclave key, the self-signed certificate and the identity lookup, with the simulator software key, in the access group `$(AppIdentifierPrefix)dev.herdrremote.shared`.
  - Add the keychain store (decision 6).
  - Add `HerdrApi` on URLSession with the timeouts of `docs/api.md`, "Timeouts on the phone": events 30 s idle, live none, enroll 130 s, actions 150 s, others 15 s, connect 10 s.
  - Add endpoint failover, `hello` with the clock offset and endpoints, the action rule (decision 5), and Forget, which deletes the key, the certificate and the store.
- **Implements:** decisions 3, 4, 5 and 6.
- **Invariant:**
  - The private key is never exported (SR-4).
  - The software key exists only in simulator builds.
  - An action never posts twice.
- **Verified today:** `net/DeviceKey.kt`, `net/Tls.kt`, `net/Api.kt:54-225`, `net/Store.kt`.
- **Depends on:** 2.
- **Check:**
  - An XCTest in the simulator against the manual harness: hello, spaces, a refused connection to the next endpoint, and a forced drop after the send that gives "Outcome unknown" with one request in the harness audit.
  - `grep` shows the software key only inside `#if targetEnvironment(simulator)`.

### Step 4. The Pane design system
- **Change:** port the color tokens (light and dark), the type scale with JetBrains Mono, and the components that the screens use from `ui/theme/` to SwiftUI.
- **Implements:** parity item 13.
- **Invariant:** the tokens match `docs/design.md`.
- **Verified today:** `ui/theme/PaneTheme.kt`, `ui/theme/PaneComponents.kt`.
- **Depends on:** 1.
- **Check:** a preview screen in both color modes, compared by eye with `design/viewer.html`.

### Step 5. Pairing
- **Change:**
  - Add the pair screen: the QR scanner (decision 7), the paste field, and a pinned handshake with 7767 that reads the server SPKI and sends no HTTP request.
  - Then show the code, then enroll with a 130 s long-poll and the Android error texts.
  - The device name is the model and the hardware identifier, 1 to 64 printable characters.
- **Implements:** parity item 1, decision 7.
- **Invariant:** a wrong pin never shows a code and never enrolls. The token goes only to 7767.
- **Verified today:** `net/Api.kt:254-326`, `ui/PairScreen.kt`.
- **Depends on:** 3 and 4.
- **Check:** in the simulator, pair with `HR_MANUAL_AUTOALLOW=1` by a pasted link. The code equals the one in the harness. An expired link and a wrong pin give the Android texts.

### Step 6. Events and lists
- **Change:**
  - Add the SSE loop with a backoff from 1 s to 30 s, and `hello` beside the stream (`watch/AgentsRepository.kt:60-90`).
  - Add the reducer state, Spaces, Space and All agents (blocked first), the tab bar with the blocked count, the stale banner, and the connect log.
  - Fetch `/v1/spaces` on each visit and when the agent set changes (`ui/ListScreens.kt:136`).
- **Implements:** parity items 3 and 4.
- **Invariant:** a silent stream reconnects after 30 s. Old data stays, dimmed, with its age.
- **Verified today:** `watch/AgentsRepository.kt`, `ui/ListScreens.kt`, `ui/App.kt`.
- **Depends on:** 5.
- **Check:** in the simulator against the harness, a probe agent's status change shows within 2 s. A stopped harness shows the banner, and a restart reconnects.

### Step 7. The agent screen
- **Change:**
  - Add the transcript with every behaviour in criterion 5.
  - Add the prompt card: Approve, Deny, the question keys and Next, with the results, the refusals and a resync after `expired_request`.
  - Add the message box (4000 characters) with the skill picker after "/".
  - Add copy (decision 16).
- **Implements:** parity items 5, 7 and 8, decision 16.
- **Invariant:** one tap gives one request. A refused action shows its reason and keeps the buttons.
- **Verified today:** `ui/AgentScreen.kt`, `ui/AgentViewModel.kt`.
- **Depends on:** 6.
- **Check:** in the simulator, with Claude and Codex probe agents (`docs/testing.md`, "Probe agents"), approve, deny, answer a two-question prompt, and send a message. Each shows in `audit.log` once.

### Step 8. The live view
- **Change:** add the WebView of decision 8, the 64 KiB chunks with a wait for `written`, the reopen past 1 MiB, the reopen on a stream end and on an SSE reconnect, and the `#14161E` background.
- **Implements:** parity item 6, decision 8.
- **Invariant:** the page loads only the three bundled files and has no input path.
- **Verified today:** `ui/LiveView.kt`, `assets/xterm/live.html`.
- **Depends on:** 6.
- **Check:** in the simulator, the live view follows a probe agent in color. `yes | head -c 50000000` in a probe terminal recovers. A navigation to an outside URL from the page is cancelled (a unit test of the policy).

### Step 9. Lock, privacy and settings
- **Change:**
  - Add the gate with `LAContext.deviceOwnerAuthentication`, at start and after 5 minutes away. A device with no passcode opens with the Android note. Other errors stay locked.
  - Add the cover of decision 17.
  - Add the settings screen: the Mac, the endpoint kinds, the watching toggle and Forget this Mac.
- **Implements:** parity items 10, 11 and 12, decision 17.
- **Invariant:** nothing shows and no action runs while locked.
- **Verified today:** `Gate.kt`, `MainActivity.kt`, `ui/SettingsScreen.kt`.
- **Depends on:** 7.
- **Check:** in the simulator, the app switcher shows the blank card. A clock jump of 5 minutes in the gate unit test locks.

### Step 10. Free alerts and the content extension
- **Change:**
  - Add local alerts from `Notifier` (decisions 9, 11 and 13) and the notification permission. A denied permission keeps the toggle off, with the reason.
  - Add the target `HerdrRemoteNotificationContent` (proposed), with decision 10.
- **Implements:** parity item 9 (free part), decisions 9 to 11.
- **Invariant:**
  - No prompt or `msg` in a title or body.
  - No buttons without a fresh prompt on an unlocked device.
  - One alert for each episode.
- **Verified today:** `watch/WatchService.kt:192-358`, `watch/Reducer.kt:60-102`.
- **Depends on:** 7.
- **Check:** in the simulator, a probe agent blocks while the app is open, and one alert comes. Expanded, it shows the prompt and the buttons, and Approve gives "Applied". Deny after a Mac keystroke gives the `mac_in_use` text with the buttons back.

### Step 11. The iPad, free signing
- **Change:** no code, unless the checks find a fault.
  - The user signs in to Xcode with an Apple ID and turns on Developer Mode on the iPad.
  - Install with `xcodebuild -allowProvisioningUpdates DEVELOPMENT_TEAM=<team>` and `xcrun devicectl device install app`.
  - Write the 7-day re-install in `docs/deployment.md`.
- **Implements:** acceptance criterion 10.
- **Invariant:** a free team signs the app, the content extension and the keychain access group.
- **Verified today:** no device is connected and there is no signing identity.
- **Depends on:** 10, and the user's steps.
- **Check:**
  - The iPad pairs by QR with the real daemon.
  - `devices ls` lists it.
  - The lock, the blank card, rotation, Split View and the content extension work.
  - If a Personal Team cannot sign the access group for an extension, stop and report.

### Step 12. **Paid:** the daemon push sender
- **Change:**
  - Add `daemon/push.go` (proposed) with the endpoints of decision 12, which store `push_token` and `push_env` as optional fields of `Device`.
  - Add the Go `Notifier` port, fed from the state changes in `daemon/state.go`.
  - Add the APNs client: the JWT is cached under 60 minutes. `apns-push-type: alert`, priority 10, the collapse ID is the address, and the expiry is short.
  - The client drops a token on a 400 `BadDeviceToken` or a 410.
  - Add the admin CLI `herdr-remoted push setup --key <p8> --key-id <id> --team-id <id> --topic dev.herdrremote` and `push status`. The key is copied to the state folder with mode 0600.
  - `devices rm` drops the token.
  - Write ADR 0026 and update `docs/api.md`, `docs/architecture.md`, `docs/operations.md` and `docs/deployment.md`.
- **Implements:** decisions 12 to 14.
- **Invariant:**
  - A device with no token gets no push, so Android is unchanged.
  - A push failure fails toward no alert.
  - No log holds the key, the JWT, the token, a name or content.
  - The daemon connects out only to the APNs hosts.
- **Verified today:** `daemon/devices.go:12-17`, `daemon/state.go:425-460`, `daemon/main.go:15-90`. There is no outbound client today.
- **Depends on:** 2, for the shared rules.
- **Check:** `go test ./...` with a fake APNs server (`httptest`, HTTP/2). It checks the payload fields, the JWT header, a blocked episode with one push, the finish push, the passive clear, a 410 that drops the token, and a revoked device that gets no push.

### Step 13. **Paid:** push in the app and the service extension
- **Change:**
  - Add the target `HerdrRemoteNotificationService` (proposed), with decision 12.
  - The watching toggle registers for remote alerts and calls `PUT /v1/push`. Off calls `DELETE`. Forget calls `DELETE` first.
  - The app sends the token again at each launch.
  - The entitlement `aps-environment` goes into a second entitlements file. A build setting selects that file, so a free team can still sign the default build.
- **Implements:** decisions 12 and 13.
- **Invariant:** the service extension never adds the prompt. A fetch failure or a device before the first unlock leaves the generic title.
- **Verified today:** none of this exists.
- **Depends on:** 10 and 12.
- **Check:** in the simulator, `xcrun simctl push booted dev.herdrremote <payload.json>` shows the alert with the agent name. A broken pin shows the generic title.

### Step 14. **Paid:** push on the device
- **Change:**
  - The user buys the membership and makes an APNs key.
  - Run `herdr-remoted push setup`.
  - Build with the push entitlement and install on the iPad.
- **Implements:** acceptance criterion 11.
- **Invariant:** the same as step 13.
- **Verified today:** not possible without the membership.
- **Depends on:** 11 and 13, and the membership.
- **Check:** with the app closed, a probe block alerts within about 5 s. Do the same after a reboot and the first unlock. Approve from the expanded alert applies. A finish alert comes.

### Step 15. Closing docs
- **Change:**
  - Update `README.md`, `HANDOFF.md` and `AGENTS.md` (the iOS commands, four checks before each commit).
  - Update `docs/testing.md` (the iOS layers and the on-device checks) and `docs/design.md` (iOS notes).
  - Update `docs/product.md` (the platforms, FR-14 and SR-4 on iOS, the gaps).
- **Implements:** decision 18.
- **Invariant:** the docs describe what was built, and the gaps are named.
- **Depends on:** the last step built.
- **Check:** `scripts/format.sh --check`, and a read of each changed doc.

## 6. Testing strategy

| Criterion | Check |
| --- | --- |
| 1 | `swift test --package-path ios/Core` (proposed) |
| 2 | `cd ios && xcodegen generate && xcodebuild test -scheme HerdrRemote -destination 'platform=iOS Simulator,name=iPhone 17' CODE_SIGNING_ALLOWED=NO`, and the same with `name=iPad mini (A17 Pro)` (proposed) |
| 3 | The three commands in `AGENTS.md` (verified) |
| 4 to 9 | The simulator against the manual harness (`docs/testing.md`) with probe agents |
| 10 | The iPad, steps 11 and 14 |
| 11 | `go test ./...` with the fake APNs server (proposed), then the iPad with the membership |
| 12 | A read of the docs, and `scripts/format.sh --check` |

- **Setup:** the manual harness needs the real `hc` and `herdr-app`. The simulator reaches the Mac as `127.0.0.1`. The iPad needs the LAN or Tailscale.
- **Not covered by automatic tests:** the Secure Enclave (no simulator support), the real APNs delivery and the lock screen. These are on-device checks.

## 7. Constraints and open questions

- **Blocker (user, steps 12 to 14):** buy the Apple Developer Program membership (USD 99 a year, about ₹8,700 in India)?
  - Steps 12 and 13 can be built and checked in the simulator without it.
  - Step 14 cannot run without it.
  - The user chooses one: build 12 and 13 now, or build only steps 1 to 11 and 15.
- **Non-blocking (user, step 11):** the iPad must run iPadOS 17 or later. Check in Settings, General, About.
- **Non-blocking (assumption, step 1):** the ATS keys that URLSession needs for a pinned, self-signed key on an IP address and on a MagicDNS name. Step 1 settles it.
- **Non-blocking (assumption, step 10):** iOS gives a backgrounded app about 30 s to finish a task (from memory). The free path does not depend on the exact value.
- **Non-blocking (assumption, step 11):** a Personal Team can sign a keychain access group that the app and an extension share. If not, the content extension cannot reach the key. Then a free build opens the app at the agent from the alert, and the plan records that gap.
- **Non-blocking (assumption, step 13):** `xcrun simctl push` runs the service extension in the simulator.
- **Ready:** steps 1 to 11 and 15. **Blocked on the membership:** step 14. **The user's choice:** steps 12 and 13.

## 8. For the implementer

- Before you start, check that `main` is still at `866f953` or that the files in section 2 did not change.
- Work in step order. End each step with its check and a commit. Update the docs for that step in the same commit.
- Write "As built" notes in a dated section at the end of this file.
- **Stop and report:**
  - if step 1 shows that URLSession cannot pin or present the identity
  - if a free team cannot sign the access group (step 11)
  - if any check needs a change to the Android app or to the security model.
- **Resolve alone:** view layout, file names inside `ios/`, test names and the exact Info.plist text.

## 9. Verification status

- **Inspected:** `herdr-remote` at `866f953`. Every Android and daemon reference in sections 2 and 5 was read in the code.
- **From memory or forum reports, not from Apple documentation:**
  - the ATS rules for custom trust
  - the background task time
  - the `DataScannerViewController` chip requirement
  - the simulator Secure Enclave
  - `simctl push` with a service extension
  - the content extension's dynamic actions and action handling.
- **The membership price:** Apple's enrollment page gives USD 99. The Indian figure comes from developer forum reports.
- **Review:** this draft is the input to the second round.

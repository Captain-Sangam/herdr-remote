# Plan: the iOS app

## 1. Outcome

- **Task:** add a native iOS app to this repo, with full feature parity with the Android app. It runs on iPhone and iPad. The user tests on an iPad.
- **Decision:** a Swift 6 and SwiftUI app for iOS and iPadOS 17 or later, with no third-party packages. It is a second client of the same daemon API and the same pairing flow.
- **What changes:**
  - a new `ios/` folder: an xcodegen spec, a local Swift package of pure logic, the app and two notification extensions
  - an additive push sender in the daemon, for background alerts only
  - `scripts/format.sh`, CI, the docs and two new ADRs.
- **What a free Apple ID gives:**
  - Parity items 1 to 8 and 10 to 13 work fully.
  - Item 9, background watching, works only while the app is open, and for the short time iOS gives after the user leaves it.
  - An install stops after 7 days. The user then installs it again from Xcode.
- **What the paid membership adds:**
  - Alerts while the app is closed and after a reboot, through Apple's push service (APNs).
  - Installs that last one year.
- **Cost:** the Apple Developer Program costs USD 99 a year, in local currency at enrollment.
  - Indian developers report ₹8,700 a year. Check the price on the enrollment screen.
  - In India, enrollment is only through the Apple Developer app.
  - One membership covers every app on every Apple platform.
- **Gaps that no iOS API closes, free or paid:**
  - The haptic tick in the transcript: iPads have no haptic engine. It works on an iPhone.
  - Android's "sensitive" clipboard flag: iOS has no such flag.
  - A guaranteed 5 s alert: APNs is best effort. About 5 s is typical.
  - The removal of a delivered alert when an agent stops waiting: a quiet replacement alert takes its place.
  - The prompt in an expanded alert needs the per-app setting Show Previews = "When Unlocked" (the iOS default). Under "Always", the alert shows a hint and no prompt.

## 2. Current behaviour

Revision inspected: `herdr-remote` at `866f953` on `main`.

### Daemon (`daemon/`)

- It accepts any P-256 client certificate on both ports. `mutualTLS` checks `isP256` and then the per-port `verify` (`daemon/tls.go:85-115`).
- No daemon code is Android-specific. The API is in `docs/api.md`.
- A decision continues on a background context after the phone drops (`daemon/actions.go:549-556`).
- A repeated `request_id` waits for and returns the recorded result, with no second key (`daemon/actions.go:440-448`).
- The device store is `daemon/devices.go:12-17`: `id`, `name`, `spki_sha256`, `added`.
- The SSE subscribers are in `daemon/state.go:425-460`. The CLI is `daemon/main.go:15-90`.
- The daemon has no outbound network client and no push. ADR 0013 records that the user ruled out a push service.

### Android app (`android/app/src/main/java/dev/herdrremote/`)

This is the parity reference.

- **Key and TLS:**
  - The key is P-256 in Android Keystore, with `setUserAuthenticationRequired(false)` (`net/DeviceKey.kt:29`).
  - `PinnedTrust` and `DeviceKeyManager` are in `net/Tls.kt`.
- **Pairing:**
  - The link parse, the 6-digit code and the SPKI hash are in `net/Pairing.kt`.
  - `testdata/pairing-vectors.json` holds the shared code vectors.
  - The enroll handshake and long-poll are at `net/Api.kt:254-326`.
- **API client (`net/Api.kt`):**
  - the clients and timeouts at `:54-96`
  - endpoint failover with the last good first at `:103-127`
  - one action POST at `:195-225`. The next endpoint is tried only before `requestHeadersStart` (`:41-46`).
- **Events:**
  - The SSE loop has a backoff from 1 s to 30 s (`watch/AgentsRepository.kt:60-90`).
  - The parser and reducer are at `watch/Reducer.kt:30-58`.
  - The alert rules (`Notifier`) are at `watch/Reducer.kt:60-102`.
- **Screens:**
  - The lists are in `ui/ListScreens.kt`. The tab bar with the blocked count is in `ui/App.kt`.
  - The agent screen, transcript, prompt card, skill picker and copy are in `ui/AgentScreen.kt`.
  - The haptic tick is at `ui/AgentScreen.kt:404-406`. The sensitive clip is at `:651-665`.
  - The texts and actions are in `ui/AgentViewModel.kt`: the result texts at `:51-128`, and "Outcome unknown" at `:59` and `:105`.
  - Settings and Forget are in `ui/SettingsScreen.kt`. The endpoint kind is at `:50`.
- **Live view (`ui/LiveView.kt`):**
  - 64 KiB chunks and a 1 MiB backlog limit (`:36-37`)
  - the bridge name `hr` (`:204`). The page calls `hr.written(n)` (`assets/xterm/live.html:69-73`).
- **Watch service (`watch/WatchService.kt`):**
  - A blocked alert's title is "`<name>` needs approval" or "needs attention". Its collapsed text is `msg` (`:288-314`).
  - The expanded text holds the prompt block, cut to 1500 characters (`:305`).
  - Approve and Deny carry the turn and the prompt hash (`:339-358`).
  - `decide` shows the result in the alert. After a refusal while the agent still waits, the buttons come back (`:214-248`).
  - A locked phone gets a redacted alert (`:57-84`).
- **Lock and privacy:** `Gate.kt` and `MainActivity.kt`. The blank recents card is at `MainActivity.kt:30-38`.
- **Design:** the Pane theme is in `ui/theme/`. The JetBrains Mono fonts are in `res/font/`.

### Tooling

- `scripts/format.sh` covers md, json, yaml, js, html, css, go, kt and sh. It has no Swift. It pins shfmt for `Darwin_arm64` and `Linux_x86_64`.
- The CI format job runs on `ubuntu-24.04` (`.github/workflows/ci.yaml`).
- The Mac has these installed:
  - Xcode 27.0 and Swift 6.4
  - iOS 27 simulators for iPhone and iPad
  - `xcodegen`.

  It has no signing identity and no provisioning profile.
- The manual harness is `TestManualServe` (`daemon/manual_test.go`). It runs a scratch daemon on 17766 and 17767. `HR_MANUAL_AUTOALLOW=1` allows each pending phone. `HR_MANUAL_PAGE` sets the transcript page size (`docs/testing.md`, "Manual harness").

## 3. Decisions

| # | Decision | Why | Strongest alternative, and why it lost |
| --- | --- | --- | --- |
| 1 | Swift 6 and SwiftUI. iOS and iPadOS 17 or later. No third-party packages. | It is the smallest native stack, as in ADR 0015. iOS 17 has `scrollPosition` and `ScrollTargetBehavior` for the transcript. | A shared cross-platform codebase: a non-goal, and "more toolchain" in ADR 0015. |
| 2 | Commit `ios/project.yml` (xcodegen). Ignore the generated `.xcodeproj`. Put pure logic in the local package `ios/Core`, tested with `swift test` on the Mac. | Fast tests with no simulator. No project-file merge noise. | A committed `.xcodeproj`: large, noisy diffs. |
| 3 | The device key is a Secure Enclave P-256 key, `AfterFirstUnlockThisDeviceOnly`, with no user presence, in a shared keychain access group. The app builds a self-signed certificate in DER and signs it with the key. The key and the certificate form a keychain identity. Simulator builds only use a software key, under `#if targetEnvironment(simulator)`. | It mirrors `net/DeviceKey.kt:29`, so alerts work while locked. The key cannot leave the hardware (SR-4). Apple DTS (forum thread 797509) describes an EC key plus a certificate as a keychain identity, with the Secure Enclave. The simulator has no Secure Enclave. | A CryptoKit `SecureEnclave.P256` key: it cannot form a `SecIdentity`. |
| 4 | URLSession for every call: the API, SSE, the live stream and enroll. Ephemeral configuration, TLS 1.3 minimum. The delegate answers the client challenge with the identity, and the server challenge with the SPKI pin only. The app builds only `https` URLs and refuses every redirect. | URLSession gives HTTP, SSE through `bytes(for:)`, streams and both TLS challenges. | Network.framework: it needs a hand-written HTTP client. It is the fallback only if step 1 fails. |
| 5 | Enroll is one URLSession request. The app reads the server certificate in the server-trust challenge of that request, checks the pin, and shows the 6-digit code. The long-poll then waits for Allow. | Android needs a separate raw-socket handshake (`net/Api.kt:261-285`) only because OkHttp does not expose the handshake. One request shows the code before Allow too. | A separate handshake with `NWConnection`: a second transport for one step. |
| 6 | An action tries the next endpoint only if the request never started. That is: a connect-phase error, and no `requestStartDate` in the task's transaction metrics. In every other case the app shows "Outcome unknown" and never posts again. | It matches `net/Api.kt:195-225` and `docs/architecture.md` 4.4. `countOfBytesSent` excludes headers, so it cannot prove "never sent". The daemon's `request_id` check also stops a second key (SR-6). | A new asynchronous action API: the daemon already finishes a decision after a drop (`daemon/actions.go:549-556`). |
| 7 | All phone state is one keychain item (a generic password, JSON), `AfterFirstUnlockThisDeviceOnly`, in the shared access group. It holds the pin, endpoints, last good, port, device ID, Mac name, the watching toggle and the clock offset. | The extensions must read it, also on a locked device. A Personal Team can sign an access group. | `UserDefaults`: extensions cannot read it without App Groups. |
| 8 | Scan the QR code with `AVCaptureMetadataOutput`, beside a paste field. | It works on every iPad with a camera. iPadOS 17 also runs on A10 iPads. | `DataScannerViewController`: it needs an A12 or later. |
| 9 | The live view is a `WKWebView` with a non-persistent data store. A `WKURLSchemeHandler` serves the existing `android/app/src/main/assets/xterm/` folder from the bundle, with a Content-Security-Policy header. A `WKContentRuleList` blocks every load outside that scheme. A user script defines `window.hr.written(n)` over a message handler. | One copy of xterm.js and `live.html`, with no change to the Android asset. A scheme handler alone does not stop loads over built-in schemes. The page still cannot send input (SR-13). | A native terminal renderer: much more code. |
| 10 | No prompt and no `msg` in a notification title or body. The title is "`<name>` needs approval", "needs attention" or "finished". The body is empty. The category has `hiddenPreviewsShowTitle` and no static actions. | iOS has no screen-lock event, so it cannot redact a delivered alert as `watch/WatchService.kt:57-84` does. Under "Always", a body shows on the lock screen. With no static actions, the Apple Watch and the collapsed banner never show buttons (ADR 0019). | `msg` in the body: it breaks SR-11 under "Always". |
| 11 | A notification content extension shows the prompt and the buttons when the user expands an alert. It does this only if the app's `showPreviewsSetting` is `whenAuthenticated` and a probe keychain item (`WhenUnlockedThisDeviceOnly`) is readable. Then it fetches `/v1/terminals/{t}/prompt` over mTLS. It shows `msg` and the block, cut to 1500 characters. It sets Approve and Deny, with `authenticationRequired`, for the turn and hash shown. It handles the tap and shows the result. After a refusal while the agent waits, the buttons come back under the reason. A stale alert shows "No longer waiting", with no buttons. Under "Always" or "Never", it shows a hint to set Show Previews to "When Unlocked". | It gives the parity of `watch/WatchService.kt:192-248`. The extension lives while the user looks at it, so no background limit applies. Under "When Unlocked", iOS itself asks for an unlock before it expands an alert on the lock screen. The probe alone reads as unlocked for about 10 s after a lock. | The app handles the tap in the background: limited time, and no place for the result. |
| 12 | If dynamic actions do not show for a category with no static actions, the extension shows its own buttons (`UNNotificationExtensionUserInteractionEnabled`), under the same gate. | A fallback inside the same design. | Static actions: they show on the Watch and on the redacted banner. |
| 13 | Free path for item 9: while the app runs, the SSE stream feeds a Swift port of `Notifier`. The app posts local alerts, one for each agent, with the address as the identifier. `Clear` removes the agent's alert. A `willPresent` handler shows alerts while the app is open. In the background, a background task keeps the stream for the time iOS gives, then closes it. | The closest equivalent on a free account. | A background audio or location keep-alive: it misuses those modes, drains the battery, and iOS still ends it. Background app refresh: minutes to hours. |
| 14 | **Paid path for item 9.** The daemon sends APNs alerts. See the list under this table. | APNs is the only path that alerts with the app closed and after a reboot. The payload has no name and no content (SR-9, SR-11). Android never registers a token, so Android does not change. | Local Push Connectivity: Wi-Fi only (no Tailscale), and an entitlement that Apple grants on request. |
| 15 | With push on, the daemon is the only alert source, and the app posts no local alerts. The app uses push only after `PUT /v1/push` succeeds. If the Mac has no APNs key, `PUT` returns 409 `push_off`, and the app keeps its local alerts. | One alert for each episode, and never zero. | Both sources at once: two alerts for each block. |
| 16 | An agent can stop waiting by another path. Then the daemon sends a quiet replacement with the same collapse ID: "`<name>` no longer waits", at the passive level. `Refresh` sends nothing, because the content extension fetches the prompt fresh. | A server cannot remove a delivered alert. The fresh fetch already makes a stale Approve impossible. | A background push that wakes the app: iOS throttles these. |
| 17 | Format `.swift` files in `scripts/format.sh` with `xcrun swift-format` (default config, `lint --strict` for `--check`). Move the CI format job to a macOS arm64 runner. iOS tests run locally, like the Android tests. | One format command and one hook. The repo is public, so macOS minutes are free. `format.sh` already pins shfmt for `Darwin_arm64`. | A second CI job only for Swift: two jobs to keep in step. |
| 18 | Copy uses `UIPasteboard` with `.localOnly`. The haptic tick uses `UISelectionFeedbackGenerator`, which does nothing on an iPad. | The closest equivalents of `ui/AgentScreen.kt:651-665` and `:404-406`. | An expiry on the copy: a new behaviour that Android does not have. |
| 19 | An opaque cover goes over the app at `sceneWillResignActive` and comes off at active. Screenshots stay allowed. | The app switcher snapshot comes after resign-active (ADR 0024 parity). | SwiftUI `scenePhase` only: it can come too late. |
| 20 | Write two new ADRs. ADR 0025 covers the iOS app (decisions 1 to 13 and 15 to 19). ADR 0026 covers push through APNs and changes ADR 0013. `docs/product.md` puts iOS in scope. | `AGENTS.md`: a new decision gets an ADR, in the same commit as the code. | One ADR: push is a separate, paid, reversible decision. |

**Decision 14, the paid push path:**

- `PUT /v1/push {token, env}` and `DELETE /v1/push` on 7766. The daemon stores them as optional fields of the device.
- A Go port of `Notifier` runs for each device with a token.
- The daemon signs an ES256 JWT with a `.p8` key (mode 0600) and sends over HTTP/2 from the Go standard library.
- The payload has:
  - the generic title "An agent needs attention" or "An agent finished"
  - the category and `mutable-content`
  - the address and the kind.
- The address is the `apns-collapse-id`.
- A notification service extension fetches `/v1/terminals/{t}` over mTLS and puts the agent name in the title. On a failure, the generic title stays.

## 4. Scope and acceptance criteria

**Delivered:**

- `ios/`, with the app and the extensions
- the daemon push sender, if the user chooses the paid steps
- the format and CI change
- the docs.

**Excluded:**

- changes to the Android app, Herdr.app or the mod
- an App Store or TestFlight release
- Mac layouts
- Local Push Connectivity
- features that Android does not have.

**Acceptance criteria:**

1. `swift test --package-path ios/Core` passes. It covers:
   - the link parse, the shared pairing vectors and the SPKI DER
   - UUIDv7, the SSE parser, the reducer and `Notifier`
   - the texts, the transcript merge and fold, and the gate
   - the action-retry rule, the certificate DER and the push payload parse.
2. `xcodebuild test` passes on an iPhone simulator and on an iPad simulator.
3. These all pass, and the Android app does not change:
   - `cd daemon && go test ./...`
   - the Android tests and build
   - `scripts/format.sh --check`.
4. In the simulator, the app pairs with the manual harness by a pasted link. It shows the code that the harness shows. A status change shows within 2 s.
5. The transcript has every Android behaviour:
   - the folds, follow and "Bottom ↓"
   - older pages, and a restart after a cursor 409
   - a reload each 1 s while the agent works, the working row and the model
   - the scroll bar, the stop at a message start, and copy.
6. The live view:
   - shows color and zooms with two fingers
   - follows a resize, and recovers after a backlog over 1 MiB and after a reconnect
   - loads nothing outside the bundle.
7. Approve, Deny, a question key, Next and a message give the Android results and refusal texts. A drop after the send shows "Outcome unknown", and the server sees one request.
8. The app locks at start and after 5 minutes away. The app switcher shows a blank card.
9. On a free build, alerts come while the app runs:
   - one alert for each episode, and a finish alert
   - no prompt or `msg` in the title or body
   - the prompt and the buttons only under the gate of decision 11.
10. On the iPad with free signing:
    - QR pairing works, and the key is in the Secure Enclave.
    - The lock and the blank card work.
    - The content extension works over a LAN endpoint.
    - A lock during the prompt fetch, or after it shows, leaves no prompt on the lock screen, under each Show Previews setting.
    - The layout works in both orientations and in Split View.
11. **Paid:**
    - With the app closed, and after a reboot and the first unlock, an alert comes in about 5 s.
    - Approve works from the expanded alert.
    - A Go test shows that the payload has no name, prompt or message.
12. ADR 0025, ADR 0026 (if paid), and the API, architecture, deployment, testing, operations and product docs describe the result.

**Failure cases to handle (no others):**

- a pin mismatch, which refuses the connection
- a redirect, which the app refuses
- enroll 403 or 410
- no reachable endpoint
- every 409 reason, and "Outcome unknown"
- a silent SSE stream (30 s)
- a live backlog over 1 MiB, and a cursor 409
- notifications denied
- a keychain that is not readable before the first unlock: the extensions keep the generic alert
- `push_off` from the daemon: the app keeps local alerts
- an APNs 400 `BadDeviceToken` or 410: the daemon drops the token
- APNs not reachable: no alert and no retry storm.

## 5. Implementation plan

- Steps 1 to 5, 7 to 11 and 16 need no membership.
- Steps 6 and 12 need the user at the iPad.
- Steps 13 to 15 are paid. Steps 13 and 14 can be built and checked without the membership. Step 15 cannot.

### Step 1. Skeleton, format, CI and a transport spike

- **Change:**
  - Add `ios/project.yml` (proposed): the app target `HerdrRemote`, bundle `dev.herdrremote`, iOS 17, iPhone and iPad.
  - Add `ios/Core/Package.swift` (proposed) and a minimal `ios/App/` (proposed).
  - Add a folder reference to `android/app/src/main/assets/xterm`, and the two fonts from `android/app/src/main/res/font/`.
  - Add `ios/*.xcodeproj`, `ios/build/` and `ios/Core/.build/` to `.gitignore`.
  - Add `.swift` to `scripts/format.sh`. Move the CI format job to a macOS arm64 runner.
  - Spike in the simulator with a software key against the manual harness. Present the identity, pin the SPKI, refuse TLS 1.2 and refuse a redirect. Then call `/v1/hello`. Do it for an IP address and for a MagicDNS name.
  - Record the ATS keys that the spike needs in `Info.plist`, with the reason in ADR 0025.
  - Write ADR 0025, and change `docs/product.md` section 5.
- **Implements:** decisions 1, 2, 4, 17 and 20.
- **Invariant:** a pin mismatch or a redirect fails toward refusing. No code builds an `http` URL.
- **Verified today:** there is no `ios/` folder. `format.sh` has no Swift. The CI format job runs on Ubuntu.
- **Depends on:** nothing.
- **Check:**
  - `cd ios && xcodegen generate && xcodebuild build -scheme HerdrRemote -destination 'platform=iOS Simulator,name=iPhone 17' CODE_SIGNING_ALLOWED=NO`
  - The spike gets `/v1/hello` from `HR_MANUAL_DIR=<dir> go test -run TestManualServe -count=1 -timeout 3h .` (in `daemon/`). A wrong pin fails. A redirect fails.
  - `scripts/format.sh --check`
  - If URLSession cannot pin or present the identity, stop and report. Decision 4 then falls back to Network.framework.

### Step 2. Core logic

- **Change:** add to `ios/Core` (proposed):
  - the link parse and the 6-digit code
  - the SPKI DER: the fixed 26-byte P-256 header plus the 65-byte point
  - UUIDv7, the SSE parser and the reducer
  - `Notifier`, with the rules of `watch/Reducer.kt:79-97`
  - the texts (`ui/AgentViewModel.kt:51-128`), the transcript merge (`mergeTail`) and the fold
  - the gate (`Gate.kt`) and the endpoint kind (`ui/SettingsScreen.kt:50`)
  - the action-retry rule, as a pure function of the error and "request started"
  - the certificate DER builder, which signs through a closure
  - the push payload parse and the title text, for the service extension.
- **Implements:** decisions 3, 6, 13 and 14.
- **Invariant:** the texts and rules match the Android sources word for word. An unknown error counts as "started".
- **Verified today:** the JVM tests `PairingTest`, `ReducerTest`, `TextsTest`, `GateTest`, `SentMarkTest` and `PinTest` define the behaviour.
- **Depends on:** 1.
- **Check:** `swift test --package-path ios/Core`. It reads `testdata/pairing-vectors.json` and has a case for each JVM test case.

### Step 3. Key, identity, store and API client

- **Change:**
  - Add the Secure Enclave key, the certificate and the identity lookup, with the simulator software key. Use the access group `$(AppIdentifierPrefix)dev.herdrremote.shared`.
  - Add the keychain store (decision 7).
  - Add `HerdrApi` on URLSession with the timeouts of `docs/api.md`, "Timeouts on the phone":
    - events: 30 s idle
    - live: none
    - enroll: 130 s
    - actions: 150 s
    - others: 15 s
    - connect: 10 s.
  - Add endpoint failover, `hello` with the clock offset and the endpoints, and the action rule (decision 6).
  - Add Forget, which deletes the key, the certificate and the store.
- **Implements:** decisions 3, 4, 6 and 7.
- **Invariant:** the private key is never exported (SR-4). The software key exists only in simulator builds. An action never posts twice.
- **Verified today:** `net/DeviceKey.kt`, `net/Tls.kt`, `net/Api.kt:54-225`, `net/Store.kt`.
- **Depends on:** 2.
- **Check:**
  - An XCTest in the simulator with a fake TLS server (an `NWListener` with a test identity). The test covers four cases:
    - a refused first endpoint moves to the second
    - a server that reads the request and closes gives "Outcome unknown"
    - the server counts exactly one request across all endpoints
    - a redirect is refused.
  - An XCTest against the manual harness: `hello` and `spaces`.
  - `grep` shows the software key only inside `#if targetEnvironment(simulator)`.

### Step 4. The Pane design system

- **Change:** port the color tokens (light and dark), the type scale with JetBrains Mono, and the components that the screens use from `ui/theme/`.
- **Implements:** parity item 13.
- **Invariant:** the tokens match `docs/design.md`.
- **Verified today:** `ui/theme/PaneTheme.kt`, `ui/theme/PaneComponents.kt`.
- **Depends on:** 1.
- **Check:** a preview screen in both color modes, compared by eye with `design/viewer.html`.

### Step 5. Pairing

- **Change:**
  - Add the pair screen: the QR scanner (decision 8) and the paste field.
  - Add enroll as one request (decision 5). Show the code from the trust challenge, then wait up to 130 s. Use the Android error texts.
  - The device name is the model and the hardware identifier, 1 to 64 printable characters.
  - Add `NSCameraUsageDescription` and `NSLocalNetworkUsageDescription`.
- **Implements:** parity item 1, decisions 5 and 8.
- **Invariant:** a wrong pin never shows a code and never sends the token. The token goes only to the enroll port.
- **Verified today:** `net/Api.kt:254-326`, `ui/PairScreen.kt`.
- **Depends on:** 3 and 4.
- **Check:** in the simulator, pair with the harness and `HR_MANUAL_AUTOALLOW=1` by a pasted link. The code equals the harness's code. An expired link and a wrong pin give the Android texts.

### Step 6. Spike on the iPad (needs the user)

- **Change:**
  - Add the content extension target `HerdrRemoteNotificationContent` (proposed), with only a read of the store item.
  - Sign with the user's Personal Team: `xcodebuild -allowProvisioningUpdates DEVELOPMENT_TEAM=<team>`. Install with `xcrun devicectl device install app`.
- **Implements:** an early test of decisions 3, 4 and 7 on real hardware.
- **Invariant:** the same as step 3.
- **Verified today:** no device is connected, and there is no signing identity.
- **Depends on:** 5. The user must first:
  - sign in to Xcode with an Apple ID
  - connect the iPad and turn on Developer Mode
  - click Allow on the Mac.
- **Check:**
  - The iPad pairs by a pasted link with the real daemon. `herdr-remoted devices ls` lists it.
  - `/v1/hello` works over the Secure Enclave identity.
  - A local test alert's content extension reads the store item.
- **Stop:** stop and report in two cases. The key cannot form an identity that URLSession presents. Or a Personal Team cannot sign the shared access group.
- **If the user's device is not ready:** skip this step, note it under As built, and continue. Its checks then run first in step 12.

### Step 7. Events and lists

- **Change:**
  - Add the SSE loop with a backoff from 1 s to 30 s, and `hello` beside the stream (`watch/AgentsRepository.kt:60-90`).
  - Add the reducer state, and the Spaces, Space and All agents screens (blocked first).
  - Add the tab bar with the blocked count, the stale banner and the connect log.
  - Fetch `/v1/spaces` on each visit and when the agent set changes (`ui/ListScreens.kt:136`).
- **Implements:** parity items 3 and 4.
- **Invariant:** a silent stream reconnects after 30 s. Old data stays, dimmed, with its age.
- **Verified today:** `watch/AgentsRepository.kt`, `ui/ListScreens.kt`, `ui/App.kt`.
- **Depends on:** 5.
- **Check:** in the simulator against the harness:
  - A probe agent's status change shows within 2 s.
  - A stopped harness shows the banner.
  - A restarted harness reconnects.

### Step 8. The agent screen

- **Change:**
  - Add the transcript with every behaviour in criterion 5.
  - Add the prompt card: Approve, Deny, the question keys and Next. Show the results and the refusals. Resync the clock after `expired_request`.
  - Add the message box (4000 characters), with the skill picker after "/".
  - Add copy (decision 18).
- **Implements:** parity items 5, 7 and 8, decision 18.
- **Invariant:** one tap gives one request. A refused action shows its reason and keeps the buttons.
- **Verified today:** `ui/AgentScreen.kt`, `ui/AgentViewModel.kt`.
- **Depends on:** 7.
- **Check:** in the simulator, with the probe agents of `docs/testing.md`:
  - Approve, deny, answer a two-question prompt and send a message. Each shows once in the harness `audit.log`.
  - With `HR_MANUAL_PAGE=4096`, scroll back across three pages.
  - Open and close a fold.
  - Scroll up, see "Bottom ↓" and tap it.
  - Make the cursor stale (a new transcript file), and see the transcript restart from the end.
  - Make a fling stop at a message start (a UI test).

### Step 9. The live view

- **Change:** add the web view of decision 9:
  - 64 KiB chunks, each after a wait for `written`
  - a reopen past 1 MiB, on a stream end and on an SSE reconnect
  - the `#14161E` background.
- **Implements:** parity item 6, decision 9.
- **Invariant:** the page loads only the three bundled files and has no input path.
- **Verified today:** `ui/LiveView.kt`, `assets/xterm/live.html`.
- **Depends on:** 7.
- **Check:**
  - In the simulator, the view follows a probe agent in color.
  - The view redraws after a pane resize on the Mac.
  - The view recovers after `yes | head -c 50000000` in a probe terminal.
  - A pinch zooms.
  - A UI test injects script that tries `fetch`, an image and a WebSocket to an outside host. All three are blocked.

### Step 10. Lock, privacy and settings

- **Change:**
  - Add the gate with `LAContext.deviceOwnerAuthentication`, at start and after 5 minutes away. A device with no passcode opens with the Android note. Other errors stay locked.
  - Add the cover of decision 19.
  - Add the settings screen with these items:
    - the Mac and the endpoint kinds
    - the watching toggle
    - the Show Previews hint
    - Forget this Mac.
- **Implements:** parity items 10, 11 and 12, decision 19.
- **Invariant:** nothing shows and no action runs while locked.
- **Verified today:** `Gate.kt`, `MainActivity.kt`, `ui/SettingsScreen.kt`.
- **Depends on:** 8.
- **Check:** in the simulator, the app switcher shows a blank card. The gate unit test locks after 5 minutes.

### Step 11. Free alerts and the content extension

- **Change:**
  - Add the local alerts of decision 13, with the notification permission and `willPresent`. A denied permission keeps the toggle off, with the reason.
  - Complete the content extension with decisions 10, 11 and 12.
- **Implements:** parity item 9 (the free part), decisions 10 to 13.
- **Invariant:**
  - There is no prompt or `msg` in a title or body.
  - There are no buttons without a fresh prompt under the gate.
  - Each episode gives one alert.
- **Verified today:** `watch/WatchService.kt:192-358`, `watch/Reducer.kt:60-102`.
- **Depends on:** 8. Also 6 if it ran.
- **Check:** in the simulator, while the app is open:
  - A probe block gives one alert.
  - The expanded alert shows the prompt and the buttons. Approve gives "Applied".
  - Deny right after a Mac keystroke gives the `mac_in_use` text, with the buttons back.
  - An alert for an agent that no longer waits shows "No longer waiting".

### Step 12. The iPad, free signing (needs the user)

- **Change:** no code, unless a check finds a fault. Write the install steps and the 7-day re-install in `docs/deployment.md`.
- **Implements:** acceptance criterion 10.
- **Invariant:** a free team signs the app, the content extension and the access group.
- **Depends on:** 11, and the user's steps of step 6.
- **Check:**
  - Run the checks of step 6 first, if step 6 was skipped.
  - Pair by QR with the real daemon.
  - Check the lock, the blank card, rotation and Split View.
  - Check the content extension over a LAN endpoint.
  - Under each Show Previews setting, lock the iPad during the prompt fetch, and after the prompt shows. No prompt shows on the lock screen.

### Step 13. **Paid:** the daemon push sender

- **Change:**
  - Add `daemon/push.go` (proposed) with the endpoints of decision 14. Store `push_token` and `push_env` as optional fields of `Device`.
  - `PUT /v1/push` returns 409 `push_off` if the Mac has no APNs key.
  - Add the Go `Notifier` port, fed from the state changes in `daemon/state.go`.
  - Add the APNs client:
    - a JWT cached under 60 minutes
    - `apns-push-type: alert`, priority 10, a short expiry
    - the address as the collapse ID
    - a drop of the token on a 400 `BadDeviceToken` or a 410.
  - Add `herdr-remoted push setup --key <p8> --key-id <id> --team-id <id> --topic dev.herdrremote` and `push status`. The setup copies the key to the state folder with mode 0600.
  - Make `devices rm` drop the token.
  - Write ADR 0026. Update `docs/api.md`, `docs/architecture.md`, `docs/operations.md` and `docs/deployment.md`.
- **Implements:** decisions 14 to 16.
- **Invariant:**
  - A device with no token gets no push, so Android does not change.
  - A push failure fails toward no alert.
  - No log holds the key, the JWT, the token, a name or content.
  - The daemon connects out only to the APNs hosts.
- **Verified today:** `daemon/devices.go:12-17`, `daemon/state.go:425-460`, `daemon/main.go:15-90`. There is no outbound client.
- **Depends on:** 2, for the shared rules.
- **Check:** `go test ./...` with a fake APNs server (`httptest`, HTTP/2). It checks:
  - the payload fields, and that the payload has no name, prompt or `msg`
  - the JWT header
  - one push for each blocked episode, the finish push and the passive clear
  - a 410 that drops the token
  - a revoked device that gets no push
  - `push_off` with no key.

### Step 14. **Paid:** push in the app and the service extension

- **Change:**
  - Add the target `HerdrRemoteNotificationService` (proposed), with the fetch of decision 14 and the Core parse of step 2.
  - The watching toggle registers for remote alerts and calls `PUT /v1/push`. Off calls `DELETE`. Forget calls `DELETE` first. The app sends the token again at each launch.
  - Put the entitlement `aps-environment` in a second entitlements file. A build setting selects it, so a free team can still sign the default build.
- **Implements:** decisions 14 and 15.
- **Invariant:** the service extension never adds the prompt. A failed fetch, or a device before the first unlock, leaves the generic title.
- **Verified today:** none of this exists.
- **Depends on:** 11 and 13.
- **Check:**
  - The Core unit tests of the payload parse and the titles.
  - `xcodebuild build` with and without the push entitlement file.
  - `xcrun simctl push` checks the content extension only. It does not run a service extension. The service extension is checked in step 15.

### Step 15. **Paid:** push on the device

- **Change:**
  - The user buys the membership and makes an APNs key.
  - The user runs `herdr-remoted push setup`.
  - Build with the push entitlement and install on the iPad.
- **Implements:** acceptance criterion 11.
- **Invariant:** the same as step 14.
- **Depends on:** 12 and 14, and the membership.
- **Check:**
  - With the app closed, a probe block alerts in about 5 s, with the agent name from the service extension.
  - The same works after a reboot and the first unlock.
  - Approve from the expanded alert applies.
  - A finish alert comes.

### Step 16. Closing docs

- **Change:** update these files:
  - `README.md` and `HANDOFF.md`
  - `AGENTS.md`: the iOS commands, and four checks before each commit
  - `docs/testing.md`: the iOS layers and the on-device checks
  - `docs/design.md`: the iOS notes
  - `docs/product.md`: the platforms, FR-14 and SR-4 on iOS, and the gaps.
- **Implements:** decision 20.
- **Invariant:** the docs describe what was built, and they name the gaps.
- **Depends on:** the last step built.
- **Check:** `scripts/format.sh --check`, and a read of each changed doc.

## 6. Testing strategy

| Criterion | Check |
| --- | --- |
| 1 | `swift test --package-path ios/Core` (proposed) |
| 2 | `cd ios && xcodegen generate && xcodebuild test -scheme HerdrRemote -destination 'platform=iOS Simulator,name=iPhone 17' CODE_SIGNING_ALLOWED=NO`. Run it again with `name=iPad mini (A17 Pro)` (proposed). |
| 3 | The three commands in `AGENTS.md` (verified) |
| 4 to 9 | The simulator against the manual harness, with probe agents (`docs/testing.md`) |
| 10 | The iPad, steps 6 and 12 |
| 11 | `go test ./...` with the fake APNs server (proposed), then the iPad in step 15 |
| 12 | A read of the docs, and `scripts/format.sh --check` |

- **Setup:**
  - The manual harness needs the real `hc` and `herdr-app`.
  - The simulator reaches the Mac as `127.0.0.1`.
  - The iPad needs the LAN or Tailscale.
- **Not covered by automatic tests:**
  - the Secure Enclave (the simulator has none)
  - real APNs delivery and the service extension
  - the lock screen.

  These are checks on the iPad.

## 7. Constraints and open questions

- **Blocker (the user, steps 13 to 15): buy the Apple Developer Program membership?** It costs USD 99 a year, about ₹8,700 in India.
  - Steps 13 and 14 can be built and checked without it.
  - Step 15 cannot run without it.
  - Choose one:
    - build steps 13 and 14 now
    - build only the free steps (1 to 12 and 16).
- **Blocker (the user, steps 6 and 12): get the iPad ready?**
  1. Sign in to Xcode with an Apple ID (Xcode, Settings, Accounts).
  2. Connect the iPad and trust this Mac.
  3. Turn on Developer Mode on the iPad (Settings, Privacy and Security).

  Without these steps, step 6 is skipped and step 12 waits.
- **Non-blocking (the user, step 6):** the iPad must run iPadOS 17 or later (Settings, General, About).
- **Non-blocking (assumption, step 1):** the ATS keys that a pinned, self-signed key needs on an IP address and on a MagicDNS name. Step 1 settles this.
- **Non-blocking (assumption, step 11):** a backgrounded app gets about 30 s to finish a task (from memory). The free path does not depend on the exact value.
- **Non-blocking (assumption, step 11):** dynamic actions show for a category with no static actions (from memory). Decision 12 is the fallback.
- **Non-blocking (assumption, step 6):** a Personal Team can sign an access group that the app and an extension share. One Apple DTS experiment supports this. If it fails, the content extension cannot reach the key. Then a free build opens the app at the agent from the alert, and the plan records the gap.
- **Ready:** steps 1 to 5, 7 to 11 and 16.
- **Blocked:**
  - steps 6 and 12 until the user's device is ready
  - step 15 until the membership
  - steps 13 and 14 until the user's choice.

## 8. For the implementer

- Before you start, check that `main` is at `866f953`, or that the files in section 2 did not change.
- Work in step order. End each step with its check and a commit. Update the docs of that step in the same commit.
- Write "As built" notes in a dated section at the end of this file.
- **Stop and report in these cases:**
  - Step 1 or step 6 shows that URLSession cannot pin or present the identity.
  - A free team cannot sign the access group (step 6 or 12).
  - A check needs a change to the Android app or to the security model.
- **Resolve alone:** view layout, file names inside `ios/`, test names and the exact `Info.plist` text.
- **Do not:** pair a device yourself (`AGENTS.md`: `pair allow` needs the user), print `server.key`, or log content.

## 9. Verification status

- **Inspected:** `herdr-remote` at `866f953`. Every Android and daemon reference in sections 2 and 5 was read in the code.
- **Sources for platform facts:**
  - The membership price: Apple's enrollment page gives USD 99. The Indian figure comes from developer forum reports.
  - The identity from a key and a certificate: Apple DTS forum thread 797509.
  - The simulator has no Secure Enclave: an Apple DTS forum answer.
  - `simctl push` does not run a service extension: one agent's reading of the Xcode 14 release notes.
  - From memory: the ATS rules, the background task time, dynamic actions with an empty category, and the `DataScannerViewController` chip requirement.
- **Review:** two agents checked the draft. This plan adds the fixes from their check. The agents did not re-check these fixes:
  - enroll as one request
  - the request-start rule
  - the Show Previews gate
  - the content rule list
  - the push payload fields
  - `push_off`
  - the device spike.

## Decisions before implementation (2026-11-15)

- **Membership (steps 13 to 15):** the user chose to build steps 13 and 14 now. Step 15 waits for the membership. The implementer builds steps 13 and 14 and skips step 15, and lists step 15 as left under As built.
- **iPad setup (steps 6 and 12):** the user will have the iPad ready early. At step 6 the implementer builds and installs the spike on the iPad if a device is connected and signing works. Then it stops and reports `blocked at step 6`, with the exact checks for the user. This is a planned pause: the orchestrator runs the checks with the user and then sends the implementer the rest of the plan. If no device is connected or signing fails at step 6, the implementer writes what it found under As built and stops at step 6 in the same way.
- **Bundle ID:** if a free team cannot register `dev.herdrremote` (taken by another team), the implementer adds a build setting for the bundle ID prefix, uses a unique one on the device build only, and records it under As built.

## As built (2026-11-15)

Implementer: ios-app-impl. Base `866f953`. The files in section 2 matched the code at the start.

### Steps

| Step | Status  | Commit    | Check and result                                                                                                                                                                                                       |
| ---- | ------- | --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1    | Done    | `61522c2` | Simulator build passes. `SpikeTests` (5) pass against the harness: hello over an IP address and a DNS name, a wrong pin fails, a 301 is not followed, TLS 1.2 fails. `scripts/format.sh --check` passes.               |
| 2    | Done    | `558a549` | `swift test --package-path ios/Core`: 26 tests pass (28 after step 5). Go parses the Swift certificate DER, and its signature checks.                                                                                  |
| 3    | Done    | `9fb42ec` | `ApiTests` (7) pass: a refused endpoint moves on, a close after the request gives "Outcome unknown" with one request, a redirect is refused, a silent endpoint moves on after 10 s, hello and spaces on the harness. |
| 4    | Done    | `553231c` | The preview screen (`-panePreview`) in dark and light mode, checked by eye from simulator screenshots. Two layout faults fixed: a key label wrapped, and the tmux chips ran into the home-indicator area.             |
| 5    | Done    | `4b100f8` | `PairingTests` and `PairUITests` pass against the harness. The app showed "119, 639" and the harness showed 119639. The 410, expired-link and wrong-pin texts are the Android ones.                                    |
| 6    | Blocked | `46bb6dc` | The content extension stub builds and is embedded. No device is connected, and this Mac has no signing identity and no Xcode account. See "Step 6 checks for the user".                                               |

- Before each commit: the daemon tests, the Android tests and build, `scripts/format.sh --check`, and from step 2 `swift test --package-path ios/Core`. All passed.

### Spike result (step 1)

- URLSession presents a keychain identity (a key and a certificate in the keychain) and pins the SPKI. Decision 4 holds. No fallback to Network.framework.
- ATS does not apply to IP address literals or to `localhost`.
- ATS refuses a DNS name with a self-signed key, also when the delegate accepts the trust (`-9802`, "ATS failed system trust"). An exception domain with `NSExceptionAllowsInsecureHTTPLoads` fixes it. `Info.plist` has `ts.net` with `NSIncludesSubdomains`. ADR 0025 gives the reason.
- Tailscale was stopped on this Mac, so no MagicDNS name was tested. The stand-in was `localtest.me` (it resolves to 127.0.0.1) under a temporary exception of the same form. Hello passed, and a wrong pin failed over the name too.

### Assumptions

- Until the app can pair, the simulator tests use the harness phone key (`client.key` and `client.crt`) as the test identity. The test process reads them. Nothing printed them.
- The simulator tests need a signed build: `CODE_SIGNING_ALLOWED=NO` gives `errSecMissingEntitlement` (`-34018`). The test command drops it and adds `-collect-test-diagnostics never`. The build check keeps `CODE_SIGNING_ALLOWED=NO`.
- URLSession has no connect timeout. A per-task probe cancels a task that has sent and received nothing after 10 s. Plain GETs use only the 15 s read timeout.
- Each action has its own URLSession, so no stale connection can make URLSession send it again.
- The iOS part of `docs/architecture.md` is a new section 9, so no anchors change.
- `PromptField` is not in step 4. Only the agent screen (step 8) uses it.
- The Core alert enum is `AlertKind`, because SwiftUI has `Alert`.
- A push title says "needs approval" when the agent's `kind` is `permission`, else "needs attention".
- Until step 7, a paired app shows a stand-in screen: the endpoints, "Check the connection" (`/v1/hello`), "Send a test alert in 5 s" and "Forget this Mac".
- Debug builds take `-panePreview` (the component preview) and `-reset` (forget at launch, for UI tests).
- The pairing tests set the link's endpoints to `127.0.0.1`, as `PairingEmulatorTest` does with `10.0.2.2`.
- The code check ran on a harness without auto-allow. `herdr-remoted pair cancel` closed the window. Nobody ran `pair allow`.
- `HR_BUNDLE_ID` (default `dev.herdrremote`) sets the bundle ID now, and the extension adds `.content`. Registration was not tried. Change it only if a free team cannot register `dev.herdrremote`.

### Deviations

- Criterion 2 names `CODE_SIGNING_ALLOWED=NO` for `xcodebuild test`. The keychain tests cannot pass with it, so the test command uses the default ad-hoc signing (`docs/testing.md`, "iOS tests").
- The MagicDNS check of step 1 did not run (Tailscale stopped). It moves to the iPad checks.

### Notes

- The CI format job runs on `macos-26` with Xcode 26.6. This Mac has Xcode 27. Their swift-format versions can format differently. If CI fails on a Swift file that passes here, select Xcode 27 on the runner.
- The scratch harness state (in the session scratchpad) holds test devices. It is not the real daemon.

### Step 6 checks for the user

Do these on the Mac and the iPad, in order.

1. Sign in to Xcode with an Apple ID (Xcode, Settings, Accounts). Note the Personal Team ID.
2. Connect the iPad, trust this Mac, and turn on Developer Mode (Settings, Privacy and Security). The iPad restarts.
3. Run `xcrun devicectl list devices`. The iPad must show with a Reality that is not `simulated`. Note its identifier.
4. Build: `cd ios && xcodegen generate && xcodebuild build -scheme HerdrRemote -destination 'generic/platform=iOS' -derivedDataPath build -allowProvisioningUpdates DEVELOPMENT_TEAM=<team>`.
   - If Xcode cannot register `dev.herdrremote`, add `HR_BUNDLE_ID=dev.herdrremote.<something unique>`.
   - If signing fails on the keychain group `<team>.dev.herdrremote.shared`, stop. A free team cannot sign the shared access group (section 7).
5. Install: `xcrun devicectl device install app --device <identifier> build/Build/Products/Debug-iphoneos/HerdrRemote.app`.
6. On the iPad, trust the developer (Settings, General, VPN and Device Management).
7. Pair: on the Mac, click **Pair a phone**. On the iPad, scan the QR code or paste the link. If the codes match, click **Allow** on the Mac. Then `herdr-remoted devices ls` lists the iPad.
8. On the iPad, tap **Check the connection**. Expect "Connected to herdr-remoted `<version>`." This is `/v1/hello` over the Secure Enclave identity. If it fails with a TLS error, stop: the key cannot form an identity that URLSession presents.
9. Tap **Send a test alert in 5 s**, allow notifications, and leave the app. Expand the alert. Expect "Store item read. Paired: yes. Mac: `<endpoint>`." If it says "Store item not readable." on an unlocked iPad, stop: the extension cannot read the shared access group.
10. If Tailscale is on, make the MagicDNS name the only reachable endpoint and tap **Check the connection** again. Expect the same text. This is the ATS check of step 1.

### Left

- Step 6: the checks above, by the user.
- Steps 7 to 14 and 16, in a later turn. Step 15 waits for the membership.
- ADR 0026 (step 13).

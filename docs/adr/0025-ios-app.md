# ADR 0025: The iOS app

- **Status:** Accepted, 2026-11-15.
- **Decision:** 25. It is not in the decision table of the technical design. The plan is `.agents/tasks/ios-app/plan.md`, decisions 1 to 13 and 15 to 19.

## Context

- The user asked for a native iOS app with the same features as the Android app. It must run on iPhone and iPad. The user tests on an iPad.
- The daemon has no Android-specific code. Any client with a P-256 key can pair and call the API ([API](../api.md)).
- iOS has no foreground service and no screen-lock event. A free Apple ID gives installs that stop after 7 days, and no push.

## Decision

### Stack and layout

- Swift 6 and SwiftUI, for iOS and iPadOS 17 or later. No third-party packages.
- `ios/project.yml` is an xcodegen spec. The generated `.xcodeproj` is not in git.
- `ios/Core` is a local Swift package with the pure logic. `swift test --package-path ios/Core` runs its tests on the Mac.
- `ios/Shared` holds the code that the app and its extensions share: TLS, the keychain and the store.
- The app bundles `android/app/src/main/assets/xterm/` and the two JetBrains Mono fonts from the Android tree. There is one copy of each.

### Key and TLS

- The device key is a Secure Enclave P-256 key, `AfterFirstUnlockThisDeviceOnly`, with no user presence, in a shared keychain access group. Simulator builds use a software key, only under `#if targetEnvironment(simulator)`.
- The app builds a self-signed certificate for the key in DER. The key and the certificate form a keychain identity.
- Every call uses URLSession with an ephemeral configuration and TLS 1.3 as the minimum.
  - The client certificate challenge gets the identity.
  - The server trust challenge passes only if the leaf key's SPKI SHA-256 equals the pin. Names and addresses play no part.
  - The delegate refuses every redirect. The app treats the 3xx reply as an error.
  - The app builds only `https` URLs.
- Enroll is one request. The app reads the server key in the trust challenge of that request, checks the pin and shows the 6-digit code. Then the long-poll waits for Allow.
- An action tries the next endpoint only if the request never started: a connect-phase error, and no `requestStartDate` in the task metrics. In every other case the app shows "Outcome unknown" and never posts again.
- All phone state is one keychain item (a generic password that holds JSON), `AfterFirstUnlockThisDeviceOnly`, in the shared access group. The extensions read it.

### App Transport Security

- ATS does not apply to IP address literals or to `localhost`. A pinned connection to an IP address needs no ATS key.
- ATS applies its own system trust check to a DNS name, also when the delegate accepts the server trust. The connection then fails with `-9802` ("ATS failed system trust").
- So `Info.plist` has one exception domain, `ts.net`, with `NSIncludesSubdomains` and `NSExceptionAllowsInsecureHTTPLoads`. The daemon lists no DNS names except the MagicDNS name, which ends in `.ts.net`.
  - Apple documents `NSExceptionAllowsInsecureHTTPLoads` as the key that lets an app loosen server trust for a domain.
  - The key also allows plain HTTP to that domain. The app builds no `http` URL, so it never uses that.
  - The pin check still runs. The spike shows that a wrong pin fails over the name too.

### Screens, alerts and privacy

- The QR scanner is `AVCaptureMetadataOutput`, beside a paste field.
- The live view is a `WKWebView` with a non-persistent data store. A `WKURLSchemeHandler` serves the bundled xterm files, and a `WKContentRuleList` blocks every other load.
- No prompt and no `msg` go in a notification title or body. The title is "`<name>` needs approval", "needs attention" or "finished".
- A notification content extension shows the prompt and Approve or Deny when the user expands an alert. It does this only under Show Previews "When Unlocked", and only while a probe keychain item (`WhenUnlockedThisDeviceOnly`) is readable.
- Without push, the app posts local alerts from the SSE stream while it runs, and for the short time iOS gives after the user leaves it.
- With push on, the daemon is the only alert source. ADR 0026 will record push.
- Copy uses `UIPasteboard` with `.localOnly`. The haptic tick uses `UISelectionFeedbackGenerator`.
- An opaque cover goes over the app at `sceneWillResignActive`. Screenshots stay allowed ([ADR 0024](0024-allow-screenshots.md)).

### Format and CI

- `scripts/format.sh` formats `.swift` files with `xcrun swift-format` and the default configuration. `--check` runs `lint --strict`.
- The CI format job runs on `macos-26` (arm64), because swift-format comes with Xcode.
- The iOS tests run locally, like the Android tests.

## Alternatives

| Alternative                     | Why it lost                                                                  |
| ------------------------------- | ---------------------------------------------------------------------------- |
| A shared cross-platform app     | A non-goal, and more toolchain ([ADR 0015](0015-android-stack.md)).          |
| A committed `.xcodeproj`        | Large, noisy diffs.                                                          |
| CryptoKit `SecureEnclave.P256`  | It cannot form a `SecIdentity`.                                              |
| Network.framework for HTTP      | It needs a hand-written HTTP client. URLSession passed the spike.            |
| `NSAllowsArbitraryLoads`        | It turns ATS off for every host. One exception domain is enough.             |
| `DataScannerViewController`     | It needs an A12 chip or later. iPadOS 17 also runs on A10 iPads.             |
| `msg` in the notification body  | Under Show Previews "Always", the body shows on the lock screen (SR-11).     |
| A background keep-alive (audio) | It misuses the mode, drains the battery, and iOS still ends it.              |
| SwiftUI `scenePhase` only       | It can come after the app switcher snapshot, so the cover can come too late. |

## Consequences

- The iOS app has the parity items of the Android app, with these gaps:
  - An iPad has no haptic engine.
  - iOS has no "sensitive" clipboard flag.
  - Without push, alerts come only while the app runs or just after.
  - A free install stops after 7 days. The user then installs it again from Xcode.
- Keychain tests in the simulator need a signed build. `CODE_SIGNING_ALLOWED=NO` gives `errSecMissingEntitlement` (`-34018`), so the tests use the default ad-hoc signing.
- Local Xcode and the CI runner can have different swift-format versions. A version change can change the format.

## Sources

- The plan: `.agents/tasks/ios-app/plan.md`.
- The step 1 spike: `ios/Tests/SpikeTests.swift`, against the manual harness ([testing](../testing.md#manual-harness)).
- Apple, "NSExceptionAllowsInsecureHTTPLoads" and "Performing manual server trust authentication".

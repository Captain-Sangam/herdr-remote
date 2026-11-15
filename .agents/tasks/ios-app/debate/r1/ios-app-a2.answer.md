⏺ ROUND 1 ios-app-a2

  Position.
  Swift 6 and SwiftUI, iOS/iPadOS 17 minimum, zero third-party packages. ios/project.yml (xcodegen, project not committed) has three targets: the app, a notification service extension (NSE) and a
  notification content extension. They share ios/Core, a local Swift package of pure logic.
  - mTLS: a Secure Enclave P-256 key (AfterFirstUnlockThisDeviceOnly, no user presence, shared keychain access group), a hand-built DER self-signed certificate signed by that key, a SecIdentity from
    the keychain, and URLSession delegate challenges with TLS 1.3 minimum.
  - Live view: WKWebView loads the existing android/app/src/main/assets/xterm/ by folder reference, with a non-persistent data store.
  - Alerts: the daemon sends a content-free APNs push; the NSE fetches the agent name over mTLS; the content extension fetches the prompt and adds Approve and Deny only on an unlocked device.

  Reasoning.
  - No daemon change for the key: it accepts any P-256 certificate (daemon/tls.go:85, :98-115). The pin and the pairing code hash SPKI DER (net/Pairing.kt:56-70), so iOS prepends the fixed 26-byte
    P-256 SPKI header to the raw point and tests it against testdata/pairing-vectors.json.
  - Key policy matches Android: setUserAuthenticationRequired(false) (net/DeviceKey.kt:31) so alerts work while locked. SR-4 holds because the key cannot be exported.
  - Push is the only parity path: Android holds /v1/events in a foreground service (watch/WatchService.kt:104, :128), which iOS does not have. Local Push Connectivity works only on named Wi-Fi
    networks and needs an Apple-granted entitlement (from memory), so it loses Tailscale.
  - Redaction: Android reposts alerts redacted at screen-off (WatchService.kt:62-84), which iOS cannot do. So the banner never holds the prompt. It shows only in the content extension, which first
    checks that protected data is readable. This holds SR-11 under any Show Previews setting.
  - Buttons: they carry the turn and prompt_hash of the prompt shown (WatchService.kt:331-338), with .authenticationRequired.

  ┌─────┬────────────────────────────────────────────────────────────────────────────────────────────┬────────────────────────────────────────────────────────┐
  │  #  │                                       iOS mechanism                                        │                       Membership                       │
  ├─────┼────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
  │ 1   │ DataScannerViewController plus paste; camera permission                                    │ free                                                   │
  ├─────┼────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
  │ 2   │ As above; UserDefaults for net/Store.kt:10; failover as net/Api.kt:106-119                 │ free                                                   │
  ├─────┼────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
  │ 3   │ URLSession.bytes, hand-parsed SSE, 30 s idle timeout (Api.kt:95); port of watch/Reducer.kt │ free                                                   │
  ├─────┼────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
  │ 4   │ SwiftUI lists, TabView badge                                                               │ free                                                   │
  ├─────┼────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
  │ 5   │ ScrollView with scrollPosition and a custom ScrollTargetBehavior                           │ free                                                   │
  ├─────┼────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
  │ 6   │ callAsyncJavaScript awaits the write callback; 64 KiB and 1 MiB as ui/LiveView.kt:36-37    │ free                                                   │
  ├─────┼────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
  │ 7   │ One POST per tap, 150 s, no retry (Api.kt:91-92); UUIDv7 port                              │ free                                                   │
  ├─────┼────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
  │ 8   │ TextField, 4000 limit, picker                                                              │ free                                                   │
  ├─────┼────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
  │ 9   │ While open: SSE plus local notifications. Closed or after reboot: APNs                     │ paid: the aps-environment entitlement and the APNs key │
  ├─────┼────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
  │ 10  │ LAContext.deviceOwnerAuthentication, 5 min (Gate.kt:12)                                    │ free                                                   │
  ├─────┼────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
  │ 11  │ Opaque cover when scenePhase is not active; ephemeral session, no URLCache                 │ free                                                   │
  ├─────┼────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
  │ 12  │ SwiftUI; the toggle registers or removes the push token                                    │ free                                                   │
  ├─────┼────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
  │ 13  │ Pane tokens ported to SwiftUI                                                              │ free                                                   │
  └─────┴────────────────────────────────────────────────────────────────────────────────────────────┴────────────────────────────────────────────────────────┘

  - All items: a free install expires after 7 days; paid gives 1 year.
  - Gaps that no API closes:
    - Haptic tick (ui/AgentScreen.kt:406): no Taptic Engine on an iPad; works on an iPhone.
    - Sensitive copy (AgentScreen.kt:661): no such flag; closest is .localOnly plus an expiry.
    - Removing an alert when an agent leaves blocked: closest is a quiet replacement push with the same apns-collapse-id. A stale Approve is still refused by the hash guard.
    - APNs delivery is best effort, so 5 s is typical, not guaranteed.
    - No ongoing "watching" notification.

  Outline.
  1. ios/Core: Pairing, Uuid7, Reducer and Notifier, SSE parser, DER certificate, Texts. swift test covers the shared vectors.
  2. Key, identity and Api; then pairing; lists; transcript; live view; actions; lock; settings.
  3. Local alerts and the content extension (free).
  4. Paid:
     - Daemon push.go, additive: PUT and DELETE /v1/push {token} stored per device (daemon/devices.go:12), a Go port of Notifier, an ES256 JWT, and HTTP/2 to APNs from the stdlib.
     - New Mac secret: the .p8 key, mode 0600, never logged.
     - Payload: generic text, the terminal address and the kind only.
     - The NSE.
  5. A new ADR (iOS stack, and push changing ADR 0013), docs/product.md sections 5 and 8, the api, architecture, deployment and testing docs, swift format in scripts/format.sh, and a macOS CI job.

  Android never registers a token, so its behaviour is unchanged.

  Invariants:
  - Pin mismatch fails toward refusing.
  - An NSE fetch failure, or a device before first unlock, fails toward the generic alert.
  - A locked device or a failed prompt fetch fails toward no prompt and no buttons.
  - An action timeout fails toward outcome_unknown and never a second key press. A fake-server test asserts one request.
  - No push token, or APNs down, fails toward no alert.

  Verify:
  - swift test --package-path ios/Core
  - cd ios && xcodegen && xcodebuild test -scheme HerdrRemote -destination 'platform=iOS Simulator,name=iPad mini (A17 Pro)'
  - Simulator: pair by pasted link against the real daemon, then xcrun simctl push booted dev.herdrremote payload.json.
  - iPad: xcodebuild -allowProvisioningUpdates, then xcrun devicectl device install app. Check QR pairing, the Secure Enclave key, the lock, the recents card, Approve from an alert, and a closed-app
    alert after a reboot (paid).
  - go test ./... for push.go with a fake APNs server.

  Assumptions.
  - All iOS claims here are from memory; I opened no Apple documentation.
  - A keychain identity forms from a Secure Enclave key plus an added certificate, and extensions can use it through the access group.
  - The simulator may lack Secure Enclave keys, so it needs a software-key fallback under #if targetEnvironment(simulator).
  - A Personal Team can sign keychain sharing and three App IDs.
  - A delegate trust override satisfies ATS for IP addresses and MagicDNS names.
  - Extensions inherit the app's local-network permission (TN3179).
  - URLSession does not replay a POST.
  - The iPad runs iPadOS 17 or later.

  Confidence. 72

  END ROUND 1

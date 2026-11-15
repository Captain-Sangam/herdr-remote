# Brief: ios-app

## Name

`ios-app`

## Task

Build a native iOS app that gives the same features as the Android app (`android/`, package `dev.herdrremote`) against the same daemon (`herdr-remoted`) and the same pairing flow. It runs on iPhone and iPad; the user tests on an iPad. The user requires 100% feature parity. Done means: an iOS app in this repo that pairs with the Mac by QR code, and does everything in the parity list below, with the security requirements SR-1 to SR-13 of `docs/product.md` held on iOS. Where parity needs a paid Apple Developer Program membership, the plan designs that path in full, marks each step that needs the membership, and states its price. The user then decides whether to buy it. Builds and tests run from the command line on this Mac. The docs and a new ADR describe the iOS app.

## Repos

- **herdr-remote** (primary): `/Users/zeus/code/herdr-remote`, branch `main`, HEAD `866f953`. Clean tree.
  - `daemon/`: the Go daemon. It accepts any P-256 client certificate on both ports (`daemon/tls.go:98-115`, `isP256` at `:85`). No daemon code is Android-specific. The API is `docs/api.md`.
  - `android/app/src/main/java/dev/herdrremote/`: the parity reference, about 5,800 lines. The parity list:
    1. Pairing: QR scan or a pasted link, a pinned handshake on 7767 to show the 6-digit code, the enroll long-poll (`net/Pairing.kt`, `net/Api.kt` `Enroll`, `ui/PairScreen.kt`).
    2. The P-256 hardware key and the mTLS client: TLS 1.3 only, SPKI pin, no name checks (`net/DeviceKey.kt`, `net/Tls.kt`). Endpoint failover with the last good first; `/v1/hello` refreshes endpoints and the clock offset (`net/Api.kt`, `net/Store.kt`).
    3. The SSE stream, 30 s read timeout, the reducer, the stale banner (`watch/AgentsRepository.kt`, `watch/Reducer.kt`).
    4. Spaces, Space, All agents (blocked first), the tab bar with the blocked count (`ui/ListScreens.kt`, `ui/App.kt`).
    5. The transcript: folds, follow and "Bottom ↓", older pages, fling to a message start with a haptic tick, scroll bar, reload each 1 s while working, the working row, the model, long-press copy marked sensitive (`ui/AgentScreen.kt`, `ui/AgentViewModel.kt`).
    6. The live view: xterm.js in a web view that loads only bundled files and sends no input, 64 KiB chunks with write-callback flow control, a reopen past 1 MiB backlog, pinch zoom (`ui/LiveView.kt`, `assets/xterm/`).
    7. Approve, Deny, question keys and Next, the result and refusal texts, no retry after send, UUIDv7 on the Mac clock (`ui/AgentViewModel.kt`, `net/Uuid7.kt`).
    8. Messages up to 4000 characters, with the skill picker after "/" (`ui/AgentScreen.kt`).
    9. Background watching: blocked alerts with the prompt and Approve and Deny, finish alerts, one alert per agent, redaction on a locked phone, start after a reboot (`watch/WatchService.kt`, `watch/StartReceiver.kt`, ADR 0013, ADR 0019).
    10. The app lock at start and after 5 minutes away (`Gate.kt`, `MainActivity.kt`).
    11. Privacy: blank recents card, screenshots allowed, content in memory only (`MainActivity.kt`, ADR 0024).
    12. Settings: the paired Mac, endpoint kinds, the watching toggle, Forget this Mac (`ui/SettingsScreen.kt`).
    13. The Pane design system (`ui/theme/`, `docs/design.md`, `design/`).
  - `mod/herdr-remote/`: the pairing overlay. Platform-neutral.
  - `testdata/pairing-vectors.json`: the 6-digit code vectors that the Go and JVM tests share.
  - `scripts/format.sh`, `.githooks/pre-commit`, `.github/workflows/ci.yaml`: no Swift today.

## Binding rules

- `AGENTS.md`: run the three checks before each commit; commit on `main`, never push; a new design decision gets a new ADR, and the docs that describe the code change in the same commit; drive Herdr only with `hc` and `herdr-app`; do not print `server.key`; the recents card stays blank in every build.
- `docs/product.md` section 5 lists iOS, and a relay server or push service, as out of scope. ADR 0013 records that the user ruled out a relay and a push service. This task reverses the first. The second may change only if parity needs Apple's push service and the user buys a membership; that change needs a new ADR, and the push must carry no content (SR-11).
- `docs/product.md` section 8 (SR-1 to SR-13) and `docs/architecture.md` section 6: the security model. The phone key never leaves secure hardware; one user action gives at most one key press or message; the lock screen shows no prompt.

## Criteria

The user gave one: 100% feature parity. Under it, I optimise for: (1) every item in the parity list works on iOS; a gap is acceptable only where no iOS API, free or paid, can close it, and the plan names it; (2) the security requirements hold unchanged; (3) the smallest system, as ADR 0015 chose for Android: native stack, few dependencies; (4) no change in behaviour for the Android app; daemon changes only where iOS parity needs them, additive, with no effect on Android.

## Non-goals

- Restructuring or rewriting the Android app, or a shared cross-platform codebase.
- Changes to Herdr.app or its shipped mods.
- An App Store or TestFlight release. Mac layouts. iPad-specific layouts beyond what the app needs to work well on an iPad.
- New features that Android does not have.

## Known facts

- Xcode 27.0, Swift 6.4, an iOS 27 simulator runtime with iPhone simulators, and `xcodegen` are installed. There is no code-signing identity, no provisioning profile and no connected iPhone.
- **The test device is an iPad** (user, 2026-11-15). Its iPadOS version is not known yet.
- **The user has no paid Apple Developer Program membership today, and requires 100% parity** (user, 2026-11-15). If parity needs a membership, the plan says which items need it, what it unlocks, and its price, and the user decides. Until then the app is signed with a free Apple ID (a Personal Team):
  - an install on a device runs for 7 days, then needs a new install from Xcode on this Mac;
  - no push notifications (APNs) and no Local Push Connectivity;
  - simulator builds and tests need no signing.
- The membership costs USD 99 a year, charged in local currency at enrollment (developer.apple.com/support/enrollment). Indian developers report ₹8,700 a year; in India, enrollment is only through the Apple Developer app. One membership covers every app on every Apple platform.
- iOS has no equivalent of a foreground service: a backgrounded app cannot keep `/v1/events` open. So item 9 probably needs APNs for parity, and so a paid membership. (Platform facts: agents confirm or correct them with sources.)

## Questions

1. **For the user (non-blocking):** the iPad's iPadOS version (Settings, General, About). The plan states the minimum version it needs. Signing in to Xcode with an Apple ID, and Developer Mode on the iPad, are steps only the user can do.
2. **For the agents:** the stack and project layout (SwiftUI, deployment target, `xcodegen` or a committed project, dependencies); mTLS with a Secure Enclave key, which needs a self-signed certificate and an identity; the QR scanner (needs camera permission on iOS); the live view in WKWebView with the shared xterm assets; what differs on an iPad (sizes, rotation, multitasking, no haptics on most iPads); Swift format, tests from the command line and the shared vectors; the full parity map, item by item.
3. **For the agents:** item 9 at full parity: alerts within 5 s while the app is closed and after a reboot, Approve and Deny from the alert, finish alerts, redaction on a locked device. Design the path (for example a content-free APNs push from the daemon, then a notification service extension that fetches the prompt over mTLS), with the daemon changes and the secrets it adds on the Mac. Say what a free account gives until a membership exists.
4. **For the agents:** every parity item that needs a paid membership (for example the 7-day install limit, push, keychain or app-group sharing with an extension), and what it unlocks for each. The price is a known fact above.

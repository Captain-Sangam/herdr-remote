• ROUND 1 ios-app-a1

  Position.

  Use Swift 6, SwiftUI, iOS/iPadOS 17+, and ios/project.yml with XcodeGen; separate Core, App, Extensions and Tests. This follows the small native stack criterion (ADR 0015:10 (docs/adr/0015-android-
  stack.md:10)). Generate a permanent Secure Enclave P-256 SecKey, self-sign a DER certificate, add it to Keychain, then retrieve kSecClassIdentity for TLS. Apple documents this identity workflow. Use
  Network.framework, bundled xterm in WKWebView, and paid APNs for background alerts. Literal parity has platform limits below.

  Reasoning.

  “Free” means Personal Team provisioning, which expires after seven days. Membership costs USD 99/year; India enrollment uses Apple’s Developer app, with the local price shown there. Ordinary
  Keychain Sharing is available free; avoid App Groups.

   Item and reference                                                  iOS mechanism                                                       Signing
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   1. Pairing (android/app/src/main/java/dev/herdrremote/ui/           AVFoundation QR with camera permission; paste; pinned               Free
   PairScreen.kt:80)                                                   preliminary handshake, code, 130-second enrollment.
  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────
   2. TLS/endpoints (android/app/src/main/java/dev/herdrremote/net/    Secure Enclave identity; TLS 1.3; SPKI-only verification; last-     Free
   Api.kt:54)                                                          good endpoint first; hello refreshes endpoints/clock.
  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────
   3. Events (android/app/src/main/java/dev/herdrremote/watch/         Actor-owned SSE, 30-second inactivity watchdog, reducer,            Free
   AgentsRepository.kt:60)                                             reconnect/backoff, stale banner.
  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────
   4. Lists (android/app/src/main/java/dev/herdrremote/ui/             SwiftUI Spaces/Space/Agents; blocked-first sorting and tab          Free
   ListScreens.kt:132)                                                 count.
  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────
   5. Transcript (android/app/src/main/java/dev/herdrremote/ui/        SwiftUI plus UIKit scrolling: folds, follow/Bottom, pagination,     Free
   AgentScreen.kt:343)                                                 message snapping, scrollbar, one-second refresh, working/model
                                                                       rows, copy.
  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────
   6. Live (android/app/src/main/java/dev/herdrremote/ui/              Nonpersistent WKWebView; bundle-only scheme, CSP/resource           Free
   LiveView.kt:36)                                                     blocking; no input bridge; 64-KiB callback acknowledgments, 1-
                                                                       MiB reopen, pinch.
  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────
   7. Actions (android/app/src/main/java/dev/herdrremote/ui/           Approve/Deny/options/Next; identical results/refusals; Mac-clock    Free
   AgentViewModel.kt:229)                                              UUIDv7; no retransmission after first write.
  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────
   8. Composer (android/app/src/main/java/dev/herdrremote/ui/          Native editor, 4000-character validation, slash skill picker.       Free
   AgentScreen.kt:179)
  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────
   9. Watching (android/app/src/main/java/dev/herdrremote/watch/       APNs, service/content extensions; blocked/finish episodes, per-     Needs paid membership: APNs entitlement/provider credentials
   WatchService.kt:129)                                                agent replacement, authenticated actions.
  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────
   10. Lock (android/app/src/main/java/dev/herdrremote/Gate.kt:7)      LocalAuthentication at launch and after five minutes away.          Free
  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────
   11. Privacy (android/app/src/main/java/dev/herdrremote/             Synchronous scene cover before snapshots; screenshots permitted;    Free
   MainActivity.kt:30)                                                 memory-only content and web storage.
  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────
   12. Settings (android/app/src/main/java/dev/herdrremote/ui/         Mac/endpoints, watching control, Forget removes credentials/        Free; background delivery needs membership
   SettingsScreen.kt:57)                                               subscription.
  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────────  ──────────────────────────────────────────────────────────────
   13. Pane (docs/design.md:12)                                        Port tokens, fonts and components; adaptive widths, rotation and    Free
                                                                       multitasking.

  Item 5 has unavoidable differences: unsupported iPad haptics become visual feedback; Android’s sensitive clipboard flag has no direct equivalent, so use local-only pasteboard with expiry.

  Item 9 cannot guarantee five-second delivery: APNs is best effort. Free builds watch only while active. Reboot recovery uses system push delivery; protected credentials require first unlock. Offline
  LAN watching additionally needs paid membership and Apple approval for Local Push Connectivity; it cannot replace APNs everywhere.

  Paid path: provision app/extensions, enable APNs, create its provider key, store it on the Mac with mode 0600. Add device-authenticated token registration, rotation, disable/revoke cleanup and
  daemon episode tracking matching Reducer.kt:79 (android/app/src/main/java/dev/herdrremote/watch/Reducer.kt:79). Send priority-10 generic alerts with opaque identifiers, per-agent collapse IDs and
  expiry. NSE fetches metadata over mTLS; failure leaves generic text without actions.

  Never put prompts in notification bodies: preview settings are read-only. Prototype authenticated expanded UI that fetches prompts into memory and binds decisions to the displayed turn/hash. Add
  asynchronous action acceptance/result polling because existing actions allow 150 seconds (docs/api.md:92).

  Outline.

  Order: identity/privacy prototypes → transport/pairing → reducer/screens/live → paid notification path → docs/ADR, scripts and macOS CI. Keep the mod unchanged. TLS already accepts P-256 clients
  (tls.go:98 (daemon/tls.go:98)); daemon additions must preserve Android behavior.

  Pin/key failures refuse connection; stale prompts/audit failures send no key; ambiguous writes become outcome unknown; notification failures expose no prompt. Preserve SR-1–13 (docs/product.md:158).

  Proposed commands:

  xcodegen generate --spec ios/project.yml
  xcodebuild -project ios/HerdrRemote.xcodeproj -scheme HerdrRemote \
    -destination 'platform=iOS Simulator,id=<UDID>' \
    -derivedDataPath ios/build CODE_SIGNING_ALLOWED=NO test
  swift format lint --recursive ios

  Repeat xcodebuild test with the iPad destination, signing enabled and -allowProvisioningUpdates.

  Test shared pairing vectors (testdata/pairing-vectors.json:2), fragmented HTTP/SSE, uncertain writes, revocation and hostile terminal bytes. Simulator checks cover both sizes; iPad checks cover
  hardware TLS, lock snapshots, notification previews under every setting, reboot, force-quit, LAN/Tailscale, rotation and multitasking.

  Assumptions.

  • Confirm iPadOS version; user handles Xcode login and Developer Mode.
  • Inline notification redaction/lifetime remains a feasibility gate. Foreground review is a disclosed fallback, never counted as parity.

  Confidence.

  80

  END ROUND 1

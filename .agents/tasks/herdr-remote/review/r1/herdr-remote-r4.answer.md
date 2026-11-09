• ROUND 1 herdr-remote-r4

  1. herdr-remote-r4-1 — android/app/src/main/java/dev/herdrremote/MainActivity.kt:57. Every unsuccessful canAuthenticate result unlocks the app, including errors other than “no screen lock.” Authentication failure therefore exposes controls.
     Likelihood: unknown; depends on device authentication failures.
     Severity: 9; bypasses lost-phone protection.
     Label: implementation defect. Disposition: new.
     Fix: remain locked on authentication errors.

  2. herdr-remote-r4-2 — android/app/src/main/java/dev/herdrremote/net/Api.kt:81; android/app/src/main/java/dev/herdrremote/ui/AgentViewModel.kt:61; android/app/src/main/java/dev/herdrremote/ui/AgentViewModel.kt:155. If a message executes but its response disappears, exhausted endpoints produce “not reachable.” Another Send creates a fresh UUID and duplicates
     delivery.
     Likelihood: 6; network changes or response timeouts.
     Severity: 8; repeats potentially consequential agent instructions.
     Label: implementation defect. Disposition: new.
     Fix: report ambiguous POST failures as outcome unknown and retain request IDs.

  3. herdr-remote-r4-3 — android/app/src/main/java/dev/herdrremote/net/Api.kt:119; android/app/src/main/java/dev/herdrremote/net/Api.kt:77; daemon/actions.go:156. Actions use endpoint retry. A lost refusal response can be retried automatically and subsequently dispatched because refused IDs are forgotten, contradicting TDD.md:416.
     Likelihood: 5; multiple endpoints plus a lost response.
     Severity: 7; an initially refused action executes later without another tap.
     Label: requirement miss. Disposition: new.
     Fix: separate POST execution from read endpoint fallback.

  4. herdr-remote-r4-4 — daemon/devices.go:96; daemon/daemon.go:177. Failed removal persistence deletes the device from memory but returns before closing connections. Retrying cannot find it; streams continue, and restart reloads the pairing.
     Likelihood: 3; disk-full or filesystem errors.
     Severity: 8; a lost phone retains access.
     Label: implementation defect. Disposition: new.
     Fix: make removal transactional and close streams despite persistence failure.

  5. herdr-remote-r4-5 — android/app/src/main/java/dev/herdrremote/ui/LiveView.kt:76; android/app/src/main/java/dev/herdrremote/ui/LiveView.kt:113. Leaving an idle view cancels its coroutine without closing the blocking network read. Leaving two views consumes both daemon slots (daemon/live.go:80).
     Likelihood: 8; ordinary navigation around idle agents.
     Severity: 6; leaked streams/processes prevent subsequent live viewing.
     Label: implementation defect. Disposition: new.
     Fix: cancel the OkHttp call and close its response on disposal/reopen.

  6. herdr-remote-r4-6 — daemon/pair.go:85; daemon/pair.go:243; daemon/pair.go:216. A LAN client with its own P-256 certificate can stall an enrollment body indefinitely. No body deadline exists; after Shutdown times out, only the listener closes, leaving active connections.
     Likelihood: 4; deliberate enrollment-window abuse.
     Severity: 6; unauthenticated resource exhaustion survives window expiry.
     Label: implementation defect. Disposition: new.
     Fix: bound body reads and forcibly close remaining enrollment connections.

  7. herdr-remote-r4-7 — android/app/src/main/java/dev/herdrremote/watch/WatchService.kt:94. Default notification settings expose the tab name, observed in TDD.md:862, contradicting generic-only criterion 12 (TDD.md:470).
     Likelihood: 10; recorded default behavior.
     Severity: 6; project information appears while locked.
     Label: requirement miss. Disposition: new.
     Fix: make primary notification content generic.

  8. herdr-remote-r4-8 — daemon/transcript.go:471; daemon/transcript.go:491. A resolved but unreadable transcript returns an error; malformed records produce an empty transcript. Neither falls through to terminal text (TDD.md:480).
     Likelihood: 4; damaged files or format changes.
     Severity: 4; approval context disappears.
     Label: implementation defect. Disposition: new.
     Fix: distinguish cursor errors from source failures and fall back.

  Completeness

  Delivered:

  • S1 delivered: toolchain/skeleton (scripts/setup-android.sh:10).
  • S2 delivered: typed subprocess access (daemon/hc.go:503).
  • S4 delivered: reads/SSE (daemon/api.go:14).
  • S5 delivered: daemon streaming (daemon/live.go:95).
  • S8 delivered: mod/install (scripts/install.sh:46).

  Remaining:

  • S3 partial: finding 4; cancel/expiry unaudited (daemon/pair.go:187).
  • S6 partial: finding 8.
  • S7 partial: target validation precedes replay lookup (daemon/actions.go:325).
  • S9 partial: finding 1.
  • S10 partial: finding 7.
  • S11 partial: findings 2, 3, 5.
  • S12 partial: outstanding acceptance (TDD.md:878).
  • C1 partial: reboot unverified (TDD.md:890).
  • C2 partial: physical pairing unverified (TDD.md:880).
  • C3 partial: space membership remains captured (android/app/src/main/java/dev/herdrremote/ui/ListScreens.kt:106).
  • C4 partial: finding 8.
  • C5 partial: finding 5.
  • C6 partial: finding 3.
  • C7 partial: finding 2.
  • C8 partial: swipe-away check outstanding (TDD.md:886).
  • C9 partial: finding 4.
  • C10 partial: cellular/Tailscale unverified (TDD.md:883).
  • C11 partial: finding 1.
  • C12 partial: finding 7.
  • C13 partial: cancel/expiry lack audit (daemon/pair.go:187).

  Unexplained changes: none identified.

  Bot comments

  None.

  END ROUND 1

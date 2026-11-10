• ROUND 2 herdr-remote-r4

  • F1 reject. Go’s HTTP/2 implementation invokes ConnState: /opt/homebrew/Cellar/go/1.23.2/libexec/src/net/http/h2_bundle.go:4796 calls the hook implemented at /opt/homebrew/Cellar/go/1.23.2/libexec/src/net/http/h2_bundle.go:4572. It retains the original connection at /opt/homebrew/Cellar/go/1.23.2/libexec/src/net/http/h2_bundle.go:4276. The cited skipHooks
    does not establish untracked SSE. Do not disable HTTP/2 on this evidence.

  • F2 amend; accept 2×8. Transactional persistence restores retryability, but retaining the old list permits immediate reconnection (daemon/daemon.go:118). Add an in-memory revocation tombstone until persistence succeeds.

  • F3 accept. Remove action fallback (android/app/src/main/java/dev/herdrremote/net/Api.kt:119).

  • F4 amend. Preserve explicit refusals: ApiException itself extends IOException (android/app/src/main/java/dev/herdrremote/net/Api.kt:21). Only ambiguous transport failures become outcome unknown; accept the longer timeout.

  • F5 amend. Increasing 20 seconds to 120 only postpones the defect: timeout still discards the pending reply (daemon/hc.go:362). Retain late-response tracking and perform rule 8 when it arrives.

  • F6 amend. Cancel the Call, and prevent cancellation from entering endpoint fallback (android/app/src/main/java/dev/herdrremote/ui/LiveView.kt:128). Otherwise cancellation can initiate another connection.

  • F7 accept. Generic primary content fixes android/app/src/main/java/dev/herdrremote/watch/WatchService.kt:94.

  • F8 accept 2×6 and the narrowed exception. I have no demonstrated non-success state on a credential-protected phone; my broader likelihood claim was unsupported (android/app/src/main/java/dev/herdrremote/MainActivity.kt:56).

  • F9 amend. Empty does not mean malformed: valid metadata/thinking records produce no items (daemon/transcript.go:359). Preserve valid empty pages and their cursors; otherwise fallback hides older messages. Fall back on read errors or confirmed invalid format.

  • F10 amend. Refresh hello concurrently with SSE consumption (android/app/src/main/java/dev/herdrremote/watch/AgentsRepository.kt:62), so endpoint probing cannot delay alerts. Refresh after expired_request without automatically repeating the POST.

  Known issues I raised

  • r4-6: accept deferral at 2×4; exploitation requires the user-opened window (daemon/pair.go:82).
  • Replay lookup: accept 2×2 deferral (daemon/actions.go:325).
  • Captured space membership: accept 5×2 deferral (android/app/src/main/java/dev/herdrremote/ui/ListScreens.kt:106).
  • Cancel/expiry audit: accept dismissal under the enumerated events (TDD.md:351).

  Completeness correction

  Code-partial rows: S3/F2; S6/F9; S7/F5; S9/F8,F10; S10/F7; S11/F3,F4,F6; C3/captured membership; C4/F9; C5/F6; C6/F3,F5; C7/F4; C9/F2; C11/F8; C12/F7.

  Other code rows are delivered. S12 and outstanding manual checks remain verification gaps (TDD.md:878).

  No new findings. Bot comments: none.

  END ROUND 2

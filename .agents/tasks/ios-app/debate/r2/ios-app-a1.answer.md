• ROUND 2 ios-app-a1

  Position.

  I withhold consensus pending the retry and privacy fixes below. I concede URLSession for HTTP, SSE, live and enrollment. Retain Network.framework for step 5’s handshake-only preflight (.agents/
  tasks/ios-app/debate/draft.md:203), and as an action-transport fallback if the no-retry requirement cannot be demonstrated.

  Reasoning.

  • ATS: Apple explicitly permits relaxed HTTPS trust through NSExceptionAllowsInsecureHTTPLoads; iOS 17 supports IP/CIDR exceptions. Keep the pin and TLS 1.3. Correct step 1’s invariant: the
    exception permits plaintext, so application code must reject HTTP and redirects.

  • Async API: withdrawn. The daemon detaches after dispatch (daemon/actions.go:553), and unknown-outcome handling (docs/architecture.md:344) already covers disconnection. This concession does not
    establish unlimited extension lifetime.

  • Identity source: Apple DTS thread 797509 describes generated EC key plus certificate → Keychain identity and explicitly includes Secure Enclave. Self-signing that certificate is my engineering
    inference, not Apple’s demonstrated end-to-end example.

  • Free signing: confirmed for ordinary access groups. Apple DTS’s Personal Team experiment shows the permitted prefix wildcard. Keep step 11’s actual app-to-extension sharing test; the excluded
    com.apple.token group is unrelated.

  Outline.

  • Decision 5; step 3: countOfBytesSent excludes headers, whereas Android marks requestHeadersStart (android/app/src/main/java/dev/herdrremote/net/Api.kt:43). Zero alone cannot prove “never sent.”
    Count actual HTTP requests under injected failures and redirects: the proposed single-audit check misses retransmission because known IDs return without redispatch (daemon/actions.go:444).

  • Decision 10; steps 10–11: the initial unlock probe (.agents/tasks/ios-app/debate/draft.md:82) leaves a race: probe succeeds, device locks, fetch completes. Require cancellation/redaction evidence
    during fetch and after rendering, including “Always” previews. The unlocked happy-path check cannot establish SR-11.

  • Decision 8; step 8: custom-scheme handling does not intercept built-in schemes. Add CSP/resource blocking and attempted fetch/image/WebSocket tests; navigation-policy testing alone (.agents/tasks/
    ios-app/debate/draft.md:241) cannot establish bundle-only loading.

  • Step 13: replace the simctl-only NSE check (.agents/tasks/ios-app/debate/draft.md:311). Apple’s Xcode 14 notes distinguish actual APNs delivery supporting service extensions from locally simulated
    pushes. Unit-test the handler, then verify through paid APNs sandbox delivery.

  Assumptions.

  Move hardware identity, sharing and notification-privacy feasibility checks before substantial screen implementation.

  Confidence.

  88

  END ROUND 2

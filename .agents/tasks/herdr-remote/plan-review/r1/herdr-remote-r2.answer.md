• ROUND 1 herdr-remote-r2

  1. herdr-remote-r2-1 — Decision 10, Step 7; infeasible; new.
     Evidence: Mac input changes the prompt after validation but before delayed injection; terminal.keys has no conditional parameter (schema; plan §7). The per-terminal lock cannot exclude Mac input.
     Likelihood 6: concurrent local use. Severity 9: approves an unseen action.
     Correction: Treat §4.6 as unresolved. Require action-bound approval support or explicit acceptance of a weaker guarantee; test changes during the injection delay.

  2. herdr-remote-r2-2 — Decisions 10/12, Step 7; security gap; new.
     Evidence: restart after dispatch but before status changes loses the consumed marker; replay can dispatch again. /message has no idempotency identifier at all (plan lines 88, 252).
     Likelihood 6: connection loss, retries or daemon restart. Severity 8: repeats potentially destructive agent work.
     Correction: Persist action identifiers before dispatch, retain uncertain outcomes, and prohibit automatic redispatch. Test crash recovery and dropped responses for both endpoints.

  3. herdr-remote-r2-3 — Step 3; under-specified; new.
     Evidence: another process under the same Mac UID can read server.key, change devices.json, or invoke pair allow; 0700/0600 do not separate same-user processes (lines 176–177).
     Likelihood 5: an agent or compromised local application accesses these paths. Severity 9: bypasses pairing confirmation.
     Correction: Explicitly declare same-UID processes trusted. If resistance is required, redesign the privilege boundary; filesystem permissions cannot deliver it.

  4. herdr-remote-r2-4 — Decision 14, Steps 9–11; security gap; new.
     Evidence: backgrounding a transcript can leave its recent-apps screenshot visible after the biometric gate expires. No policy covers previews, notification content or logs. Android documents this
     exposure under FLAG_SECURE.
     Likelihood 7: ordinary backgrounding followed by phone loss. Severity 7: exposes project data.
     Correction: Blank protected previews, use generic notifications, prohibit sensitive logging, and test these surfaces while locked.

  5. herdr-remote-r2-5 — Decisions 4/7, Step 3; under-specified; new.
     Evidence: implementing SPKI checks in VerifyPeerCertificate allows resumed handshakes to skip revocation checks; Go documents this. Request checks still protect data, but §4.9 fails.
     Likelihood 5: conventional custom-verifier implementation. Severity 5: revoked keys retain handshake access.
     Correction: Specify VerifyConnection against current device state, or disable tickets; test resumed connections after revocation.

  6. herdr-remote-r2-6 — Decision 4, Step 9; under-specified; new.
     Evidence: a correct SPKI with a certificate lacking the current LAN/Tailscale address still fails OkHttp hostname verification. A TrustManager alone does not replace it.
     Likelihood 8: first connection or address change. Severity 7: pairing/connectivity fails.
     Correction: Specify the certificate identity and endpoint-independent hostname-verification policy, restricted to the pinned peer; test LAN, Tailscale and changed addresses.

  7. herdr-remote-r2-7 — Steps 3/8; infeasible; new.
     Evidence: pair wait can block for 120 seconds, but herdr.d.ts:10 (/Users/zeus/.config/herdr-app/herdr.d.ts:10) limits exec to 30 seconds. Scanning after that loses the pending-device watcher.
     Likelihood 8: user takes over 30 seconds. Severity 7: pairing stalls.
     Correction: Use bounded waits below 30 seconds with explicit timeout/retry results; test enrollment near expiry.

  8. herdr-remote-r2-8 — Decision 13, Step 10; under-specified; new.
     Evidence: declaring only specialUse and POST_NOTIFICATIONS omits mandatory foreground-service permissions; Android 14+ requires them.
     Likelihood 7: literal implementation. Severity 6: background alerts cannot start.
     Correction: Specify manifest permissions/type, foreground startup, persistent notification, and recovery restrictions. Test Android 14/15 with notification permission denied and service
     termination.

  9. herdr-remote-r2-9 — Steps 5/11; plan defect; new.
     Evidence: unrestricted base64 delivery can outpace rendering; xterm.js buffers writes and discards overflow.
     Likelihood 7: verbose builds or hostile output. Severity 6: frozen or incorrect terminal display.
     Correction: Bound queues and acknowledge rendering before further delivery; test sustained output, slow rendering and disconnect cleanup.

  10. herdr-remote-r2-10 — Decision 9, Step 6; plan defect; new.
     Evidence: taking the last 2 MB usually starts inside a JSONL record; only partial trailing lines are handled. The resulting parse failure can replace a valid transcript with terminal text.
     Likelihood 9: sufficiently long sessions. Severity 6: transcript/paging requirement fails.
     Correction: Specify record-aligned pagination, oversized-record handling and session-bound cursors; test multi-page files.

  Completeness

   Brief requirement     Status and steps
  ━━━━━━━━━━━━━━━━━━━━  ━━━━━━━━━━━━━━━━━━━━━
   QR pairing            Partial: 3, 8, 9
  ────────────────────  ─────────────────────
   Spaces                Delivered: 4, 10
  ────────────────────  ─────────────────────
   Agents in a space     Delivered: 4, 10
  ────────────────────  ─────────────────────
   All-agents view       Delivered: 4, 10
  ────────────────────  ─────────────────────
   Transcript            Partial: 6, 11
  ────────────────────  ─────────────────────
   Approve or deny       Partial: 7, 11
  ────────────────────  ─────────────────────
   Live terminal view    Partial: 5, 11
  ────────────────────  ─────────────────────
   Send a message        Partial: 7, 11
  ────────────────────  ─────────────────────
   Safe and secure       Partial: 3, 7, 9–11
  ────────────────────  ─────────────────────
   Wi-Fi or Tailscale    Partial: 3, 9, 12

  END ROUND 1

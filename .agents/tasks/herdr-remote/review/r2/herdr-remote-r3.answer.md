⏺ ROUND 2 herdr-remote-r3

  Fix list

  - F1 accept. Also set TLSNextProto on the enrollment server (pair.go:85-90) so both listeners behave alike. Edge: the keepalive recheck must look the device up again from r.TLS.PeerCertificates[0], not reuse the ctxDevice value captured at request time. OkHttp's pool health probe discards a connection the registry closed, so http/1.1 reuse adds nothing new.
  - F2 accept.
  - F3 accept. Edge: the phone already makes a fresh UUID per tap (AgentViewModel.kt:138,155), so the daemon's known-id replay (actions.go:347-353) is never exercised once fallback goes. Amend: a user-initiated "Check again" on an unknown outcome re-POSTs the same request_id, which is not an automatic retry and returns the recorded result per 3.5.3. That is
    also how a message outcome gets resolved without a second prompt.
  - F4 accept. Classify connect-phase failures (ConnectException, UnknownHostException, timeout before the handshake) as "not reachable"; everything else is "Outcome unknown".
  - F5 accept. Edge: the per-terminal lock (actions.go:360-362) is then held up to 120 s, so a second phone's decision on that terminal waits that long before already_answered; acceptable. Whether hc serialises a pending terminal.keys ahead of a terminal.send on the same actions connection is unknown.
  - F6 accept, score agreed; LiveView.kt:107-126 has no cancel path. Edge: the daemon frees the slot only when the socket close propagates, so the reopen right after cancel can see one live_limit 409; the 500 ms backoff at LiveView.kt:99 absorbs it.
  - F7 accept. Challenge conceded: I marked C12 delivered from the As built note. Criterion 12 says the lock screen shows only "An agent needs attention", and WatchService.kt:94-96 puts a.label in the private title, which Android shows under its default setting. The plan is also inconsistent with itself: 3.7 Privacy says the private version shows the tab name.
    C12 is partial; the fix plus the decision 14 amendment resolves both.
  - F8 accept. MainActivity.kt:57-65 unlocks on every non-success code; only BIOMETRIC_ERROR_NONE_ENROLLED means no screen lock.
  - F9 amend. "First page with no items falls back" is wrong when a 2 MiB page holds only non-message records; fall back only when the page has no items and no next_cursor, else page back.
  - F10 accept. Edge: hello() replaces store.endpoints; keep the as-built "last good first even if unlisted" rule.

  My known issues

  - r3-4 pid reuse: accept as known.
  - r3-6 empty 200: accept as known, but if the same-id "Check again" of F3 lands, release (actions.go:156-161) must store the refusal body, since that retry path returns.
  - live.go defer order, unmetered refusal lines, install.sh order: accept as known.

  Scores: no disputes.

  New findings: none material.

  END ROUND 2

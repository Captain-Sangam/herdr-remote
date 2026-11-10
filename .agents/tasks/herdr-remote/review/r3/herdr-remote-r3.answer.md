⏺ ROUND 3 herdr-remote-r3

  Fix 1 (b360936): fixed. Api.kt:83-87 marks started at requestHeadersStart, which fires after the TLS handshake, so a connect-phase failure moves on and anything later throws (Api.kt:144-147). AgentViewModel.kt:190 keeps an ApiException reason and maps every other failure to the outcome-unknown text; the action client reads for 150 s (Api.kt:98).

  Fix 2 (cb5e759): fixed in code. WatchService.kt:328 makes the private title generic and drops the content text. The decision 14 and section 3.7 amendment named in the spec is not in the diff: no .md file changed between f858d27 and HEAD.

  Fix 3 (c8682c0): fixed. AgentsRepository.kt:305 launches hello() beside the stream after each connect; AgentViewModel.kt:217-219 resyncs after expired_request without a re-post; candidates() still puts the last good endpoint first.

  Fix 4 (429c512): fixed. LiveView.kt:255-258,264,270-271 keep the Call under the instance monitor, cancel it in stop() and reopen(), and openCall returns null once the generation moved or the scope ended; LiveView.kt:143 stops the next-host loop on a cancelled call.

  Fix 5 (05f8ce8): fixed. transcript.go:758-783 returns nil on a read error or a first page with no item after paging to the start, transcript.go:732-741 then serves terminal text, and a cursor still gets 409. The test at transcript_test.go:798-846 covers all four cases.

  Fix 6 (62562af): fixed. devices.go:62 hides a marked key from both the handshake and the middleware, daemon.go:625-636 closes connections and live views before drop, and devices.go:703-709 restores the list and keeps the mark on a failed write, with the audit line only after success. devices ls still shows the marked device until the write succeeds, which
  the retry needs.

  Fix 7 (05423de): fixed in code. actions.go:479 waits keyWait of 120 s, so the rule-8 idle check runs when hc delivers a queued key; actions_test.go:494-536 proves the short wait skips the check and the full wait runs it. The section 7 amendment in the spec is not in the diff.

  Fix 8 (0e367cb): fixed. Gate.kt:21-25 opens without a prompt only on BIOMETRIC_ERROR_NONE_ENROLLED; MainActivity.kt:51-55 stays locked with the code and a retry for every other result.

  END ROUND 3

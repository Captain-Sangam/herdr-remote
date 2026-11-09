  IMPLEMENT herdr-remote-impl2 fix1

  Fix 1, done. android/.../net/Api.kt, android/.../ui/AgentViewModel.kt, TextsTest.kt, new SentMarkTest.kt. An action moves to the next endpoint only while OkHttp has not started to write the request (an EventListener sets a per-call mark at requestHeadersStart, which covers connect refused, unknown host and a timeout before the TLS handshake ended). After
  that the failure is rethrown; the ViewModel shows "Outcome unknown. Check the transcript before you send again" for anything that is not an ApiException. Action read timeout 150 s. Reads keep their fallback. Check: 2 new JVM tests (a refused connection is not marked; a request with no reply is marked), new text test, testDebugUnitTest assembleDebug green.
  Commit b360936.

  Fix 2, done. WatchService.kt: both versions say "An agent needs attention"; the tab name is gone from the notification. No JVM test possible; build green. Commit cb5e759.

  Fix 3, done. AgentsRepository.kt: /v1/hello is launched in its own coroutine beside each new event stream, before the read loop. AgentViewModel.kt: after an expired_request reply, hello() runs without re-posting. Endpoint order unchanged (last good first). Build green. Commit c8682c0.

  Fix 4, done. Api.kt (live() returns the Call; open(call) runs it), LiveView.kt (the feed keeps the Call, cancels it in stop() and reopen(); a cancelled call returns without trying the next host). Build green. Emulator check against the TestManualServe harness with an hr-probe-live Claude agent: before the fix, leaving the Live tab left the hc attach
  --observe process running (1 after leaving); after the fix, two Live views opened and left, 0 attach processes, app process alive throughout. Commit 429c512.

  Fix 5, done. daemon/transcript.go, transcript_test.go. Unreadable file or no parsing record: terminal text. First request pages back to the first item or the file start. A later empty page keeps its cursor. Other-file cursor still 409 cursor. New test failed before (empty claude page), passes after; go test ./... green (84 s). Commit 05f8ce8.

  Fix 6, done. daemon/devices.go (mark then drop; bySPKI ignores marked ids), daemon/daemon.go (revoke: mark, close connections and live views, write, audit on success). New test with an unwritable state dir failed before (SSE survived), passes after; suite green. Commit 62562af.

  Fix 7, done. daemon/actions.go (keyWait field, default 120 s), actions_test.go. Test asserts the default and that the wait bounds the key (short wait: outcome_unknown, one idle read; full wait: applied, two idle reads). Fails to build without the fix; suite green. Commit 05423de.

  Fix 8, done. Gate.kt (check()), MainActivity.kt (only BIOMETRIC_ERROR_NONE_ENROLLED opens; other errors stay locked with the code and the Unlock button retries), PinTest.kt. New JVM test passes; build green. Commit 0e367cb.

  Final HEAD: 429c512.

  END IMPLEMENT fix1

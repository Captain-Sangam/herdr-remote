  IMPLEMENT herdr-remote-impl3 fix1

  Fix 1: done. Gate.kt (new Gate.Observer, a DefaultLifecycleObserver with an injectable clock), HerdrRemoteApp.kt (registers it once in onCreate), MainActivity.kt (own observer, onDestroy and imports removed; onResume still reads Gate.locked), new GateTest.kt. Check: 3 JVM tests drive a LifecycleRegistry through ON_START, ON_STOP, ON_START at 5 minutes and
  assert the lock. Emulator: unlocked with a PIN, watching on, task removed with am stack remove (the swipe-from-recents path; process pid 4500 stayed alive on the foreground service), waited 5 min 25 s, reopened: the credential prompt appeared before any content. Commit e0dabd6.

  Fix 2: done. AgentScreen.kt: the prompt block gets heightIn(max = screenHeightDp / 3) plus verticalScroll, horizontal scroll kept. Check: build. Emulator with a 45-line heredoc permission prompt: the block is a bounded scroll box and Approve, Deny and Reload sit at mid-screen (y 1396 to 1522 of 2400). Note: hc put that command in the agent's msg title, so
  the block itself held only the question lines; the cap was not pushed past its limit. Commit e96ec0b.

  Fix 3: done. AgentViewModel.kt: refreshPrompt(manual = true); decide() calls it with manual = false; the pure Texts.refreshError(current, e, manual) decides the note. TextsTest.kt: one new test. Check: 19 JVM tests pass, build green. Emulator: with the network off, Approve showed "Outcome unknown…" and it stayed after the failed refresh. Commit f1d3a40.

  Fix 4: done. Api.kt (events() returns a Call), AgentsRepository.kt (keeps the call, stop() cancels it, a cancelled read returns instead of trying the next host, openCall refuses after stop()), HerdrRemoteApp.kt (forgetMac() stops WatchService first), WatchService.kt (onDestroy joins the collectors and cancels the ongoing notification). The last item came
  from the emulator check: the state reset during stopService() reposted "Connecting to the Mac…" after the system had removed it, leaving an undismissable notification. Check: build; emulator against the scratch daemon: after Forget the Pair screen shows, the service is gone, no notification remains, a new blocked probe raised no alert, and the daemon sent no
  SSE keepalives afterwards. One idle pooled TLS connection stayed until OkHttp's 5-minute eviction (observed closing), not the stream. Commit fca5008 (amended once to add the onDestroy change).

  Final HEAD: fca5008 on main, working tree clean except .agents/. Emulator shut down (PIN I set was cleared, the debug app stays installed, unpaired), scratch daemon stopped, probes hr-probe-1 to -4 killed and removed. A finished hr-probe terminal t2t from an earlier run was left as found.

  END IMPLEMENT fix1

⏺ ROUND 3 herdr-remote-r5

  Fix 1 (G1): fixed. One observer on the process lifecycle from HerdrRemoteApp.kt:21, so no activity can remove it (Gate.kt:43-52); MainActivity.kt:33-38 reads Gate.locked in onResume, which runs after Android has dispatched the process ON_START during the activity's start. GateTest.kt:34-47 covers the 700 ms case.

  Fix 2 (G3): fixed. The block is capped at a third of the screen with vertical scroll (AgentScreen.kt:68-69), so the Approve and Deny row at :71-75 stays within reach. The title at :65 has no bound, but it carries hc's one-line summary (the fixture's longest msg is a 120-character path line, daemon/testdata/events-probe.jsonl; the daemon passes it through
  uncut, daemon/state.go:343), and the 45-line heredoc probe kept the buttons at mid-screen, so it cannot push them out on the evidence. Cheap insurance, not a finding: maxLines = 3 with ellipsis on that title, since the block below shows the full command.

  Fix 3: fixed. The refresh after a decision keeps the note (AgentViewModel.kt:131-134, :152). The event-driven refresh at :96 still reports its errors, but it only fires on an agent update, which needs a working stream, so it cannot run while the Mac is unreachable.

  Fix 4 (G2): fixed. stop() cancels the stream call (AgentsRepository.kt:49-54), the blocking read then throws and the loop returns on isCanceled (:81), and a stale loop cannot open a new call (:57-59, guarded by the same monitor as start()). forgetMac stops the service first (HerdrRemoteApp.kt:32), and WatchService.onDestroy joins the collectors before
  clearing the ongoing notification (WatchService.kt:70-74); the collectors need no main-thread work, so the join cannot deadlock.

  No regressions.

  END ROUND 3

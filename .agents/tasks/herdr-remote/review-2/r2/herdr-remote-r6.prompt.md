You are herdr-remote-r6.

Round two of this review, one short round. Same rules (read-only, no suites, no key files, no hc actions). The user's instruction still applies: only things that matter.

One reviewer reported zero findings; the other reported three. The orchestrator verified all three in the code:
- G1 (herdr-remote-r6-1) The app gate can be bypassed. MainActivity.kt:28-33 registers its ProcessLifecycleOwner observer per activity and removes it in onDestroy (:85-87). ProcessLifecycleOwner dispatches ON_STOP about 700 ms after the last activity stops. The orchestrator's sharper trigger: with background watching on (the foreground service keeps the process alive), swiping the app away from recents destroys the activity inside that 700 ms, so Gate.onBackground never runs, backgroundedAt stays null, and every later open shows content without authentication until the process dies. Scored L6 S7. Proposed fix: register the observer once in HerdrRemoteApp.onCreate.
- G2 (herdr-remote-r6-2) "Forget this Mac" (HerdrRemoteApp.kt:28-32) calls repository.stop(), which cancels the job, but the blocking readUtf8Line in AgentsRepository.kt:79-82 keeps the SSE stream (keepalives prevent the 30 s timeout), events refill the cleared state, and WatchService is not stopped. Scored L4 S5 (rare trigger; actions toward a newly paired Mac still pass that Mac's own guard and prompt). Proposed fix: keep and cancel the SSE Call as the live view now does, and stop the watch service before clearing the pairing.
- G3 (herdr-remote-r6-3) AgentScreen.kt:58-70: the prompt block Text has no height bound and only horizontal scroll, the surrounding Column does not scroll, so a tall block (a long multi-line command or heredoc) pushes Approve and Deny off screen. Scored L5 S6. Proposed fix: bound the block's height with vertical scroll, keeping the buttons visible.

Answer accept, reject or amend for G1 to G3 with evidence (path:line), and dispute a score only with evidence. No new findings unless one meets the bar (severity 6+ or an unambiguous requirement miss). Budget: 300 words.

For G1, say whether the swipe-from-recents trigger holds with the watch service running, and whether that changes your likelihood. For G2, argue or accept the lower likelihood.

Your answer opens with a line containing only `ROUND 2 herdr-remote-r6` and closes with a line containing only `END ROUND 2`. Put nothing after the closing line.

⏺ ROUND 2 herdr-remote-r5

  G1: accept. I missed it. MainActivity.kt:40 adds the observer to ProcessLifecycleOwner per activity and :86 removes it in onDestroy. ProcessLifecycleOwner posts its pause runnable with a 700 ms delay and only dispatches ON_STOP from that runnable or from a later activityStopped once the pause was sent, so a swipe from recents destroys the activity and
  removes the observer before ON_STOP is delivered. Gate.onBackground (Gate.kt:18) never runs, backgroundedAt stays null, and the next activity reads Gate.locked as false (MainActivity.kt:25, :49) and onForeground(null) cannot lock (Gate.kt:25-26). With WatchService holding the process, that lasts indefinitely. Criterion 11 is broken on the most common way to
  leave an app. L6 S7 holds. The fix in HerdrRemoteApp.onCreate is right: ON_START is dispatched during the activity's onStart, before onResume reads the gate.

  G2: accept. I missed it. AgentsRepository.stop() (:43-47) cancels the job, but read() (:77-99) has no suspension point: readUtf8Line (:82), _state.update and tryEmit are all blocking or synchronous, and : ka every 10 s (daemon/api.go:118-122) keeps it under the 30 s timeout (Api.kt:70). The cancelled coroutine only ends at delay (:72) after the stream dies.
  Meanwhile _state is refilled and WatchService (started only via SettingsScreen.kt:63, stopped only at :76) keeps notifying. forgetMac (HerdrRemoteApp.kt:28-32) touches neither. S5 is fair: a duplicate reader and stale alerts, with the new Mac's guard (daemon/actions.go:385-396) refusing any misdirected action. Below the user's bar, but correct.

  G3: accept. I missed it. AgentScreen.kt:48 is a non-scrolling Column(fillMaxSize); the block Text at :64 has only horizontalScroll and no height bound; the Approve and Deny row at :67-71 comes after it inside the same Card column. A block near the terminal's row count (a heredoc Bash prompt, terminal.read lines 0 returns the visible screen,
  daemon/actions.go:260) consumes the remaining height and the buttons are laid out past it with no way to reach them. Criterion 6 fails on that prompt. L5 S6 holds; the fix (bounded height with vertical scroll on the block) is the right one.

  No new findings.

  END ROUND 2

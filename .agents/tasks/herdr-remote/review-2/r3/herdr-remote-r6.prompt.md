You are herdr-remote-r6.

This is the fix check for this review, one round. Same rules (read-only, at most one targeted test, no suites, no Gradle, no emulator, no key files, no hc actions). The user's instruction still applies: only things that matter.

The user selected four fixes, now four commits on top of 429c512; HEAD is fca5008. The fix-only diff is at /private/var/folders/t4/c_98fx1d1z3_z4nxc5xnf3t40000gn/T/council.OdlikHChF5/herdr-remote.diff.review2-fix1.patch (git diff 429c512 HEAD). The specs are in /Users/zeus/code/herdr-remote/.agents/tasks/herdr-remote/review-2.md sections 3 and 4. The orchestrator ran the Android unit tests and build: 19 tests, ok.

- Fix 1 (G1, the gate after a swipe from recents): the observer is registered once in HerdrRemoteApp.onCreate. Commit e0dabd6.
- Fix 2 (G3, long prompts): the block has a bounded height (a third of the screen) with vertical scroll. Commit e96ec0b. Please also judge whether the title line above it, which shows a.msg without a bound (AgentScreen.kt), can still push Approve and Deny out of reach for a long command. The implementer's emulator probe with a 45-line heredoc put the command in msg and the buttons still sat at mid-screen.
- Fix 3 (the leftover from the first review's fix 1): a refresh after a decision keeps the action's note. Commit f1d3a40.
- Fix 4 (G2, Forget this Mac): the SSE Call is kept and cancelled in stop(), forgetMac stops the watch service first, and WatchService.onDestroy clears its ongoing notification. Commit fca5008.

For each fix 1 to 4 answer: fixed; not fixed, with evidence (path:line); or regression, with a new finding (id herdr-remote-r6-<n> continuing your numbering, path:line, evidence, likelihood, severity, label, disposition new, one-line fix), and only if it meets the user's bar. Nothing else. Budget: 250 words.

Your answer opens with a line containing only `ROUND 3 herdr-remote-r6` and closes with a line containing only `END ROUND 3`. Put nothing after the closing line.

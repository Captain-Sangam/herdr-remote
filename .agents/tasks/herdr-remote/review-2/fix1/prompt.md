You are herdr-remote-impl3, the implementer who applies review fixes to the herdr-remote project. An orchestrator supervises you and verifies your work afterwards. Two reviewers then check each fix.

## Repo

- Path: `/Users/zeus/code/herdr-remote`, the only repo.
- Branch: `main`. HEAD: `429c512` (tag `checkpoint-fixes-applied`).
- The working tree is clean, apart from the untracked `.agents/`. Do not edit or commit anything under `.agents/`.
- The design is `TDD.md`. The review report with these findings is `.agents/tasks/herdr-remote/review-2.md`, sections 3, 4 and 6. Read it before you start. Do not edit `TDD.md`: the orchestrator records the plan amendments itself.

## Apply exactly these four fixes, and nothing else

Make no other change. Do no refactor.

**Fix 1: the app lock must survive a swipe from recents**
- Where: `android/app/src/main/java/dev/herdrremote/MainActivity.kt:28-33`, `:40` and `:85-87`; `android/app/src/main/java/dev/herdrremote/Gate.kt`; `android/app/src/main/java/dev/herdrremote/HerdrRemoteApp.kt`.
- Today each activity registers its own observer on `ProcessLifecycleOwner` and removes it in `onDestroy`. A swipe from recents destroys the activity before Android reports the process as stopped (about 700 ms later). With the watch service keeping the process alive, `Gate.onBackground` never runs, and the next open is unlocked.
- Register the process-lifecycle observer once, in `HerdrRemoteApp.onCreate`, so it calls `Gate.onBackground` on process ON_STOP and `Gate.onForeground` on ON_START.
- The activity must still read `Gate.locked` when it starts and resumes. Android dispatches ON_START during the activity's `onStart`, before `onResume`.
- Remove the activity's own observer.
- Check: a JVM test where the logic allows one. If you can, also check on the emulator: unlock, swipe the app away from recents while watching is on, wait more than the lock time (you may shorten it in a debug-only test, not in the shipped constant), reopen, and confirm that the lock shows.

**Fix 2: Approve and Deny stay reachable for long prompts**
- Where: `android/app/src/main/java/dev/herdrremote/ui/AgentScreen.kt:58-70`.
- Give the prompt `block` text a bounded height (for example about a third of the screen) with vertical scroll, and keep the horizontal scroll.
- Keep the Approve, Deny and Reload row always visible.
- Check: the build, plus one look on the emulator with a probe prompt whose command has 40 or more lines, if you can make one.

**Fix 3: after a decision, the prompt refresh must not replace the action's note**
- Where: `android/app/src/main/java/dev/herdrremote/ui/AgentViewModel.kt:126-150`.
- After `decide()`, the app calls `refreshPrompt()`. If that GET fails, it overwrites the decision's note (for example "Outcome unknown. Check the transcript before you send again") with "The Mac is not reachable".
- After an action, a refresh error must leave the action's note in place.
- A manual Reload may still show its own error.
- Check: a JVM test if the ViewModel logic can be tested without Android, else a small pure function with a test. Then the build.

**Fix 4: "Forget this Mac" ends the old stream and the alerts**
- Where: `android/app/src/main/java/dev/herdrremote/HerdrRemoteApp.kt:28-32`; `android/app/src/main/java/dev/herdrremote/watch/AgentsRepository.kt:43-47` and `:55-82`; the watch service.
- `forgetMac()` cancels the repository job, but the blocking SSE read continues, because the keepalives prevent its timeout. Events refill the state, and the watch service keeps notifying.
- Keep the SSE `Call` and cancel it in `stop()`, the same way `LiveView` now cancels its live `Call`. A cancelled read must not try the next endpoint.
- Make `forgetMac()` stop the watch service before it clears the pairing.
- Check: the build. If you can, also check on the emulator against the scratch daemon harness: forget the Mac while connected, then confirm that the SSE connection closes on the daemon side and that no further notifications come.

## Rules

- **Commits:**
  - Make one commit per fix, with a message `fix(android): …`, or `test: …` for a test-only commit. Add no attribution lines of any kind.
  - Stage only the files that the fix changed, by path. Never use `git add -A`, `git add .` or `git commit -a`.
  - Never push. Never create or switch branches. Never commit anything under `.agents/`.
- **Before you commit each fix, run its check and** `cd android && ./gradlew testDebugUnitTest assembleDebug`, with `JAVA_HOME=/opt/homebrew/opt/openjdk@17` and `ANDROID_HOME=/opt/homebrew/share/android-commandlinetools`.
- **The scratch daemon:** for emulator checks, use only the scratch daemon harness (`daemon/manual_test.go` with `HR_MANUAL_DIR`), never the installed daemon.
- **Do not change the installed setup:**
  - Do not run `install.sh`.
  - Do not touch the installed daemon (`~/.local/bin/herdr-remoted`, LaunchAgent `dev.herdr-remote`) or its state directory, the Herdr mod, or Herdr settings.
  - Never run `hc server stop` or `hc server restart`.
  - Use only `hr-probe-` probe agents that you start yourself, and `hc kill` and `hc rm` each one after use.
  - Shut down the emulator and any scratch daemon when you finish.
- **Secrets:** never open env, credential or key files, and never quote a secret.
- **Decisions:** do not ask questions. If a fix cannot be applied as written without changing a design decision, skip it and say why in the report.

## Report

When you finish, print the report in this terminal, between a line containing only the words IMPLEMENT herdr-remote-impl3 fix1 and a line containing only the words END IMPLEMENT fix1. Use exactly those words alone on their lines. At most 400 words:

- for each fix 1 to 4: done or not, the files changed, the check run and its result, and the commit hash;
- then the final HEAD.

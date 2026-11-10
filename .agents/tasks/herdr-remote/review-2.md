# Review 2: herdr-remote

## 1. Reviewed

- **Repo:** `/Users/zeus/code/herdr-remote`, branch `main`.
- **Range:** from the empty tree to `429c512` (tag `checkpoint-fixes-applied`). This includes the eight fixes from the first review.
- **Plan:** `TDD.md`, as amended by section 6 of `review.md`.
- **User instruction:** "DONT NIT PICK. Surface only things that matter." Only findings with severity 6 or more, or clear requirement misses, are listed here.
- **Tests:**
  - `go vet ./...` is clean, and `go test ./...` passes.
  - `./gradlew testDebugUnitTest assembleDebug` passes, with 15 JVM tests.

## 2. Completeness

- **Delivered in code:** every step and criterion, apart from the two problems in fixes 1 and 2 below.
- **Criterion 11 (app lock):** broken in one common case (fix 1).
- **Criterion 6 (approve or deny):** broken for long prompts (fix 2).
- **Not verified:** the manual checks on a phone and over Tailscale. They are listed in `review.md` section 7, and nothing changed there.

## 3. Findings to fix

**1. The app lock stops working after you swipe the app away while background watching is on.**
- **Where:** `android/app/src/main/java/dev/herdrremote/MainActivity.kt:28-33`, `:40` and `:85-87`, and `android/app/src/main/java/dev/herdrremote/Gate.kt:18-26`.
- **Evidence:**
  - Each activity registers its own observer on the process lifecycle, and removes it when the activity is destroyed.
  - Android reports the process as stopped about 700 ms after the last activity stops.
  - A swipe from recents destroys the activity before that. The watch service keeps the process alive.
  - So the gate never records the time when the app went to the background. Every later open shows content without a biometric or PIN, until a normal background cycle records a time again.
  - This breaks criterion 11 on one of the most common ways to leave an app.
- **Scores:** likelihood 6, severity 7, score 42.
- **Fix:** register the process-lifecycle observer once, in `HerdrRemoteApp.onCreate`, not in each activity. Android dispatches ON_START during the activity's `onStart`, so the gate is up to date before `onResume` reads it.
- **Scope:** in scope.

**2. A long permission prompt pushes Approve and Deny off the screen.**
- **Where:** `android/app/src/main/java/dev/herdrremote/ui/AgentScreen.kt:48`, `:58-70`.
- **Evidence:**
  - The prompt text scrolls only sideways and has no height limit.
  - The screen column does not scroll, and the Approve and Deny row comes after the text in the same card.
  - A long multi-line command, such as a heredoc, fills the screen, and the buttons cannot be reached. This breaks criterion 6 for that prompt.
- **Scores:** likelihood 5, severity 6, score 30.
- **Fix:** limit the height of the prompt text, give it vertical scroll, and keep the buttons visible.
- **Scope:** in scope.

**3. Finish the first review's fix 1: an approval's "Outcome unknown" note can turn into "not reachable".**
- **Where:** `android/app/src/main/java/dev/herdrremote/ui/AgentViewModel.kt:129` and `:149`.
- **Evidence:**
  - After a decision, the app refreshes the prompt.
  - If the network is still down, that refresh replaces "Outcome unknown. Check the transcript before you send again" with "The Mac is not reachable".
  - The daemon still refuses a second key on the same prompt, so this gives a misleading message, not a double approval.
  - The fix check of the first review found this, and it was carried over to here.
- **Scores:** requirement miss against the first review's fix 1. Likelihood 3, severity 3.
- **Fix:** an error from the refresh after an action must not replace the action's note.
- **Scope:** in scope.

## 4. Known issues, accepted

**4. "Forget this Mac" leaves the old event stream and background alerts running.**
- **Where:** `android/app/src/main/java/dev/herdrremote/HerdrRemoteApp.kt:28-32`, and `android/app/src/main/java/dev/herdrremote/watch/AgentsRepository.kt:43-47` and `:79-82`.
- **Evidence:**
  - `forgetMac` cancels the loop, but the blocking read keeps the stream open, because the keepalive every 10 s prevents the timeout.
  - Events refill the agent list, and the watch service keeps notifying.
  - An action toward a newly paired Mac still goes through that Mac's own guard.
- **Scores:** likelihood 4, severity 5, score 20.
- **Why it is listed here:** the severity is below the bar you set. The fix is small: cancel the stream's `Call`, as the live view now does, and stop the watch service before clearing the pairing. Select it if you want it.

**5. The title line above the prompt has no line limit.**
- **Where:** `android/app/src/main/java/dev/herdrremote/ui/AgentScreen.kt:65`.
- **Evidence:**
  - The title shows the agent's one-line status summary (`msg`), which the daemon passes through uncut (`daemon/state.go:343`).
  - In the captured fixtures, the longest `msg` is a line of 120 characters.
  - On the emulator, a probe with a 45-line heredoc still left Approve and Deny at mid-screen.
- **Scores:** likelihood 1, severity 6.
- **Fix:** optional insurance. Give the title `maxLines = 3` with an ellipsis, because the bounded block below it shows the full command.

## 5. Dismissed

None. Both reviewers accepted all three findings after the code was checked.

## 6. Plan amendments

- **Section 3.7, the app gate:** one observer on the process lifecycle, owned by the application, not by the activity.
- **Section 3.7, the prompt card:** the prompt text has a bounded height and scrolls vertically. Approve and Deny are always visible.

## 7. Still open before release

Unchanged from `review.md` section 7. In addition, rerun `scripts/install.sh` so that the installed daemon gets the fixes. It still runs the build from before the first review's fixes.

## 8. Status

- **Selected:** fixes 1 to 3, and known issue 4.
- **Applied:** four commits on `main`, from `429c512` to `fca5008`:
  - `e0dabd6`: the app lock's observer moves to the application, with `GateTest`.
  - `e96ec0b`: the prompt block has a bounded height and scrolls vertically.
  - `f1d3a40`: the refresh after a decision keeps the action's note, with a test.
  - `fca5008`: "Forget this Mac" cancels the event stream, stops the watch service, and clears its notification.

  Android files only. No attribution lines, and nothing under `.agents/`.
- **Tests:** `./gradlew testDebugUnitTest assembleDebug` passes, with 19 JVM tests. The daemon is unchanged since `429c512`, where `go test ./...` passed.
- **Emulator checks during the fixes, against a scratch daemon:**
  - After a swipe from recents with watching on and more than 5 minutes away, the app asked for the PIN before it showed anything.
  - With a 45-line heredoc prompt, Approve and Deny stayed reachable.
  - With the network off, "Outcome unknown" stayed after a failed refresh.
  - After "Forget this Mac", the stream ended and no further alert came.
- **Fix check by the reviewers:**
  - Fixes 1, 3 and 4 are fixed.
  - Fix 2 is fixed on the evidence. The unbounded title line above the block is recorded as known issue 5, because the emulator evidence shows that it does not push the buttons out.
- **Closing check against `TDD.md` section 4, at `fca5008`:**
  - Criteria 1, 3, 4, 6, 7, 9, 11 and 13 are delivered in code and verified by tests or on the emulator.
  - Criteria 2, 5, 8 and 12 are delivered in code. Their manual parts are still open: Allow with a real phone, resize and heavy output in the live view, and the alert and lock screen on a real phone.
  - Criterion 10 (Tailscale) is not verified.
- **The installed daemon** still runs the build from before both reviews' fixes. Rerun `scripts/install.sh` to update it.

outcome: done

# Implementation verification (2026-11-08)

Implementer: herdr-remote-impl (Claude Fable 5.1), turn 07:21:22Z to 09:13:20Z (1 h 52 min). Report: implement/report.md, STATUS: done. Plan: /Users/zeus/code/herdr-remote/TDD.md (As built appended).

## Checked by the orchestrator

1. **Working tree.** `git status --porcelain` shows only `?? .agents/`, `?? DEBATE.md`, `?? PRD.md`, matching before.md (TDD.md is now committed). Pass.
2. **Commits.** 13 commits from the unborn base to `a334510`, one per step 1 to 11, one fix (`649c88f`), and the As built note. All conventional. No attribution lines (the only "Claude" match is the subject "transcripts from Claude sessions"). Nothing under `.agents/`, no PRD.md or DEBATE.md. 81 tracked files in daemon/, android/, mod/, scripts/, testdata/, TDD.md, .gitignore. No key, env, keystore or local.properties file tracked. Pass.
3. **Tests run by the orchestrator.**
   - `cd daemon && go vet ./...`: clean.
   - `cd daemon && go test ./...`: `ok herdr-remote/daemon 85.540s`.
   - `cd android && ./gradlew testDebugUnitTest assembleDebug` (JAVA_HOME openjdk@17): exit 0, 11 JVM tests in 4 classes, 0 failures; `app-debug.apk` 34.7 MB.
   - `./gradlew connectedDebugAndroidTest` on the `herdr-remote` AVD, booted headless by the orchestrator and shut down after: 2 tests, `liveWebViewRefusesNetworkUrls` passed, `pairsThenPinDecidesTheHandshake` skipped (needs a live pairing link from the manual harness; the implementer reports it passed with the harness; not re-run here).
4. **As built note.** Present at the end of TDD.md, covers all 12 steps, the re-verification of section 2, captured event shapes, assumptions, deviations, checks run and checks left for the user. Pass.
5. **Installed environment.**
   - `launchctl print gui/501/dev.herdr-remote`: running, pid 59292, program `~/.local/bin/herdr-remoted`.
   - Listening: `*:7766` only; 7767 closed.
   - State dir `~/Library/Application Support/herdr-remote` is 0700; `server.key`, `audit.log`, `daemon.log`, `admin.sock` 0600; `server.crt` 0644; no `devices.json` (no phone paired).
   - `audit.log` holds 2 lines (`enroll`, `deny`) with keys only `action`, `state`, `ts`: no token or code.
   - `daemon.log` holds handshake refusals from the checks and the start line with the public pin.
   - Mod at `~/.config/herdr-app/mods/herdr-remote/` with `.herdr-remote-owned`. No other file under `~/.config/herdr-app` changed apart from Herdr's own data/telemetry; `shared.json` and other mods untouched.
   - No `hr-probe-*` terminals, scratch daemons or emulators left. Two idle Gradle daemons from the build (13:10) remain; they stop on their own.
   - Homebrew: `gradle 9.8.2` installed and pulled `openjdk 27.0.1.0.3` plus dependency upgrades; `android-commandlinetools` cask; SDK under `/opt/homebrew/share/android-commandlinetools`.
6. **Step checks.** The retained transcript (1081 lines) is truncated; it shows two earlier JVM runs with "11 tests completed, 1 failed", later fixed (the orchestrator's run passes 11 of 11). The rest of the manual checks are taken from the As built note, cross-checked against the audit log and the installed state where possible.

## Discrepancies and notes for the review

- **Criterion 12 vs Android behaviour (plan drift).** With the Android default (sensitive content allowed on the lock screen), a `VISIBILITY_PRIVATE` notification shows its private text on the lock screen: the tab name and "permission". The command never appears. TDD decision 14 implies this; criterion 12 ("shows only 'An agent needs attention'") overstates it. Either change the criterion or make the private text generic too.
- **Commit `0d14b24` does not build alone** (Android steps 9 and 10 split by file after the fact). History is not bisectable at that point.
- **Additions not in the plan:** `pair --json` also returns `text` (the QR content); the app has a "paste the pairing link" field; the mod accepts `herdr-remote.pair`, `.allow`, `.deny`, `.cancel` through `herdr-app do`. All stay inside the same-user trust boundary (decision 18), but `.allow` lets any local process allow a pending phone, as `herdr-remoted pair allow` already does.
- **Step 8 "60 s then Allow" ended in Deny** by the implementer's choice, so no real phone was paired with the user's daemon.
- **Checks left for the user** (from As built): Allow for a real phone; an overlay screenshot (the window drew no new frame during the run); real phone install and camera scan; Tailscale over cellular; resize and heavy output in the Live tab; notifications-denied toggle; swipe-away alert; recents card; macOS firewall prompt; Codex launcher `--no-daemon`; LaunchAgent after a Mac reboot; final audit/daemon log grep.

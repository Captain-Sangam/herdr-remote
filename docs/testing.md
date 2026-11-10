# Testing

This page tells how herdr-remote is tested. It covers the unit tests, the emulator tests, the probe agents and the checks on a real phone over `adb`.

## Test layers

| Layer            | Where                                | Runs                                       |
| ---------------- | ------------------------------------ | ------------------------------------------ |
| Go tests         | `daemon/*_test.go`                   | `go test ./...`, with a fake `hc`          |
| JVM tests        | `android/app/src/test/`              | `./gradlew testDebugUnitTest`              |
| Emulator tests   | `android/app/src/androidTest/`       | `./gradlew connectedDebugAndroidTest`      |
| Manual harness   | `daemon/manual_test.go`              | A scratch daemon against the real `hc`     |
| Probe agents     | Throwaway Claude and Codex terminals | Real prompts for approve, deny and markers |
| On-device checks | A real phone over `adb`              | The acceptance criteria, by hand           |
| Format check     | `scripts/format.sh --check`          | Local, the pre-commit hook and CI          |

## Commands

Run these before each commit:

```sh
(cd daemon && go test ./...)
(cd android && JAVA_HOME=/opt/homebrew/opt/openjdk@17 \
  ANDROID_HOME=/opt/homebrew/share/android-commandlinetools \
  ./gradlew testDebugUnitTest assembleDebug)
scripts/format.sh --check
```

- `go test ./...` takes about 80 s, because one test waits 60 s on the enrollment long-poll.
- Tests that need a real core use their own `HC_SESSION` and scratch XDG paths. They never use the user's core.

## Go tests

The tests use a fake `hc` binary that records each request, and a fake `ioreg`. The fixtures are in `daemon/testdata/`:

- `events-probe.jsonl`: hc events captured from probe agents.
- `screen-claude-cmd.txt`, `screen-claude-edit.txt`, `screen-codex-cmd.txt`, `screen-codex-edit.txt`: the prompt screens.

`testdata/pairing-vectors.json` at the repo root holds the 6-digit code vectors. The Go and the JVM tests both use it.

| Area                   | What the tests prove                                                                                                                                                                                               |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Subprocess helper, hc  | No method outside the list, no `force: true`, no `caller`, no other key, no `machine/` target, no program outside the helper list, a bare environment.                                                             |
| TLS and pairing        | Enrollment through the real 7767 listener, also with Allow after 60 s. Deny, cancel and expiry store nothing. A second key is refused. 7767 is closed outside a window.                                            |
| Revocation             | An unpaired key fails on 7766. After `devices rm`, a resumed session fails. Streams end within 2 s. A revoke that fails to save still revokes.                                                                     |
| Read API and events    | Hello, agents, spaces from `herdr-app ls`, a space rename, SSE, `: ka` within 11 s, `agent_removed`, 503 when hc is lost, and recovery.                                                                            |
| Live view              | The subprocess ends after a disconnect, a size event and a revoke. The limit of 2 live views.                                                                                                                      |
| Transcripts            | Claude and Codex parse. A partial last line, a record larger than a page, three pages back with no loss. A stale cursor. A symlink out of the roots. The guardian rollout. Daemon-mode Codex and the fallbacks.    |
| Approvals and messages | Each guard presses no key. Two decisions press one key. A repeated request ID, an expired ID, the rate limit. A restart, a failed audit write, an `ioreg` failure. `mac_in_use` and `uncertain`. Message cleaning. |
| Audit                  | After a full pairing, `audit.log` has no token and no code.                                                                                                                                                        |

## JVM tests

| Test           | What it proves                                                                                                    |
| -------------- | ----------------------------------------------------------------------------------------------------------------- |
| `PairingTest`  | The pairing link parse, refusals for a bad version, an expired link and missing fields, the code vectors, UUIDv7. |
| `PinTest`      | The pin in the trust manager and in the hostname verifier. The gate after 5 minutes.                              |
| `GateTest`     | The app lock at start, after 5 minutes away, and its one process-lifecycle observer.                              |
| `ReducerTest`  | Snapshot, change, `agent_removed` and missed. Blocked first. One notification for each episode.                   |
| `SentMarkTest` | A refused connection was never sent. A request with no reply may have arrived.                                    |
| `TextsTest`    | The text of each result and reason, the note after a refresh, the note kinds, Wi-Fi or Tailscale endpoints.       |

## Emulator tests

The AVD is `herdr-remote` (API 35, 3-button navigation on).

1. Start the emulator:

   ```sh
   emulator -avd herdr-remote -no-window &
   ```

2. Start the manual harness with auto-allow (see [Manual harness](#manual-harness)).
3. Run the tests, with the pairing link from the harness daemon:

   ```sh
   cd android && ./gradlew connectedDebugAndroidTest \
     -Pandroid.testInstrumentationRunnerArguments.pairText='<link>'
   ```

| Test                  | What it proves                                                                                                                                            |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `PairingEmulatorTest` | Pairing with the dev daemon at `10.0.2.2`, with the code known before Allow. A certificate name that does not match connects. A wrong pin never connects. |
| `WebViewTest`         | The live WebView refuses network, file and content URLs and serves only the bundled assets.                                                               |

- `PairingEmulatorTest` skips itself if `pairText` is not given.
- To keep the emulator app paired after the test, add `-Pandroid.testInstrumentationRunnerArguments.keep=1` and `-Pandroid.injected.androidTest.leaveApksInstalledAfterRun=true`.

## Manual harness

`TestManualServe` in `daemon/manual_test.go` runs a scratch daemon for manual checks. It skips itself unless `HR_MANUAL_DIR` is set.

```sh
cd daemon
HR_MANUAL_DIR=/tmp/hr go test -run TestManualServe -count=1 -timeout 3h .
curl --cert /tmp/hr/client.crt --key /tmp/hr/client.key -k https://127.0.0.1:17766/v1/agents
```

- It runs a daemon on ports 17766 and 17767 against the real `hc` and `herdr-app`, with a scratch state directory.
- It pairs a generated phone key and writes `client.crt`, `client.key` and `port` into `HR_MANUAL_DIR`.
- It serves until a file named `stop` appears in that directory.
- `HR_MANUAL_AUTOALLOW=1` allows every pending phone, for the emulator tests.
- `HR_MANUAL_PAGE` sets the transcript page size in bytes.

## Probe agents

Use probe agents to make real prompts. Use a scratch folder. Remove each probe after use with `hc kill` and `hc rm`.

- **Claude command:** `hc run --bg --name hr-probe-claude --cwd <repo> -- claude --permission-mode default "Use Bash to run: touch <scratch>/ok"`.
- **Claude edit:** the same, with "Use the Write tool to create <scratch>/edit.txt containing hi".
- **Codex command:** `hc run --bg --name hr-probe-codex --cwd <repo> -- codex --no-daemon -s read-only -a on-request "Run: touch <scratch>/ok"`.
- **Codex edit:** the same, with "Use apply_patch to create <scratch>/edit.txt containing hi".
- **A plain agent:** a terminal shows on the phone as an agent if it prints `ESC ] 7501 ; state=idle BEL`, for example `printf '\033]7501;state=idle\007'`.

[Interfaces](interfaces.md) lists what each prompt must show.

## Acceptance criteria

The product is done when these 13 criteria pass on the emulator and on a real phone.

1. `herdr-remoted` runs under a LaunchAgent and starts again after a reboot. A TLS client without a certificate, or with an unpaired one, fails the handshake on 7766. Port 7767 is closed outside a pairing window.
2. The overlay shows the paired devices. **Pair a phone** shows a QR code.
   - A scan shows a 6-digit code on the phone that matches the Mac.
   - After Allow, the phone is paired.
   - Deny, no Allow, or 120 s stores nothing.
   - A second phone with the same token is refused.
3. The app lists spaces with their Herdr names. A space lists its agents with status. The all-agents view lists every local agent. A status change on the Mac shows on the phone within 2 s.
4. The transcript shows user and assistant messages and tool calls, and pages back across at least three pages. This is true for a Claude Code agent and for a Codex agent with `--no-daemon`. Other agents show labelled terminal text. A Codex in daemon mode also shows the `--no-daemon` hint.
5. The live view shows the terminal in color and follows new output within about 1 s on the LAN. It recovers after a pane resize on the Mac, after heavy output, and after a network change.
6. The user can approve or deny a blocked Claude Code or Codex permission prompt, for a command or a file edit, from the phone. The agent then continues. The daemon presses no key, and the phone says why, in these cases:
   - the prompt changed
   - someone answered it on the Mac
   - another phone or a retry already answered it
   - the Mac had keyboard or mouse input within `mac_idle_seconds` (a heuristic that does not see SSH or input from other agents)
   - the daemon restarted during the action (the result is `outcome_unknown`).
7. A message from the phone reaches an idle or working agent as a new, untagged prompt. The daemon sends it at most once for each request, also after a retry or a daemon restart. If the outcome is unknown, the phone says so and does not send again. A blocked agent refuses the message, with a reason.
8. With background watching on and the app closed, a newly blocked agent causes one notification within 5 s, while the service runs and is connected. Watching starts again after a phone reboot and an app update.
9. A revoke on the Mac ends the open streams of that phone within 2 s. Later handshakes fail, also when the phone tries to resume a TLS session.
10. The same pairing works over Tailscale from a cellular connection.
11. After 5 minutes in the background, the app asks for a biometric or the device credential before it shows anything.
12. The recents screen shows no app content. The lock screen and the notification show only "An agent needs attention".
13. `audit.log` has a line for every pairing event, revocation, decision and message. It contains no token, code, transcript text, terminal bytes or message text.

### Check for each criterion

| Criterion | Check                                                                                                                                                                |
| --------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1         | Go tests. `openssl s_client -connect 127.0.0.1:7766` without a certificate fails. `nc -z 127.0.0.1 7767` fails outside a window. `launchctl print` after a reboot.   |
| 2         | Go tests. Manual pairing: with Allow, with Deny, with a timeout, with 60 s before the scan, and with 60 s before Allow.                                              |
| 3         | Go tests. Manual: change the status of an agent and time it on the phone.                                                                                            |
| 4         | Go tests. Manual: a Claude agent. A `codex --no-daemon` agent. A plain `codex` agent (terminal text and the hint). An agent before its first prompt (terminal text). |
| 5         | Go tests. Manual: resize, heavy output (`yes \| head -c 50000000` in a watched terminal), network change.                                                            |
| 6         | Go tests. Probe agents: approve and deny each prompt type. Answer one on the Mac first, then tap on the phone. Tap while you type on the Mac (`mac_in_use`).         |
| 7         | Go tests. Manual: send a message to an idle probe, and check that the prompt has no `<hc` tag.                                                                       |
| 8         | Manual: swipe the app away, block a probe agent, and look for one notification. Run `adb reboot`, then block a probe agent again.                                    |
| 9         | Go tests for resumption and streams. Manual: revoke during a live view.                                                                                              |
| 10        | Manual over Tailscale on cellular, with Wi-Fi off.                                                                                                                   |
| 11        | JVM gate tests. Manual: a cold start and 5 minutes away.                                                                                                             |
| 12        | Manual: lock the phone and read the notification. Open recents and look at the card.                                                                                 |
| 13        | Go audit test. Manual `grep` of `audit.log` and `daemon.log` after the full run.                                                                                     |

## Checks on a real phone

### Status on 2026-11-10

The phone is a Samsung S26 (SM-S942B), Android 17, 3-button navigation, dark mode, font scale 0.9. It runs the latest APK, paired as `SM-S942B`, with "Notify me" on.

| Check                                                                                                                                             | Result |
| ------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| Layout at font scale 0.9 and 1.3, with the keyboard open: nothing under the navigation bar, no tap target under 48 dp.                            | Pass   |
| A status change shows in about 1.2 s.                                                                                                             | Pass   |
| Transcripts for Claude and for `--no-daemon` Codex.                                                                                               | Pass   |
| The live view: in color, follows a pane resize, recovers after 50 MB of output.                                                                   | Pass   |
| Approve, deny and a message: applied on the Mac and in `audit.log`.                                                                               | Pass   |
| A tap while the Mac is in use gives "Someone is using the Mac" (the limit is 10 s of idle).                                                       | Pass   |
| "Notify me": the two prompts, no crash.                                                                                                           | Pass   |
| An alert 3 to 4 s after a block, once, with only "An agent needs attention".                                                                      | Pass   |
| The recents card is blank.                                                                                                                        | Pass   |
| The app asks for the fingerprint on a cold start and after 5 minutes away.                                                                        | Pass   |
| After a phone reboot, the watcher starts by itself and alerts come.                                                                               | Pass   |
| Revoke: the Mac drops the phone in 0.21 s. Pairing again works.                                                                                   | Pass   |
| Wi-Fi off: the app connects over Tailscale.                                                                                                       | Pass   |
| After a Mac reboot, the LaunchAgent runs, and `audit.log` holds no secrets or content. See [operations](operations.md#checks-after-a-mac-reboot). | Open   |
| The screens in light mode, by eye. FLAG_SECURE blocked a check of the colours.                                                                    | Open   |

### How the phone was tested

The checks used `adb` from a Mac with the phone on USB.

- **Screens:** `uiautomator dump` gives the view tree with the bounds of each element. It shows the layout, the text and the tap target sizes, also when screenshots are black.
- **Live view:** the WebView debugging port shows the xterm.js page. A debug build only, because Android turns on WebView debugging only for a debuggable app.
- **Screenshots:** FLAG_SECURE blocks screenshots of the app, and the recents card is blank. This is by design. Do not turn FLAG_SECURE off in any build.
- **Agents:** probe agents on the Mac, or a plain terminal that prints `ESC ] 7501 ; state=idle BEL`.
- **The emulator** was paired once to reproduce a crash, then revoked on the Mac. Its app still thinks that it is paired.

Useful commands. `adb` is in `$ANDROID_HOME/platform-tools/`, which is not on the `PATH` by default:

| Task                   | Command                                                                |
| ---------------------- | ---------------------------------------------------------------------- |
| List devices           | `adb devices`                                                          |
| Install the APK        | `adb install -r app/build/outputs/apk/debug/app-debug.apk`             |
| Dump the screen layout | `adb shell uiautomator dump /sdcard/ui.xml && adb pull /sdcard/ui.xml` |
| Set the font scale     | `adb shell settings put system font_scale 1.3`                         |
| Dark or light mode     | `adb shell cmd uimode night yes`, or `no`                              |
| Reboot the phone       | `adb reboot`                                                           |
| Inspect the live view  | Open `chrome://inspect` in Chrome on the Mac, then pick the WebView.   |

## Format check and CI

- `scripts/format.sh` formats Markdown, JSON, YAML, JavaScript, HTML and CSS with Prettier, Go with `gofmt`, Kotlin with ktfmt and shell with shfmt. The tool versions are pinned.
- `scripts/format.sh --check` changes nothing. It lists unformatted files and exits 1 if there are any.
- `.githooks/pre-commit` formats the staged files. Turn it on once per clone with `git config core.hooksPath .githooks`. It refuses a file that also has unstaged changes.
- `.github/workflows/ci.yaml` runs `scripts/format.sh --check` on each push and pull request. The repo has no remote yet, so it has not run.
- `.prettierignore` lists the paths that no tool formats.

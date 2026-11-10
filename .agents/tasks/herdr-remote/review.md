# Review: herdr-remote

## 1. Reviewed

- **Repo:** `/Users/zeus/code/herdr-remote`, branch `main`.
- **Range:** from the empty tree (`4b825dc642cb6eb9a060e54bf8d69288fbee4904`) to `f858d27` (tag `checkpoint-pre-review`). The whole repo is new code.
- **Plan:** `TDD.md` at the repo root, with its "As built (2026-11-08)" section.
- **Tests run before the review:**
  - `go vet ./...` is clean, and `go test ./...` passes.
  - `./gradlew testDebugUnitTest assembleDebug` passes, with 11 JVM tests.
  - On the emulator, the WebView test passes. The pairing test skips without a live pairing link.

## 2. Completeness

- **Delivered:**
  - steps 1, 2, 4, 5, 8
  - criteria 1, 2, 8 (in code), 10 (in code), 13.
- **Partial for a code reason.** Each row maps to a fix below:
  - step 3 and criterion 9: a revoke that fails to save (fix 6)
  - step 6 and criterion 4: the transcript fallback (fix 5)
  - step 7 and criterion 6: the queued key (fix 7)
  - step 9 and criterion 11: the app lock (fix 8)
  - step 9 and criterion 3: the `/v1/hello` refresh (fix 3)
  - step 10 and criterion 12: the lock-screen text (fix 2)
  - step 11 and criteria 5 and 7: the action retries and timeouts (fix 1), and the live-view streams (fix 4).
- **Drifted:**
  - The overlay also accepts `herdr-app do herdr-remote.pair`, `.allow`, `.deny` and `.cancel`.
  - `pair --json` also returns the QR text, and the app has a "paste the pairing link" field.
  - These stay inside the same-user trust boundary (decision 18) and are recorded in As built. No change is needed.
- **Not verified here:** step 12 and every check that needs a physical phone, Tailscale or a GUI dialog. Section 7 lists them.
- **Changes that no step explains:** none, apart from the test harness `daemon/manual_test.go`. It runs only with `HR_MANUAL_DIR` set.

## 3. Findings to fix

Requirement misses come first, then the rest by score. The score is likelihood times severity. The fix threshold is 12.

**1. The app retries approvals and messages by itself, and reports an unclear result as "not reachable".**
- **Where:**
  - `android/app/src/main/java/dev/herdrremote/net/Api.kt:47`, `:67-82` and `:119-125`
  - `android/app/src/main/java/dev/herdrremote/ui/AgentViewModel.kt:61`
  - `daemon/actions.go:156-161`
- **Evidence:**
  - `decide` and `message` run through `withEndpoint`. After an `IOException` it posts the same body to the next address of the Mac.
  - The daemon forgets a refused request ID. So a refusal whose response is lost (for example `mac_in_use`) is evaluated again on the second address, and can press a key without a new tap.
  - The action read timeout is 15 s. The daemon may take up to 20 s for a key and 60 s for a message, because hc waits until nobody types at the Mac.
  - Every such timeout becomes "The Mac is not reachable". A user who then sends again creates a new request ID, and the agent gets the prompt twice.
  - Section 3.7 of the plan says "The app never retries an action by itself" and "Outcome unknown. Check the transcript before you send again".
- **Scores:** requirement miss. Likelihood 5, severity 8.
- **Fix:**
  - An action may move to the next endpoint only when the connection failed before the request was sent: connect refused, unknown host, or a timeout before the TLS handshake ended.
  - After the request may have been sent, there is no automatic retry.
  - An `ApiException` (an explicit reply from the daemon) keeps its reason.
  - Any other failure shows "Outcome unknown. Check the transcript before you send again".
  - The action read timeout becomes 150 s.
- **Rejected:** a "Check again" button that re-posts the same request ID. It is useful, but the plan does not ask for it.
- **Scope:** in scope.

**2. The lock screen shows the tab name.**
- **Where:** `android/app/src/main/java/dev/herdrremote/watch/WatchService.kt:94-96`.
- **Evidence:** the private version of the notification has the tab name and "permission". Android shows the private version on the lock screen under its default setting ("show sensitive content"). Criterion 12 and PRD SR-11 say the lock screen shows only "An agent needs attention". The plan's section 3.7 contradicts its own criterion.
- **Scores:** requirement miss. Likelihood 10, severity 6.
- **Fix:** make the private and the public text both "An agent needs attention". Show the tab name only inside the app. Amend decision 14 and section 3.7.
- **Scope:** in scope.

**3. The app never refreshes the clock offset or the endpoint list.**
- **Where:** `android/app/src/main/java/dev/herdrremote/ui/PairScreen.kt:83` and `android/app/src/main/java/dev/herdrremote/net/Api.kt:117`.
- **Evidence:**
  - `/v1/hello` runs once, at pairing.
  - A phone clock that drifts more than 5 minutes gets `expired_request` on every action, and nothing resyncs it.
  - A Mac that gets a new address is reachable only through the old list.
  - Step 9 says the app refreshes the endpoints from `/v1/hello`.
- **Scores:** requirement miss. Likelihood 3, severity 4.
- **Fix:**
  - Call `/v1/hello` on each SSE connect and reconnect. Run it alongside the event stream, so it never delays an alert.
  - Call it again after an `expired_request`, without an automatic re-post.
  - Keep the rule that the last endpoint that worked comes first.
- **Scope:** in scope.

**4. The live view never closes a stream that the user leaves.**
- **Where:** `android/app/src/main/java/dev/herdrremote/ui/LiveView.kt:76-83` and `:107-128`.
- **Evidence:**
  - `stop()` and `reopen()` cancel coroutines. But the stream read is a blocking call with no read timeout, so it does not stop, and the response is not closed.
  - Each idle view that the user leaves keeps its `hc attach` on the Mac.
  - After two such views, the daemon refuses new live views with `live_limit`.
- **Scores:** likelihood 8, severity 6, score 48.
- **Fix:**
  - Keep the OkHttp `Call`, and cancel it in `stop()` and in `reopen()`.
  - A cancelled stream exits and does not try the next endpoint.
  - One `live_limit` refusal right after a cancel is acceptable, because the retry backoff absorbs it.
- **Scope:** in scope.

**5. A transcript that cannot be read shows an error, or an empty page, instead of terminal text.**
- **Where:** `daemon/transcript.go:471-500`.
- **Evidence:**
  - If the resolved file cannot be read, `filePage` returns an error.
  - If the records do not parse, the page is empty.
  - Step 6 says that a source that cannot resolve passes to terminal text.
- **Scores:** likelihood 4, severity 4, score 16.
- **Fix:**
  - Fall back to terminal text on a read error, or when the file gives no record that parses.
  - On the first request, page back until there is at least one item or the start of the file.
  - Valid pages with no items (for example only metadata records) keep their cursor.
- **Scope:** in scope.

**6. A revoke that fails to save leaves the phone connected, and lets it come back.**
- **Where:** `daemon/devices.go:91-100` and `daemon/daemon.go:176-191`.
- **Evidence:**
  - `remove` deletes the device from memory, then writes the file. If the write fails, `revoke` returns before it closes the connections and the live views.
  - A retry of `devices rm` cannot find the device anymore.
  - A daemon restart loads the old file, with the device.
- **Scores:** likelihood 2, severity 8, score 16.
- **Fix:**
  - Mark the device as revoked in memory first. The handshake check and the per-request check then treat it as unknown.
  - Close its connections and live views.
  - Then write the file without the device. On success, drop the mark.
  - On failure, keep the mark and return the error, so a retry of `devices rm` finds the device and writes again.
- **Rejected:** write the file first, then change memory. If the write fails, the phone stays connectable.
- **Scope:** in scope.

**7. An approval key can wait in hc after the daemon gave up, and then skip the Mac-in-use check.**
- **Where:** `daemon/actions.go:441-461`.
- **Evidence:**
  - The daemon waits 20 s for `terminal.keys`. But hc holds the key while someone types at the Mac, and presses it when the Mac is quiet, also after the daemon stopped waiting.
  - In that case the daemon records `outcome_unknown` and never runs the idle check after the key, so the key can land on a later prompt.
- **Scores:** likelihood 2, severity 7, score 14.
- **Fix:**
  - Wait up to 120 s for hc's reply, so the idle check runs. The per-terminal lock is held that long, which is acceptable.
  - The phone's 150 s action timeout (fix 1) covers this wait.
  - Amend the plan's section 7 to name this queued-key window.
- **Rejected:** keep waiting for a late reply without a limit, to run the check whenever it arrives. That holds a goroutine and the terminal lock without bound, for a case that needs more than 120 s of continuous typing.
- **Scope:** in scope.

**8. The app lock opens on any error from the biometric check.**
- **Where:** `android/app/src/main/java/dev/herdrremote/MainActivity.kt:55-65`.
- **Evidence:** every result of `canAuthenticate` other than success unlocks the app. The plan allows this only for a phone with no screen lock (`BIOMETRIC_ERROR_NONE_ENROLLED`).
- **Scores:** likelihood 2, severity 6, score 12.
- **Fix:** unlock only on `BIOMETRIC_ERROR_NONE_ENROLLED`. On any other result, stay locked, and show the error with a retry.
- **Scope:** in scope.

## 4. Known issues, accepted

These are below the fix threshold. They are recorded, not fixed.

9. **A reused process ID can map an agent to an old Claude session.** A stale `~/.claude/sessions/<pid>.json` and a reused pid give the old transcript. Likelihood 1, severity 5. A fix would check that the file's `cwd` equals the terminal's cwd.
10. **A duplicate request in flight gets an empty reply.** If the first copy is refused, the duplicate gets `200` with no body (`daemon/actions.go:156-172`). Likelihood 2, severity 3. Fix 1 removes the main way this happens.
11. **A stalled enrollment connection can outlive the pairing window.** It needs an attacker on the LAN during a window that the user opened (`daemon/pair.go:85-90`, `:205-218`). Likelihood 2, severity 4.
12. **An unpaired client can write refusal lines to the audit log during a pairing window.** Likelihood 2, severity 3.
13. **The live handler waits for the process before it cancels it** (`daemon/live.go:105-127`). The process context comes from the request, so it ends when the client disconnects. Likelihood 1, severity 3.
14. **`install.sh` loads the LaunchAgent before it checks the mod marker.** A stop there leaves the daemon installed. Likelihood 1, severity 2.
15. **A retry of a decision after the terminal ended gets 404**, not the recorded result. The target check runs before the request ID lookup. Likelihood 2, severity 2.
16. **An open space screen keeps the agent list that it had when it opened** (`ListScreens.kt:106`). A new agent shows only after the user goes back. Status changes still show. Likelihood 5, severity 2.
17. **Commit `0d14b24` does not build alone.** The Android files were split across commits after the build. This affects history only.

## 5. Dismissed

18. **"An event stream survives a revoke over HTTP/2."** This claim is wrong. The net/http server skips its own state hooks when it hands a connection to HTTP/2 (`net/http/server.go:1996-2000`, Go 1.23). But the HTTP/2 server then calls the same `ConnState` hook with the TLS connection (`net/http/h2_bundle.go:4570-4574`, called at `:4796` and `:6011`). So `connRegistry.onState` records HTTP/2 connections, and a revoke closes them.
19. **"Pairing cancel and expiry are not audited."** Section 3.6 of the plan does not list them.

## 6. Plan amendments

- **Decision 5 and section 3.7:**
  - The action read timeout is 150 s.
  - An action may change endpoints only if the connection failed before the request was sent.
  - After that, the app never retries. Any failure that is not an explicit daemon reply shows "Outcome unknown".
- **Decision 7:** a revoke marks the device as revoked in memory before it writes the file. A failed write keeps the mark and returns the error.
- **Decision 14, section 3.7 and criterion 12:** the notification text is "An agent needs attention" in both the private and the public version. The tab name appears only in the app.
- **Section 3.5:** the daemon waits up to 120 s for `terminal.keys`.
- **Section 3.6, transcript rules:**
  - A read error, or a file with no record that parses, falls back to terminal text.
  - The first request pages back until it has an item.
- **Section 3.7:** the app calls `/v1/hello` on each SSE connect and after `expired_request`.
- **Section 7:** name the queued-key window. hc can hold a key for up to 120 s while someone types at the Mac.

## 7. Still open before release

These need a person, a physical phone, Tailscale or a GUI dialog. They come from the As built note:

- Pair a real phone and click Allow. Look at the overlay in the Herdr window.
- Install the APK on a phone, and scan the QR code with the camera.
- Install Tailscale on the Mac and the phone, and test over cellular.
- In the live view: resize the pane while an agent prints, and send heavy output through it.
- Deny notifications, swipe the app away, and check the recents card and the lock screen.
- Allow incoming connections for `herdr-remoted` when the macOS firewall asks.
- Set Herdr's Codex launcher to `["codex", "--no-daemon"]`.
- Restart the Mac, then check the LaunchAgent.
- Check `audit.log` and `daemon.log` after a full run.

## 8. Status

- **Selected:** fixes 1 to 8.
- **Applied:** eight commits on `main`, from `f858d27` to `429c512`, one for each fix:
  - `05423de` fix 7
  - `62562af` fix 6
  - `05f8ce8` fix 5
  - `b360936` fix 1
  - `c8682c0` fix 3
  - `cb5e759` fix 2
  - `0e367cb` fix 8
  - `429c512` fix 4

  Each commit touches only the files of its fix, has no attribution lines, and nothing under `.agents/`.
- **Tests after the fixes:**
  - `go vet ./...` is clean, and `go test ./...` passes.
  - `./gradlew testDebugUnitTest assembleDebug` passes, with 15 JVM tests (up from 11).
- **Fix check by the reviewers:** fixes 2 to 8 are fixed.
- **Fix 1 is partly fixed.** Decisions and messages no longer retry by themselves, and an unclear send shows "Outcome unknown". But after a decision, the app refreshes the prompt (`AgentViewModel.kt:149`). If the network is still down, that refresh replaces the "Outcome unknown" note with "The Mac is not reachable" (`AgentViewModel.kt:129`).
  - Messages do not refresh, so this affects decisions only.
  - A second tap is still refused by the daemon's guard, so the result is a misleading message, not a double key.
  - It is carried into the next fix turn.
- **Not done yet:**
  - The plan amendments in section 6 are not yet appended to `TDD.md`. They will be recorded with the second review.
  - The installed daemon still runs the build from before the fixes. Rerun `scripts/install.sh` to update it.
- **Checkpoint:** tag `checkpoint-fixes-applied` on `429c512`.

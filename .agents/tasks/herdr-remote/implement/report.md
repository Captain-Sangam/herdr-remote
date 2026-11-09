  IMPLEMENT herdr-remote-impl implement

  Section 2 re-verified first: versions, prompt markers and keys for both agents and both prompt types, attach stream to a pipe, transcript mappings, untagged send from a bare environment, bare-environment access, HIDIdleTime. No difference. Claude auto mode showed no prompt for touch.

  1. Toolchain: done. go version, adb version (full path), ./gradlew assembleDebug pass. Files: scripts/setup-android.sh, .gitignore, daemon/go.mod, android/ skeleton. SDK at the cask root /opt/homebrew/share/android-commandlinetools, AVD herdr-remote.
  2. hc client: done. go test passes: method allowlist, force:false, no caller, three keys only, machine/ and name targets refused, programs outside the list refused, idle parser. Files: daemon/run.go, hc.go, idle.go, fake_test.go, tests, testdata/events-probe.jsonl plus four prompt screens.
  3. Identity and pairing: done. Go tests incl. real 7767 listener, 60 s long-poll, deny/cancel/expiry store nothing, second SPKI refused, bad names, 7767 closed after the window, unpaired/RSA/no-cert handshakes fail, resumption after devices rm fails, audit has no token/code. openssl s_client without a cert: "alert certificate required". Files: tls.go,
     devices.go, pair.go, admin.go, audit.go, daemon.go, main.go, pair_test.go, testdata/pairing-vectors.json.
  4. Read API: done. Tests for snapshot/status/missed, space rename, : ka within 11 s, agent_removed, hc lost 503 and recovery, revoke ends SSE. Manual curl passed. Files: state.go, api.go, api_test.go, manual_test.go (harness).
  5. Live: done. Subprocess count returns to zero after disconnect, size event and revoke; third view 409 live_limit. Manual curl -N: 6 KiB colored bytes. Files: live.go, live_test.go.
  6. Transcripts: done. All ten listed cases pass. Manual on probes: claude, codex, daemon-mode hint, fresh agent, stale cursor 409. Files: transcript.go, transcript_test.go.
  7. Approvals and messages: done. All listed cases pass (stale/changed/markers/question/pi/machine, concurrency, same id, expired id, 429, restart with rotation, audit failure, ioreg failure, dropped-response retry, mac_in_use, uncertain, message cleaning, blocked refusal). Manual on probes through the API: approve/deny both prompt types both agents,
     answered-on-Mac first, untagged message, blocked refuses. Files: actions.go, actions_test.go.
  8. Mod and install: done. herdr-app check passes; marker stops/replaces; 7767 closed with overlay open; herdr-app do herdr-remote.pair opens a window; scan after 60 s reached pending with code; deny stored nothing; closing the overlay cancels; launchctl print running. Files: mod/herdr-remote/{mod.json,main.js}, scripts/install.sh.
  9. Android pairing: done. JVM tests (link parse, vectors, pin in TM and HV, gate); emulator test pairs with the scratch daemon at 10.0.2.2, code before Allow, hello with non-matching name, wrong pin fails.
  10. Android lists and service: done. Reducer and de-dup tests; on the emulator: spaces with real names, blocked first, alert at once in background, reboot restarts service and alerts, network off/on recovers in 5 s, PIN gate at cold start, lock screen never shows the command.
  11. Agent screen: done. WebView refuses network URLs (emulator test); approve from the phone applied; deny refused with mac_in_use while the user typed; live tab opens/ends one hc attach --observe.
  12. Acceptance: done where this Mac allows; the rest listed in As built.

  Deviations and assumptions: pair --json also returns text; the 60 s overlay check ended with Deny, not Allow (I do not grant a phone access to the user's daemon); Android commit 9 does not build alone (split by file); QR library skip2/go-qrcode; sub_agents in herdr-app ls is a string array; bare env for every subprocess; prompt hash NUL-joined; rate limit
  2/s; paste-link field in the app; code from a pinned handshake before the long-poll; lock-screen text depends on the phone's sensitive-content setting. Left for the user: Allow click and overlay screenshots (window drew no frame all afternoon), real phone, Tailscale, resize and heavy-output live checks, notifications-denied toggle text, recents card,
  firewall prompt, Codex launcher setting.

  Final commit: a334510

  STATUS: done

  END IMPLEMENT implement

⏺ ROUND 1 herdr-remote-r3

  Findings

  herdr-remote-r3-1 — daemon/daemon.go:115-126, daemon/daemon.go:209-238, android/.../net/Api.kt:39-48
  - Evidence: the paired server is started with ServeTLS and no TLSNextProto, so Go advertises h2. OkHttp's default protocols are h2 then http/1.1 and Clients.base never restricts them, so the phone negotiates h2. For an ALPN connection Go runs setState(StateActive, skipHooks) ($GOROOT/src/net/http/server.go:1994-1999), so connRegistry.onState never registers
    it and closeAll at daemon.go:188 closes nothing. After devices rm, the revoked phone's /v1/events stream keeps flowing, with every blocked msg (the command text), and : ka every 10 s stops the phone's 30 s timeout from ever firing. Live views still end via live.go:67. The Go tests pass only because pair_test.go:108 builds an http.Transport with
    TLSClientConfig, which is http/1.1 only.
  - Likelihood 9: every real phone connection.
  - Severity 8: criterion 9 and decision 7 fail for the stream that carries the most data; a lost phone keeps watching until it disconnects.
  - Label: implementation defect. Disposition: new.
  - Fix: set TLSNextProto to an empty map on both servers (or OkHttp .protocols(listOf(HTTP_1_1))), and recheck devices.bySPKI in handleEvents on each keepalive tick.

  herdr-remote-r3-2 — android/.../net/Api.kt:47,67-82, android/.../ui/AgentViewModel.kt:61, daemon/actions.go:441,546
  - Evidence: action calls read-time out at 15 s, but the daemon budgets 20 s for a key and 60 s for a send, both behind hc's "nobody typing for 3 s" wait. On timeout withEndpoint re-posts the same id to the next host (good), then throws UnreachableException, shown as "The Mac is not reachable." The plan (3.6, 3.7) requires "Outcome unknown. Check the
    transcript before you send again" and a 5-minute wait. A user who reads "not reachable" re-sends a message with a fresh id, and the agent gets the prompt twice.
  - Likelihood 4: any send or decision while someone types on the Mac for more than about 12 s.
  - Severity 5: duplicate prompt; for decisions the guard catches the retry.
  - Label: requirement miss. Disposition: new.
  - Fix: action read timeout above the daemon budgets (90 s), and map a failure after the body was sent to the outcome-unknown text.

  herdr-remote-r3-3 — daemon/actions.go:441-461
  - Evidence: the Keys context expires at 20 s, but hc still holds the queued key and presses it whenever the Mac goes quiet. The daemon records outcome_unknown, never runs the rule-8 idle check, and the key can land on a later prompt. Section 7 accepts a race of "milliseconds", not a 20 s queued key.
  - Likelihood 2: local input must start right after the idle check and continue 20 s.
  - Severity 7: a key on a prompt nobody saw, unaudited as uncertain.
  - Label: implementation defect. Disposition: new.
  - Fix: wait for hc's reply well past its quiet rule (120 s) so rule 8 runs; amend section 7 to name the queued-key window.

  herdr-remote-r3-4 — daemon/transcript.go:98-121
  - Evidence: source 1 maps foreground.pid to ~/.claude/sessions/<pid>.json with no check of that file's cwd or age against the terminal. A sessions file left by an exited Claude whose pid macOS reused for a new agent shows the old session as this agent's transcript, the exact case decision 9 calls worse than terminal text.
  - Likelihood 1: pid wrap on a long-running Mac.
  - Severity 5: the user approves after reading the wrong transcript.
  - Label: implementation defect. Disposition: new.
  - Fix: require the sessions file's cwd to equal foreground.cwd and its mtime to be after created_ms.

  herdr-remote-r3-5 — android/.../ui/PairScreen.kt:83, android/.../net/Api.kt:117, daemon/actions.go:317
  - Evidence: /v1/hello runs once, at pairing, inside runCatching. The clock offset and endpoint list never refresh. A phone whose clock drifts more than 5 minutes gets expired_request on every action; the text says "Try again" and nothing resyncs. A Mac that changes address is reachable only through the stored list.
  - Likelihood 3.
  - Severity 4: approvals unusable until re-pairing.
  - Label: implementation defect. Disposition: new.
  - Fix: call hello() on each SSE connect and after expired_request.

  herdr-remote-r3-6 — daemon/actions.go:156-172,349-352
  - Evidence: a duplicate request id that arrives while the first copy is in flight waits on done; if the first ends in a refusal, release closes done with res nil, so the waiter writes nothing and Go sends 200 with an empty body, which the phone reports as outcome unknown.
  - Likelihood 2: the next-host re-post of finding 2 overlapping a refusal.
  - Severity 3.
  - Label: implementation defect. Disposition: new.
  - Fix: release stores the refusal body instead of nil.

  One-liners: live.go:105-127 defers cmd.Wait to run before cancel, so a write error before context teardown blocks on a live hc attach. pair.go:271,277 lets unauthenticated clients write unmetered fsynced refusal lines during a window, the only writer outside the rate limit that feeds the rotation argument in 3.5.10. install.sh:42-49 loads the LaunchAgent
  before the mod-marker check, so a stop leaves the daemon installed.

  Completeness

  - Delivered: S1, S2, S3, S4, S5, S6, S7, S8, S10, C1, C2 (Allow on a real phone left to the user), C3, C4, C5 (resize and heavy output left to the user), C6, C8, C11, C12 (as-built lock-screen note), C13.
  - S9 partial: endpoint refresh from /v1/hello happens once (r3-5).
  - S11 partial: unknown-outcome text and action timeout (r3-2).
  - S12 partial: C10 and the phone checks not run, as recorded.
  - C7 partial: at-most-once holds in the daemon, but the phone invites a duplicate (r3-2).
  - C9 missing for the real client: handshakes and resumption fail, live views end, SSE does not (r3-1).
  - C10 not run: no Tailscale.
  - Changes no step explains: none. daemon/manual_test.go, testdata/pairing-vectors.json and server-test.crt are the as-built test tooling.

  Bot comments

  None.

  END ROUND 1

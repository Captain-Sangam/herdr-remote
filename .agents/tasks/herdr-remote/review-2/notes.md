# Review notes (second review)

## User notes, verbatim
"DONT NIT PICK. Surface only things that matter"

This means the following:
- Report at most five findings.
- Report a finding only if it meets one of these:
  - severity 6 or more, meaning a security hole, lost or wrong data, an action pressed on the wrong prompt, a broken requirement the user will hit, or a crash or hang;
  - an unambiguous requirement miss against TDD.md.
- Do not report style, naming, missing comments, test-only gaps, theoretical races that need several unlikely events, or anything already listed below.
- Zero findings is an acceptable answer.

## What is under review
- The whole repo `/Users/zeus/code/herdr-remote`, from the empty tree (`4b825dc642cb6eb9a060e54bf8d69288fbee4904`) to `429c512`.
- That includes the eight review fixes in commits `05423de` to `429c512`.
- The code diff is `/private/var/folders/t4/c_98fx1d1z3_z4nxc5xnf3t40000gn/T/council.OdlikHChF5/herdr-remote.diff.review2.patch`. It leaves out the docs and the vendored files.

## The plan
- `TDD.md` at the repo root, with its As built section.
- The first review report, `/Users/zeus/code/herdr-remote/.agents/tasks/herdr-remote/review.md`, changed the plan in its section 6, "Plan amendments". These changes are now in the code:
  - action timeout 150 s and no automatic retry after send;
  - revoke marks first;
  - generic notification text;
  - 120 s key wait;
  - transcript fallback and paging;
  - `/v1/hello` refresh.

  Treat the amended behaviour as the plan.

## Settled. Do not re-argue
- Same-user processes are trusted (decision 18).
- There is no relay. There are two TLS listeners.
- Daemon-mode Codex shows terminal text.
- Questions are view-only.
- The Mac-in-use guard is a heuristic, and its residual race is accepted.
- The decisions in the TDD table.

## Already known and accepted (review.md section 4). Do not report again
- a reused pid mapped to an old Claude session;
- an empty 200 for an in-flight duplicate after a refusal;
- stalled enrollment connections that outlive the window;
- unmetered refusal lines during a pairing window;
- the live handler defer order;
- `install.sh` loads the agent before the marker check;
- a retry after the terminal ended gets 404;
- an open space screen does not add new agents;
- commit `0d14b24` does not build alone.

## Already dismissed
- "SSE survives revoke over HTTP/2": the HTTP/2 server calls `ConnState` with the TLS connection (`go/src/net/http/h2_bundle.go:4570-4574`).
- Missing audit lines for pairing cancel and expiry: not required.

## Verified before this review
- `go vet` is clean, and `go test ./...` passes (84 s).
- `./gradlew testDebugUnitTest assembleDebug` passes, with 15 JVM tests.
- The installed daemon still runs the build from before the fixes. Do not judge the installed binary. Judge the code.

## Environment limits
- There is no phone and no Tailscale.
- Do not run the full suites, Gradle, the emulator, `install.sh`, or anything that touches the installed daemon, its state directory, the mod, or hc terminals.
- At most one targeted Go test.

## Known open item from the first review's fix check. Do not report it
After a decision with an unknown outcome, `refreshPrompt()` (`android/app/src/main/java/dev/herdrremote/ui/AgentViewModel.kt:149`) can replace the "Outcome unknown" note with "The Mac is not reachable" (`:129`) when the network is still down. It is queued for the next fix turn.

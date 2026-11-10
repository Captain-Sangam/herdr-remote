
## 5. Plan review, round 3 (the drafts of PRD.md and TDD.md)

The orchestrator wrote the corrected plan as drafts of the two final documents. Both reviewers checked them against the earlier plan, the brief and the installed tools, and gave a verdict.

### Fable reviewer: `VERDICT: READY`

- **All round-two items: fixed.**
  - It verified the Codex lock evidence again. The daemon holds the lock of t1v's thread, and the thread's time (UUIDv7, 19:31:21Z) matches t1v's status time (19:31:25Z).
  - t1v had no prompt, so it falls to terminal text as designed.
  - The daemon keeps the locks of idle threads for over an hour, so "live" was the right test.
- **Small new findings:**
  - The SSE stream had no event for an agent that finished or was removed, so the phone could not drop a dead agent until the next snapshot (likelihood 8, severity 3).
  - The SHA-256 of a short message such as "yes" or "continue" can be guessed, against the no-content logging rule (5, 2).
  - "Do not overwrite a folder that exists" meant a reinstall never updates the mod, and the check would pass on the old copy (6, 2).
  - Three details were still guessed: how "input during the call" is computed, the status code of `/v1/enroll` after a cancel, and the shape of `status` in `GET /v1/terminals/{t}` (7, 2).
  - The PRD listed three approval results where the TDD had four.
- **Facts.** The tool facts in the TDD match what it could see: the lock holders, rollout names, `HIDIdleTime`, `hc events`, and the `exec` limit.

### Astra reviewer: `VERDICT: NOT READY`

- **Fixed:** the enrollment timeout, the overlay button, the testing row, the idle-check order, logging in every build, the cursor wording, and the listener-specific checks.
- **Not fixed: Codex attribution.**
  1. Start a Herdr Codex terminal A, and give it no prompt.
  2. Start a daemon-mode Codex B outside Herdr, in the same folder, and prompt it.
  3. B's rollout is the only candidate, and A is the only Herdr terminal there, so the rule gives B's transcript to A.

  Neither `source: cli` nor an app-server lock ties a thread to a Herdr terminal. This contradicts the PRD's promise of transcripts "identified for certain".
- **Not fixed: replay retention.** Request IDs were arbitrary UUIDs with no creation time that the daemon could check. After the daemon forgot an ID, it could not tell a replay from a new request. Two rotations could also drop IDs that were still valid. The PRD still promised that a message arrives "once".
- **New, severity 7:** the PRD said every unpaired TLS connection is rejected, but the TDD needs an unpaired phone to complete TLS on the enrollment port. Built literally from the PRD, pairing is impossible.

### What the orchestrator verified

- **No exact binding.** The files in `~/.codex/tui-thread-reference-capabilities/` are empty markers, so no shipped interface ties a daemon-mode terminal to its thread.
- **More holes in the folder match**, beyond the reviewer's example:
  - a daemon-mode session that exited keeps its lock;
  - a resumed thread has an old timestamp.

  Each can make it show another session's transcript.
- **Herdr's launcher.** Herdr's own launcher for Codex runs plain `codex` (`shared.json`, `new-menu.agents[1].run = ["codex"]`), so daemon mode is the default for the user's agents. `hc --skill` already says Codex under hc must run with `--no-daemon`.
- **The other findings** were checked against the drafts and are correct.

### Final resolution (made after round 3, at the round cap)

- **Codex transcripts.** The folder match for daemon-mode Codex is removed.
  - A Codex in daemon mode shows terminal text, with the hint "Start Codex with `--no-daemon` for the full transcript".
  - The setup tells the user to set Herdr's Codex launcher to `["codex", "--no-daemon"]` once. `install.sh` prints this, but does not change Herdr settings.
  - **Reason:** a wrong transcript is worse than terminal text, because the user reads it before an approval. Three rounds kept finding new ways the match could be wrong. This sides with the Astra reviewer's position and overrides the Fable reviewer's "ready" on this point.
- **Replay.** Request IDs are now UUIDv7, so each one carries its time.
  - The daemon refuses an ID more than 5 minutes from its clock, and remembers each ID for 10 minutes.
  - The phone corrects its clock with `server_time_ms` from `/v1/hello`, so clock skew does not matter.
  - A rate limit of 2 actions a second for each device keeps 10 minutes of audit lines far below the 10 MiB rotation size, so a rotation cannot drop a valid ID.
  - Consumed markers live until the terminal's turn changes.
  - The PRD now says "at most once", with "outcome unknown" as a possible result.
- **Handshake guarantee.** The PRD and the TDD now limit "only paired phones complete the handshake" to the API port 7766, and describe the enrollment port as the exception. That port opens only in a window that the user starts, and it is authorized by the one-time token.
- **Fable's small findings, all adopted:**
  - an `agent_removed` event, with a reducer test;
  - the message length in the audit log, in place of a hash;
  - a hidden marker file, so `install.sh` updates its own mod folder and stops on a folder it does not own;
  - exact definitions: `uncertain` when the idle time after the key is less than the time since the idle check; `/v1/enroll` returns 403 for Deny or a wrong token and 410 for expiry or cancel; `GET /v1/terminals/{t}` returns the agent object plus `cols` and `rows`;
  - four results in the PRD's approval flow (five, with "refused").

**Not re-reviewed:** these final fixes were made after the third and last round, so the reviewers did not check them again. The TDD's verification status says so.

## 6. Summary of positions

| Topic | Claude planner | Codex planner | Fable reviewer | Astra reviewer | Final |
|---|---|---|---|---|---|
| Live view | Stream through a PTY | 4 Hz text snapshots | Accepted the stream | Flow control needed | Stream through a pipe, with flow control |
| Claude approve key | `1` | `enter` when Yes is selected | Accepted | Accepted | `1`, verified |
| Enrollment | Path middleware | Mac confirmation, separate listener | Timeouts, overlay button | Per-listener checks | Separate listener, Mac Allow, 6-digit code |
| Duplicate actions | 10-minute cache | Durable IDs | Rebuild from audit | Enforceable expiry | Audit log, UUIDv7 IDs, rate limit |
| Codex transcript | Folder heuristic | Explicit ID or manual pick | Elimination rule: ready | Not an identity | Exact IDs only; `--no-daemon` setup |
| Race with the Mac user | Not raised | Named the race | Guard is a heuristic | Guard runs last, fails closed | Idle guard, `uncertain`, accepted residual |
| Questions | Digits 1 to 9 | Mac only | Not raised | Not raised | Mac only |

## 7. Timing

All times are on 2026-11-08, IST.

| Stage | Start | End | Duration |
|---|---|---|---|
| Brief written and approved | 01:05 | 01:13 | about 8 min |
| Planning round 1 | 01:15 | 01:19 | 4 min 20 s |
| Orchestrator verification (attach, approval probes) | 01:19 | 01:26 | about 7 min |
| Planning round 2 | 01:27 | 01:30 | 3 min 10 s |
| Plan written | 01:30 | 01:34 | about 4 min |
| Review round 1 | 01:35 | 01:45 | 9 min 52 s |
| Orchestrator verification (Codex daemon, edit prompts, tagging) | 01:45 | 01:52 | about 7 min |
| Review round 2 | 01:53 | 01:57 | 4 min 23 s |
| Drafts of PRD and TDD | 01:58 | 02:03 | about 5 min |
| Review round 3 | 02:04 | 02:08 | 4 min 36 s |
| Final fixes and publication | 02:08 | 02:20 | about 12 min |

⏺ ROUND 2 herdr-remote-r1

  Round-one findings
  - r1-1: amend. Source 3 as written still fails on this Mac: t1v (daemon mode, cwd /Users/zeus/code, created 19:29:33Z) has two source: cli candidates after created_ms (19:33:34Z, which belongs to
    t1x running --no-daemon, and 19:40:47Z), so "several" sends t1v to terminal text. Fix: subtract rollouts already bound to other terminals by sources 1 and 2 before the exactly-one test. Also, the
    daemon closes idle rollouts (pid 14881 now holds none), so mtime scanning is right and lsof on the daemon is not.
  - r1-2, r1-3, r1-4, r1-6, r1-7, r1-8, r1-10: accept.
  - r1-5: accept; auto mode stays an assumption, correctly stated.
  - r1-9: withdrawn. The untagged-send probe refutes it.

  r2 findings
  - r2-1: amend. HIDIdleTime prints here (2168 s idle now). It sees local HID and, unverified, Screen Sharing injection; it does not see SSH, hc attach from another terminal, or hc keys from agents,
    which only the core's 3 s quiet rule covers. False refusals: a mouse jiggler or any HID device with periodic reports keeps idle under 10 s forever, so the phone can never approve. Fix: make the
    threshold a setting, return the measured idle in the 409, audit refusals. A sleeping display is fine, idle only grows.
  - r2-2: amend, see r1-12.
  - r2-3, r2-4, r2-5, r2-6, r2-7, r2-8, r2-9, r2-10: accept. Two-listener enrollment: no new issue.

  New findings

  herdr-remote-r1-11 · Decision 9 source 3, Step 6 · plan defect · new. Evidence above (t1v, t1x). Likelihood 7: the user runs several Codex agents per folder today. Severity 4. Fix: exclude
  explicitly bound rollouts; Step 6 test "two candidates, one bound elsewhere".

  herdr-remote-r1-12 · Decisions 10 and 17 · plan defect · new. The audit line {ts, device_id, action, terminal, turn, request_id, result} has no prompt_hash or decision, so the consumed marker {turn,
  prompt_hash} cannot be rebuilt on restart as decision 10 claims; a turn-only marker is weaker. Nothing says the dispatching line is fsynced before the key, or that a failed write refuses the
  action, or that reload reads the rotated file. Likelihood 8. Severity 4. Fix: add both fields; fsync then press; audit write failure gives 503 and no key; reload both files.

  herdr-remote-r1-13 · Decisions 5 and 6, Step 9 · plan defect · new. /v1/enroll long-polls up to 120 s, but decision 5 sets read timeouts only for SSE and live; OkHttp's 10 s default aborts before
  Allow. Section 6 row 2 delays scanning, not Allow, so the Check misses it. Likelihood 7. Severity 4. Fix: 130 s read timeout on enroll; row 2 adds "60 s before Allow".

  herdr-remote-r1-14 · Decision 6 step 1, Step 8 · plan defect · new. Opening the overlay always runs pair --json, so revoking a lost phone opens a pairing window and port 7767 for 120 s. Likelihood
  10. Severity 3. Fix: a "Pair a phone" button opens the window; the device list does not.

  herdr-remote-r1-15 · Section 6 row 4 vs Decisions 3 and 16 · plan defect · new. "A plain shell" has reported: false, so it is not an agent and not a valid target; the manual check cannot run.
  Likelihood 10. Severity 2. Fix: test terminal text with a reporting agent before its first prompt, when no sessions file exists.

  Nothing else material remains.

  END ROUND 2

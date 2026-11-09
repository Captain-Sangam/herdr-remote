⏺ ROUND 3 herdr-remote-r1

  Round-two items in TDD.md
  - r1-1, r1-11, Astra amendment: fixed (3.6 source 3; Step 6 tests). Verified now: lsof +D shows pid 14881 holding one lock; its UUIDv7 time (19:31:21Z) matches t1v's status_ms (19:31:25Z) and
    ~/.codex/tui-thread-reference-capabilities/<id>. It has no rollout, so t1v is unprompted and falls to terminal text as designed. The daemon keeps idle-thread locks (held over an hour), so "live"
    is the right test. Each --no-daemon codex binary holds its own lock; the ChatGPT app-server holds guardian locks, excluded by source: cli.
  - r1-12, Astra r2-2: fixed (3.5 rules 3, 6, 9, 10; audit line has prompt_hash and decision; Step 7 tests rotation and a failed write).
  - r1-13, Astra r1-4: fixed (decision 5; Step 3 test; section 6 row 2).
  - r1-14: fixed (decision 6; 3.3 rules 1 and 2; 3.8; Step 8 check).
  - r1-15: fixed (section 6 row 4).
  - Astra r2-1: fixed (3.5 rule 5; criterion 6; section 7).
  - Astra r2-4, r2-10, r2-11: fixed (decision 14 and SR-9; Step 6 invariant; decision 4 and Step 3 first test).

  New findings

  herdr-remote-r1-16 · 3.4 /v1/events · under-specified · new. The agent event shape has no way to say a terminal finished or was removed, although decision 16 reacts to removed events; the phone
  cannot drop a dead agent until the next snapshot, and the Step 10 reducer test has no such case. Likelihood 8. Severity 3. Fix: agent carries state: "gone", or a removed event with address; add the
  reducer case.

  herdr-remote-r1-17 · 3.6 audit line, decision 12 · security gap · new. payload_sha256 of a short message ("yes", "continue") is reversible by guessing, against SR-9. Likelihood 5. Severity 2. Fix:
  log the text length only, or an HMAC with a per-install random key.

  herdr-remote-r1-18 · Step 8 · plan defect · new. "Do not overwrite a folder that exists" means a reinstall never updates the mod, and the Check passes on the old copy. Likelihood 6. Severity 2. Fix:
  overwrite the files the script owns; herdr-app check after the copy.

  herdr-remote-r1-19 · 3.5 rule 8, 3.4 · under-specified · new. Still guessed: how "input during the call" is computed (idle after the key less than the time since the pre-check); the /v1/enroll
  status after pair cancel (403 or 410); whether status in GET /v1/terminals/{t} is the hc object or the state string. Likelihood 7. Severity 2. Fix: one line each.

  PRD and TDD agree, except PRD 6.2 step 6 lists three results where the TDD has four (uncertain, outcome_unknown missing). Installed-tool facts in section 2 match what I can see: lock holders,
  rollout naming by thread id, HIDIdleTime, hc events, the exec limit.

  VERDICT: READY

  END ROUND 3

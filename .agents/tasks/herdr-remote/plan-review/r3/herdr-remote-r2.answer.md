• ROUND 3 herdr-remote-r2

  Fixed:

  • herdr-remote-r1-13 and the r1-4 amendment: TDD decision 5 specifies 130 seconds for enrollment; Step 3 tests delayed Allow.
  • herdr-remote-r1-14: Pairing requires the button; Step 8 checks that opening the device list leaves 7767 closed.
  • herdr-remote-r1-15: Section 6, criterion 4 now tests unprompted agents and ambiguous Codex sessions.
  • herdr-remote-r2-1: TDD §3.5 moves the idle check under the lock, refuses unreadable idle state, audits refusals and exposes the threshold. Section 7 identifies the remaining limitations.
  • herdr-remote-r2-4: Decision 14 and PRD SR-9 prohibit content logging in every build.
  • herdr-remote-r2-10: Step 6 explicitly permits the cursor 409.
  • herdr-remote-r2-11: Decision 4 separates enrollment authorization from device-store authorization; Step 3 tests an unpaired key through 7767.

  Not fixed:

  • herdr-remote-r1-1 / herdr-remote-r1-11, including my amendment: TDD §3.6 still cannot establish identity. Start an unprompted Herdr Codex terminal A, then start and prompt daemon-mode Codex B
    outside Herdr in the same folder. B supplies the sole eligible rollout; A is the sole Herdr peer. The elimination rule assigns B’s transcript to A. source: cli and an app-server lock do not
    establish a Herdr terminal binding. This contradicts PRD FR-8’s “identified for certain.” Require an exact or explicit binding; otherwise use terminal text. Add this external-CLI fixture.

  • herdr-remote-r1-12 / herdr-remote-r2-2: Fsync and stored prompt hashes are fixed, but replay retention is not. TDD §3.5 rule 10 reloads only ten minutes; §3.6 specifies arbitrary UUIDs without a
    verifiable creation time. After forgetting an ID, the daemon cannot distinguish its replay from a new request. Two rotations can also discard still-valid IDs. Define enforceable expiry and
    preserve all accepted IDs through it; preserve consumed markers until their prompt changes. Test late replay and multiple rotations. PRD FR-13 and acceptance criterion 7 also still promise arrival
    “once,” whereas TDD correctly permits an unknown outcome and at-most-once dispatch.

  herdr-remote-r2-12 — PRD success measure/SR-1 versus TDD decision 4; plan defect; disposition new.
  Evidence: PRD lines 35 and 147 reject every unpaired TLS connection. TDD correctly requires an unpaired phone to complete TLS on 7767 before enrollment. Both requirements cannot hold.
  Likelihood 10: every first pairing exercises the contradiction.
  Severity 7: implementing the PRD literally prevents pairing.
  Correction: Scope the rejection guarantee to 7766 and explicitly describe the temporary, token-authorized enrollment exception in both documents’ outcome statements.

  VERDICT: NOT READY
  Blocking: incorrect Codex attribution, incomplete replay retention/expiry, and the contradictory enrollment guarantee.

  END ROUND 3

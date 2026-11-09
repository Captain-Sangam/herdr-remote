• ROUND 2 herdr-remote-a2

  Position:

  I concede streaming through hc attach --observe (orchestrator probe 1), Claude approval with 1 (probe 3), explicit transcript resolution with terminal-text fallback (probe 5), and dropping openapi.yaml. I reject decision 10’s state-only idempotency claim. Decision 4 also needs correction before pairing can work.

  Reasoning:

  The probes remove my streaming and Claude-key objections. They also establish explicit transcript identities without persisted manual bindings.

  A permitted duplicate-action sequence is: the broker injects an approval; the agent has not yet processed input or redrawn; the response connection drops; a retry acquires the lock and observes the same blocked status, turn, and prompt. It injects again. This is an execution-schedule inference, not an observed probe: terminal.keys promises key delivery, with
  no guarantee that the agent changes status or screen before return. Serializing requests does not consume an action.

  Outline:

  • Decision 4; step 3: TLS authentication cannot inspect an HTTP path or body token before HTTP parsing. Rejecting every unknown key during the handshake also rejects new phones. Keep the paired listener strict and add a temporary enrollment-only listener, advertised in the QR. Test successful fresh enrollment as well as rejection.

  • Decision 10; step 7: Durably consume a broker-issued action ID before injection. On ambiguous dispatch, return “outcome unknown” without reinjection. Add a test that holds screen/status unchanged after injection, drops the response, then retries; assert one key. Existing stale-turn/hash tests miss this failure.

  • Decision 8; steps 5 and 11: Size events and terminal bytes travel on separate connections; no cross-stream ordering is specified. Define resize recovery, preferably restarting the observer for a fresh paint at the new dimensions. Test resizing during output. Counting subprocesses and checking bytes cannot detect rendering corruption. The underlying API
    explicitly distinguishes ordered resize support (terminal.attach.resize_notifications).

  • Decision 14; steps 9–11: The biometric timeout is claimed but lacks an explicit implementation and verification step. Test background timeout, activity recreation, and action rejection before unlock.

  • Step 4: Layout refresh depends only on core terminal events. Add periodic herdr-app ls reconciliation and a rename/move test; spaces and tabs belong to the window (herdr-app --skill).

  Assumptions:

  • The orchestrator’s probe results are authoritative.
  • Background delivery remains conditional on a running, connected service; criterion 8 needs that qualification.

  Confidence:

  92

  END ROUND 2

# ADR 0008: Live view from hc attach in xterm.js

- **Status:** Accepted, 2026-11-08.
- **Decision:** 8 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- The user asked for a live view of the terminal "like screen sharing".
- The planners disagreed in round 1:
  - The Claude planner wanted to stream `hc attach --observe` through a PTY (`creack/pty`) over a WebSocket.
  - The Codex planner wanted `terminal.read` snapshots at 4 Hz, and to measure before it decoded the framed stream.
- The orchestrator verified that `hc attach --observe` writes the first paint and then the live bytes, with colors, to a plain pipe. It needs no TTY and no decode of the framed protocol.
- xterm.js drops data past its write buffer, so unbounded writes lose output.
- After a resize, the old stream no longer fits the terminal.

## Decision

- `GET /v1/terminals/{t}/live` runs `hc attach --observe --history 500 <t>` and streams its stdout.
- The phone shows xterm.js in a locked-down WebView at the terminal's own `cols` and `rows`, scaled to the screen width, with pinch zoom.
- Flow control:
  - base64 chunks of at most 64 KiB
  - each chunk waits for the xterm.js `write` callback
  - at more than 1 MiB pending, Kotlin drops the queue and reopens the stream.
- On a `size` event, the daemon ends the live streams of that terminal. The phone reopens one for a fresh paint.
- A device can have at most 2 live views.

## Alternatives

| Alternative                  | Why it lost                                   |
| ---------------------------- | --------------------------------------------- |
| `terminal.read` at 4 Hz      | Plain text, and 250 ms steps.                 |
| A PTY around `attach`        | Not needed. A pipe is enough.                 |
| Unbounded writes to xterm.js | xterm.js drops data past its buffer limit.    |
| WebSocket                    | See [ADR 0005](0005-http-sse-and-streams.md). |

## Consequences

- The stream is colored and incremental, with no PTY and no protocol decode.
- A fresh paint from `attach` makes it safe to drop a backlog.
- The phone never writes to the terminal. Each stream that ends kills its subprocess.
- A third live view gets 409 `live_limit`.
- On the phone tested, the live view followed a pane resize and recovered after 50 MB of output.

## Sources

- [Technical design, decision 8](../history/technical-design.md#31-decision-table)
- [Debate, what the orchestrator verified after round 1](../history/debate.md#what-the-orchestrator-verified-after-round-1)

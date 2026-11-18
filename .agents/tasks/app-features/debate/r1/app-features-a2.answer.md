⏺ ROUND 1 app-features-a2

  Position. Build next the features that close the loop from the phone: today the user can see a block and press one key, but cannot redirect an agent without the Mac. The theme is "decide and redirect in one visit". First come the features that need no rule change (reply from the alert, deny with a reason, triage facts, saved replies). Then come four that
  need a security change, in order of value per risk: interrupt, typed and Codex answers, send an image, start or stop an agent.

  Reasoning.
  - Value. The done alert carries only a title (android/.../watch/WatchService.kt:316-322), and only blocked alerts have actions (WatchService.kt:312). So every "finished" costs an unlock, a navigation and a typed message.
  - Deny is a dead end. Deny sends esc (daemon/actions.go:493) and Claude goes idle (docs/interfaces.md:116). A message to a blocked agent is refused (daemon/actions.go:651-654). The user must deny, wait, then type, with no guidance.
  - Unused facts. hc gives status.progress and created_ms (docs/interfaces.md:71-74). The daemon drops progress (daemon/hc.go:77-82) and does not send CreatedMS (daemon/hc.go:60, docs/api.md:40).
  - Unused hc commands. hc --help lists interrupt, put, run and kill. ADR 0003 excludes them (docs/adr/0003-allowed-hc-use.md), and allowedMethods enforces the list (daemon/hc.go:44-52).
  - Security fit. A message already lets the phone direct an agent that runs commands. So interrupt and image add little new power, and start adds more. Each reuses the request_id, audit and rate-limit path (daemon/actions.go:98-166).
  - Size. Features 1 to 4 and 10 reuse the 11 routes (daemon/api.go:15-25).
  - Parity. Every feature is a plain HTTP call plus platform UI. Alert actions depend on how the iOS plan delivers alerts.

  Outline.

  1. Read and reply from the alert. The done alert expands to show the agent's last text and a Reply field. The user answers without opening the app.
     - Why: removes the most frequent trip.
     - Changes: phone only. Fetch the last transcript page on Done, as fetch does for prompts (WatchService.kt:163). Reply calls /message.
     - Security: none new. The lock screen stays empty (WatchService.kt:277-278). Reply needs an unlocked device.
     - Size: S. Depends on: nothing.
  2. Deny with a reason. After Deny, the composer opens with focus and sends when the agent is idle.
     - Why: a bare deny stalls the agent.
     - Changes: phone only. Two taps give one key and one message, so SR-6 holds.
     - Security: none.
     - Size: S. Depends on: nothing.
  3. Interrupt. Needs a security change. A Stop button on a working agent, with a confirm step. The result shows what hc reports.
     - Why: a wrong-headed agent in auto mode wastes hours, and a message only queues behind the work.
     - Changes: POST /v1/terminals/{t}/interrupt, terminal.interrupt added to allowedMethods, a new ADR.
     - Rule: "Interrupts" out of scope (docs/product.md:69), ADR 0003.
     - Risk: lost work in progress, or a stop on the wrong turn.
     - Safeguard: state must be working, turn check, Mac-idle guard, one request_id, audit line.
     - Size: M. Depends on: nothing.
  4. Triage facts in the lists. Each row shows the time in its state and the progress. The user sees "blocked 40 min" at a glance.
     - Why: the lists order by state only (android/.../ui/ListScreens.kt:51,256).
     - Changes: the daemon adds since_ms and progress to agent. The phone shows them.
     - Security: none, no content.
     - Size: S. Depends on: nothing.
  5. Saved replies and a kept draft. Chips above the composer hold the user's own short prompts. A draft survives leaving the screen.
     - Why: typing is the slow part. The draft is plain remember state (android/.../ui/AgentScreen.kt:109).
     - Changes: phone only. Replies are stored on the phone, the draft in memory.
     - Security: the stored text is written by the user, not transcript text (SR-12).
     - Size: S. Depends on: nothing.
  6. Typed and Codex answers. Needs a security change. The user picks "Type something" and types the answer, or answers a Codex question.
     - Why: these prompts say "Answer this on the Mac" (android/.../ui/AgentViewModel.kt:241).
     - Changes: probes for the screens, then a decision variant with one key and one text under one request_id.
     - Rule: out of scope (docs/product.md:68), SR-6, ADR 0021.
     - Risk: text lands in the wrong field.
     - Safeguard: prompt hash and turn checks, re-read the screen for the text field before the send, else refuse.
     - Size: L. Depends on: probes.
  7. Send an image. Needs a security change. The user attaches a screenshot or photo to a message.
     - Why: a visual bug seen on the phone needs no description.
     - Changes: POST /v1/terminals/{t}/file, terminal.put, then a message with the landed path.
     - Rule: SR-2 (write files).
     - Risk: an arbitrary file on the Mac.
     - Safeguard: images only by magic bytes, a size cap, hc picks the path, audit with size only.
     - Size: M. Depends on: nothing.
  8. Start an agent. Needs a security change. In a space, the user starts Claude Code or Codex from a fixed list, with a first prompt.
     - Why: new work without the Mac.
     - Changes: POST /v1/spaces/{id}/agents, hc run --bg --label herdr.space=<id> in the space folder.
     - Rule: SR-2, "Start or stop" out of scope (docs/product.md:70).
     - Risk: program start from the phone.
     - Safeguard: a launcher list in the daemon config, no argv from the phone, a cap on live agents, the Mac-idle guard, audit.
     - Size: L. Depends on: nothing.
  9. End an agent. Needs a security change. The user ends a finished or stuck agent after a confirm step.
     - Why: frees a stuck session.
     - Changes: terminal.kill and a route.
     - Rule: same as 8.
     - Risk: lost session.
     - Safeguard: confirm, never while working unless interrupted first, audit.
     - Size: M. Depends on: 3.
  10. Search. The user filters agents by name and finds text in the loaded transcript.
      - Why: helps with many agents.
      - Changes: phone only, in memory.
      - Security: none.
      - Size: S. Depends on: nothing.

  Assumptions.
  - The user's answer to the brief's question is the default: out-of-scope items may be listed.
  - ADR 0019 allows actions from an unlocked alert without the app gate. I did not read it.
  - terminal.interrupt and terminal.put exist as RPC methods. I saw only the CLI help and docs/interfaces.md:39.
  - The transcript always holds the agent's last text at the done event.
  - Not investigated: a push relay for iOS alerts, several Macs, a home-screen widget, token or cost display, "always allow" options.

  Confidence. 72

  END ROUND 1

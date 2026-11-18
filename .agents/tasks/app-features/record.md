# Council record: app-features

Invocation: `/council plan using astra and fable. Check and tell me what othre features can we build into the mobile app\n\nThe final output to be a list of features and descriptions for each`

## Roster

| Name | Kind | Model | Terminal | Tab |
| --- | --- | --- | --- | --- |
| app-features-a1 | codex | gpt-6-astra | t113 | tbd91ae5f20ff26f48861dc205c0e4094d |
| app-features-a2 | claude | fable | t114 | tb6298d6ff7aa6a06bacb3271afced9e5d |

## Paths

- Task folder: /Users/zeus/code/herdr-remote/.agents/tasks/app-features
- Scratch: /private/var/folders/t4/c_98fx1d1z3_z4nxc5xnf3t40000gn/T/council.T7GrMOyMTE
- Primary repo: /Users/zeus/code/herdr-remote (branch main, HEAD 247206a)
- Plan file: /Users/zeus/code/herdr-remote/.agents/tasks/app-features/plan.md

## Stages

- 2026-11-18 20:08 brief: started. Roster as given: astra, fable. Expanded to codex with gpt-6-astra and claude with fable.
- 2026-11-18 20:09 brief: approved by the user as written. Out-of-scope features allowed, each marked "needs a security change" (the default).
- 2026-11-18 20:10 debate round 1: started.
- 2026-11-18 20:13 debate round 1: ended. Both answered (14:40:22Z to 14:43:05Z). Tree unchanged against the baseline.

## Round 1 synthesis

**Adaptation.** The user asked for a list of features, not an implementation plan. The Outline heading asked for a ranked feature list, the budget was 1000 words, and the plan template puts the feature list where the steps go.

**app-features-a1 (Codex).** Theme: shorten the cycle of check, understand, unblock. Nine features: a decision queue (S), catch up since last read (M), stop the current turn (M, security change), full tool details (M), action receipts (M), transcript search (S), helpers under the parent agent (M), Codex choice questions (L, security change), launch Mac-side presets (L, security change). Confidence 85.

**app-features-a2 (Claude).** Theme: decide and redirect in one visit. Ten features: read and reply from the finished alert (S), deny with a reason (S), interrupt (M, security change), time in state and progress on the rows (S), saved replies and a kept draft (S), typed and Codex answers (L, security change), send an image (M, security change), start an agent (L, security change), end an agent (M, security change), search (S). Confidence 72.

**Agreements.** Both list interrupt as the first security-change feature, with a new ADR, an allowlist entry, a turn check, the Mac-idle guard and an audit line. Both list search (S), Codex or typed answers (L) and a way to start agents from a fixed list on the Mac (L, last).

**Verified.**
- `allowedMethods` has no interrupt, put, run or kill (`daemon/hc.go:44-52`). `docs/interfaces.md:39` lists `terminal.run`, `terminal.kill` and `terminal.put` as unused methods. `hc interrupt` exists in the CLI ("stop its current turn, not the process", `hc --skill`). The RPC method name for it is not verified.
- The finished alert has a title and the agent title only, and no actions (`WatchService.kt:313-320`). Only the blocked alert has actions (`WatchService.kt:309`). ADR 0019 already allows Approve and Deny from an unlocked alert with `setAuthenticationRequired(true)`. So a Reply action fits the same safeguard, with an amendment to ADR 0019.
- Deny presses `esc` (`daemon/actions.go:493`), and Claude goes idle (`docs/interfaces.md:116`). The daemon refuses a message to a blocked agent (`daemon/actions.go:651-655`). So a deny with a reason is two user actions: one key, then one message (SR-6 holds).
- The daemon `Status` struct drops `progress` (`daemon/hc.go:76-81`). The agent shape has no time in state (`docs/api.md:40`). `hc get` returns `status_ms` (seen on the live terminal t113).
- Tool calls and results are cut at 2000 characters (`daemon/transcript.go:21`, `docs/api.md:46`).
- Request IDs stay 10 minutes (`daemon/actions.go:20`).
- Sub-agents take the parent's placement (`daemon/state.go:315-318`).

**Disputes and gaps.**
- Action receipts (a1): the app already says "outcome unknown, check the transcript", and the transcript shows whether a message arrived. Low value. Draft drops it, round 2 asks a1.
- Helpers under the parent (a1): value is not shown. Round 2 asks a1.
- Catch up, tool details and the decision queue (a1) were not in a2's list. Round 2 asks a2.
- Reply from the alert on iOS: alerts with the app closed need the paid Apple program (HANDOFF). Round 2 asks a2 about parity and size.
- Send an image (a2): it writes a file on the Mac (SR-2). Round 2 asks a2 if the value justifies the rule change.

**Decision so far.** Rank by value first, then security fit, then size. Features with no rule change come first, except interrupt, which both rank high for value.
- 2026-11-18 20:15 debate round 2: started. Draft at debate/draft.md.
- 2026-11-18 20:18 debate round 2: ended. Both answered (14:45:54Z to 14:47:06Z). Tree unchanged against the baseline.

## Round 2 synthesis and final resolution

**app-features-a1 (Codex), confidence 89.** Accepted Reply first, provisionally. Asked to swap Deny with a reason (2) and the decision queue (5): the message box is already enabled when the agent is not blocked (`AgentScreen.kt:199-200`, verified), so Deny with a reason adds only the focus. Held that receipts have one real case: a lost decision response, then the terminal exits, and the transcript is gone with it. Accepted the deferral. Conceded that the helpers view's value is inferred. Plan check: catch-up cannot use time stamps (several items share one, `daemon/transcript.go:383`), typed answers as one key plus one text break SR-6, interrupt semantics need a probe, the image safeguards are weak, and launch and end need durable dispatch records.

**app-features-a2 (Claude), confidence 80.** Accepted the three added features. Moved Saved replies above Catch up (S, no open dependency) and Helpers to 10 (no rule change). Kept Send an image at 13. Plan check: a finished alert stays after the agent works again (`Reducer.kt:82-96`, verified), the request ID must be made at send time (5-minute window, `daemon/actions.go:19`, verified), Codex after `esc` needs a probe (`docs/interfaces.md:154`, verified), three contradictions in the draft's groups and SR-6 line, the space folder for launches comes from `herdr-app ls` (`docs/interfaces.md:88`, verified), and the Status struct is at `daemon/hc.go:77-82`.

**Resolution.**
- Decision queue moves to 2 and Deny with a reason to 5 (a1's evidence holds). Saved replies moves above Catch up (a2, accepted: Catch up now also needs a daemon item ID).
- Helpers stays 11, below typed answers, because value ranks first and its value is unmeasured. It moves to the no-rule-change group. a2's dissent (rank 10) is recorded.
- Typed answers become two confirmed steps, each one key or one text, so SR-6 holds (a1). The out-of-scope rule still needs a change.
- Receipts stay out. The reason now names the terminal-exit case and calls it rare.
- All plan-check points went into the final plan: stale finished alerts, request ID at send, transcript fallback, preview cap and notification history (Reply); Codex probe (Deny); interrupt probe and residual race (Stop); item ID (Catch up); image decode, size and count caps, clean-up (Image); fixed launchers, space folder, durable dispatch (Start, End).
- No third round: no open fact would change the ranking. The agents did not check the final text.
- 2026-11-18 20:19 plan: written to plan.md. 14 features, ranked. Linted with the language skill (warnings only, for labels).

## Timing

| Stage | Start | End | Duration |
| --- | --- | --- | --- |
| Brief | 20:08 | 20:10 (approved) | about 2 min, plus the wait for approval |
| Panes | 20:10 | 20:11 | about 1 min |
| Debate round 1 | 14:40:22Z (20:10 local) | 14:43:05Z | 2 min 43 s |
| Debate round 2 | 14:45:54Z | 14:47:06Z | 1 min 12 s |
| Synthesis and plan | 14:47Z | 14:49Z | |

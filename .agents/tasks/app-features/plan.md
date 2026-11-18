# Plan: the next features for the mobile app

## 1. Outcome

- **Result:** a ranked list of 14 features for the herdr-remote mobile app. Each feature says what the user can do, why it helps, what it changes, its security effect and its size.
- **Theme:** decide and redirect an agent in one visit to the phone. Today the phone can see a block and press one key. To redirect an agent, the user must go back to the Mac.
- **Order:** value away from the desk first, then fit with the security model, then size.
- **Security groups:**
  - Nine features need no change to a security rule: 1, 2, 4 to 9 and 11.
  - Five features need a security change: 3, 10, 12, 13 and 14. Each names the rule, the risk and the safeguard.
  - Feature 3 (stop the current turn) needs a security change, but its value is high. So it ranks third.
- **Practical effect:** the first five features remove most trips to the Mac for a finished or a blocked agent. Each feature is a separate piece of work with its own plan.

## 2. Current behaviour

Revision inspected: `herdr-remote` on `main` at `247206a`.

- **Alerts:** the watch service posts one alert for each agent.
  - A blocked alert shows the prompt. It has Approve and Deny when the prompt is actionable (`android/app/src/main/java/dev/herdrremote/watch/WatchService.kt:288-314`).
  - A finished alert shows only the agent name and its title. It has no actions (`WatchService.kt:316-322`).
  - A finished alert stays when the agent starts to work again. The alert rules give no clear for `done` to `working` (`watch/Reducer.kt:82-96`).
  - Alert actions need an unlocked phone (`setAuthenticationRequired(true)`, ADR 0019). A locked phone shows only the agent name.
- **Decisions:** Deny presses `esc` (`daemon/actions.go:493`).
  - After `esc`, Claude Code goes idle and its turn stays the same (`docs/interfaces.md:116`).
  - After `esc`, the Codex turn goes up by one (`docs/interfaces.md:154`). Its state after `esc` is not recorded.
- **Messages:**
  - A message has up to 4000 characters (`daemon/actions.go:21`).
  - The daemon refuses a message to a blocked agent (`daemon/actions.go:651-655`).
  - The message box is off while the agent is blocked, and on at all other times (`android/.../ui/AgentScreen.kt:199-200`).
  - The draft is plain `remember` state. It is lost when the user leaves the screen (`AgentScreen.kt:109`).
- **Request IDs:** the daemon accepts a request ID for 5 minutes from its UUIDv7 time. It remembers the ID for 10 minutes (`daemon/actions.go:19-20`).
- **Questions:** the phone answers Claude Code questions with one key for each option. "Type something", Codex questions and login prompts are view-only: "Answer this on the Mac" (`android/.../ui/AgentViewModel.kt:241`, ADR 0021).
- **Lists:** the rows sort by state only, in this order: blocked, error, done, working, idle (`android/.../ui/ListScreens.kt:48`).
- **Agent facts:**
  - The agent shape is `{address, space, tab, where, app, state, kind, msg, turn, title}` (`docs/api.md:40`).
  - The daemon `Status` struct drops the `progress` field of hc (`daemon/hc.go:77-82`).
  - `hc get` also returns `status_ms`, the time of the last status change.
- **Transcript:**
  - The daemon sends pages of `{role, kind, text, ts}` (`docs/api.md:44`).
  - Items from one record share one time stamp (`daemon/transcript.go:383-394`).
  - Tool calls and results are cut at 2000 characters (`daemon/transcript.go:21`).
  - The phone loads older pages on request (`AgentViewModel.kt:186`).
- **hc calls:**
  - The daemon sends only `hello`, `terminal.list`, `terminal.get`, `terminal.read`, `events.subscribe`, `terminal.send` and `terminal.keys` (`daemon/hc.go:44-52`, ADR 0003).
  - hc also has `terminal.run`, `terminal.kill` and `terminal.put` (`docs/interfaces.md:39`).
  - The CLI command `hc interrupt` stops the current turn, not the process. Its RPC method is not verified.
- **Sub-agents:** the daemon reads `sub_agents` from `herdr-app ls` and gives each sub-agent the place of its parent (`daemon/state.go:298-318`).
- **Spaces:** `herdr-app ls` gives the folder of each space (`docs/interfaces.md:88`).

## 3. Decisions

| # | Decision | Why | Strongest rejected alternative |
|---|---|---|---|
| 1 | Rank by value away from the desk, then security fit, then size. | These are the brief's criteria, in its order. | Rank all rule-safe features first. It puts a high-value interrupt below small comforts. |
| 2 | Put Reply on the finished alert first. | Each finished agent today costs an unlock, a navigation and a typed message. ADR 0019 already allows actions on an unlocked alert. | Open the app on a tap. That is today's behaviour. |
| 3 | Put the decision queue second and Deny with a reason fifth. | The message box is already on after a deny (`AgentScreen.kt:199-200`). So Deny with a reason adds only the focus. The queue saves a navigation for each decision. | Deny with a reason second. Its value is smaller than first thought. |
| 4 | Put Stop the current turn third, as a security change. | An agent on a wrong path in auto mode can work for a long time. A message waits behind the work. | Keep interrupts out of scope. It lost on value. |
| 5 | Make typed answers two confirmed steps: one key, then one text. | Each step is one dispatch, so SR-6 holds. | One key and one text under one request. It breaks SR-6. |
| 6 | Put helpers under their parent eleventh, in the no-rule-change group. | Its value is not measured, and value ranks first. | Rank 10, above typed answers, because it crosses no rule. Recorded as a dissent. |
| 7 | Do not list action receipts. | The app already says "outcome unknown, check the transcript". A receipt adds value only when the terminal exits before the user checks. That case is rare. | A receipt lookup for each request ID. Deferred. |
| 8 | Put the three features that start, end or write on the Mac last: 12, 13 and 14. | They cross SR-2, the rule that the phone cannot start programs or write files. | Leave them out. The brief asks to list them, marked. |
| 9 | Give each feature its own plan, and a new ADR when it changes a rule. | `AGENTS.md` asks for an ADR for each design decision. | One plan for all features. It is too large to review. |

## 4. Scope and acceptance criteria

- **Delivered:** the ranked feature list in section 5.
- **Excluded:** code, the iOS parity plan, store releases, and the open bugs in `docs/operations.md`.
- **Done when:**
  1. Each feature has a description, its value, its changes, its security effect, a size and its dependencies.
  2. Each feature that crosses a security rule names the rule, the risk and the safeguard.
  3. Each claim about today's code has a file reference that matches `247206a`.

## 5. Feature list

Size: S is a few days on one platform, M is about a week, L is more. The sizes are for Android. iOS parity adds to each.

### 1. Reply from the finished alert

- **What the user does:** the user expands a "finished" alert and reads the agent's last answer. The user types a reply in the alert and sends it, without the app.
- **Why:** this removes the most frequent trip to the Mac. The frequency is an assumption, not a measurement.
- **Changes:** phone only.
  - On a finish, the watch service gets the last transcript page, as it gets the prompt for a block.
  - The alert shows the last answer, cut to a fixed length. With no transcript, it shows the title, as today.
  - The alert gets a Reply action (Android `RemoteInput`). The action calls `POST /v1/terminals/{t}/message`.
  - The phone makes the request ID when the user sends, not when the alert posts. The daemon accepts an ID for only 5 minutes.
  - The alert goes away when the agent starts to work again. Then a late Reply cannot land in new work.
- **Security:** no rule change.
  - The action needs an unlocked phone, as Approve does. A locked phone shows only the agent name.
  - An amendment to ADR 0019 records the new action. It also says that the answer text sits in the notification shade, and possibly in the notification history.
- **Size:** S on Android. On iOS, the feature follows the delivery of iOS alerts.
- **Depends on:** nothing.
- **Done when:** a reply from the alert arrives as one prompt, once. A locked phone shows no answer text. The alert goes away when the agent works again.

### 2. Next blocked agent after a decision

- **What the user does:** after Approve, Deny or an answer, the phone offers the next blocked agent. The user works through all blocked agents without the list.
- **Why:** today each blocked agent costs a navigation.
- **Changes:** phone only, from the agent list that it has.
- **Security:** no rule change. One decision for each tap. No bulk approval.
- **Size:** S.
- **Depends on:** nothing.
- **Done when:** the next agent is the first blocked agent in list order, and its prompt loads fresh.

### 3. Stop the current turn (needs a security change)

- **What the user does:** the user taps Stop on a working agent and confirms. The phone shows the result that hc reports.
- **Why:** an agent on a wrong path in auto mode wastes time. A message waits behind the work.
- **Changes:**
  - A probe of the hc interrupt method: its RPC name, the key it presses, and the state after it.
  - `POST /v1/terminals/{t}/interrupt` on the daemon, and the interrupt method in `allowedMethods`.
  - A Stop key with a confirm step on the phone.
- **Rule:** "Interrupts" is out of scope (`docs/product.md:69`). ADR 0003 limits the hc calls.
- **Risk:** lost work in progress, or a stop on the wrong turn.
- **Safeguard:**
  - The daemon checks that the agent is `working` and that the turn matches, just before it sends.
  - The Mac-idle guard applies. One request ID. One audit line.
  - The same race as for approvals stays (ADR 0010). hc has no send on a condition (`docs/interfaces.md:36`).
- **Size:** M.
- **Depends on:** the probe, and a new ADR.
- **Done when:** one Stop sends one interrupt and shows its result. A retry with the same request ID sends nothing. A turn that changed is refused.

### 4. Time in state and progress on each row

- **What the user does:** each agent row shows how long the agent has been in its state, for example "blocked 40 min". It also shows the progress that hc reports.
- **Why:** the user sees at a glance which agent waits longest.
- **Changes:** the daemon adds `status_ms` and `progress` to the agent shape. The phone shows them.
- **Security:** no rule change. No content.
- **Size:** S.
- **Depends on:** nothing.
- **Done when:** a row shows the time since the last status change, and the time matches `hc get`.

### 5. Deny with a reason

- **What the user does:** after Deny, the message box gets the focus with the hint "Tell the agent why". The user types the reason and taps Send.
- **Why:** a bare Deny leaves the agent idle with no guidance.
- **Changes:** phone only.
  - A probe of the Codex state after `esc`, so that Send turns on for Codex too.
- **Security:** no rule change. It is two user actions: one key, then one message (SR-6).
- **Size:** S.
- **Depends on:** the Codex probe.
- **Done when:** after Deny, Send stays off while the agent is blocked, then sends once.

### 6. Saved replies and a kept draft

- **What the user does:** chips above the message box hold the user's own short replies, for example "continue" or "run the tests". A draft stays when the user leaves the screen.
- **Why:** typing is the slow part on a phone.
- **Changes:** phone only. The phone stores the saved replies. The draft stays in memory.
- **Security:** no rule change. The stored text is the user's own text, not transcript text (SR-12).
- **Size:** S.
- **Depends on:** nothing.
- **Done when:** a chip puts its text in the box, and the draft stays after a visit to another screen.

### 7. Catch up from the last read message

- **What the user does:** the transcript opens at the first message that the user has not seen, with a mark. "Latest" jumps to the end.
- **Why:** after a meeting, the user reads only what is new.
- **Changes:**
  - The daemon gives each transcript item a stable ID, for example the record offset and the block number. Time stamps cannot do this, because the items of one record share one time stamp.
  - The phone stores, for each agent, the ID of the last item that showed on the screen. It stores no text.
  - A new session file resets the mark.
- **Security:** no rule change. The phone stores only IDs (SR-12).
- **Size:** M.
- **Depends on:** the item ID in the transcript API.
- **Done when:** the mark sits after the last item that showed, and a new session opens at the end.

### 8. Search

- **What the user does:** the user filters agents by name. The user finds a word in the loaded transcript.
- **Why:** with 10 agents, the list is long. A long transcript is slow to scroll.
- **Changes:** phone only, in memory. The search says which pages it covers.
- **Security:** no rule change.
- **Size:** S.
- **Depends on:** nothing.
- **Done when:** a match scrolls into view, and "no match" names the loaded range.

### 9. Full tool step details

- **What the user does:** in an open fold, the user taps a tool step. The phone shows its full command, edit or output.
- **Why:** the cut at 2000 characters often hides the part that matters.
- **Changes:** a daemon endpoint returns one transcript item in full, up to a size cap. The phone shows it on request.
- **Security:** no rule change. The text is transcript text, in memory only.
- **Size:** M.
- **Depends on:** the item ID from feature 7.
- **Done when:** the detail shows the full text up to the cap, and says when it is cut.

### 10. Typed answers and Codex questions (needs a security change)

- **What the user does:** the user picks "Type something" and types the answer. The user also answers Codex questions.
- **Why:** these prompts now say "Answer this on the Mac".
- **Changes:**
  - Probes of each prompt screen, for Claude Code and Codex.
  - Two confirmed steps. First, one key picks the option. Then the daemon reads the screen again and finds the text field. Second, one text goes into the field.
- **Rule:** free-text answers and Codex questions are out of scope (`docs/product.md:68`, ADR 0021).
- **Risk:** the text lands in the wrong field, because the screen changed between the two steps.
- **Safeguard:** each step has its own request ID, prompt hash and turn check. The daemon refuses the text step if the text field is not on the screen.
- **Size:** L.
- **Depends on:** the probes, and a new ADR.
- **Done when:** a typed answer reaches the right field once. An unknown screen stays view-only.

### 11. Helpers under their parent agent

- **What the user does:** the user sees which helper agents belong to a parent agent. The user opens the helper that needs attention.
- **Why:** a council or a large task starts several helpers. The flat list hides which task they belong to.
- **Changes:** the daemon adds the parent address to the agent shape, from `sub_agents`. The phone groups the rows.
- **Security:** no rule change. Read-only.
- **Size:** S to M.
- **Depends on:** nothing. The daemon already reads `sub_agents`.
- **Done when:** a helper shows under its parent, and a blocked helper shows on the parent row.

### 12. Start an agent from a launcher list (needs a security change)

- **What the user does:** in a space, the user starts Claude Code or Codex from a fixed list, with a first prompt.
- **Why:** new work starts without the Mac.
- **Changes:**
  - A list of launchers in the daemon config on the Mac. Each launcher fixes the program and its arguments.
  - `POST /v1/spaces/{id}/agents`, and `terminal.run` in the space folder from `herdr-app ls`, with the space label.
  - The first prompt goes in as a message, as literal text.
- **Rule:** SR-2, and "Start or stop of agents" is out of scope (`docs/product.md:70`).
- **Risk:** the phone starts programs on the Mac.
- **Safeguard:**
  - Only launchers that the Mac config names. No program, argument or folder comes from the phone.
  - A cap on live agents.
  - A durable dispatch record for each request ID, so a retry never starts a second agent. An unknown outcome shows as unknown.
  - One audit line.
- **Size:** L.
- **Depends on:** a new ADR.
- **Done when:** a launch starts one agent in the right space. A launcher that is not in the config is refused. A retry starts nothing.

### 13. Send an image (needs a security change)

- **What the user does:** the user adds a screenshot or a photo to a message.
- **Why:** a visual bug that the user sees on the phone needs no description.
- **Changes:** a daemon endpoint, `terminal.put`, then a message with the path where the file landed.
- **Rule:** SR-2 (the phone cannot write files). SR-6 needs a line for one file and one message from one user action.
- **Risk:** an unwanted file on the Mac, or a full disk.
- **Safeguard:**
  - The daemon decodes the image and refuses a file that does not decode.
  - Caps on size, dimensions and the number of files. The daemon removes old files.
  - hc picks the path. Nothing runs the file.
  - The audit line has the size only.
- **Size:** M.
- **Depends on:** a new ADR.
- **Done when:** an image arrives once next to the agent. A file that is not an image is refused.

### 14. End an agent (needs a security change)

- **What the user does:** the user ends a finished or stuck agent after a confirm step.
- **Why:** a stuck session holds a tab and a context.
- **Changes:** `terminal.kill` and a route.
- **Rule:** SR-2, and "Start or stop of agents" is out of scope.
- **Risk:** a lost session, or the end of the wrong terminal.
- **Safeguard:**
  - A confirm step.
  - The daemon checks the address and the state again just before it ends the agent. It never ends a `working` agent. Stop the turn first.
  - A durable dispatch record and one audit line.
- **Size:** M.
- **Depends on:** feature 3, and a new ADR.
- **Done when:** an ended agent leaves the list. A working agent cannot be ended.

## 6. Testing strategy

- **Daemon changes** (features 3, 4, 7, 9 to 14): Go tests with the fake hc in `daemon/fake_test.go`. Run `cd daemon && go test ./...`.
- **Phone logic** (features 2, 5 to 8): JVM tests, as in `TextsTest`. Run the app test command in `AGENTS.md`.
- **Alerts and screens** (features 1, 2, 5): the on-device checks over `adb` in `docs/testing.md`.
- **New hc behaviour** (features 3, 5, 10, 12 to 14): run a probe agent for each new method or screen first. `docs/testing.md` describes the probes.

## 7. Constraints and open questions

- **Constraint:** each feature keeps SR-12 (transcript text in memory only).
- **Constraint:** each feature keeps SR-6 (one user action, at most one key or one message), except feature 13. Feature 13 needs an amendment to SR-6.
- **Open (non-blocking):** the RPC method for an interrupt. Feature 3 probes it first.
- **Open (non-blocking):** the Codex state after `esc`. Feature 5 probes it first.
- **Open (non-blocking):** iOS alerts with the app closed need the paid Apple program. Feature 1 on iOS works only where iOS alerts work.
- **Ready:** every feature is ready for its own plan. None is blocked.

## 8. For the implementer

- Take one feature at a time, through its own `/council plan`.
- Before you start, check that the references in section 2 still match the code.
- For features 3, 10 and 12 to 14, write the ADR first.
- For features 3, 5 and 10, run the probe before you write code.

## 9. Verification status

- **Inspected:** `herdr-remote` at `247206a`. Each reference in section 2 was checked against the code.
- **From the CLI only:** `hc interrupt` and `status_ms`. The RPC method for an interrupt is not verified.
- **Agent check:** both agents checked the draft. This final list applies their checks. The agents did not check the final text.

# Plan: the next features for the mobile app

## 1. Outcome

- **Result:** a ranked list of 14 features for the herdr-remote mobile app. Each feature has a description, its value, what it changes, its security effect and a size.
- **Theme:** decide and redirect an agent in one visit to the phone. Today the phone can see a block and press one key. To redirect an agent, the user must go back to the Mac.
- **Order:** value away from the desk first, then fit with the security model, then size.
  - Features 1, 2 and 4 to 9 need no change to a security rule. Most use the 11 routes that exist.
  - Feature 3 (stop the current turn) needs a security change, but it has the highest value of that group. So it ranks third.
  - Features 10 to 14 need a security change. Each names the rule, the risk and the safeguard.
- **Practical effect:** the first five features remove most trips to the Mac for a finished or a blocked agent. Each feature is a separate piece of work with its own plan.

## 2. Current behaviour

Revision inspected: `herdr-remote` on `main` at `247206a`.

- **Alerts:** the watch service posts one alert for each agent.
  - A blocked alert shows the prompt and has Approve and Deny when the prompt is actionable (`android/app/src/main/java/dev/herdrremote/watch/WatchService.kt:288-314`).
  - A finished alert shows only the agent name and its title, with no actions (`WatchService.kt:316-322`).
  - Alert actions need an unlocked phone (`setAuthenticationRequired(true)`, ADR 0019). A locked phone shows only the agent name.
- **Decisions:** Deny presses `esc` (`daemon/actions.go:493`). After `esc`, Claude Code goes idle (`docs/interfaces.md:116`).
- **Messages:** up to 4000 characters (`daemon/actions.go:21`). The daemon refuses a message to a blocked agent (`daemon/actions.go:651-655`). The draft is plain `remember` state and is lost when the user leaves the screen (`android/.../ui/AgentScreen.kt:109`).
- **Questions:** the phone answers Claude Code questions with one key for each option. "Type something", Codex questions and login prompts are view-only: "Answer this on the Mac" (`AgentViewModel.kt:241`, ADR 0021).
- **Lists:** the rows sort by state only: blocked, error, done, working, idle (`android/.../ui/ListScreens.kt:48`).
- **Agent facts:** the agent shape is `{address, space, tab, where, app, state, kind, msg, turn, title}` (`docs/api.md:40`). The daemon `Status` struct drops `progress` (`daemon/hc.go:76-81`). `hc get` also returns `status_ms`, the time of the last status change.
- **Transcript:** pages of `{role, kind, text, ts}` (`docs/api.md:44`). Tool calls and results are cut at 2000 characters (`daemon/transcript.go:21`). The phone loads older pages on request (`AgentViewModel.kt:186`).
- **hc calls:** the daemon sends only `hello`, `terminal.list`, `terminal.get`, `terminal.read`, `events.subscribe`, `terminal.send` and `terminal.keys` (`daemon/hc.go:44-52`, ADR 0003). hc also has `terminal.run`, `terminal.kill` and `terminal.put` (`docs/interfaces.md:39`) and the CLI command `hc interrupt`, which stops the current turn but not the process.
- **Request IDs:** the daemon keeps them for 10 minutes, so a retry never repeats a key or a message (`daemon/actions.go:20`).
- **Sub-agents:** a sub-agent takes the placement of its parent (`daemon/state.go:315-318`).

## 3. Decisions

| # | Decision | Why | Strongest rejected alternative |
|---|---|---|---|
| 1 | Rank by value away from the desk, then security fit, then size. | These are the brief's criteria, in its order. | Rank all rule-safe features first. It lost because it puts a high-value interrupt below small comforts. |
| 2 | Put Reply on the finished alert first. | Every finished agent today costs an unlock, a navigation and a typed message. ADR 0019 already allows actions on an unlocked alert. | Open the app on a tap. That is today's behaviour. |
| 3 | Put Stop the current turn third, as a security change. | A wrong agent in auto mode can work for a long time. A message only queues behind the work. | Keep interrupts out of scope. It lost on value. |
| 4 | Do not list action receipts. | The app already says "outcome unknown, check the transcript". The transcript shows whether a message arrived. | A receipt lookup for each request ID. Deferred: little value over the transcript. |
| 5 | Put the four features that start, end or write on the Mac last. | They cross SR-2, the rule that the phone cannot start programs or write files. | Leave them out. The user asked to list them, marked. |
| 6 | Each feature gets its own plan, and a new ADR when it changes a rule. | `AGENTS.md` asks for an ADR for each design decision. | One plan for all features. Too large to review. |

## 4. Scope and acceptance criteria

- **Delivered:** the ranked feature list in section 5.
- **Excluded:** code, the iOS parity plan, store releases, and the open bugs in `docs/operations.md`.
- **Done when:**
  1. Each feature has a name, a description, its value, its changes, its security effect, a size and its dependencies.
  2. Each feature that crosses a security rule names the rule, the risk and the safeguard.
  3. Each claim about today's code has a file reference that matches `247206a`.

## 5. Feature list

Size: S is a few days on one platform, M is about a week, L is more. The sizes are for Android. iOS parity adds to each.

### 1. Reply from the finished alert

- **What the user does:** the user expands a "finished" alert and reads the agent's last answer. The user types a reply in the alert and sends it, without opening the app.
- **Why:** this is the most frequent trip to the Mac.
- **Changes:** phone only. On a finish, the watch service gets the last transcript page, as it gets the prompt for a block. The alert gets a Reply action (Android `RemoteInput`) that calls `POST /v1/terminals/{t}/message`.
- **Security:** no rule change. The action needs an unlocked phone, as Approve does. A locked phone shows only the name. An amendment to ADR 0019 records the new action.
- **Size:** S. **Depends on:** nothing.
- **Done when:** a reply from the alert arrives as one prompt, once, and a locked phone shows no answer text.

### 2. Deny with a reason

- **What the user does:** after Deny, the message box opens with the focus. The user types why, and Send works when the agent is no longer blocked.
- **Why:** a bare Deny leaves the agent idle with no guidance.
- **Changes:** phone only.
- **Security:** no rule change. It is two user actions: one key, then one message (SR-6).
- **Size:** S. **Depends on:** nothing.
- **Done when:** after Deny, Send stays off until the agent leaves `blocked`, then sends once.

### 3. Stop the current turn (needs a security change)

- **What the user does:** the user taps Stop on a working agent and confirms. The phone shows what hc reports.
- **Why:** an agent on a wrong path in auto mode wastes time, and a message waits behind the work.
- **Changes:** `POST /v1/terminals/{t}/interrupt` on the daemon, the hc interrupt method in `allowedMethods`, a Stop key on the phone.
- **Rule:** "Interrupts" is out of scope (`docs/product.md:69`). ADR 0003 limits the hc calls.
- **Risk:** lost work in progress, or a stop on the wrong turn.
- **Safeguard:** the agent must be `working`, the turn must match, the Mac-idle guard applies, one request ID, and an audit line.
- **Size:** M. **Depends on:** a new ADR, and a probe of the hc interrupt method.
- **Done when:** one Stop stops one turn, a retry with the same request ID stops nothing more, and a stale turn is refused.

### 4. Time in state and progress on each row

- **What the user does:** each agent row shows how long it has been in its state ("blocked 40 min") and the progress that hc reports.
- **Why:** the user sees at a glance which agent waits longest.
- **Changes:** the daemon adds `status_ms` and `progress` to the agent shape. The phone shows them.
- **Security:** no rule change. No content.
- **Size:** S. **Depends on:** nothing.
- **Done when:** a row shows the time since the last status change, and it matches `hc get`.

### 5. Next blocked agent after a decision

- **What the user does:** after Approve, Deny or an answer, the phone offers the next blocked agent. The user works through the queue without the list.
- **Why:** several blocked agents today cost a navigation each.
- **Changes:** phone only, from the agent list it has.
- **Security:** no rule change. One decision for each tap. No bulk approval.
- **Size:** S. **Depends on:** nothing.
- **Done when:** the next agent is the first blocked one in list order, and it shows its own prompt fresh.

### 6. Catch up from the last read message

- **What the user does:** the transcript opens at the first message that the user has not seen, with a mark. "Latest" jumps to the end.
- **Why:** after a meeting, the user reads only what is new.
- **Changes:** the phone stores, for each agent, the time stamp of the last item it showed. It stores no text (SR-12). A new session resets the mark.
- **Security:** no rule change. The phone stores only a time stamp.
- **Size:** M. **Depends on:** a way to tell that the session changed.
- **Done when:** the mark sits after the last seen item, and a new session starts at the end.

### 7. Saved replies and a kept draft

- **What the user does:** chips above the message box hold the user's own short replies ("continue", "run the tests"). A draft stays when the user leaves the screen.
- **Why:** typing is the slow part on a phone.
- **Changes:** phone only. The replies are stored on the phone. The draft stays in memory.
- **Security:** no rule change. The stored text is the user's own, not transcript text.
- **Size:** S. **Depends on:** nothing.
- **Done when:** a chip puts its text in the box, and the draft survives a visit to another screen.

### 8. Search

- **What the user does:** the user filters agents by name, and finds a word in the loaded transcript.
- **Why:** with 10 agents, the list is long. A long transcript is slow to scroll.
- **Changes:** phone only, in memory. The search says which pages it covers.
- **Security:** no rule change.
- **Size:** S. **Depends on:** nothing.
- **Done when:** a match scrolls into view, and "no match" names the loaded range.

### 9. Full tool step details

- **What the user does:** in an open fold, the user taps a tool step to see its full command, edit or output.
- **Why:** 2000 characters often hide the part that matters.
- **Changes:** a daemon endpoint that returns one transcript item in full, with a size cap. The phone shows it on demand.
- **Security:** no rule change. The text is transcript text, in memory only.
- **Size:** M. **Depends on:** a stable way to name one item.
- **Done when:** the detail shows the full text up to the cap, and says when it is cut.

### 10. Typed answers and Codex questions (needs a security change)

- **What the user does:** the user picks "Type something" and types the answer, or answers a Codex question.
- **Why:** these prompts now say "Answer this on the Mac".
- **Changes:** probes of each screen, then a decision with one key and one text under one request ID.
- **Rule:** out of scope (`docs/product.md:68`), SR-6, ADR 0021.
- **Risk:** the text lands in the wrong field.
- **Safeguard:** the prompt hash and turn checks, and the daemon reads the screen for the text field before it sends. Else it refuses.
- **Size:** L. **Depends on:** the probes and a new ADR.
- **Done when:** a typed answer reaches the right field once, and an unknown screen stays view-only.

### 11. Helpers under their parent agent

- **What the user does:** the user sees which helper agents belong to a parent, and opens the one that needs attention.
- **Why:** a council or a large task starts several helpers. The flat list hides which task they belong to.
- **Changes:** the daemon adds the parent to the agent shape. The phone groups the rows.
- **Security:** no rule change. Read-only.
- **Size:** M. **Depends on:** the parent fact from hc.
- **Done when:** a helper shows under its parent, and a blocked helper shows on the parent row.

### 12. Start an agent from a launcher list (needs a security change)

- **What the user does:** in a space, the user starts Claude Code or Codex from a fixed list, with a first prompt.
- **Why:** new work starts without the Mac.
- **Changes:** a list of launchers in the daemon config, `POST /v1/spaces/{id}/agents`, and `terminal.run` with the space label.
- **Rule:** SR-2 and "Start or stop of agents" out of scope (`docs/product.md:70`).
- **Risk:** the phone starts programs on the Mac.
- **Safeguard:** only the launchers that the Mac config names, no command text from the phone, a cap on live agents, one request ID, an audit line.
- **Size:** L. **Depends on:** a new ADR.
- **Done when:** a launch starts one agent in the right space, and a launcher that is not in the config is refused.

### 13. Send an image (needs a security change)

- **What the user does:** the user adds a screenshot or a photo to a message.
- **Why:** a visual bug that the user sees on the phone needs no description.
- **Changes:** a daemon endpoint, `terminal.put`, then a message with the path where the file landed.
- **Rule:** SR-2 (the phone cannot write files).
- **Risk:** a file of any kind on the Mac.
- **Safeguard:** images only (checked by their first bytes), a size cap, hc picks the path, an audit line with the size only.
- **Size:** M. **Depends on:** a new ADR.
- **Done when:** an image arrives once next to the agent, and a non-image is refused.

### 14. End an agent (needs a security change)

- **What the user does:** the user ends a finished or stuck agent after a confirm step.
- **Why:** a stuck session holds a tab and a context.
- **Changes:** `terminal.kill` and a route.
- **Rule:** SR-2 and "Start or stop of agents" out of scope.
- **Risk:** a lost session.
- **Safeguard:** a confirm step, never while `working` (stop the turn first), an audit line.
- **Size:** M. **Depends on:** feature 3 and a new ADR.
- **Done when:** an ended agent leaves the list, and a working agent cannot be ended.

## 6. Testing strategy

- **Daemon changes** (features 3, 4, 9 to 14): Go tests with the fake hc in `daemon/fake_test.go`. Run `cd daemon && go test ./...`.
- **Phone logic** (features 2, 5 to 8): JVM tests like `TextsTest`. Run the app test command in `AGENTS.md`.
- **Alerts and screens** (features 1, 2, 5): the on-device checks over `adb` in `docs/testing.md`.
- **Security changes:** probe agents, as in `docs/testing.md`, for each new hc method before the code uses it.

## 7. Constraints and open questions

- **Constraint:** each feature keeps SR-6 (one user action, at most one key or one message) and SR-12 (transcript text in memory only).
- **Open (non-blocking):** the name of the hc RPC method for an interrupt. Feature 3 probes it first.
- **Open (non-blocking):** iOS alerts with the app closed need the paid Apple program. Feature 1 on iOS works only where iOS alerts work.
- **Open (non-blocking):** how the phone tells that a session changed, for feature 6.
- All features are ready for their own plan. None is blocked.

## 8. For the implementer

- Take one feature at a time, through its own `/council plan`.
- Before you start, check that the references in section 2 still match the code.
- For features 3 and 10 to 14, write the ADR first.

## 9. Verification status

- **Inspected:** `herdr-remote` at `247206a`. Each reference in section 2 was checked against the code.
- **From the CLI only:** `hc interrupt` and `status_ms`. The RPC method for an interrupt is not verified.
- **Agent check:** this draft goes to both agents in round 2.

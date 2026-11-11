# ADR 0022: The skill picker reads the transcript's skill list

- **Status:** Accepted, 2026-11-12. Amended the same day: the skill folders add the skills that only the user may start.
- **Decision:** 22. It is not in the decision table of the technical design.

## Context

- The user asked to pick a skill on the phone, as with "/" in Claude Code.
- A message that starts with `/name` runs that skill in Claude Code ([ADR 0012](0012-messages.md)).
- Claude Code writes the agent's skills to its transcript, in `skill_listing` records with a `names` list ([interfaces](../interfaces.md#claude-code)).
- That list is for the model. It leaves out a skill with `disable-model-invocation: true`, such as `council`. Only the user can start such a skill.

## Decision

- `GET /v1/terminals/{t}/skills` returns `{skills[]}` for a Claude Code agent. Other agents have none. The list has:
  1. The names from the `skill_listing` records of the agent's transcript file. An initial record (`isInitial: true`) starts the list again. A later record adds names. The file is the one that the transcript uses ([ADR 0009](0009-transcripts.md)).
  2. Then each skill folder (a subfolder with a `SKILL.md`) in `~/.claude/skills` and in `.claude/skills` under the agent's current folder. The folder name is the command. This adds the skills that only the user may start.
- The phone lists the skills when the draft starts with "/" and has no space. The list filters as the user types. A tap puts `/name ` in the draft.
- A "/" key beside Send starts an empty draft with "/".
- The phone gets the list once for each agent screen.

## Alternatives

| Alternative                                  | Why it lost                                                                     |
| -------------------------------------------- | ------------------------------------------------------------------------------- |
| Only the skill folders                       | Plugin skills are not in them. The transcript has the list that the agent uses. |
| Only the transcript's list                   | It leaves out `council` and the other skills that only the user may start.      |
| Add built-in commands (`/compact`, `/clear`) | Not asked for.                                                                  |

## Consequences

- The names reach the phone. Descriptions do not.
- A Codex agent shows "No skills match."
- A plugin skill that only the user may start is not in the list.

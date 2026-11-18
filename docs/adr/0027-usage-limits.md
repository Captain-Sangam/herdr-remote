# ADR 0027: Read usage limits from local files

- **Status** Accepted, 2026-11-18.
- **Decision** 27.

## Context

The phone needs the usage and reset time for Claude Code and Codex windows.

## Decision

- Serve `GET /v1/usage` on the paired API port. Use the same mutual TLS checks as the other GET requests.
- Read Claude Code usage from the JSON file that its status line writes. The `claude_usage_file` setting defaults to `~/.cache/herdr-slack/claude-usage.json`. Read at most 64 KiB.
- Read at most the last 1 MiB of the newest Codex rollout by modification time. Use the last `token_count` event with non-null `rate_limits` in that tail. Resolve the rollout path inside `~/.codex/sessions/` before reading it.
- Read both sources on request. The daemon makes no network calls to either provider and reads no credentials. The response carries only window names, percentages, reset times, the reading time and the optional Codex plan. The daemon writes no usage data to logs or the audit log.

## Consequences

- Claude Code numbers update only while a Claude Code session draws its status line. A refresh can return the same reading.
- A source with no usable reading returns `null`. The other source can still return data.
- Codex usage reflects the newest rollout. It can be absent if that rollout has no rate-limit event in its tail.

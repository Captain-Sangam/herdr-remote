# API

`herdr-remoted` serves a small HTTPS API to the phone. This page gives the endpoints, the shapes, the errors and the timeouts.

- **Related:** [architecture](architecture.md), [ADR 0004](adr/0004-two-tls-listeners.md) (TLS), [ADR 0005](adr/0005-http-sse-and-streams.md) (HTTP, SSE and streams).

## Ports and TLS

| Port | Listener   | Open                                | Client check                                                                         |
| ---- | ---------- | ----------------------------------- | ------------------------------------------------------------------------------------ |
| 7766 | Paired     | Always                              | The client SPKI SHA-256 must be in the device store. Each request looks it up again. |
| 7767 | Enrollment | Only while a pairing window is open | One well-formed P-256 certificate. The token, bound to that SPKI, authorizes.        |

- Both listeners share one P-256 self-signed server key. The phone pins its SPKI SHA-256.
- Both listeners use TLS 1.3 only, `RequireAnyClientCert` and no session tickets.
- All client checks run in `VerifyConnection`, which runs on every handshake, also on a resumption.
- The listeners use TCP keepalive (the Go default, 15 s).

## Endpoints

All endpoints except `/v1/enroll` are on port 7766. Bodies are JSON unless the table says otherwise.

| Method and path                                    | Request                                           | Response                                                                                                             |
| -------------------------------------------------- | ------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `POST /v1/enroll` (7767 only)                      | `{token, name}`                                   | 200 `{device_id}` after Allow. 403 on Deny, a wrong token, or a token bound to another key. 410 on expiry or cancel. |
| `GET /v1/hello`                                    | none                                              | `{daemon_version, server_time_ms, endpoints[]}`                                                                      |
| `GET /v1/spaces`                                   | none                                              | `[{id, name, agents[]}]`                                                                                             |
| `GET /v1/usage`                                    | none                                              | `{claude, codex}`. A source with no data is `null`.                                                                  |
| `GET /v1/agents`                                   | none                                              | `[agent]`                                                                                                            |
| `GET /v1/terminals/{t}`                            | none                                              | `agent` plus `{cols, rows}`                                                                                          |
| `GET /v1/events`                                   | none                                              | SSE. See [Events](#events).                                                                                          |
| `GET /v1/terminals/{t}/live`                       | none                                              | Chunked `application/octet-stream` from `hc attach --observe --history 500`.                                         |
| `GET /v1/terminals/{t}/transcript?cursor=<opaque>` | none                                              | `{source, hint?, model?, items[], next_cursor}`. `source` is `claude`, `codex` or `terminal`.                        |
| `GET /v1/terminals/{t}/prompt`                     | none                                              | `{app, kind, msg, block, question?, options?, turn, prompt_hash, actionable, reason?}`                               |
| `GET /v1/terminals/{t}/skills`                     | none                                              | `{skills[]}`: a Claude Code agent's skill names, from its transcript and its skill folders. Empty for other agents.  |
| `POST /v1/terminals/{t}/decision`                  | `{request_id, decision, key?, turn, prompt_hash}` | `{result}`: `applied`, `sent_unconfirmed`, `uncertain` or `outcome_unknown`                                          |
| `POST /v1/terminals/{t}/message`                   | `{request_id, text}`                              | `{result}`: `sent` or `outcome_unknown`                                                                              |

## Shapes

- **`agent`:** `{address, space, tab, where, app, state, kind, msg, turn, title}`.
  - `state` is the hc status word: `idle`, `working`, `done`, `blocked` or `error`.
  - `kind` and `msg` are set only when the agent is blocked.
- **Usage source** has `as_of`, an optional `plan`, and `windows`.
  - `as_of` is the time of the source reading in Unix milliseconds.
  - Each window has `name`, `used_percent` and `resets_at`. Names include `5h` and `7d`. `resets_at` is in Unix milliseconds.
  - The daemon reads both sources on each request and returns any available data. It writes no audit entry for this GET.
- **Transcript page:** `model` is the last model that the page names: `message.model` in Claude Code, `turn_context` in Codex. Terminal text has none.
- **Transcript item:** `{role, kind, text, ts}`.
  - `kind` is `text`, `tool_call` or `tool_result`.
  - Tool calls and tool results are cut to 2000 characters. Thinking blocks are not sent.
  - `ts` is milliseconds since the epoch.
- **`request_id`:** a UUIDv7 that the phone makes for each user action.
  - The daemon refuses an ID more than 5 minutes from its clock (`expired_request`).
  - The phone sets the UUIDv7 time from its own clock plus the offset to `server_time_ms` from `/v1/hello`.
- **`decision`:** `approve` or `deny` for a permission prompt. `answer` for a Claude Code question, with `key` set to one of the prompt's `options` ([ADR 0021](adr/0021-answer-questions.md)).
- **`question`:** a question's `block` above its options. The phone shows it, and the options as keys, so no option shows twice.
- **`options`:** a question's keys, `[{key, label, detail?}]`.
  - `key` is a digit from `1` to `9`. After a multi-select, the last key is `tab`, with the label `Next`.
  - `label` is the option's text as the screen shows it.
  - `detail` is the description under the option. It is empty when the screen repeats the label.
- **`text`:** at most 4000 characters. The daemon removes C0 and C1 control characters, but keeps `\n` and `\t`.

[Architecture: approvals](architecture.md#35-approvals) gives the rules behind `prompt_hash` and each result.

## Events

`GET /v1/events` is a Server-Sent Events stream.

| Event           | Data               | When                                                      |
| --------------- | ------------------ | --------------------------------------------------------- |
| `snapshot`      | All agents         | On connect, and after an hc `missed` notification.        |
| `agent`         | One `agent`        | Each change of an agent.                                  |
| `agent_removed` | `{address}`        | A terminal finishes, is removed, or stops being an agent. |
| `: ka`          | A comment, no data | Every 10 s, so the phone can detect a dead connection.    |

## Errors

Errors are `{error, reason}`.

| Status | Use                                                                                                                                                                                                      |
| ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 400    | Bad target, bad body, text too long, an answer `key` that is not an option.                                                                                                                              |
| 401    | Unknown or revoked device (after the handshake).                                                                                                                                                         |
| 404    | No such terminal.                                                                                                                                                                                        |
| 409    | Guard failure. `reason` is `stale`, `changed`, `already_answered`, `mac_in_use`, `not_actionable`, `blocked`, `not_agent`, `cursor`, `expired_request` or `live_limit`. `mac_in_use` also has `idle_ms`. |
| 429    | More than 2 actions (decisions and messages) in one second from one device.                                                                                                                              |
| 503    | hc unavailable, or an audit write failed.                                                                                                                                                                |

## Timeouts on the phone

| Call         | Read timeout | Notes                                                                                   |
| ------------ | ------------ | --------------------------------------------------------------------------------------- |
| `/v1/events` | 30 s         | Then the phone reconnects, calls `/v1/hello`, and reopens live views.                   |
| Live view    | None         | An idle terminal sends no bytes. The phone reopens each live view when SSE reconnects.  |
| `/v1/enroll` | 130 s        | The call long-polls until Allow, Deny or the 120 s expiry.                              |
| Actions      | 150 s        | `retryOnConnectionFailure(false)`. The daemon can wait up to 120 s for `terminal.keys`. |
| Other calls  | 15 s         |                                                                                         |

## Pairing link

The QR code holds this link:

```text
herdr-remote://pair?v=1&pin=<b64url SPKI sha256>&ep=<addr>,<addr>&p=7766&ep_port=7767&t=<token>&exp=<unix>
```

| Field     | Content                                                                                                       |
| --------- | ------------------------------------------------------------------------------------------------------------- |
| `v`       | The link version, `1`.                                                                                        |
| `pin`     | The base64url SHA-256 of the server SPKI.                                                                     |
| `ep`      | The private IPv4 addresses of active interfaces, any `100.64.0.0/10` address, and the MagicDNS name if known. |
| `p`       | The paired port, 7766.                                                                                        |
| `ep_port` | The enrollment port, 7767.                                                                                    |
| `t`       | The one-time token: 32 random bytes, kept in memory only.                                                     |
| `exp`     | The expiry, in Unix seconds.                                                                                  |

The phone refuses a link with a bad version, a past expiry or a missing field.

## Admin socket

The CLI talks to the daemon through `admin.sock`: one JSON request line and one reply. [Operations: admin CLI](operations.md#admin-cli) lists the commands.

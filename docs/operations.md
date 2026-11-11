# Operations

This page tells how to run herdr-remote day to day: the admin CLI, the logs, a revoke, and what to do when something fails.

- **Related:** [deployment](deployment.md), [architecture: security and privacy model](architecture.md#6-security-and-privacy-model).

## Admin CLI

The CLI talks to the running daemon through `admin.sock`. Each command returns within 1 s.

| Command                                      | Result                                                                                                  |
| -------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `herdr-remoted serve`                        | Runs the daemon. The LaunchAgent uses it.                                                               |
| `herdr-remoted pair --json`                  | Opens a pairing window. Returns `{matrix, expires_at, text}`.                                           |
| `herdr-remoted pair status --json`           | `{state, device_name?, code?}`. `state` is `none`, `open`, `pending`, `allowed`, `denied` or `expired`. |
| `herdr-remoted pair allow`, `deny`, `cancel` | Changes the state of the pairing window.                                                                |
| `herdr-remoted devices ls [--json]`          | `[{id, name, added}]`                                                                                   |
| `herdr-remoted devices rm <id>`              | Revokes the device.                                                                                     |
| `herdr-remoted version`                      | Prints the daemon version.                                                                              |

- Same-user programs can run these commands. That is inside the trust boundary ([ADR 0018](adr/0018-trust-boundary.md)).
- WARNING: Do not let an agent run `pair allow` by itself. Allow gives a phone full access to the agents.

## Revoke a phone

Revoke a phone that is lost, sold or no longer in use.

1. Open the overlay: `herdr-app do mod.open --name herdr-remote`.
2. Click **Revoke** next to the phone.

Or use the CLI:

```sh
herdr-remoted devices ls
herdr-remoted devices rm <id>
```

- The daemon marks the device as revoked in memory, closes its connections and live views, and then writes `devices.json` ([ADR 0007](adr/0007-revocation.md)).
- A failed write keeps the mark and returns the error. So a retry writes again.
- A revoke does not open the enrollment port.
- On the phone tested, the Mac dropped the phone in 0.21 s. The phone then could pair again.
- A revoked phone keeps its last data on the screen with "Not connected" and retries without end. It does not say that the Mac removed it.

## Audit log

`audit.log` in the state directory has one JSON line for each pairing event, revocation, decision, refusal and message ([ADR 0017](adr/0017-logging-and-audit.md)).

**Line:** `{ts, device_id, action, state, terminal, turn, request_id, decision, prompt_hash, length, idle_ms, result}`, with only the fields that apply.

| Field     | Content                                                                    |
| --------- | -------------------------------------------------------------------------- |
| `action`  | `enroll`, `allow`, `deny`, `revoke`, `decision`, `message` or `refusal`.   |
| `state`   | `dispatching` before a key or a send. Then a second line has the `result`. |
| `result`  | The result of a decision or a message. For a `refusal`, the reason.        |
| `length`  | For a message, the number of characters. Never the text or a hash of it.   |
| `idle_ms` | For a `mac_in_use` refusal, the measured idle time.                        |

- `dispatching` lines are fsynced before any key or send. If the write fails, the daemon presses no key and sends nothing.
- The daemon rebuilds request IDs and consumed markers from the last 10 minutes of `audit.log` and `audit.log.1` on start.
- The log rotates at 10 MiB to `audit.log.1`.

**Never logged anywhere:** tokens, pairing codes, keys, transcript text, terminal bytes, message text.

Commands to read the log:

```sh
cd ~/Library/Application\ Support/herdr-remote
tail -n 20 audit.log
grep -c . audit.log
grep '"action":"decision"' audit.log | tail
```

## Daemon log

- `daemon.log` holds the stdout and stderr of the daemon.
- The start line has the version, the port and the server pin.
- The daemon rotates it at 10 MiB.

## Checks after a Mac reboot

1. Check that the daemon runs:

   ```sh
   launchctl print gui/$(id -u)/dev.herdr-remote | grep state
   ```

2. Check that port 7766 is open and port 7767 is closed:

   ```sh
   nc -z 127.0.0.1 7766 && echo 7766 open
   nc -z 127.0.0.1 7767 || echo 7767 closed
   ```

3. Check that `audit.log` holds no secrets or content. Read some lines by eye.

## Troubleshooting

| Symptom on the phone or the Mac                                       | Cause                                                                                                          | Fix                                                                                                          |
| --------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| "Someone is using the Mac right now"                                  | The Mac had keyboard or mouse input in the last `mac_idle_seconds`.                                            | Wait, then tap again. The limit is 10 s of idle time.                                                        |
| Every approval says that the Mac is in use                            | A device, such as a mouse jiggler, sends input all the time.                                                   | Set `mac_idle_seconds` to 0 in `config.json`, then restart the daemon.                                       |
| A Codex agent shows terminal text and a `--no-daemon` hint            | Codex runs in daemon mode.                                                                                     | Set Herdr's Codex launcher to `["codex", "--no-daemon"]`. See [deployment](deployment.md#codex-transcripts). |
| A working agent shows only one screen of terminal text                | hc returns only the visible screen while an agent works or is blocked.                                         | None. The full text shows when the agent is idle.                                                            |
| "Answer this on the Mac"                                              | The prompt is a login, a Codex question, or from an unknown agent. Or Claude Code changed its question layout. | Answer it on the Mac. The phone answers permission prompts and Claude Code questions only.                   |
| "Outcome unknown. Check the transcript before you send again"         | The connection or the daemon stopped during the action.                                                        | Read the transcript. Send again only if the agent did not get the action.                                    |
| "Not connected to the Mac" banner                                     | The phone cannot reach any endpoint.                                                                           | Check Wi-Fi or Tailscale on both devices. The phone retries by itself.                                       |
| "Not connected" for good, after a revoke                              | The Mac revoked the phone.                                                                                     | Use Forget this Mac in Settings, then pair again.                                                            |
| The phone stopped connecting after the Mac got a new Wi-Fi address    | The phone learns addresses only while it is connected.                                                         | Use Tailscale, or pair again.                                                                                |
| No alerts                                                             | The watch service is off, not connected, or notifications are denied.                                          | Turn on "Notify me" in Settings. Allow notifications. Keep the battery-optimization exemption.               |
| `install.sh` stops: the mod folder "was not installed by this script" | The folder has no `.herdr-remote-owned` marker.                                                                | Move the folder away, then run `install.sh` again.                                                           |
| macOS asks to allow incoming connections                              | The firewall sees `herdr-remoted` for the first time.                                                          | Click Allow.                                                                                                 |
| A phone cannot connect to port 7767                                   | The port is open only during a pairing window.                                                                 | Click **Pair a phone** in the overlay first.                                                                 |

## Known issues

### Accepted after the code reviews

The two code reviews of 2026-11-09 accepted these issues (likelihood times severity):

- A reused process ID can map an agent to an old Claude session (1 × 5).
- A duplicate request in flight gets an empty reply after a refusal (2 × 3).
- A stalled enrollment connection can outlive the pairing window (2 × 4).
- An unpaired client can write refusal lines during a pairing window (2 × 3).
- The live handler waits for the process before it cancels it (1 × 3).
- `install.sh` loads the LaunchAgent before it checks the mod marker (1 × 2).
- A retry of a decision after the terminal ended gets 404 (2 × 2).
- An open space screen does not add new agents (5 × 2).
- Commit `0d14b24` does not build alone (history only).
- The title line above the prompt block has no line limit (1 × 6). The optional fix is `maxLines = 3`.

### Open

- A revoked phone does not say that the Mac removed it, and does not offer to pair again.
- Without Tailscale, a new Wi-Fi address on the Mac needs a new pairing.
- `scripts/format.sh` has shfmt checksums only for Apple silicon Macs and x86_64 Linux. Intel Macs and arm64 Linux need a line each.
- The repo has no git remote, so `ci.yaml` has not run on GitHub.
- The light mode of the app was not checked by eye on the phone.

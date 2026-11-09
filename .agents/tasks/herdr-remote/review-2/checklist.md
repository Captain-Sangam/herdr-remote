# Review checklist

One row per implementation step and per acceptance criterion of `TDD.md`. For each one, say delivered, partial, missing or drifted (delivered differently from the plan), with evidence.

## Steps

| Row | Promise (TDD section 5) | Where to look |
|---|---|---|
| S1 | Toolchain script, Go module, empty Compose app, .gitignore | `scripts/setup-android.sh`, `daemon/go.mod`, `android/`, `.gitignore` |
| S2 | Subprocess helper with fixed programs; three `hc connect` connections; typed calls only; no force, no caller, keys 1/y/esc; MacIdle; event fixtures | `daemon/run.go`, `daemon/hc.go`, `daemon/idle.go`, `daemon/testdata/` |
| S3 | State dir and modes; TLS settings per decision 4 with checks per listener; pairing window, token binding, code, long-poll; admin socket CLI; audit log rules | `daemon/tls.go`, `devices.go`, `pair.go`, `admin.go`, `audit.go`, `daemon.go`, `main.go` |
| S4 | Read API and SSE per section 3.4, `agent_removed`, keepalive; spaces per decision 16; error codes | `daemon/api.go`, `daemon/state.go` |
| S5 | Live stream via `hc attach --observe`, 2 per device, ends on disconnect, size or revoke | `daemon/live.go` |
| S6 | Three transcript sources in order, path rules, pages and cursor per section 3.6 | `daemon/transcript.go` |
| S7 | Approval guard per section 3.5 (order, marker, idle guard, fsync, one key, results, retention); messages per decision 12; UUIDv7 window; rate limit | `daemon/actions.go`, `daemon/audit.go` |
| S8 | Mod per section 3.8; install.sh steps 1 to 6 with marker | `mod/herdr-remote/`, `scripts/install.sh` |
| S9 | Keystore key, key manager, trust manager and hostname verifier pinning, TLS 1.3, endpoint fallback, Pair screen, BiometricPrompt gate, FLAG_SECURE | `android/app/src/main/java/dev/herdrremote/net/`, `MainActivity.kt`, `ui/` |
| S10 | Lists, SSE client (30 s timeout) and reducer, stale banner, foreground service and manifest, receivers, private notifications, de-dup | `android/.../watch/`, `AndroidManifest.xml`, `ui/` |
| S11 | Transcript tab, Live tab (WebView lock-down, flow control, reopen), prompt card and results, message box with no auto retry | `android/.../ui/`, `assets/xterm/live.html` |
| S12 | Acceptance run (no code) | As built "Checks run" and "Left for the user" |

## Acceptance criteria (TDD section 4)

| Row | Criterion | Where to look |
|---|---|---|
| C1 | LaunchAgent, handshake refusal on 7766, 7767 closed outside a window | install.sh, tls.go, pair.go |
| C2 | Overlay, QR, matching code, Allow, failure cases, second phone refused | mod, pair.go, Android Pair screen |
| C3 | Spaces, agents, status within 2 s | state.go, api.go, Android lists |
| C4 | Transcripts for Claude and `--no-daemon` Codex, three pages, labelled terminal text, hint | transcript.go, Android Transcript tab |
| C5 | Live view in color, about 1 s, recovers from resize, heavy output, network change | live.go, LiveView.kt, live.html |
| C6 | Approve and deny for command and edit prompts; every refusal case | actions.go, Android prompt card |
| C7 | Untagged message, at most once, blocked refusal, unknown outcome | actions.go, Android message box |
| C8 | Background alert within 5 s, restarts after reboot and update | WatchService, StartReceiver, manifest |
| C9 | Revoke ends streams within 2 s; later handshakes and resumptions fail | devices.go, tls.go, live.go, api.go |
| C10 | Tailscale over cellular | endpoints in pair.go, Android endpoint fallback |
| C11 | Gate after 5 minutes in the background | MainActivity.kt |
| C12 | Recents blank; lock screen generic | MainActivity.kt, WatchService |
| C13 | Audit log complete and free of secrets and content | audit.go and its callers |

## Changes no step explains
List anything in the diff that no step above accounts for.

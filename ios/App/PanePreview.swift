#if DEBUG
  import Core
  import SwiftUI

  /// Every Pane component on one screen, to compare by eye with design/viewer.html in both
  /// modes. Launch the app with `-panePreview` to see it.
  struct PanePreview: View {
    @State private var on = true
    @State private var name = "iPad"
    @State private var tab = 0
    @Environment(\.pane) private var c

    var body: some View {
      VStack(spacing: 0) {
        PaneHeader("~/spaces/herdr-remote", onBack: {}, online: true)
        PaneSubline {
          Text("3 agents")
          Text("·")
          Text("1 blocked").foregroundStyle(c.blocked)
        }
        PaneBanner(text: "Not connected to the Mac: last update 48 s ago")
        PaneTabs(tabs: ["transcript", "live"], selected: tab) { tab = $0 }
        ScrollView {
          VStack(alignment: .leading, spacing: PaneSpace.s) {
            PaneBox(title: "needs you (1)") {
              StatusRow(
                state: "blocked", title: "herdr-remote / main", meta: "claude-code · turn 41",
                detail: "approve Bash: touch x"
              ) {}
              StatusRow(state: "working", title: "niti / api", meta: "codex") {}
              StatusRow(state: "done", title: "email / codex", meta: "codex") {}
              StatusRow(state: "error", title: "web / dev", meta: "pi") {}
              StatusRow(state: "idle", title: "notes", meta: "claude-code") {}
            }
            PaneBox(title: "permission", color: c.blocked) {
              Text("! Bash: touch x").pane(PaneType.body.with(bold: true)).foregroundStyle(
                c.blocked
              )
              .padding(.horizontal, PaneSpace.m)
              CodeBlock(text: "touch /tmp/x && echo -> done\n--- a/x\n+++ b/x", maxHeight: 120)
              HStack(spacing: PaneSpace.s) {
                KeyButton("Approve", key: "y", kind: .go) {}
                KeyButton("Deny", key: "n", kind: .stop) {}
                KeyButton("Reload", key: "r", kind: .quiet) {}
              }
              .padding(PaneSpace.s)
            }
            PaneNote(text: Texts.result("applied"), kind: .ok)
            PaneNote(text: Texts.result("outcome_unknown"), kind: .warn)
            PaneNote(text: "This prompt was already answered.", kind: .error)
            HStack {
              StatusMeter(states: ["blocked", "working", "done", "idle"])
              Spacer()
              Spinner()
              PaneSwitch(isOn: $on)
            }
            PaneField(label: "This phone's name", text: $name)
            KeyButton("Forget this Mac", kind: .stop) {}
            KeyButton("Pair with the pasted link", enabled: false) {}
            Text("# Pair with your Mac").pane(PaneType.display)
            Text("610 712").pane(PaneType.code).foregroundStyle(c.done)
            Text("-> != => // ligatures off").pane(PaneType.body)
          }
          .padding(PaneSpace.s)
        }
        TmuxBar(
          items: ["spaces", "agents", "settings"], selected: 0, onSelect: { _ in }, alerts: [1: 2])
      }
    }
  }
#endif

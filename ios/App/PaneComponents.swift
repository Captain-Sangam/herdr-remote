import Core
import SwiftUI

// The Pane building blocks (docs/design.md, section Components), as in
// `ui/theme/PaneComponents.kt`. Screens compose these. design/viewer.html shows each one.

private let tags = [
  "idle": "IDLE", "working": "WORK", "done": "DONE", "blocked": "BLOCKED", "error": "ERROR",
]
private let glyphs: [String: Character] = [
  "blocked": "!", "error": "x", "done": "+", "working": ">",
]

/// A 1 pt rule, solid or dashed.
struct Rule: View {
  var color: Color
  var dashed = false

  var body: some View {
    GeometryReader { g in
      Path { p in
        p.move(to: CGPoint(x: 0, y: 0.5))
        p.addLine(to: CGPoint(x: g.size.width, y: 0.5))
      }
      .stroke(color, style: StrokeStyle(lineWidth: PaneSpace.line, dash: dashed ? [4, 3] : []))
    }
    .frame(height: PaneSpace.line)
  }
}

extension View {
  /// A 1 pt rule under the view.
  func bottomRule(_ color: Color, dashed: Bool = false) -> some View {
    overlay(alignment: .bottom) { Rule(color: color, dashed: dashed) }
  }

  /// A 1 pt rule over the view.
  func topRule(_ color: Color) -> some View {
    overlay(alignment: .top) { Rule(color: color) }
  }
}

/// A working status blinks like a terminal cursor: 530 ms bright, 530 ms dim. The wall clock
/// keeps every blink on the screen in step. Nothing blinks with Reduce Motion on.
struct Blink<Content: View>: View {
  let state: String
  @ViewBuilder let content: () -> Content
  @Environment(\.accessibilityReduceMotion) private var still

  var body: some View {
    if state != "working" || still {
      content()
    } else {
      TimelineView(.periodic(from: .now, by: 0.53)) { t in
        let on = Int64(t.date.timeIntervalSince1970 * 1000) / 530 % 2 == 0
        content().opacity(on ? 1 : 0.35)
      }
    }
  }
}

/// A braille spinner: the frame changes each 80 ms.
struct Spinner: View {
  var style = PaneType.head
  var color: Color?
  @Environment(\.pane) private var c
  @Environment(\.accessibilityReduceMotion) private var still
  private static let frames = Array("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")

  var body: some View {
    TimelineView(.periodic(from: .now, by: 0.08)) { t in
      let i = still ? 0 : Int(t.date.timeIntervalSince1970 / 0.08) % Self.frames.count
      Text(String(Self.frames[i])).pane(style).foregroundStyle(color ?? c.accent)
    }
  }
}

/// Inverse-video status tag: BLOCKED in the background colour on the status colour.
struct StatusTag: View {
  let state: String
  @Environment(\.pane) private var c

  var body: some View {
    Blink(state: state) {
      Text(tags[state] ?? "IDLE")
        .pane(PaneType.caption.with(bold: true))
        .foregroundStyle(c.bg)
        .lineLimit(1)
        .padding(.horizontal, 6)
        .background(c.status(state))
    }
  }
}

/// The row lead: [!] blocked, [x] error, [+] done, [>] working, [ ] idle. Never colour alone.
struct StatusGlyph: View {
  let state: String
  @Environment(\.pane) private var c

  var body: some View {
    Blink(state: state) {
      Text("[\(String(glyphs[state] ?? " "))]")
        .pane(PaneType.body.with(bold: true))
        .foregroundStyle(c.status(state))
    }
    .accessibilityLabel(state)
  }
}

/// One bar for each agent of a space, in the colour of its state.
struct StatusMeter: View {
  let states: [String]
  @Environment(\.pane) private var c

  var body: some View {
    HStack(spacing: 2) {
      ForEach(Array(states.enumerated()), id: \.offset) { _, s in
        Blink(state: s) { Rectangle().fill(c.status(s)).frame(width: 7, height: 14) }
      }
    }
    .accessibilityElement()
    .accessibilityLabel(states.joined(separator: ", "))
  }
}

/// The pane: a 1 pt box with its title in the top border, like a tmux pane. Pane titles are
/// lowercase ("4 spaces", "needs you (2)", "permission"). `color` draws the border and the
/// title: the line colour for lists, a status colour for an alert. `behind` is the colour
/// behind the box, so that the title cuts the border.
struct PaneBox<Content: View>: View {
  let title: String?
  var color: Color?
  var behind: Color?
  @ViewBuilder let content: () -> Content
  @Environment(\.pane) private var c

  var body: some View {
    let edge = color ?? c.line
    let alert = color != nil
    VStack(alignment: .leading, spacing: 0, content: content)
      .frame(maxWidth: .infinity, alignment: .leading)
      .padding(.top, title == nil ? 4 : 10)
      .padding(.bottom, 4)
      .overlay(Rectangle().stroke(edge, lineWidth: PaneSpace.line))
      .overlay(alignment: .topLeading) {
        if let title {
          Text("┤ \(title) ├")
            .pane(PaneType.label.with(bold: alert))
            .foregroundStyle(alert ? edge : c.muted)
            .padding(.horizontal, 6)
            .background(behind ?? c.bg)
            .padding(.leading, 8)
            .offset(y: -9)
        }
      }
      .padding(.top, 9)
  }
}

/// A list row: status glyph, name, a meta line, an optional detail line and a trailing view
/// (the status tag by default). A blocked row has a faint red background.
struct StatusRow<Trailing: View>: View {
  let state: String
  let title: String
  let meta: String
  var detail: String?
  let action: () -> Void
  @ViewBuilder var trailing: () -> Trailing
  @Environment(\.pane) private var c

  var body: some View {
    Button(action: action) {
      HStack(spacing: PaneSpace.m) {
        StatusGlyph(state: state)
        VStack(alignment: .leading, spacing: 0) {
          Text(title).pane(PaneType.title.with(size: PaneType.body.size)).lineLimit(1)
            .foregroundStyle(c.text)
          Text(meta).pane(PaneType.caption).foregroundStyle(c.muted).lineLimit(1)
          if let detail {
            Text(detail).pane(PaneType.caption).lineLimit(1).truncationMode(.tail)
              .foregroundStyle(state == "blocked" ? c.blocked : c.text)
          }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        trailing()
      }
      .padding(.horizontal, PaneSpace.m)
      .padding(.vertical, PaneSpace.xs)
      .frame(minHeight: PaneSpace.touch)
      .background(state == "blocked" ? c.blocked.opacity(0.12) : .clear)
      .contentShape(Rectangle())
    }
    .buttonStyle(.plain)
  }
}

extension StatusRow where Trailing == StatusTag {
  init(
    state: String, title: String, meta: String, detail: String? = nil,
    action: @escaping () -> Void
  ) {
    self.init(state: state, title: title, meta: meta, detail: detail, action: action) {
      StatusTag(state: state)
    }
  }
}

/// The top line of a screen: a back key, the path ("~/spaces/herdr-remote", last part bold)
/// and the link state when `online` is set.
struct PaneHeader<Actions: View>: View {
  let path: String
  var onBack: (() -> Void)?
  var online: Bool?
  @ViewBuilder var actions: () -> Actions
  @Environment(\.pane) private var c

  var body: some View {
    let cut = path.lastIndex(of: "/").map { path.index(after: $0) } ?? path.startIndex
    HStack(spacing: 0) {
      if let onBack {
        Button(action: onBack) {
          Text("←").pane(PaneType.title).foregroundStyle(c.accent)
            .frame(width: PaneSpace.touch, height: PaneSpace.touch)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .accessibilityLabel("Back")
      } else {
        Spacer().frame(width: PaneSpace.l)
      }
      (Text(path[..<cut]).font(PaneType.title.with(bold: false).font).foregroundColor(c.muted)
        + Text(path[cut...]).font(PaneType.title.font).foregroundColor(c.text))
        .lineLimit(1).truncationMode(.tail)
        .frame(maxWidth: .infinity, alignment: .leading)
      actions()
      if let online {
        Text(online ? "● online" : "○ offline").pane(PaneType.caption)
          .foregroundStyle(online ? c.done : c.blocked)
      }
    }
    .padding(.trailing, PaneSpace.m)
    .frame(minHeight: PaneSpace.touch)
    .background(c.surface)
    .bottomRule(c.line)
  }
}

extension PaneHeader where Actions == EmptyView {
  init(_ path: String, onBack: (() -> Void)? = nil, online: Bool? = nil) {
    self.init(path: path, onBack: onBack, online: online) { EmptyView() }
  }
}

/// The line under the header: counts or the agent's state, with a dashed rule.
struct PaneSubline<Content: View>: View {
  @ViewBuilder let content: () -> Content
  @Environment(\.pane) private var c

  var body: some View {
    HStack(spacing: PaneSpace.s, content: content)
      .pane(PaneType.label)
      .foregroundStyle(c.muted)
      .padding(.horizontal, PaneSpace.l)
      .padding(.vertical, 6)
      .frame(maxWidth: .infinity, alignment: .leading)
      .bottomRule(c.line, dashed: true)
  }
}

/// Tabs as terminal labels: the selected tab is inverse on the accent colour.
struct PaneTabs: View {
  let tabs: [String]
  let selected: Int
  let onSelect: (Int) -> Void
  @Environment(\.pane) private var c

  var body: some View {
    HStack(spacing: 0) {
      ForEach(Array(tabs.enumerated()), id: \.offset) { i, tab in
        let on = i == selected
        Button {
          onSelect(i)
        } label: {
          Text(tab.lowercased()).pane(PaneType.label.with(bold: on))
            .foregroundStyle(on ? c.onAccent : c.muted)
            .padding(.horizontal, PaneSpace.l)
            .frame(maxHeight: .infinity)
            .background(on ? c.accent : .clear)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(on ? [.isSelected] : [])
      }
      Spacer(minLength: 0)
    }
    .padding(.horizontal, PaneSpace.s)
    .frame(height: PaneSpace.touch)
    .background(c.surface)
    .bottomRule(c.line)
  }
}

/// The bottom navigation, a tmux status line: "[herdr] spaces* agents(2!) settings". `alerts`
/// maps an item index to a count of blocked agents. It scrolls sideways when it does not fit.
struct TmuxBar: View {
  let items: [String]
  let selected: Int
  let onSelect: (Int) -> Void
  var alerts: [Int: Int] = [:]
  @Environment(\.pane) private var c

  var body: some View {
    ScrollView(.horizontal, showsIndicators: false) {
      HStack(spacing: 0) {
        Text("[herdr]").pane(PaneType.label.with(bold: true)).foregroundStyle(c.bg)
          .padding(.horizontal, PaneSpace.m)
          .frame(height: PaneSpace.touch)
          .background(c.done, ignoresSafeAreaEdges: [])
        ForEach(Array(items.enumerated()), id: \.offset) { i, name in
          let on = i == selected
          let n = alerts[i] ?? 0
          Button {
            onSelect(i)
          } label: {
            (Text(name.lowercased() + (on ? "*" : ""))
              + Text(n > 0 ? "(\(n)!)" : "").foregroundColor(c.blocked).bold())
              .pane(PaneType.label.with(bold: on))
              .foregroundStyle(on ? c.text : c.muted)
              .padding(.horizontal, 9)
              .frame(height: PaneSpace.touch)
              .background(on ? c.accent.opacity(0.18) : .clear, ignoresSafeAreaEdges: [])
              .contentShape(Rectangle())
          }
          .buttonStyle(.plain)
          .accessibilityAddTraits(on ? [.isSelected] : [])
        }
      }
    }
    .scrollBounceBehavior(.basedOnSize)
    .frame(height: PaneSpace.touch)
    .background(c.surface2)
    .topRule(c.line)
  }
}

/// What a `KeyButton` does: Go is the main action, Stop denies or destroys, Quiet is minor.
enum KeyKind {
  case go, plain, stop, quiet
}

/// A square key: 1 pt border, at least 48 pt high, an optional key hint in an inverse chip
/// ("[y] Approve"). Go is filled (Approve, Scan the QR code, Unlock). Plain is a normal action.
/// Stop is Deny or Forget this Mac. Quiet is a minor action (Reload).
struct KeyButton: View {
  let label: String
  var key: Character?
  var kind = KeyKind.plain
  var enabled = true
  var detail: String?
  var alignment = HorizontalAlignment.center
  let action: () -> Void
  @Environment(\.pane) private var c

  init(
    _ label: String, key: Character? = nil, kind: KeyKind = .plain, enabled: Bool = true,
    detail: String? = nil, alignment: HorizontalAlignment = .center, action: @escaping () -> Void
  ) {
    self.label = label
    self.key = key
    self.kind = kind
    self.enabled = enabled
    self.detail = detail
    self.alignment = alignment
    self.action = action
  }

  var body: some View {
    let fg: Color =
      switch kind {
      case .go: c.bg
      case .plain: c.accent
      case .stop: c.blocked
      case .quiet: c.muted
      }
    let edge: Color =
      switch kind {
      case .go: c.done
      case .quiet: c.line
      default: fg
      }
    Button(action: action) {
      HStack(spacing: PaneSpace.s) {
        if alignment != .leading { Spacer(minLength: 0) }
        if let key {
          Text(String(key)).pane(PaneType.label.with(bold: true))
            .foregroundStyle(kind == .go ? c.done : c.bg)
            .frame(width: 20, height: 20)
            .background(kind == .go ? c.bg : fg)
            .accessibilityHidden(true)
        }
        VStack(alignment: .leading, spacing: 0) {
          Text(label).pane(PaneType.label.with(bold: true)).foregroundStyle(fg)
            .lineLimit(1).fixedSize()
          if let detail {
            Text(detail).pane(PaneType.caption).foregroundStyle(c.muted)
          }
        }
        .padding(.vertical, detail == nil ? 0 : PaneSpace.xs)
        if alignment != .trailing { Spacer(minLength: 0) }
      }
      .padding(.horizontal, PaneSpace.l)
      .frame(minHeight: PaneSpace.touch)
      .background(kind == .go ? c.done : .clear)
      .overlay(Rectangle().stroke(edge, lineWidth: PaneSpace.line))
      .contentShape(Rectangle())
    }
    .buttonStyle(.plain)
    .disabled(!enabled)
    .opacity(enabled ? 1 : 0.45)
  }
}

/// What a `PaneNote` reports.
typealias NoteKind = Core.NoteKind

/// The result of an action, in words ("Done. The agent continues."): a 3 pt bar in the colour
/// of the result on the surface colour. Use the texts in `Texts`.
struct PaneNote: View {
  let text: String
  let kind: NoteKind
  @Environment(\.pane) private var c

  var body: some View {
    let bar: Color =
      switch kind {
      case .ok: c.done
      case .warn: c.error
      case .error: c.blocked
      }
    HStack(alignment: .firstTextBaseline, spacing: PaneSpace.s) {
      Text(kind == .ok ? "+" : kind == .warn ? "!" : "x").pane(PaneType.label.with(bold: true))
        .foregroundStyle(bar).accessibilityHidden(true)
      Text(text).pane(PaneType.label).foregroundStyle(c.text)
      Spacer(minLength: 0)
    }
    .padding(.leading, 13)
    .padding(.trailing, PaneSpace.m)
    .padding(.vertical, 6)
    .background(alignment: .leading) {
      HStack(spacing: 0) {
        bar.frame(width: 3)
        c.surface
      }
    }
  }
}

/// A full-width warning strip under the header ("Not connected to the Mac: last update 48 s
/// ago").
struct PaneBanner: View {
  let text: String
  @Environment(\.pane) private var c

  var body: some View {
    Text(text).pane(PaneType.label.with(bold: true)).foregroundStyle(c.bg)
      .padding(.horizontal, PaneSpace.l)
      .padding(.vertical, 6)
      .frame(maxWidth: .infinity, alignment: .leading)
      .background(c.error)
  }
}

/// An on/off switch drawn as "[ on ]" and "[ off ]".
struct PaneSwitch: View {
  @Binding var isOn: Bool
  @Environment(\.pane) private var c

  var body: some View {
    Button {
      isOn.toggle()
    } label: {
      Text(isOn ? "[ on ]" : "[ off ]").pane(PaneType.body.with(bold: true))
        .foregroundStyle(isOn ? c.done : c.muted)
        .frame(minWidth: PaneSpace.touch, minHeight: PaneSpace.touch)
        .contentShape(Rectangle())
    }
    .buttonStyle(.plain)
    .accessibilityAddTraits(.isToggle)
    .accessibilityValue(isOn ? "on" : "off")
  }
}

/// A text field in a pane, with its label in the top border ("This phone's name").
struct PaneField: View {
  let label: String
  @Binding var text: String
  var placeholder = ""
  var singleLine = true
  @Environment(\.pane) private var c

  var body: some View {
    PaneBox(title: label) {
      TextField(
        "", text: $text,
        prompt: Text(placeholder).font(PaneType.body.font).foregroundColor(c.muted),
        axis: singleLine ? .horizontal : .vertical
      )
      .pane(PaneType.body)
      .foregroundStyle(c.text)
      .autocorrectionDisabled()
      .textInputAutocapitalization(.never)
      .padding(.horizontal, PaneSpace.m)
      .padding(.vertical, 6)
    }
  }
}

/// A command, a diff or prompt text, exactly as the agent shows it: no wrapping, scrolls both
/// ways, at most `maxHeight` high, so that the keys under it stay in reach.
struct CodeBlock: View {
  let text: String
  var maxHeight: CGFloat = .infinity
  @Environment(\.pane) private var c

  var body: some View {
    ScrollView([.horizontal, .vertical]) {
      Text(text).pane(PaneType.label).foregroundStyle(c.text)
        .fixedSize()
        .padding(.leading, PaneSpace.l)
        .padding(.trailing, PaneSpace.m)
        .padding(.vertical, PaneSpace.s)
        .frame(maxWidth: .infinity, alignment: .leading)
        .textSelection(.enabled)
    }
    .frame(maxHeight: maxHeight)
    .fixedSize(horizontal: false, vertical: maxHeight == .infinity)
    .background(alignment: .leading) {
      HStack(spacing: 0) {
        c.line.frame(width: 2)
        c.surface
      }
    }
  }
}

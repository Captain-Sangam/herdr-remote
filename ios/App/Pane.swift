import SwiftUI
import UIKit

// Pane, the herdr-remote design system (docs/design.md), for iOS. The tokens match
// `ui/theme/PaneTheme.kt`. Every screen reads its colours, type and spacing from here
// (`@Environment(\.pane)`, `PaneType`, `PaneSpace`) and builds on the components in
// PaneComponents.swift. Do not hard-code a colour or a text size in a screen.

extension Color {
  init(hex: UInt32) {
    self.init(
      .sRGB, red: Double(hex >> 16 & 0xFF) / 255, green: Double(hex >> 8 & 0xFF) / 255,
      blue: Double(hex & 0xFF) / 255)
  }
}

/// The colour tokens. The names match docs/design.md and design/mockups.js.
struct PaneColors: Sendable {
  let bg, surface, surface2, line, text, muted, accent, onAccent: Color
  let idle, working, done, blocked, error: Color
  let isDark: Bool

  /// The colour of an hc agent state. An unknown state is idle.
  func status(_ state: String) -> Color {
    switch state {
    case "blocked": blocked
    case "working": working
    case "done": done
    case "error": error
    default: idle
    }
  }

  static let light = PaneColors(
    bg: Color(hex: 0xEEF1F6), surface: Color(hex: 0xFFFFFF), surface2: Color(hex: 0xE2E7EF),
    line: Color(hex: 0xB7C0D1), text: Color(hex: 0x222836), muted: Color(hex: 0x5B667D),
    accent: Color(hex: 0x2E7D8C), onAccent: Color(hex: 0xFFFFFF), idle: Color(hex: 0x7A8499),
    working: Color(hex: 0x3A6EA5), done: Color(hex: 0x4F7F2E), blocked: Color(hex: 0xB23A48),
    error: Color(hex: 0xB4561E), isDark: false)

  /// The main mode.
  static let dark = PaneColors(
    bg: Color(hex: 0x191C26), surface: Color(hex: 0x1F2330), surface2: Color(hex: 0x272C3B),
    line: Color(hex: 0x3B4258), text: Color(hex: 0xD8DEE9), muted: Color(hex: 0x7D879E),
    accent: Color(hex: 0x88C0D0), onAccent: Color(hex: 0x191C26), idle: Color(hex: 0x6C7891),
    working: Color(hex: 0x81A1C1), done: Color(hex: 0xA3BE8C), blocked: Color(hex: 0xBF616A),
    error: Color(hex: 0xD08770), isDark: true)

  /// The live view's terminal colours, the same in both modes (assets/xterm/live.html).
  static let terminalBg = Color(hex: 0x14161E)
  static let terminalFg = Color(hex: 0xD8DEE9)
}

extension EnvironmentValues {
  @Entry var pane = PaneColors.dark
}

/// One style of the type scale: JetBrains Mono (SIL OFL 1.1), the only family, in two weights.
/// Ligatures are off, so a command shows exactly as typed ("->" stays two characters). Sizes
/// follow Dynamic Type, as sp sizes follow the font scale on Android.
struct PaneStyle: Sendable {
  let size: CGFloat
  let line: CGFloat
  let bold: Bool

  func with(bold: Bool) -> PaneStyle { PaneStyle(size: size, line: line, bold: bold) }
  func with(size: CGFloat) -> PaneStyle { PaneStyle(size: size, line: line, bold: bold) }

  var uiFont: UIFont {
    let name = bold ? "JetBrainsMono-Bold" : "JetBrainsMono-Regular"
    let base =
      UIFont(name: name, size: size)
      ?? .monospacedSystemFont(ofSize: size, weight: bold ? .bold : .regular)
    let off = base.fontDescriptor.addingAttributes([
      .featureSettings: [
        [
          UIFontDescriptor.FeatureKey.type: kLigaturesType,
          UIFontDescriptor.FeatureKey.selector: kCommonLigaturesOffSelector,
        ],
        [
          UIFontDescriptor.FeatureKey.type: kContextualAlternatesType,
          UIFontDescriptor.FeatureKey.selector: kContextualAlternatesOffSelector,
        ],
      ]
    ])
    return UIFontMetrics.default.scaledFont(for: UIFont(descriptor: off, size: size))
  }

  var font: Font { Font(uiFont) }

  /// The space between lines that gives the scale's line height.
  var spacing: CGFloat {
    max(0, UIFontMetrics.default.scaledValue(for: line) - uiFont.lineHeight)
  }
}

/// The type scale. Sizes and line heights come from the mockups.
enum PaneType {
  /// 44/52 bold: the pairing code.
  static let code = PaneStyle(size: 44, line: 52, bold: true)
  /// 18/26 bold: the title of a full-screen flow ("Pair with your Mac").
  static let display = PaneStyle(size: 18, line: 26, bold: true)
  /// 16/24 bold: an empty-state title.
  static let head = PaneStyle(size: 16, line: 24, bold: true)
  /// 14/20 bold: the header path and row names.
  static let title = PaneStyle(size: 14, line: 20, bold: true)
  /// 13/20: body text, field input.
  static let body = PaneStyle(size: 13, line: 20, bold: false)
  /// 12/18: buttons, tabs, pane titles, transcript and code.
  static let label = PaneStyle(size: 12, line: 18, bold: false)
  /// 11/16: meta lines, status tags, hints.
  static let caption = PaneStyle(size: 11, line: 16, bold: false)
}

/// Spacing in points. One character cell is about 8 pt, one text row 20 pt.
enum PaneSpace {
  static let xs: CGFloat = 4
  static let s: CGFloat = 8
  static let m: CGFloat = 10
  static let l: CGFloat = 12
  static let xl: CGFloat = 16
  /// The minimum size of anything the user taps.
  static let touch: CGFloat = 48
  /// Every border and rule.
  static let line: CGFloat = 1
}

extension View {
  /// Text in a Pane style.
  func pane(_ style: PaneStyle) -> some View {
    font(style.font).lineSpacing(style.spacing)
  }
}

/// The app theme: dark is the main mode, light follows the system setting.
struct PaneTheme: ViewModifier {
  @Environment(\.colorScheme) private var scheme

  func body(content: Content) -> some View {
    let colors = scheme == .dark ? PaneColors.dark : PaneColors.light
    content
      .environment(\.pane, colors)
      .foregroundStyle(colors.text)
      .tint(colors.accent)
      .background(colors.bg.ignoresSafeArea())
  }
}

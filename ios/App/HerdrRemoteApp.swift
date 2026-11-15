import Core
import SwiftUI

@main
struct HerdrRemoteApp: App {
  var body: some Scene {
    WindowGroup {
      Root().modifier(PaneTheme())
    }
  }
}

struct Root: View {
  @State private var model = AppModel()

  var body: some View {
    #if DEBUG
      if ProcessInfo.processInfo.arguments.contains("-panePreview") {
        PanePreview()
      } else {
        main
      }
    #else
      main
    #endif
  }

  @ViewBuilder private var main: some View {
    if model.api != nil {
      PairedScreen(model: model)
    } else {
      PairScreen(model: model)
    }
  }
}

/// What a paired app shows until the lists come (step 7 of the plan): the Mac, a check of
/// /v1/hello over the device identity, and Forget.
struct PairedScreen: View {
  let model: AppModel
  @State private var note: (text: String, kind: NoteKind)?
  @State private var busy = false
  @Environment(\.pane) private var c

  var body: some View {
    VStack(spacing: 0) {
      PaneHeader("~/spaces")
      ScrollView {
        VStack(alignment: .leading, spacing: PaneSpace.s) {
          PaneBox(title: "paired mac") {
            ForEach(model.store.state.endpoints, id: \.self) { ep in
              Text("\(ep) · \(endpointKind(ep))").pane(PaneType.body)
                .padding(.horizontal, PaneSpace.m).padding(.vertical, PaneSpace.xs)
            }
            Text("Device id \(model.store.state.deviceId ?? "-")").pane(PaneType.caption)
              .foregroundStyle(c.muted)
              .padding(.horizontal, PaneSpace.m).padding(.vertical, PaneSpace.xs)
          }
          if let note {
            PaneNote(text: note.text, kind: note.kind).accessibilityIdentifier("hello-note")
          }
          KeyButton("Check the connection", enabled: !busy) {
            busy = true
            Task {
              do {
                let h = try await model.api?.hello()
                note = ("Connected to herdr-remoted \(h?.version ?? "").", .ok)
              } catch {
                note = (Texts.error(error), .error)
              }
              busy = false
            }
          }
          KeyButton("Forget this Mac", kind: .stop) { model.forget() }
        }
        .padding(PaneSpace.s)
      }
    }
  }
}

import Core
import SwiftUI
import UserNotifications

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
/// /v1/hello over the device identity, a test alert for the content extension, and Forget.
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
          KeyButton("Send a test alert in 5 s") {
            Task { note = (await TestAlert.send(), .ok) }
          }
          KeyButton("Forget this Mac", kind: .stop) { model.forget() }
        }
        .padding(PaneSpace.s)
      }
    }
  }
}

/// A local alert in the category of the content extension (step 6 of the plan): expanded, it
/// shows whether the extension reads the store item.
enum TestAlert {
  static func send() async -> String {
    let center = UNUserNotificationCenter.current()
    guard (try? await center.requestAuthorization(options: [.alert, .sound])) == true else {
      return "Notifications are off for Herdr Remote. Turn them on in Settings."
    }
    center.setNotificationCategories([
      UNNotificationCategory(
        identifier: "agent", actions: [], intentIdentifiers: [], options: [.hiddenPreviewsShowTitle]
      )
    ])
    let content = UNMutableNotificationContent()
    content.title = "Test alert"
    content.categoryIdentifier = "agent"
    let request = UNNotificationRequest(
      identifier: "test", content: content,
      trigger: UNTimeIntervalNotificationTrigger(timeInterval: 5, repeats: false))
    do {
      try await center.add(request)
    } catch {
      return "The test alert failed: \(error.localizedDescription)"
    }
    return "A test alert comes in 5 s. Leave the app, then expand the alert."
  }
}

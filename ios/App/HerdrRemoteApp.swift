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
  var body: some View {
    #if DEBUG
      if ProcessInfo.processInfo.arguments.contains("-panePreview") {
        PanePreview()
      } else {
        Text("Herdr Remote")
      }
    #else
      Text("Herdr Remote")
    #endif
  }
}

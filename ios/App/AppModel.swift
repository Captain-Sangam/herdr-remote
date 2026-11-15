import Core
import Foundation
import Observation

/// The app's state: the store and, once paired, the API.
@MainActor @Observable
final class AppModel {
  let store = Store()
  private(set) var api: HerdrApi?

  init() {
    #if DEBUG
      // UI tests start from the pair screen.
      if ProcessInfo.processInfo.arguments.contains("-reset") { store.forget() }
    #endif
    api = HerdrApi.paired(store)
  }

  /// After enroll: the store holds the Mac. `/v1/hello` sets the clock offset and the endpoints.
  func didPair() {
    api = HerdrApi.paired(store)
    let api = api
    Task { _ = try? await api?.hello() }
  }

  /// Forget this Mac: the key, the certificate and the store go.
  func forget() {
    store.forget()
    api = nil
  }
}

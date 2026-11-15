// swift-tools-version: 6.0
// The pure logic of the iOS app (ADR 0025): no UIKit, no keychain, no network.
// `swift test --package-path ios/Core` runs its tests on the Mac.
import PackageDescription

let package = Package(
  name: "Core",
  platforms: [.iOS(.v17), .macOS(.v14)],
  products: [.library(name: "Core", targets: ["Core"])],
  targets: [
    .target(name: "Core"),
    .testTarget(name: "CoreTests", dependencies: ["Core"]),
  ]
)

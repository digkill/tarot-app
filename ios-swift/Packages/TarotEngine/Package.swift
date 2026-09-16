// swift-tools-version: 6.0
//
// The rules of French Tarot as a pure Swift library: no UI, no networking, no
// persistence. Kept as its own package so it builds and tests in seconds on the
// Mac with `swift test`, and so the app, the AI opponents and any future server
// validation all share one implementation of the rules.
import PackageDescription

let package = Package(
    name: "TarotEngine",
    platforms: [.iOS(.v17), .macOS(.v14)],
    products: [
        .library(name: "TarotEngine", targets: ["TarotEngine"]),
    ],
    targets: [
        .target(name: "TarotEngine"),
        .testTarget(name: "TarotEngineTests", dependencies: ["TarotEngine"]),
    ]
)

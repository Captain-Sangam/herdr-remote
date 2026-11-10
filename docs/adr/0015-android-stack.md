# ADR 0015: The Android stack

- **Status:** Accepted, 2026-11-08.
- **Decision:** 15 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- The user asked for an Android app, built with the Android command-line tools from Homebrew, without Android Studio.
- The app needs TLS with a client key, SSE, a chunked stream, a QR scanner, a biometric prompt and a terminal view.
- The smallest system was a criterion.

## Decision

- Kotlin and Jetpack Compose. minSdk 31, targetSdk 35.
- OkHttp for TLS, SSE and streams.
- kotlinx.serialization.
- The Google Play services code scanner (no camera permission).
- androidx.biometric.
- xterm.js 5.x as a bundled asset.
- One activity, ViewModels, no DI framework.

As built:

- Screens are a `Screen` sealed class. There is no navigation library.
- The app parses SSE by hand over OkHttp.
- The screens use the Pane design system on Material 3 ([design](../design.md)).

## Alternatives

| Alternative             | Why it lost                              |
| ----------------------- | ---------------------------------------- |
| Termux terminal-view    | A JitPack dependency and more glue code. |
| React Native or Flutter | More toolchain.                          |

## Consequences

- The phone needs Google Play services for the scanner.
- The app is sideloaded with `adb`. There is no Play Store release.
- androidx.biometric 1.1.0 pulls `fragment` 1.2.5, which rejects the permission request code. The build pins `fragment` 1.8.6 (`d3cd35f`).

## Sources

- [Technical design, decision 15 and step 1](../history/technical-design.md#31-decision-table)

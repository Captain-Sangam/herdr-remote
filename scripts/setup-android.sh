#!/bin/sh
# Installs the Android toolchain for herdr-remote on this Mac (TDD step 1).
# Outside the repo it changes only Homebrew packages and the SDK directory.
set -eu
export NONINTERACTIVE=1

SDK="${ANDROID_HOME:-/opt/homebrew/share/android-commandlinetools}"
IMAGE="system-images;android-35;google_apis_playstore;arm64-v8a"

brew list --cask android-commandlinetools >/dev/null 2>&1 || brew install --cask android-commandlinetools
brew list gradle >/dev/null 2>&1 || brew install gradle

yes | sdkmanager --sdk_root="$SDK" --licenses >/dev/null || true
sdkmanager --sdk_root="$SDK" "platform-tools" "platforms;android-35" "build-tools;35.0.0" "emulator" "$IMAGE"

if ! avdmanager list avd 2>/dev/null | grep -q 'Name: herdr-remote$'; then
  echo no | avdmanager create avd -n herdr-remote -k "$IMAGE" -d pixel_6
fi

printf 'sdk.dir=%s\n' "$SDK" >"$(dirname "$0")/../android/local.properties"

echo "Done. For adb and the emulator on your PATH:"
echo "  export ANDROID_HOME=$SDK"
echo "  export PATH=\"\$ANDROID_HOME/platform-tools:\$ANDROID_HOME/emulator:\$PATH\""

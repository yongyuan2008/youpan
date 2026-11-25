#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"
bundle="build/有盼.app"
mkdir -p "$bundle/Contents/MacOS" "$bundle/Contents/Resources"
cp Info.plist "$bundle/Contents/Info.plist"
cp assets/Calendar.icns "$bundle/Contents/Resources/"
# The Swift 6.4 CLI ships no SwiftUIMacros plugin for the default macOS 27 SDK.
# Use the installed compatibility SDK when present, and otherwise follow xcrun.
compat_sdk="/Library/Developer/CommandLineTools/SDKs/MacOSX26.sdk"
if [[ -d "$compat_sdk" ]]; then sdk="$compat_sdk"; else sdk="$(xcrun --sdk macosx --show-sdk-path)"; fi
mkdir -p build/tests
test_profile="$(mktemp -d "$PWD/build/tests/isolated-profile.XXXXXX")"
xcrun swiftc -swift-version 5 -parse-as-library -O -sdk "$sdk" -target "$(uname -m)-apple-macosx13.0" \
  core/CountdownCore.swift core/AppearanceCore.swift core/PlannerCore.swift core/tests/CountdownCoreTests.swift -o build/tests/CountdownCoreTests
RIZI_COUNTDOWN_TEST_DIRECTORY="$test_profile" build/tests/CountdownCoreTests
for arch in arm64 x86_64; do
  xcrun swiftc -swift-version 5 -parse-as-library -O -sdk "$sdk" -target "$arch-apple-macosx13.0" \
    -module-name DaysCountdown core/CountdownCore.swift core/AppearanceCore.swift core/PlannerCore.swift ui/*.swift -o "build/DaysCountdown-$arch"
done
xcrun lipo -create build/DaysCountdown-arm64 build/DaysCountdown-x86_64 -output "$bundle/Contents/MacOS/DaysCountdown"
codesign --force --sign - "$bundle"
codesign --verify --deep --strict "$bundle"
plutil -lint "$bundle/Contents/Info.plist"
xcrun lipo -info "$bundle/Contents/MacOS/DaysCountdown"

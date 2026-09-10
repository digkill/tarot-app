#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WS="$ROOT/ios-native/Tarot.xcworkspace"
OUT="$ROOT/build/ios-native"
ARCHIVE="$OUT/Tarot.xcarchive"
IPA_DIR="$OUT/ipa"
PLIST="$OUT/ExportOptions.plist"

if lsof -nP -iTCP:8081 -sTCP:LISTEN >/dev/null 2>&1; then
  echo "Metro is listening on 8081. Stop it. This script is Release-only." >&2
  exit 1
fi

if [[ ! -d "$WS" ]]; then
  echo "Missing native project. Run: scripts/sync-ios-native.sh" >&2
  exit 1
fi

mkdir -p "$OUT"
cat > "$PLIST" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>method</key>
    <string>app-store-connect</string>
    <key>destination</key>
    <string>export</string>
    <key>signingStyle</key>
    <string>automatic</string>
    <key>teamID</key>
    <string>39CP3623CD</string>
    <key>uploadSymbols</key>
    <true/>
    <key>manageAppVersionAndBuildNumber</key>
    <false/>
</dict>
</plist>
EOF

rm -rf "$ARCHIVE" "$IPA_DIR"
mkdir -p "$IPA_DIR"

xcodebuild -workspace "$WS" -scheme Tarot -configuration Release \
  -destination 'generic/platform=iOS' \
  -archivePath "$ARCHIVE" \
  -allowProvisioningUpdates \
  archive

APP="$ARCHIVE/Products/Applications/Tarot.app"
if [[ ! -f "$APP/main.jsbundle" ]]; then
  echo "Release archive is missing embedded main.jsbundle" >&2
  exit 1
fi
file "$APP/main.jsbundle" | grep -q 'Hermes JavaScript bytecode' || {
  echo "main.jsbundle is not Hermes bytecode" >&2
  exit 1
}

xcodebuild -exportArchive \
  -archivePath "$ARCHIVE" \
  -exportPath "$IPA_DIR" \
  -exportOptionsPlist "$PLIST" \
  -allowProvisioningUpdates

echo "Archive: $ARCHIVE"
echo "IPA: $IPA_DIR/Tarot.ipa"

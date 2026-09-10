#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/ios"
DST="$ROOT/ios-native"

if [[ ! -d "$SRC/Tarot.xcworkspace" ]]; then
  echo "Missing $SRC/Tarot.xcworkspace. Run: npx expo prebuild --platform ios" >&2
  exit 1
fi

mkdir -p "$DST"
rsync -a --delete \
  --exclude /build \
  --exclude '*.xcarchive' \
  --exclude .DS_Store \
  "$SRC/" "$DST/"

# RN Codegen writes to <ios>/build/generated — keep it, do not use that dir for xcarchive.
if [[ -d "$SRC/build/generated" ]]; then
  mkdir -p "$DST/build"
  rsync -a --delete "$SRC/build/generated/" "$DST/build/generated/"
fi

cat > "$DST/.xcode.env.updates" << 'EOF'
# ios-native is Release/embedded-JS only. Never wait for Metro.
unset SKIP_BUNDLING
EOF

echo "Synced $SRC -> $DST"

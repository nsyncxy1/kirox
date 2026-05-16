#!/bin/bash
set -e

# 先确保 build/appicon.png 是正确的图标（wails build -clean 会用默认图标覆盖）
cp frontend/assets/appicon.png build/appicon.png

# Wails build
~/go/bin/wails build "$@"

# 用 iconutil 生成正确的 icns 替换 Wails 生成的（Go icns 库生成的 macOS 可能不识别）
APP="build/bin/Kiro注册机.app"
if [ -d "$APP" ]; then
  ICONSET=$(mktemp -d)/appicon.iconset
  mkdir -p "$ICONSET"
  SRC="frontend/assets/appicon.png"
  sips -z 16 16 "$SRC" --out "$ICONSET/icon_16x16.png" >/dev/null
  sips -z 32 32 "$SRC" --out "$ICONSET/icon_16x16@2x.png" >/dev/null
  sips -z 32 32 "$SRC" --out "$ICONSET/icon_32x32.png" >/dev/null
  sips -z 64 64 "$SRC" --out "$ICONSET/icon_32x32@2x.png" >/dev/null
  sips -z 128 128 "$SRC" --out "$ICONSET/icon_128x128.png" >/dev/null
  sips -z 256 256 "$SRC" --out "$ICONSET/icon_128x128@2x.png" >/dev/null
  sips -z 256 256 "$SRC" --out "$ICONSET/icon_256x256.png" >/dev/null
  sips -z 512 512 "$SRC" --out "$ICONSET/icon_256x256@2x.png" >/dev/null
  sips -z 512 512 "$SRC" --out "$ICONSET/icon_512x512.png" >/dev/null
  sips -z 1024 1024 "$SRC" --out "$ICONSET/icon_512x512@2x.png" >/dev/null
  iconutil -c icns "$ICONSET" -o "$APP/Contents/Resources/iconfile.icns"
  rm -rf "$(dirname "$ICONSET")"
  echo "Icon replaced with iconutil-generated icns"
fi

echo "Build complete"

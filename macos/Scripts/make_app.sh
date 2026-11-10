#!/bin/sh
# 构建 ImageResize.app（自包含 bundle，免安装运行）。
# 用法: Scripts/make_app.sh [debug|release]   （默认 release）
set -euo pipefail
cd "$(dirname "$0")/.."

CONFIG=${1:-release}
swift build -c "$CONFIG"
BIN="$(swift build -c "$CONFIG" --show-bin-path)"

APP="build/ImageResize.app"
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

cp "$BIN/ImageResize" "$APP/Contents/MacOS/ImageResize"
cp Resources/Info.plist "$APP/Contents/Info.plist"
if [ -f Resources/AppIcon.icns ]; then
  cp Resources/AppIcon.icns "$APP/Contents/Resources/AppIcon.icns"
fi

# ad-hoc 签名：本机可直接运行；分发给别人前需 Developer ID 签名 + 公证
codesign --force --sign - "$APP"

echo "已生成 $APP"

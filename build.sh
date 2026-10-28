#!/usr/bin/env bash
# Builds Chess.exe and the portable folder in dist/Chess.
# Needs Go 1.24+. The Windows icon and version info are already in
# app/rsrc_windows_amd64.syso (regenerate it with Ironite's mkres tool
# when the version or icon changes).
#
# The installer: after building, compile installer/ChessSetup.iss with
# Inno Setup 6.3+ (ISCC.exe installer\ChessSetup.iss). It packages
# dist/Chess and writes dist/Output/Chess-Setup-v<version>.exe.
set -euo pipefail
cd "$(dirname "$0")"
VERSION=$(grep -m1 -oP '^## \[\K[0-9.]+' web/CHANGELOG.md)
grep -q "const VERSION = '$VERSION'" web/game.js || { echo "web/game.js VERSION does not match CHANGELOG ($VERSION)"; exit 1; }
OUT=dist/Chess
rm -rf dist build && mkdir -p "$OUT" build
cp -r app/. build/ && cp -r web build/web
( cd build && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
    -ldflags "-H windowsgui -s -w -X main.gameID=Chess -X main.title=Chess -X main.version=$VERSION" -o "../$OUT/Chess.exe" . )
crlf() { sed 's/\r$//; s/$/\r/' "$1" > "$2"; }
crlf web/CHANGELOG.md "$OUT/CHANGELOG.txt"
crlf web/fonts/OFL.txt "$OUT/OFL.txt"
for f in README.txt CREDITS.txt; do sed "s/@VERSION@/$VERSION/g" "$f" > "build/$f"; crlf "build/$f" "$OUT/$f"; done
printf '%s\r\n' "$VERSION" > "$OUT/VERSION"
# lay out what the Inno script expects: ..\files\Games\Chess relative to installer\
mkdir -p dist/files/Games && cp -r "$OUT" dist/files/Games/
echo "Built $OUT/Chess.exe ($VERSION)"

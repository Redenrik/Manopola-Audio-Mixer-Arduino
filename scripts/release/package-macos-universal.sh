#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/release/package-macos-universal.sh <out_dir>

Builds a universal2 (amd64+arm64) macOS portable MAMA directory
including a launchable MAMA.app bundle.
USAGE
}

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

OUT_DIR="$1"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MAMA_DIR="${ROOT_DIR}/mama"

if ! command -v lipo >/dev/null 2>&1; then
  echo "lipo not found (required for universal2 packaging)" >&2
  exit 1
fi
if ! command -v sips >/dev/null 2>&1; then
  echo "sips not found (required for macOS icon packaging)" >&2
  exit 1
fi
if ! command -v iconutil >/dev/null 2>&1; then
  echo "iconutil not found (required for macOS app icon packaging)" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

mkdir -p "${OUT_DIR}"

(
  cd "${MAMA_DIR}"
  GOOS=darwin GOARCH=amd64 go build -o "${TMP_DIR}/mama-amd64" ./cmd/mama
  GOOS=darwin GOARCH=arm64 go build -o "${TMP_DIR}/mama-arm64" ./cmd/mama
)

lipo -create -output "${OUT_DIR}/mama" "${TMP_DIR}/mama-amd64" "${TMP_DIR}/mama-arm64"
cp "${MAMA_DIR}/internal/config/default.yaml" "${OUT_DIR}/config.yaml"
cp "${MAMA_DIR}/assets/icons/mama-app.png" "${OUT_DIR}/mama-app.png"
cp "${MAMA_DIR}/assets/icons/mama-tray.png" "${OUT_DIR}/mama-tray.png"

ICONSET_DIR="${TMP_DIR}/MAMA.iconset"
mkdir -p "${ICONSET_DIR}"
for spec in \
  "16 16 icon_16x16.png" \
  "32 32 icon_16x16@2x.png" \
  "32 32 icon_32x32.png" \
  "64 64 icon_32x32@2x.png" \
  "128 128 icon_128x128.png" \
  "256 256 icon_128x128@2x.png" \
  "256 256 icon_256x256.png" \
  "512 512 icon_256x256@2x.png" \
  "512 512 icon_512x512.png" \
  "1024 1024 icon_512x512@2x.png"
do
  read -r width height name <<<"${spec}"
  sips -z "${height}" "${width}" "${OUT_DIR}/mama-app.png" --out "${ICONSET_DIR}/${name}" >/dev/null
done

APP_DIR="${OUT_DIR}/MAMA.app"
APP_CONTENTS="${APP_DIR}/Contents"
APP_MACOS="${APP_CONTENTS}/MacOS"
APP_RESOURCES="${APP_CONTENTS}/Resources"
rm -rf "${APP_DIR}"
mkdir -p "${APP_MACOS}" "${APP_RESOURCES}"

cp "${OUT_DIR}/mama" "${APP_MACOS}/MAMA"
cp "${OUT_DIR}/mama-app.png" "${APP_RESOURCES}/mama-app.png"
cp "${OUT_DIR}/mama-tray.png" "${APP_RESOURCES}/mama-tray.png"
iconutil -c icns "${ICONSET_DIR}" -o "${APP_RESOURCES}/MAMA.icns"

cat > "${APP_CONTENTS}/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key>
  <string>MAMA</string>
  <key>CFBundleDisplayName</key>
  <string>MAMA Audio Mixer</string>
  <key>CFBundleIdentifier</key>
  <string>io.mama.mixer.app</string>
  <key>CFBundleVersion</key>
  <string>1</string>
  <key>CFBundleShortVersionString</key>
  <string>1.0.0</string>
  <key>CFBundleExecutable</key>
  <string>MAMA</string>
  <key>CFBundleIconFile</key>
  <string>MAMA.icns</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>LSMinimumSystemVersion</key>
  <string>14.2</string>
  <key>NSAudioCaptureUsageDescription</key>
  <string>MAMA captures app audio to apply per-app volume and mute controls.</string>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
PLIST

touch "${APP_DIR}"

cat > "${OUT_DIR}/Open Setup UI.command" <<'LAUNCH_UI'
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
config_path="$PWD/config.yaml"
exec open -a "./MAMA.app" --args -config "$config_path" "$@"
LAUNCH_UI

cat > "${OUT_DIR}/Start Mixer.command" <<'LAUNCH_MIXER'
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
app_bin="./MAMA.app/Contents/MacOS/MAMA"
config_path="./config.yaml"
if [[ ! -x "$app_bin" ]]; then
  echo "Missing app executable: $app_bin"
  exit 1
fi
pid_file=".mama.pid"
if [[ -f "$pid_file" ]]; then
  pid="$(cat "$pid_file" 2>/dev/null || true)"
  if [[ -n "${pid}" ]] && kill -0 "$pid" >/dev/null 2>&1; then
    echo "MAMA is already running in background (PID ${pid})."
    exit 0
  fi
fi
nohup "$app_bin" -config "$config_path" -open=false -start-hidden=true "$@" >/dev/null 2>&1 &
echo $! > "$pid_file"
echo "MAMA started in background (PID $(cat "$pid_file"))."
LAUNCH_MIXER

cat > "${OUT_DIR}/Stop Mixer.command" <<'STOP_MIXER'
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
pid_file=".mama.pid"
if [[ ! -f "$pid_file" ]]; then
  echo "No running MAMA instance found (missing .mama.pid)."
  exit 0
fi
pid="$(cat "$pid_file" 2>/dev/null || true)"
if [[ -z "${pid}" ]]; then
  rm -f "$pid_file"
  echo "No running MAMA instance found (empty PID file)."
  exit 0
fi
if ! kill -0 "$pid" >/dev/null 2>&1; then
  rm -f "$pid_file"
  echo "No running MAMA process for PID ${pid}."
  exit 0
fi
kill "$pid" >/dev/null 2>&1 || true
for _ in $(seq 1 20); do
  if ! kill -0 "$pid" >/dev/null 2>&1; then
    rm -f "$pid_file"
    echo "MAMA stopped (PID ${pid})."
    exit 0
  fi
  sleep 0.1
done
echo "MAMA process ${pid} is still running."
exit 1
STOP_MIXER

chmod +x "${OUT_DIR}/mama" "${OUT_DIR}/Open Setup UI.command" "${OUT_DIR}/Start Mixer.command" "${OUT_DIR}/Stop Mixer.command"
chmod +x "${APP_MACOS}/MAMA"

mkdir -p "${OUT_DIR}/bin"
mv "${OUT_DIR}/mama" "${OUT_DIR}/bin/mama-cli"
chmod +x "${OUT_DIR}/bin/mama-cli"

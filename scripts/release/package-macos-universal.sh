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

APP_DIR="${OUT_DIR}/MAMA.app"
APP_CONTENTS="${APP_DIR}/Contents"
APP_MACOS="${APP_CONTENTS}/MacOS"
APP_RESOURCES="${APP_CONTENTS}/Resources"
mkdir -p "${APP_MACOS}" "${APP_RESOURCES}"

cp "${OUT_DIR}/mama" "${APP_MACOS}/MAMA"
cp "${OUT_DIR}/config.yaml" "${APP_MACOS}/config.yaml"
cp "${OUT_DIR}/mama-app.png" "${APP_MACOS}/mama-app.png"
cp "${OUT_DIR}/mama-tray.png" "${APP_MACOS}/mama-tray.png"

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
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>LSMinimumSystemVersion</key>
  <string>11.0</string>
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
if [[ -d "./MAMA.app" ]]; then
  exec open -a "./MAMA.app" --args "$@"
fi
exec ./mama "$@"
LAUNCH_UI

cat > "${OUT_DIR}/Start Mixer.command" <<'LAUNCH_MIXER'
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
pid_file=".mama.pid"
if [[ -f "$pid_file" ]]; then
  pid="$(cat "$pid_file" 2>/dev/null || true)"
  if [[ -n "${pid}" ]] && kill -0 "$pid" >/dev/null 2>&1; then
    echo "MAMA is already running in background (PID ${pid})."
    exit 0
  fi
fi
nohup ./mama -open=false -start-hidden=true "$@" >/dev/null 2>&1 &
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

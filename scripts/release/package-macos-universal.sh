#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/release/package-macos-universal.sh <out_dir>

Builds a universal2 (amd64+arm64) macOS portable MAMA directory.
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

cat > "${OUT_DIR}/Open Setup UI.command" <<'LAUNCH_UI'
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
exec ./mama "$@"
LAUNCH_UI

cat > "${OUT_DIR}/Start Mixer.command" <<'LAUNCH_MIXER'
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
exec ./mama -open=false -start-hidden=true "$@"
LAUNCH_MIXER

chmod +x "${OUT_DIR}/mama" "${OUT_DIR}/Open Setup UI.command" "${OUT_DIR}/Start Mixer.command"

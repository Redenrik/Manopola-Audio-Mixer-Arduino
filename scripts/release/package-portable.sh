#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/release/package-portable.sh <goos> <goarch> <out_dir>

Builds a portable MAMA directory for Linux/macOS targets.
USAGE
}

if [[ $# -lt 3 ]]; then
  usage
  exit 1
fi

GOOS_TARGET="$1"
GOARCH_TARGET="$2"
OUT_DIR_INPUT="$3"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MAMA_DIR="${ROOT_DIR}/mama"

if [[ "${OUT_DIR_INPUT}" = /* ]]; then
  OUT_DIR="${OUT_DIR_INPUT}"
else
  OUT_DIR="${ROOT_DIR}/${OUT_DIR_INPUT}"
fi

mkdir -p "${OUT_DIR}"

BIN_NAME="mama"
if [[ "${GOOS_TARGET}" == "windows" ]]; then
  BIN_NAME="mama.exe"
fi

(
  cd "${MAMA_DIR}"
  GOOS="${GOOS_TARGET}" GOARCH="${GOARCH_TARGET}" go build -o "${OUT_DIR}/${BIN_NAME}" ./cmd/mama
)

cp "${MAMA_DIR}/internal/config/default.yaml" "${OUT_DIR}/config.yaml"

if [[ "${GOOS_TARGET}" == "linux" ]]; then
  cat > "${OUT_DIR}/open-setup-ui.sh" <<'LAUNCH_UI'
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
exec ./mama "$@"
LAUNCH_UI
  cat > "${OUT_DIR}/start-mixer.sh" <<'LAUNCH_MIXER'
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
exec ./mama -open=false -start-hidden=true "$@"
LAUNCH_MIXER
  chmod +x "${OUT_DIR}/${BIN_NAME}" "${OUT_DIR}/open-setup-ui.sh" "${OUT_DIR}/start-mixer.sh"
elif [[ "${GOOS_TARGET}" == "darwin" ]]; then
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
  chmod +x "${OUT_DIR}/${BIN_NAME}" "${OUT_DIR}/Open Setup UI.command" "${OUT_DIR}/Start Mixer.command"
fi

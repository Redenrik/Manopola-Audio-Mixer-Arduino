#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/release/package-macos-pkg.sh <portable_dir> <version> <output_pkg>

Builds a macOS .pkg installer from a macOS portable MAMA directory.
Installs /Applications/MAMA.app.
USAGE
}

if [[ $# -lt 3 ]]; then
  usage
  exit 1
fi

PORTABLE_DIR_INPUT="$1"
VERSION_INPUT="$2"
OUTPUT_PKG_INPUT="$3"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [[ "${PORTABLE_DIR_INPUT}" = /* ]]; then
  PORTABLE_DIR="${PORTABLE_DIR_INPUT}"
else
  PORTABLE_DIR="${ROOT_DIR}/${PORTABLE_DIR_INPUT}"
fi

if [[ "${OUTPUT_PKG_INPUT}" = /* ]]; then
  OUTPUT_PKG="${OUTPUT_PKG_INPUT}"
else
  OUTPUT_PKG="${ROOT_DIR}/${OUTPUT_PKG_INPUT}"
fi

if [[ ! -d "${PORTABLE_DIR}" ]]; then
  echo "portable directory not found: ${PORTABLE_DIR}" >&2
  exit 1
fi
if [[ ! -d "${PORTABLE_DIR}/MAMA.app" ]]; then
  echo "portable directory is missing MAMA.app: ${PORTABLE_DIR}/MAMA.app" >&2
  exit 1
fi

if ! command -v pkgbuild >/dev/null 2>&1; then
  echo "pkgbuild not found (required for .pkg packaging)" >&2
  exit 1
fi

VERSION="${VERSION_INPUT#v}"
if [[ -z "${VERSION}" ]]; then
  VERSION="1.0.0"
fi

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

PAYLOAD_ROOT="${TMP_DIR}/payload"
SCRIPTS_DIR="${TMP_DIR}/scripts"
mkdir -p "${PAYLOAD_ROOT}/Applications" "${SCRIPTS_DIR}"
cp -a "${PORTABLE_DIR}/MAMA.app" "${PAYLOAD_ROOT}/Applications/MAMA.app"

cat > "${SCRIPTS_DIR}/preinstall" <<'PREINSTALL'
#!/usr/bin/env bash
set -euo pipefail

# Cleanup legacy folder install layout to avoid duplicate launch entries.
legacy_dir="/Applications/MAMA"
if [[ -d "${legacy_dir}" ]]; then
  if [[ -f "${legacy_dir}/mama" && -f "${legacy_dir}/Open Setup UI.command" ]]; then
    rm -rf "${legacy_dir}"
  fi
fi
PREINSTALL
chmod +x "${SCRIPTS_DIR}/preinstall"

mkdir -p "$(dirname "${OUTPUT_PKG}")"
pkgbuild \
  --root "${PAYLOAD_ROOT}" \
  --identifier "io.mama.mixer.app.installer" \
  --version "${VERSION}" \
  --scripts "${SCRIPTS_DIR}" \
  --install-location "/" \
  "${OUTPUT_PKG}"

echo "macOS installer package ready at: ${OUTPUT_PKG}"

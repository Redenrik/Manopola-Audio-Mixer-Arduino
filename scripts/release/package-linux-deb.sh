#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/release/package-linux-deb.sh <portable_dir> <version> <output_deb> [architecture]

Builds a Debian package from a Linux portable MAMA directory.
USAGE
}

if [[ $# -lt 3 ]]; then
  usage
  exit 1
fi

PORTABLE_DIR_INPUT="$1"
VERSION_INPUT="$2"
OUTPUT_DEB_INPUT="$3"
ARCH_INPUT="${4:-}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [[ "${PORTABLE_DIR_INPUT}" = /* ]]; then
  PORTABLE_DIR="${PORTABLE_DIR_INPUT}"
else
  PORTABLE_DIR="${ROOT_DIR}/${PORTABLE_DIR_INPUT}"
fi

if [[ "${OUTPUT_DEB_INPUT}" = /* ]]; then
  OUTPUT_DEB="${OUTPUT_DEB_INPUT}"
else
  OUTPUT_DEB="${ROOT_DIR}/${OUTPUT_DEB_INPUT}"
fi

if [[ ! -d "${PORTABLE_DIR}" ]]; then
  echo "portable directory not found: ${PORTABLE_DIR}" >&2
  exit 1
fi

HAS_DPKG_DEB=false
if command -v dpkg-deb >/dev/null 2>&1; then
  HAS_DPKG_DEB=true
elif ! command -v ar >/dev/null 2>&1 || ! command -v tar >/dev/null 2>&1; then
  echo "dpkg-deb not found and ar/tar fallback tools are unavailable" >&2
  exit 1
fi

VERSION="${VERSION_INPUT#v}"
if [[ -z "${VERSION}" ]]; then
  VERSION="1.0.0"
fi

ARCHITECTURE="${ARCH_INPUT}"
if [[ -z "${ARCHITECTURE}" ]]; then
  if command -v dpkg >/dev/null 2>&1; then
    ARCHITECTURE="$(dpkg --print-architecture)"
  else
    ARCHITECTURE="amd64"
  fi
fi

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

PKG_ROOT="${TMP_DIR}/pkg"
mkdir -p "${PKG_ROOT}/DEBIAN"
mkdir -p "${PKG_ROOT}/opt/mama"
mkdir -p "${PKG_ROOT}/usr/bin"
mkdir -p "${PKG_ROOT}/usr/share/applications"
mkdir -p "${PKG_ROOT}/usr/share/icons/hicolor/256x256/apps"

cp -a "${PORTABLE_DIR}/." "${PKG_ROOT}/opt/mama/"

cat > "${PKG_ROOT}/DEBIAN/control" <<CONTROL
Package: mama-audio-mixer
Version: ${VERSION}
Section: sound
Priority: optional
Architecture: ${ARCHITECTURE}
Maintainer: MAMA Project
Depends: libgtk-3-0
Description: MAMA Audio Mixer for Arduino
 Physical USB audio mixer runtime and setup UI.
CONTROL

cat > "${PKG_ROOT}/usr/bin/mama" <<'LAUNCHER'
#!/usr/bin/env bash
set -euo pipefail
exec /opt/mama/mama "$@"
LAUNCHER

cat > "${PKG_ROOT}/usr/bin/mama-setup" <<'SETUP'
#!/usr/bin/env bash
set -euo pipefail
exec /opt/mama/open-setup-ui.sh "$@"
SETUP

chmod 0755 "${PKG_ROOT}/usr/bin/mama" "${PKG_ROOT}/usr/bin/mama-setup"

if [[ -f "${PKG_ROOT}/opt/mama/mama-app.png" ]]; then
  cp "${PKG_ROOT}/opt/mama/mama-app.png" "${PKG_ROOT}/usr/share/icons/hicolor/256x256/apps/mama-audio-mixer.png"
fi

cat > "${PKG_ROOT}/usr/share/applications/io.mama.mixer.desktop" <<'DESKTOP'
[Desktop Entry]
Type=Application
Version=1.0
Name=MAMA Audio Mixer
Comment=Configure and run MAMA Audio Mixer
Exec=/opt/mama/open-setup-ui.sh
Icon=mama-audio-mixer
Terminal=false
Categories=AudioVideo;Audio;
DESKTOP

mkdir -p "$(dirname "${OUTPUT_DEB}")"
if [[ "${HAS_DPKG_DEB}" == "true" ]]; then
  dpkg-deb --build "${PKG_ROOT}" "${OUTPUT_DEB}"
else
  echo "dpkg-deb not found; using ar/tar fallback for .deb packaging"
  FALLBACK_DIR="${TMP_DIR}/fallback"
  mkdir -p "${FALLBACK_DIR}/control" "${FALLBACK_DIR}/data"

  cp -a "${PKG_ROOT}/DEBIAN/." "${FALLBACK_DIR}/control/"
  cp -a "${PKG_ROOT}/." "${FALLBACK_DIR}/data/"
  rm -rf "${FALLBACK_DIR}/data/DEBIAN"

  printf '2.0\n' > "${FALLBACK_DIR}/debian-binary"
  (
    cd "${FALLBACK_DIR}/control"
    tar -czf "${FALLBACK_DIR}/control.tar.gz" .
  )
  (
    cd "${FALLBACK_DIR}/data"
    tar -czf "${FALLBACK_DIR}/data.tar.gz" .
  )

  rm -f "${OUTPUT_DEB}"
  (
    cd "${FALLBACK_DIR}"
    ar r "${OUTPUT_DEB}" debian-binary control.tar.gz data.tar.gz
  )
fi
echo "Linux installer package ready at: ${OUTPUT_DEB}"

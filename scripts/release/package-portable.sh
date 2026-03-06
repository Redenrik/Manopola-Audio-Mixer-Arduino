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
  cat > "${OUT_DIR}/stop-mixer.sh" <<'STOP_MIXER'
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
  chmod +x "${OUT_DIR}/${BIN_NAME}" "${OUT_DIR}/open-setup-ui.sh" "${OUT_DIR}/start-mixer.sh" "${OUT_DIR}/stop-mixer.sh"
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
  chmod +x "${OUT_DIR}/${BIN_NAME}" "${OUT_DIR}/Open Setup UI.command" "${OUT_DIR}/Start Mixer.command" "${OUT_DIR}/Stop Mixer.command"
fi

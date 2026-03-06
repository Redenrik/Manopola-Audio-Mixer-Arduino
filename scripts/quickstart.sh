#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MAMA_DIR="$ROOT_DIR/mama"
OUT_DIR="${1:-$ROOT_DIR/dist/mama-quickstart}"

usage() {
  cat <<'USAGE'
Usage: scripts/quickstart.sh [output_dir]

Builds a portable quick-start bundle containing:
- mama app binary (runtime + setup UI)
- config.yaml
- launcher scripts

Example:
  scripts/quickstart.sh
  scripts/quickstart.sh /tmp/mama-quickstart
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if ! command -v go >/dev/null 2>&1; then
  echo "error: Go toolchain not found in PATH. Install Go (1.22+; CI uses 1.24.x) and retry." >&2
  exit 1
fi

if [[ ! -d "$MAMA_DIR" ]]; then
  echo "error: expected Go module directory '$MAMA_DIR' was not found." >&2
  exit 1
fi

mkdir -p "$OUT_DIR"

echo "[1/3] Building mama app..."
(
  cd "$MAMA_DIR"
  go build -o "$OUT_DIR/mama" ./cmd/mama
)

echo "[2/3] Preparing portable config + launchers..."
cp "$MAMA_DIR/internal/config/default.yaml" "$OUT_DIR/config.yaml"

cat > "$OUT_DIR/open-setup-ui.sh" <<'LAUNCH_UI'
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
exec ./mama "$@"
LAUNCH_UI

cat > "$OUT_DIR/start-mixer.sh" <<'LAUNCH_MIXER'
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
nohup ./mama -open=false "$@" >/dev/null 2>&1 &
echo $! > "$pid_file"
echo "MAMA started in background (PID $(cat "$pid_file"))."
LAUNCH_MIXER

cat > "$OUT_DIR/stop-mixer.sh" <<'STOP_MIXER'
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

chmod +x "$OUT_DIR/mama" "$OUT_DIR/open-setup-ui.sh" "$OUT_DIR/start-mixer.sh" "$OUT_DIR/stop-mixer.sh"

cat > "$OUT_DIR/README-QUICKSTART.txt" <<'NOTES'
MAMA quick-start bundle

1) Run ./open-setup-ui.sh
2) In the browser setup UI, detect/test your board and save your mappings
3) Run ./start-mixer.sh
4) Use ./stop-mixer.sh to stop background runtime

All settings remain local to this folder via config.yaml.
Background PID is stored in .mama.pid.
NOTES

echo "[3/3] Done. Quick-start bundle available at: $OUT_DIR"
echo "Next: run '$OUT_DIR/open-setup-ui.sh'"

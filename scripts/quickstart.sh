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
exec ./mama -open=false -start-hidden=true "$@"
LAUNCH_MIXER

chmod +x "$OUT_DIR/mama" "$OUT_DIR/open-setup-ui.sh" "$OUT_DIR/start-mixer.sh"

cat > "$OUT_DIR/README-QUICKSTART.txt" <<'NOTES'
MAMA quick-start bundle

1) Run ./open-setup-ui.sh
2) In the browser setup UI, detect/test your board and save your mappings
3) Run ./start-mixer.sh

All settings remain local to this folder via config.yaml.
NOTES

echo "[3/3] Done. Quick-start bundle available at: $OUT_DIR"
echo "Next: run '$OUT_DIR/open-setup-ui.sh'"

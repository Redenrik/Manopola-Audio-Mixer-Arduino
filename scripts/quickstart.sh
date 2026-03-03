#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MAMA_DIR="$ROOT_DIR/mama"
OUT_DIR="${1:-$ROOT_DIR/dist/mama-quickstart}"

mkdir -p "$OUT_DIR"

echo "[1/4] Building mama runtime..."
(
  cd "$MAMA_DIR"
  go build -o "$OUT_DIR/mama" ./cmd/mama
)

echo "[2/4] Building mama setup UI..."
(
  cd "$MAMA_DIR"
  go build -o "$OUT_DIR/mama-ui" ./cmd/mama-ui
)

echo "[3/4] Preparing portable config + launchers..."
cp "$MAMA_DIR/internal/config/default.yaml" "$OUT_DIR/config.yaml"

cat > "$OUT_DIR/open-setup-ui.sh" <<'LAUNCH_UI'
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
exec ./mama-ui
LAUNCH_UI

cat > "$OUT_DIR/start-mixer.sh" <<'LAUNCH_MIXER'
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
exec ./mama
LAUNCH_MIXER

chmod +x "$OUT_DIR/mama" "$OUT_DIR/mama-ui" "$OUT_DIR/open-setup-ui.sh" "$OUT_DIR/start-mixer.sh"

cat > "$OUT_DIR/README-QUICKSTART.txt" <<'NOTES'
MAMA quick-start bundle

1) Run ./open-setup-ui.sh
2) In the browser setup UI, detect/test your board and save your mappings
3) Run ./start-mixer.sh

All settings remain local to this folder via config.yaml.
NOTES

echo "[4/4] Done. Quick-start bundle available at: $OUT_DIR"
echo "Next: run '$OUT_DIR/open-setup-ui.sh'"

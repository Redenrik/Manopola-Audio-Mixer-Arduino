#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_OUT="$ROOT_DIR/dist/mama-quickstart-smoketest"

cleanup() {
  rm -rf "$TEST_OUT"
}

trap cleanup EXIT

rm -rf "$TEST_OUT"
"$ROOT_DIR/scripts/quickstart.sh" "$TEST_OUT"

required_files=(
  "$TEST_OUT/mama"
  "$TEST_OUT/config.yaml"
  "$TEST_OUT/open-setup-ui.sh"
  "$TEST_OUT/start-mixer.sh"
  "$TEST_OUT/stop-mixer.sh"
  "$TEST_OUT/README-QUICKSTART.txt"
)

for file in "${required_files[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo "Missing expected file: $file" >&2
    exit 1
  fi
done

for exe in "$TEST_OUT/mama" "$TEST_OUT/open-setup-ui.sh" "$TEST_OUT/start-mixer.sh" "$TEST_OUT/stop-mixer.sh"; do
  if [[ ! -x "$exe" ]]; then
    echo "Missing executable bit: $exe" >&2
    exit 1
  fi
done

for help_cmd in \
  "$TEST_OUT/mama --help" \
  "$TEST_OUT/open-setup-ui.sh --help" \
  "$TEST_OUT/start-mixer.sh --help" \
  "$TEST_OUT/stop-mixer.sh"; do
  if ! bash -c "$help_cmd" >/dev/null; then
    echo "Help command failed: $help_cmd" >&2
    exit 1
  fi
done

echo "quickstart smoke test passed"

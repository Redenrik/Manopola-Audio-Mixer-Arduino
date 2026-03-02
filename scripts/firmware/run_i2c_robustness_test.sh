#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEST_SRC="$ROOT_DIR/firmware/slave/tests/i2c_packet_integrity_test.cpp"
OUT_BIN="$ROOT_DIR/firmware/slave/tests/i2c_packet_integrity_test"

c++ -std=c++17 -Wall -Wextra -pedantic "$TEST_SRC" -o "$OUT_BIN"
"$OUT_BIN"

#!/usr/bin/env bash
set -euo pipefail

ITERATIONS="${1:-25}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ARTIFACT_ROOT="${ROOT_DIR}/artifacts/soak"
RUN_ID="$(date -u +"%Y%m%dT%H%M%SZ")"
RUN_DIR="${ARTIFACT_ROOT}/${RUN_ID}"
SUMMARY_FILE="${RUN_DIR}/summary.txt"

mkdir -p "${RUN_DIR}"

{
  echo "run_id=${RUN_ID}"
  echo "iterations=${ITERATIONS}"
  echo "start_utc=$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
} > "${SUMMARY_FILE}"

run_check() {
  local name="$1"
  local cmd="$2"
  local attempt="$3"

  local log_file="${RUN_DIR}/${attempt}_${name}.log"
  echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] iteration=${attempt} check=${name} cmd=${cmd}" | tee -a "${SUMMARY_FILE}"

  if (cd "${ROOT_DIR}/mama" && bash -lc "${cmd}") >"${log_file}" 2>&1; then
    echo "iteration=${attempt} check=${name} status=PASS log=$(basename "${log_file}")" | tee -a "${SUMMARY_FILE}"
  else
    echo "iteration=${attempt} check=${name} status=FAIL log=$(basename "${log_file}")" | tee -a "${SUMMARY_FILE}"
    echo "end_utc=$(date -u +"%Y-%m-%dT%H:%M:%SZ")" >> "${SUMMARY_FILE}"
    echo "result=FAIL" >> "${SUMMARY_FILE}"
    exit 1
  fi
}

for i in $(seq 1 "${ITERATIONS}"); do
  run_check "unit_all" "go test ./..." "${i}"
  run_check "runtime_focus" "go test ./internal/runtime ./internal/proto" "${i}"
done

echo "end_utc=$(date -u +"%Y-%m-%dT%H:%M:%SZ")" >> "${SUMMARY_FILE}"
echo "result=PASS" >> "${SUMMARY_FILE}"
echo "Soak verification completed successfully. Artifacts: ${RUN_DIR}"

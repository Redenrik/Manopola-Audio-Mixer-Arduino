#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MAMA_DIR="${ROOT_DIR}/mama"
ARTIFACT_ROOT="${ROOT_DIR}/artifacts/readiness"
RUN_ID="$(date -u +"%Y%m%dT%H%M%SZ")"
RUN_DIR="${ARTIFACT_ROOT}/${RUN_ID}"
SUMMARY_FILE="${RUN_DIR}/summary.txt"

mkdir -p "${RUN_DIR}"

{
  echo "run_id=${RUN_ID}"
  echo "start_utc=$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  echo "root_dir=${ROOT_DIR}"
} > "${SUMMARY_FILE}"

run_check() {
  local name="$1"
  shift
  local log_file="${RUN_DIR}/${name}.log"
  echo "check=${name} status=RUNNING command=$*" | tee -a "${SUMMARY_FILE}"
  if "$@" >"${log_file}" 2>&1; then
    echo "check=${name} status=PASS log=$(basename "${log_file}")" | tee -a "${SUMMARY_FILE}"
  else
    echo "check=${name} status=FAIL log=$(basename "${log_file}")" | tee -a "${SUMMARY_FILE}"
    echo "end_utc=$(date -u +"%Y-%m-%dT%H:%M:%SZ")" >> "${SUMMARY_FILE}"
    echo "result=FAIL" >> "${SUMMARY_FILE}"
    exit 1
  fi
}

run_check "go_test" bash -lc "cd \"${MAMA_DIR}\" && go test ./..."
run_check "go_build_host" bash -lc "cd \"${MAMA_DIR}\" && go build ./..."
run_check "go_vet" bash -lc "cd \"${MAMA_DIR}\" && go vet ./..."
run_check "go_mod_verify" bash -lc "cd \"${MAMA_DIR}\" && go mod verify"

if command -v govulncheck >/dev/null 2>&1; then
  run_check "govulncheck" bash -lc "cd \"${MAMA_DIR}\" && govulncheck ./..."
else
  echo "check=govulncheck status=SKIP reason=tool_not_installed" | tee -a "${SUMMARY_FILE}"
fi

for target in "linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64" "windows/386"; do
  IFS="/" read -r goos goarch <<<"${target}"
  run_check "cross_build_${goos}_${goarch}" bash -lc "cd \"${MAMA_DIR}\" && GOOS=${goos} GOARCH=${goarch} go build ./cmd/mama"
done

echo "end_utc=$(date -u +"%Y-%m-%dT%H:%M:%SZ")" >> "${SUMMARY_FILE}"
echo "result=PASS" >> "${SUMMARY_FILE}"
echo "Production readiness checks completed successfully."
echo "Artifacts: ${RUN_DIR}"

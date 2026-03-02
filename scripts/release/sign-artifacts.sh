#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <artifact_dir>" >&2
  exit 1
fi

artifact_dir="$1"
if [[ ! -d "$artifact_dir" ]]; then
  echo "artifact directory not found: $artifact_dir" >&2
  exit 1
fi

if ! command -v cosign >/dev/null 2>&1; then
  echo "cosign not found in PATH" >&2
  exit 1
fi

if [[ -z "${COSIGN_EXPERIMENTAL:-}" ]]; then
  export COSIGN_EXPERIMENTAL=1
fi

shopt -s nullglob
files=("$artifact_dir"/*)
shopt -u nullglob

if [[ ${#files[@]} -eq 0 ]]; then
  echo "no files found to sign in $artifact_dir" >&2
  exit 1
fi

for file in "${files[@]}"; do
  if [[ ! -f "$file" ]]; then
    continue
  fi

  base_name="$(basename "$file")"
  if [[ "$base_name" == *.sig || "$base_name" == *.pem || "$base_name" == *.bundle ]]; then
    continue
  fi

  echo "signing: $base_name"
  cosign sign-blob --yes \
    --output-signature "$file.sig" \
    --output-certificate "$file.pem" \
    "$file"
done

echo "signed artifacts in $artifact_dir"

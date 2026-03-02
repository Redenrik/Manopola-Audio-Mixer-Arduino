#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <artifact_dir> [output_file]" >&2
  exit 1
fi

artifact_dir="$1"
output_file="${2:-$artifact_dir/SHA256SUMS.txt}"

if [[ ! -d "$artifact_dir" ]]; then
  echo "artifact directory not found: $artifact_dir" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  sum_cmd=(sha256sum)
elif command -v shasum >/dev/null 2>&1; then
  sum_cmd=(shasum -a 256)
else
  echo "missing checksum tool: install sha256sum or shasum" >&2
  exit 1
fi

mapfile -t files < <(find "$artifact_dir" -maxdepth 1 -type f ! -name "$(basename "$output_file")" -print | sort)

if [[ ${#files[@]} -eq 0 ]]; then
  echo "no artifacts found in $artifact_dir" >&2
  exit 1
fi

: > "$output_file"
for file in "${files[@]}"; do
  filename="$(basename "$file")"
  (
    cd "$artifact_dir"
    "${sum_cmd[@]}" "$filename"
  ) >> "$output_file"
done

echo "wrote checksums: $output_file"

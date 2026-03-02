#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 3 || $# -gt 4 ]]; then
  echo "usage: $0 <artifact-path> <version> <download-url> [output-file]" >&2
  exit 1
fi

artifact_path="$1"
version="$2"
download_url="$3"
output_file="${4:-$(dirname "$artifact_path")/update-manifest.json}"

if [[ ! -f "$artifact_path" ]]; then
  echo "artifact not found: $artifact_path" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  checksum="$(sha256sum "$artifact_path" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  checksum="$(shasum -a 256 "$artifact_path" | awk '{print $1}')"
else
  echo "sha256sum/shasum is required" >&2
  exit 1
fi

size="$(wc -c < "$artifact_path" | tr -d '[:space:]')"
name="$(basename "$artifact_path")"
generated_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

cat > "$output_file" <<JSON
{
  "version": "$version",
  "published_at": "$generated_at",
  "artifact": {
    "name": "$name",
    "url": "$download_url",
    "sha256": "$checksum",
    "size": $size
  }
}
JSON

echo "Update manifest generated at: $output_file"

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

required=(APPLE_DEVELOPER_ID_APP APPLE_NOTARY_KEY_ID APPLE_NOTARY_ISSUER_ID APPLE_NOTARY_PRIVATE_KEY)
for key in "${required[@]}"; do
  if [[ -z "${!key:-}" ]]; then
    echo "missing required env var: $key" >&2
    exit 1
  fi
done

if ! command -v xcrun >/dev/null 2>&1; then
  echo "xcrun is required for notarization" >&2
  exit 1
fi

if [[ ! -d "$artifact_dir/notarized" ]]; then
  mkdir -p "$artifact_dir/notarized"
fi

key_file="$RUNNER_TEMP/notary-api-key.p8"
printf '%s' "$APPLE_NOTARY_PRIVATE_KEY" > "$key_file"

shopt -s nullglob
zip_files=("$artifact_dir"/*.zip)
shopt -u nullglob

if [[ ${#zip_files[@]} -eq 0 ]]; then
  echo "no macOS zip artifacts found in $artifact_dir; skipping notarization"
  exit 0
fi

for zip_file in "${zip_files[@]}"; do
  base_name="$(basename "$zip_file")"
  work_dir="$RUNNER_TEMP/notary-$base_name"
  rm -rf "$work_dir"
  mkdir -p "$work_dir"

  unzip -q "$zip_file" -d "$work_dir"
  app_path=""
  while IFS= read -r candidate; do
    app_path="$candidate"
    break
  done < <(find "$work_dir" -maxdepth 3 -type d -name '*.app' | sort)

  if [[ -z "$app_path" ]]; then
    echo "no .app bundle found in $base_name; skipping"
    continue
  fi

  echo "codesigning: $app_path"
  codesign --force --deep --timestamp --options runtime --sign "$APPLE_DEVELOPER_ID_APP" "$app_path"

  signed_zip="$artifact_dir/notarized/$base_name"
  (
    cd "$work_dir"
    ditto -c -k --keepParent "$(basename "$app_path")" "$signed_zip"
  )

  echo "submitting for notarization: $signed_zip"
  xcrun notarytool submit "$signed_zip" \
    --key "$key_file" \
    --key-id "$APPLE_NOTARY_KEY_ID" \
    --issuer "$APPLE_NOTARY_ISSUER_ID" \
    --wait

  extracted_notarized_dir="$RUNNER_TEMP/notarized-$base_name"
  rm -rf "$extracted_notarized_dir"
  mkdir -p "$extracted_notarized_dir"
  unzip -q "$signed_zip" -d "$extracted_notarized_dir"

  notarized_app=""
  while IFS= read -r candidate; do
    notarized_app="$candidate"
    break
  done < <(find "$extracted_notarized_dir" -maxdepth 3 -type d -name '*.app' | sort)

  if [[ -n "$notarized_app" ]]; then
    xcrun stapler staple "$notarized_app"
    (
      cd "$extracted_notarized_dir"
      ditto -c -k --keepParent "$(basename "$notarized_app")" "$signed_zip"
    )
    echo "notarized + stapled: $signed_zip"
  fi
done

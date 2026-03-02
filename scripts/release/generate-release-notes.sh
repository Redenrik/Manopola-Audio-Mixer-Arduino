#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 3 ]]; then
  echo "usage: $0 <release-tag> [previous-tag] [output-file]" >&2
  exit 1
fi

release_tag="$1"
previous_tag="${2:-}"
output_file="${3:-release-notes-${release_tag}.md}"

if ! git rev-parse --git-dir >/dev/null 2>&1; then
  echo "must run inside a git repository" >&2
  exit 1
fi

if ! git rev-parse --verify --quiet "refs/tags/${release_tag}" >/dev/null; then
  echo "release tag not found: ${release_tag}" >&2
  exit 1
fi

if [[ -z "$previous_tag" ]]; then
  previous_tag="$(git tag --sort=-creatordate | awk -v current="$release_tag" '$0 != current {print; exit}')"
fi

range="${release_tag}"
if [[ -n "$previous_tag" ]]; then
  if ! git rev-parse --verify --quiet "refs/tags/${previous_tag}" >/dev/null; then
    echo "previous tag not found: ${previous_tag}" >&2
    exit 1
  fi
  range="${previous_tag}..${release_tag}"
fi

release_date="$(git log -1 --format=%cs "refs/tags/${release_tag}")"
mapfile -t commits < <(git log --no-merges --pretty=format:'%H%x09%s' "$range")

declare -A sections=(
  [feat]="Features"
  [fix]="Fixes"
  [docs]="Documentation"
  [ci]="CI"
  [build]="Build"
  [test]="Tests"
  [refactor]="Refactors"
  [perf]="Performance"
  [chore]="Chores"
)
section_order=(feat fix docs ci build test refactor perf chore)

declare -A grouped
other_lines=()

for entry in "${commits[@]}"; do
  sha="${entry%%$'	'*}"
  subject="${entry#*$'	'}"
  if [[ -z "$sha" || -z "$subject" ]]; then
    continue
  fi

  short_sha="$(git rev-parse --short "$sha")"
  normalized="${subject,,}"
  category=""

  for key in "${section_order[@]}"; do
    if [[ "$normalized" =~ ^${key}(\(.+\))?: ]]; then
      category="$key"
      break
    fi
  done

  line="- ${subject} (\`${short_sha}\`)"
  if [[ -n "$category" ]]; then
    grouped["$category"]+="$line"$'
'
  else
    other_lines+=("$line")
  fi
done

{
  echo "# Release ${release_tag}"
  echo
  echo "Released: ${release_date}"
  echo
  if [[ -n "$previous_tag" ]]; then
    echo "Changes since ${previous_tag}."
  else
    echo "Changes included in this release."
  fi
  echo

  wrote_section=false
  for key in "${section_order[@]}"; do
    content="${grouped[$key]:-}"
    if [[ -n "$content" ]]; then
      wrote_section=true
      echo "## ${sections[$key]}"
      echo
      printf '%s' "$content"
      echo
    fi
  done

  if [[ ${#other_lines[@]} -gt 0 ]]; then
    wrote_section=true
    echo "## Other"
    echo
    printf '%s
' "${other_lines[@]}"
    echo
  fi

  if [[ "$wrote_section" == false ]]; then
    echo "- No user-facing commits were detected in this range."
    echo
  fi

  echo "## Full Changelog"
  echo
  if [[ -n "$previous_tag" ]]; then
    echo "- ${previous_tag}...${release_tag}"
  else
    echo "- ${release_tag}"
  fi
} > "$output_file"

echo "Generated release notes: ${output_file}"

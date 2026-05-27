#!/usr/bin/env bash

set -euo pipefail

ORIGIN=${ORIGIN:-origin}

# Default: cut beta off the latest GA as-is (v1.49.0 → v1.49.0-beta.N).
# BUMP=patch|minor|major (or bugfix|feature|breaking) bumps the base first.
BUMP=${BUMP:-}

set +e
git fetch --tags "${ORIGIN}" &>/dev/null
set -e

# Latest GA tag only (vX.Y.Z); beta tags are excluded.
latest_ga_tag="$(git tag -l | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -n1)"
if [[ -z "${latest_ga_tag}" ]]; then
  echo "Error: no GA tag found (expected format vX.Y.Z)."
  echo "Create the first GA tag (e.g. with 'make tag') before cutting a beta."
  exit 1
fi
latest_ga="${latest_ga_tag#v}"

IFS='.' read -r major minor patch <<< "${latest_ga}"
case "${BUMP}" in
  "")
    ;;
  bugfix | patch)
    patch=$((patch + 1))
    ;;
  feature | minor)
    minor=$((minor + 1))
    patch=0
    ;;
  breaking | major)
    major=$((major + 1))
    minor=0
    patch=0
    ;;
  *)
    echo "Error: invalid BUMP='${BUMP}'."
    echo "Use one of: bugfix, patch, feature, minor, breaking, major."
    echo "Or unset BUMP to use the latest GA (${latest_ga_tag}) as-is."
    exit 1
    ;;
esac
target_base="${major}.${minor}.${patch}"

base_escaped="${target_base//./\\.}"
max_beta=0
existing_beta_tags=()
while IFS= read -r tag; do
  [[ -z "${tag}" ]] && continue
  if [[ "${tag}" =~ ^v${base_escaped}-beta\.([0-9]+)$ ]]; then
    num="${BASH_REMATCH[1]}"
    existing_beta_tags+=("${tag}")
    if (( num > max_beta )); then
      max_beta="${num}"
    fi
  fi
done < <(git tag -l | grep -E "^v${base_escaped}-beta\\.[0-9]+$" | sort -V)

beta_num=$((max_beta + 1))
new_tag="v${target_base}-beta.${beta_num}"

if [[ $(git status --porcelain) != "" ]]; then
  echo "Error: repo is dirty. Run git status, clean repo and try again."
  exit 1
elif [[ $(git status --porcelain -b | grep -e "ahead" -e "behind") != "" ]]; then
  echo "Error: repo has unpushed commits. Push commits to remote and try again."
  exit 1
fi

if git rev-parse "${new_tag}" >/dev/null 2>&1; then
  echo "Error: tag ${new_tag} already exists."
  if ((${#existing_beta_tags[@]} > 0)); then
    echo "Existing beta tags for v${target_base}: ${existing_beta_tags[*]}"
  fi
  echo "Delete the duplicate tag locally/remotely, or fetch tags from ${ORIGIN}, then retry."
  exit 1
fi

if [[ -n "${BUMP}" ]]; then
  echo "Latest GA: ${latest_ga_tag} → BUMP=${BUMP} → beta base: v${target_base}"
else
  echo "Latest GA: ${latest_ga_tag} (no BUMP) → beta base: v${target_base}"
fi
if ((${#existing_beta_tags[@]} > 0)); then
  echo "Existing betas: ${existing_beta_tags[*]} → next: ${new_tag}"
else
  echo "No existing beta tags for v${target_base} → creating ${new_tag}"
fi
echo "Creating beta tag ${new_tag} (pushes to ${ORIGIN})"
git tag -m "customer beta release ${new_tag}" -a "${new_tag}"
git push "${ORIGIN}" tag "${new_tag}"
echo "Pushed ${new_tag}. Watch: Actions -> beta-release"

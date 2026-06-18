#!/usr/bin/env sh
set -eu

# Builds deterministic Docker/OCI image tags for CI and release pipelines.
#
# The script writes two files:
# - OUT_FILE: newline-separated tags to pass to image build/publish tooling;
# - IMAGE_REF_FILE: one or more fully qualified image references for release
#   notes, deployment steps, or smoke checks.
#
# Release tags must start with "v". A tag such as v1.2.3 emits 1.2.3, 1.2, 1,
# and latest. Non-release builds always emit a short SHA tag, optionally add a
# sanitized branch tag, and add latest for pushes to the default branch.
OUT_FILE="${1:-.tags}"
IMAGE_REF_FILE="${2:-dist/IMAGE_RELEASE_REF.txt}"

REGISTRY="${REGISTRY:-registry.digixoil.se}"
REPO_PATH="${IMAGE_REPO_PATH:-${CI_REPO:-digixoil/lang-forge}}"
IMAGE_REPO="${IMAGE_REPO:-${REGISTRY}/${REPO_PATH}}"
EVENT="${CI_PIPELINE_EVENT:-${CI_BUILD_EVENT:-}}"
TAG="${CI_COMMIT_TAG:-}"
BRANCH="${CI_COMMIT_BRANCH:-}"
DEFAULT_BRANCH="${CI_REPO_DEFAULT_BRANCH:-main}"
SHA="${CI_COMMIT_SHA:-unknown}"

mkdir -p "$(dirname "$OUT_FILE")"
mkdir -p "$(dirname "$IMAGE_REF_FILE")"
: > "$OUT_FILE"
: > "$IMAGE_REF_FILE"

append_tag() {
    # Append a unique non-empty tag while preserving first-seen order.
    tag="$1"
    [ -z "$tag" ] && return 0
    grep -qxF "$tag" "$OUT_FILE" 2>/dev/null || printf '%s\n' "$tag" >> "$OUT_FILE"
}

sanitize_branch() {
    # Convert branch names into lowercase image-tag-safe strings.
    branch="$1"
    branch=$(printf '%s' "$branch" | tr '[:upper:]' '[:lower:]')
    branch=$(printf '%s' "$branch" | tr '/ _' '---')
    branch=$(printf '%s' "$branch" | tr -cd 'a-z0-9._-')
    while [ "${branch#-}" != "$branch" ]; do branch="${branch#-}"; done
    while [ "${branch%-}" != "$branch" ]; do branch="${branch%-}"; done
    [ -n "$branch" ] || branch="branch"
    printf '%s' "$branch"
}

if [ -n "$TAG" ]; then
    case "$TAG" in
        v*) ;;
        *)
            echo "release image tags require a Git tag starting with 'v' (got: $TAG)" >&2
            exit 1
            ;;
    esac

    tag_no_v="${TAG#v}"
    [ -n "$tag_no_v" ] || tag_no_v="$TAG"
    append_tag "$tag_no_v"

    if printf '%s' "$tag_no_v" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
        major=$(printf '%s' "$tag_no_v" | cut -d. -f1)
        minor=$(printf '%s' "$tag_no_v" | cut -d. -f1,2)
        append_tag "$major"
        append_tag "$minor"
        append_tag latest
    fi

    printf '%s\n' "${IMAGE_REPO}:${tag_no_v}" > "$IMAGE_REF_FILE"
    exit 0
fi

short_sha=$(printf '%s' "$SHA" | cut -c1-12)
append_tag "sha-${short_sha}"

if [ -n "$BRANCH" ]; then
    safe_branch=$(sanitize_branch "$BRANCH")
    append_tag "$safe_branch"
    if [ "$BRANCH" = "$DEFAULT_BRANCH" ]; then
        append_tag latest
    fi
fi

printf '%s\n' "${IMAGE_REPO}:sha-${short_sha}" > "$IMAGE_REF_FILE"

if [ "$EVENT" = "push" ] && [ "$BRANCH" = "$DEFAULT_BRANCH" ]; then
    printf '%s\n' "${IMAGE_REPO}:latest" >> "$IMAGE_REF_FILE"
fi

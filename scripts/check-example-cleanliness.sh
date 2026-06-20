#!/usr/bin/env sh
set -eu

# Verifies that example build output remains local developer state rather than
# source state. Generated folders are intentionally allowed to exist in working
# trees after `make examples-test`; this check only fails when such artifacts are
# tracked by Git and would be published with the source package.

if ! command -v git >/dev/null 2>&1; then
    printf '%s\n' "git is required for the example source-cleanliness check."
    printf '%s\n' "Install git before running make examples-cleanliness or make examples-test."
    exit 127
fi

if ! repo_root=$(git rev-parse --show-toplevel 2>/dev/null); then
    printf '%s\n' "example source-cleanliness check skipped: not inside a Git worktree"
    exit 0
fi

cd "$repo_root"

tracked=$(
    git ls-files -- \
        'examples/**/generated/**' \
        'examples/**/Generated/**' \
        'examples/**/dist/**' \
        'examples/**/bin/**' \
        'examples/**/obj/**' \
        'examples/**/*.log' \
        'examples/**/*.png' \
        'examples/**/*.ppm' \
        'examples/**/*.exe' \
        'examples/**/*.dll' \
        'examples/**/*.so'
)

if [ -n "$tracked" ]; then
    printf '%s\n' "example artifact paths are tracked and should be source-clean:"
    printf '%s\n' "$tracked"
    exit 1
fi

printf '%s\n' "example source-cleanliness check passed"

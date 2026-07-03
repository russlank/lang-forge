#!/bin/sh
set -eu

# Guard against generic environment-variable collisions. Example generation
# targets must use LF_TARGET/LANGFORGE_TARGET, not a shell or CI variable named
# TARGET. The check uses make dry-runs so it verifies command construction
# without regenerating files.

script_dir=$(CDPATH= cd "$(dirname "$0")" && pwd)
repo_root=$(CDPATH= cd "$script_dir/.." && pwd)
cd "$repo_root"

check_case() {
    dir=$1
    expected=$2
    output=$(TARGET=env-collision make -n -C "$dir" GO=go DOTNET=dotnet CXX=c++ generate 2>&1)
    if printf '%s\n' "$output" | grep -q -- '--target env-collision'; then
        printf '%s\n' "$output"
        printf 'generic TARGET leaked into LangForge generation in %s\n' "$dir" >&2
        exit 1
    fi
    if ! printf '%s\n' "$output" | grep -q -- "--target $expected"; then
        printf '%s\n' "$output"
        printf 'expected --target %s in dry-run output for %s\n' "$expected" "$dir" >&2
        exit 1
    fi
}

check_case examples/go/calc go
check_case examples/csharp/calc csharp
check_case examples/c/calc c
check_case examples/cpp/calc cpp

printf 'example LangForge target variable smoke check passed\n'

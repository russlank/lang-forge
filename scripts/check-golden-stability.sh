#!/usr/bin/env sh
set -eu

# Regenerates representative artifacts twice and byte-compares the outputs.
# This is intentionally source-only: it proves deterministic generation without
# committing bulky generated golden directories to the repository.

GO="${GO:-go}"
repo_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
tmp_root=$(mktemp -d "${TMPDIR:-/tmp}/lang-forge-stability.XXXXXX")

cleanup() {
    rm -rf "$tmp_root"
}
trap cleanup EXIT HUP INT TERM

cd "$repo_root"

compare_files() {
    compare_left=$1
    compare_right=$2
    compare_label=$3
    if ! cmp -s "$compare_left" "$compare_right"; then
        printf '%s\n' "stability mismatch: $compare_label"
        diff -u "$compare_left" "$compare_right" || true
        exit 1
    fi
}

compare_dirs() {
    left=$1
    right=$2
    label=$3
    safe_label=$(printf '%s' "$label" | tr ' /' '__')
    left_list="$tmp_root/$safe_label.left.files"
    right_list="$tmp_root/$safe_label.right.files"
    (cd "$left" && find . -type f | sort) > "$left_list"
    (cd "$right" && find . -type f | sort) > "$right_list"
    compare_files "$left_list" "$right_list" "$label file list"
    while IFS= read -r path; do
        compare_files "$left/$path" "$right/$path" "$label $path"
    done < "$left_list"
}

check_inspect() {
    spec=$1
    label=$2
    first="$tmp_root/$label.1.json"
    second="$tmp_root/$label.2.json"
    "$GO" run ./cmd/lang-forge inspect --spec "$spec" --format json > "$first"
    "$GO" run ./cmd/lang-forge inspect --spec "$spec" --format json > "$second"
    compare_files "$first" "$second" "$label inspect json"
}

check_generate() {
    spec=$1
    target=$2
    label=$3
    first="$tmp_root/$label.1"
    second="$tmp_root/$label.2"
    "$GO" run ./cmd/lang-forge generate --spec "$spec" --target "$target" --out "$first" >/dev/null
    "$GO" run ./cmd/lang-forge generate --spec "$spec" --target "$target" --out "$second" >/dev/null
    compare_dirs "$first" "$second" "$label generated output"
}

check_inspect "examples/go/calc/calc.lf" "calc"
check_inspect "examples/parser-algorithms/mysterious-conflict-ielr.lf" "ielr"
check_generate "examples/go/calc/calc.lf" "go" "calc-go"
check_generate "examples/csharp/calc/calc.lf" "csharp" "calc-csharp"
check_generate "examples/c/calc/calc.lf" "c" "calc-c"
check_generate "examples/cpp/calc/calc.lf" "cpp" "calc-cpp"

printf '%s\n' "golden stability check passed"

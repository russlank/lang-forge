#!/usr/bin/env sh
set -eu

# Exercises shared example fixtures through the Go examples. The other
# language examples use the same shared valid inputs through their Makefile
# defaults; this script adds deterministic golden and invalid-input checks
# without multiplying the full cross-target runtime cost.

GO=${GO:-go}

if command -v git >/dev/null 2>&1 && repo_root=$(git rev-parse --show-toplevel 2>/dev/null); then
    cd "$repo_root"
else
    cd "$(dirname "$0")/.."
fi

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/langforge-example-testdata.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT

check_contains() {
    report=$1
    expected=$2
    while IFS= read -r line || [ -n "$line" ]; do
        [ -z "$line" ] && continue
        if ! grep -F -- "$line" "$report" >/dev/null 2>&1; then
            printf 'missing golden fragment in %s:\n%s\n' "$report" "$line" >&2
            printf '%s\n' '--- report ---' >&2
            cat "$report" >&2
            exit 1
        fi
    done < "$expected"
}

expect_fail() {
    label=$1
    shift
    if "$@" >"$tmp_dir/$label.out" 2>"$tmp_dir/$label.err"; then
        printf 'expected %s to fail, but it succeeded\n' "$label" >&2
        exit 1
    fi
}

make -C examples/go/calc GO="$GO" run \
    INPUT=../../testdata/calc/valid/basic.calc \
    LOG="$tmp_dir/calc.log" >/dev/null
check_contains "$tmp_dir/calc.log" examples/testdata/calc/golden/report.contains
expect_fail calc-scanner make -C examples/go/calc GO="$GO" run \
    INPUT=../../testdata/calc/invalid/scanner.calc \
    LOG="$tmp_dir/calc-scanner.log"
expect_fail calc-parser make -C examples/go/calc GO="$GO" run \
    INPUT=../../testdata/calc/invalid/parser.calc \
    LOG="$tmp_dir/calc-parser.log"

make -C examples/go/datakeeper GO="$GO" run \
    INPUT=../../testdata/datakeeper/valid/basic.dks \
    LOG="$tmp_dir/datakeeper.log" >/dev/null
check_contains "$tmp_dir/datakeeper.log" examples/testdata/datakeeper/golden/report.contains
expect_fail datakeeper-scanner make -C examples/go/datakeeper GO="$GO" run \
    INPUT=../../testdata/datakeeper/invalid/scanner.dks \
    LOG="$tmp_dir/datakeeper-scanner.log"
expect_fail datakeeper-parser make -C examples/go/datakeeper GO="$GO" run \
    INPUT=../../testdata/datakeeper/invalid/parser.dks \
    LOG="$tmp_dir/datakeeper-parser.log"

make -C examples/go/draw GO="$GO" run \
    INPUT=../../testdata/draw/valid/basic.draw \
    OUTPUT="$tmp_dir/draw.png" \
    LOG="$tmp_dir/draw.log" >/dev/null
check_contains "$tmp_dir/draw.log" examples/testdata/draw/golden/report.contains
actual_signature=$(od -An -tx1 -N8 "$tmp_dir/draw.png" | tr -d ' \n')
expected_signature=$(tr -d ' \n' < examples/testdata/draw/golden/png.signature)
if [ "$actual_signature" != "$expected_signature" ]; then
    printf 'DRAW PNG signature mismatch: got %s want %s\n' "$actual_signature" "$expected_signature" >&2
    exit 1
fi
expect_fail draw-scanner make -C examples/go/draw GO="$GO" run \
    INPUT=../../testdata/draw/invalid/scanner.draw \
    OUTPUT="$tmp_dir/draw-scanner.png" \
    LOG="$tmp_dir/draw-scanner.log"
expect_fail draw-parser make -C examples/go/draw GO="$GO" run \
    INPUT=../../testdata/draw/invalid/parser.draw \
    OUTPUT="$tmp_dir/draw-parser.png" \
    LOG="$tmp_dir/draw-parser.log"

make -C examples/go/vehicle-report GO="$GO" run \
    INPUT=../../testdata/vehicle-report/valid/basic.vehicle \
    LOG="$tmp_dir/vehicle.log" >/dev/null
check_contains "$tmp_dir/vehicle.log" examples/testdata/vehicle-report/golden/report.contains
expect_fail vehicle-scanner make -C examples/go/vehicle-report GO="$GO" run \
    INPUT=../../testdata/vehicle-report/invalid/scanner.vehicle \
    LOG="$tmp_dir/vehicle-scanner.log"
expect_fail vehicle-parser make -C examples/go/vehicle-report GO="$GO" run \
    INPUT=../../testdata/vehicle-report/invalid/parser.vehicle \
    LOG="$tmp_dir/vehicle-parser.log"

printf '%s\n' "example shared testdata check passed"

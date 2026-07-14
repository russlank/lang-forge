#!/usr/bin/env sh
set -eu

# Guards the public LangForge vocabulary and generated API naming conventions.
# Private planning notes and generated output are intentionally excluded so this
# check stays focused on current guidance.

failures=0
private_doc_glob="doc/${LANGFORGE_PRIVATE_DOC_DIR:-project}/**"
private_doc_path="doc/${LANGFORGE_PRIVATE_DOC_DIR:-project}"

search_with_rg() {
    pattern=$1
    shift

    rg -n --hidden \
        --glob "!$private_doc_glob" \
        --glob '!**/Generated/**' \
        --glob '!**/generated/**' \
        --glob '!**/bin/**' \
        --glob '!**/obj/**' \
        --glob '!dist/**' \
        --glob '!.git/**' \
        --glob '!scripts/check-vocabulary.sh' \
        "$pattern" "$@" 2>/dev/null || true
}

search_with_grep() {
    pattern=$1
    shift

    for root in "$@"; do
        if [ ! -e "$root" ]; then
            continue
        fi
        if [ -f "$root" ]; then
            case "$root" in
                scripts/check-vocabulary.sh|"$private_doc_path"/*|*/Generated/*|*/generated/*|*/bin/*|*/obj/*|dist/*|.git/*)
                    continue
                    ;;
            esac
            grep -E -n -I -- "$pattern" "$root" 2>/dev/null || true
            continue
        fi
        find "$root" \
            \( -path "$private_doc_path" -o -path "$private_doc_path/*" \
                -o -path '*/Generated' -o -path '*/Generated/*' \
                -o -path '*/generated' -o -path '*/generated/*' \
                -o -path '*/bin' -o -path '*/bin/*' \
                -o -path '*/obj' -o -path '*/obj/*' \
                -o -path 'dist' -o -path 'dist/*' \
                -o -path '.git' -o -path '.git/*' \) -prune \
            -o -type f ! -path 'scripts/check-vocabulary.sh' \
            -exec grep -E -n -I -- "$pattern" {} + 2>/dev/null || true
    done
}

search_matches() {
    pattern=$1
    shift

    if command -v rg >/dev/null 2>&1; then
        search_with_rg "$pattern" "$@"
    else
        search_with_grep "$pattern" "$@"
    fi
}

check_no_matches() {
    description=$1
    pattern=$2
    shift 2

    matches=$(search_matches "$pattern" "$@")
    if [ -n "$matches" ]; then
        printf '%s\n' "vocabulary mismatch: $description"
        printf '%s\n' "$matches"
        failures=1
    fi
}

check_no_matches \
    "superseded token-source or source-style parse names; use LexemeSource and target-specific source-based APIs" \
    'Token''Source|Parse''FromSource|ParseValue''FromSource|ParseWithReducer''FromSource|ParseRecovering''FromSource' \
    README.md doc examples skills internal scripts Makefile

check_no_matches \
    "C# handwritten examples/docs should use overloads, not generated named aliases" \
    'Parser\.Parse(FromLexemeSource|ValueFromLexemeSource|WithReducerFromLexemeSource|RecoveringFromLexemeSource)|Parse(Value|Recovering)?LexemeSource' \
    README.md doc examples skills

check_no_matches \
    "C token-collection calls should use explicit _tokens APIs" \
    '(^|[^[:alnum:]_])[A-Za-z_][A-Za-z0-9_]*_parse(_value|_recovering|_value_recovering)?[[:space:]]*\([^;]*tokens' \
    examples/c examples/templates/c doc

check_no_matches \
    "ambiguous parser input vocabulary; prefer lexeme source or source text as appropriate" \
    'target-tagged|scanner/source|pulls tokens lazily|tokens lazily' \
    README.md doc examples skills internal

private_path_pattern='doc/'"${LANGFORGE_PRIVATE_DOC_DIR:-project}"'|/'home'/russlan|/'mnt'/c'

check_no_matches \
    "public files should not point readers to private planning paths" \
    "$private_path_pattern" \
    README.md doc examples skills internal scripts Makefile

check_no_matches \
    "public API guidance should avoid pre-release history framing" \
    'Compat''ibility alias|compat''ibility alias|ol''der generated|ol''der convenience|DOS-''era|histori''cal' \
    README.md doc examples skills internal/codegen scripts

if [ "$failures" -ne 0 ]; then
    exit 1
fi

printf '%s\n' "vocabulary check passed"

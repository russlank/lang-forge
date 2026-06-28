# LangForge Usage

Document id: `lang-forge-usage-v1`
Status: `active`
Last updated: `2026-06-29`
Owner: `Project maintainers`
Scope: `CLI usage guide for the current LangForge implementation`

## Build and Test

The core CLI needs Go and `make`. The complete `make ci` and example suite
also needs .NET `10.0` and GCC or another C11 compiler. See
[Requirements](requirements.md) for the full matrix.

```sh
/usr/local/go/bin/go test ./...
/usr/local/go/bin/go build -trimpath -o dist/lang-forge ./cmd/lang-forge
```

```sh
make ci
make build
make dist VERSION=0.1.0
make docker-smoke
make examples-run
```

If you are using LangForge to learn compiler construction, start with
[Learning Path](learning-path.md). If you want the stage-by-stage implementation
map, read [Compiler Pipeline](compiler-pipeline.md). For CI, release artifacts,
container images, and licensing, read [Build, Pipeline, And Docker](build-release.md).
For Makefile templates, Docker-as-the-tool workflows, and multi-parser project
layouts, read [Invocation And Layout Patterns](invocation-and-layouts.md).

## Run Without Installing LangForge

LangForge can be invoked in several interchangeable ways.

From this source checkout:

```sh
/usr/local/go/bin/go run ./cmd/lang-forge validate --spec examples/go/calc/calc.lf
```

From a local build:

```sh
make build
./dist/lang-forge validate --spec examples/go/calc/calc.lf
```

From an installed binary:

```sh
lang-forge validate --spec examples/go/calc/calc.lf
```

From a Docker image:

```sh
make docker-build
docker run --rm -v "$PWD:/workspace:ro" -w /workspace lang-forge:dev \
  validate --spec examples/go/calc/calc.lf
```

Generation needs a writable mount:

```sh
docker run --rm \
  -u "$(id -u):$(id -g)" \
  -v "$PWD:/workspace" \
  -w /workspace \
  lang-forge:dev \
  generate --spec examples/go/calc/calc.lf --target go --out examples/go/calc/generated
```

The `-u` option keeps generated files owned by the host user on Linux and WSL.
It can be omitted on platforms where user mapping is not useful.

## Validate a Combined Spec

```sh
./dist/lang-forge validate --spec examples/go/calc/calc.lf
```

Expected output:

```text
ok: 11 lexer states, 19 parser states, 10 grammar rules
```

The lexer state count can change as minimization improves, but validation
should exit `0` for the calc spec.

Validation also catches common table-generation hazards early, including lexer
rules that can match empty input, invalid Unicode scalar ranges, unsupported
scanner settings, undefined grammar symbols, parser conflicts, and symbols
reused as both tokens and nonterminals.

## Validate Legacy Split Files

```sh
./dist/lang-forge validate \
  --lex testdata/ucdt/calc/calc.l \
  --yacc testdata/ucdt/calc/calc.y
```

The current split-input parser strips
[UCDT](https://github.com/russlank/UCDT)/Pascal action blocks from Yacc rules
and infers token names from `YACC_*` references in Lex action blocks.
This support exists for source-only fixtures and migration experiments, not as
a promise to preserve UCDT syntax or byte-level behavior.
It also accepts the curated calc, DRAW, Lex meta, and Yacc meta fixture pairs
under `testdata/ucdt`.

## Inspect Tables

Human-readable summary:

```sh
./dist/lang-forge inspect --spec examples/go/calc/calc.lf --format text
```

Machine-readable output:

```sh
./dist/lang-forge inspect --spec examples/go/calc/calc.lf --format json > calc.inspect.json
```

JSON inspection includes the parsed spec model, DFA states, grammar model, SLR
or LR(1) states, action/goto tables, lookahead items when available, and
conflicts.

The text output also reports the selected parser algorithm. The default is
`lalr`.

For parser algorithm selection, start with the default LALR(1). Use `%type
ielr` when LALR reports a conflict that should be LR(1), `%type canonical` for
deep diagnosis, and `%type slr` mainly for small grammars or compatibility
checks. See
[Parser Algorithms](parser-algorithms.md) for the automata model, pseudo-code,
and LR(1)-not-SLR example.

## Generate Go Output

```sh
./dist/lang-forge generate \
  --spec examples/go/calc/calc.lf \
  --target go \
  --out examples/go/calc/generated
```

Then verify the generated package:

```sh
/usr/local/go/bin/go test ./examples/go/calc/generated
```

If `%package` is supplied for Go generation, it must already be a valid
non-keyword Go package identifier. When `%package` is omitted, the backend
derives a safe package name from the output directory.

Generated Go scanners return visible tokens by default. Hidden-channel tokens
can be included with `Scanner.IncludeHidden(true)`, but the generated parser
expects visible grammar tokens. The parser accepts token slices from
`Tokenize`, and also accepts one explicit trailing `TokenEOF` for callers that
prefer EOF-marked streams.

Generated parsers can be used in two semantic styles:

- reducer mode, where `{go: add}` becomes both the diagnostic string
  `Reduction.Action` and a generated dispatch enum such as
  `SemanticActionAdd`;
- inline mode, where a Go action block is emitted into generated `parser.go`.

Reducer mode is the default and is what the runnable examples use. See
[Generated Code And Semantics](generated-code-and-semantics.md) for the full
beginner-friendly explanation.

Generated parsers also expose recovery-oriented APIs. `ParseRecovering`
returns a possibly partial value, all syntax diagnostics, and an accepted
flag. The established `Parse` and `ParseValue` entry points still fail when
any syntax diagnostic is produced. Recovery is enabled by grammar alternatives
such as `Statement : error Semi`; see
[Parser Error Recovery](parser-error-recovery.md).

## Generate C# Output

```sh
./dist/lang-forge generate \
  --spec examples/csharp/calc/calc.lf \
  --target csharp \
  --out examples/csharp/calc/Generated
```

Then verify the generated project:

```sh
make -C examples/csharp/calc test
```

For C# generation, `%package` is a namespace such as
`LangForge.Examples.Calc.Generated`. Generated C# scanners operate over .NET
strings and validate Unicode scalar sequences. Scanner instances serialize
their mutable cursor for thread-safe shared use, and parser instances are safe
for concurrent parse calls when the reducer is safe.

## Generate C Output

```sh
./dist/lang-forge generate \
  --spec examples/c/calc/calc.lf \
  --target c \
  --out examples/c/calc/generated
```

Then verify the example project:

```sh
make -C examples/c/calc test
```

For C generation, `%package` becomes the public symbol prefix. For example,
`%package calc` produces names such as `calc_tokenize`, `calc_parse_value`,
`CALC_TOKEN_NUMBER`, and `CALC_ACTION_ADD`. Generated C output uses
conventional filenames: `tokens.h`, `scanner.h`, `scanner.c`, `parser.h`, and
`parser.c`.

The generated C scanner and parser keep their mutable state in caller-owned
structs and parse stacks. That makes independent scanner/parser instances
reentrant and suitable for threaded programs. If a program shares the same
scanner struct between threads, the caller must synchronize access to that
struct.

## Generate C++ Output

```sh
./dist/lang-forge generate \
  --spec examples/cpp/calc/calc.lf \
  --target cpp \
  --out examples/cpp/calc/generated
```

Then verify the example project:

```sh
make -C examples/cpp/calc test
```

For C++ generation, `%package` is a namespace such as
`LangForge::Examples::Calc::Generated`. Generated C++ output uses conventional
filenames: `tokens.hpp`, `scanner.hpp`, `scanner.cpp`, `parser.hpp`, and
`parser.cpp`.

The generated C++ scanner stores a `std::string_view` into caller-owned input,
so the source string must stay alive while lexemes are used. Scanner cursor
methods are protected by a mutex for shared scanner use, and parser calls use
local stacks so they are reentrant. Semantic labels such as `{cpp: add}` become
`SemanticAction::Add` values and can be connected to handwritten code with the
generated `ReducerMap`.

The C++ examples are Makefile-based rather than CMake-based. The repository
includes shared VS Code settings that tell the Microsoft C/C++ extension to use
C++17 and to index the generated include folders for the C++ example family.
Without those settings,
IntelliSense may parse `main.cpp` with an older C++ dialect and incorrectly
underline valid C++17 names such as `std::string_view` or `std::any_cast`.

## Run Example Projects

The example projects regenerate their target-specific scanner/parser code
before building. They use LangForge from source by default:

```sh
make -C examples/go/calc run
make -C examples/go/datakeeper run
make -C examples/go/draw run
make -C examples/go/vehicle-report run
make -C examples/csharp/calc run
make -C examples/csharp/datakeeper run
make -C examples/csharp/draw run
make -C examples/csharp/vehicle-report run
make -C examples/c/calc run
make -C examples/c/datakeeper run
make -C examples/c/draw run
make -C examples/c/vehicle-report run
make -C examples/cpp/calc run
make -C examples/cpp/datakeeper run
make -C examples/cpp/draw run
make -C examples/cpp/vehicle-report run
```

The examples write runnable binaries and report logs under their local `dist`
directories. Generated scanner/parser output lives under local `generated`
directories for Go, C, and C++, and `Generated` directories for C#.
Both paths are ignored by Git.

The C Makefiles validate and generate even when no C compiler is installed.
Compilation and execution are skipped with a clear message when `CC` cannot be
found. To select a compiler explicitly:

```sh
make -C examples/c/calc CC=clang test
```

The C++ Makefiles follow the same pattern. To select a compiler explicitly:

```sh
make -C examples/cpp/calc CXX=clang++ test
```

The generated-dependent Go files are compiled with the build tag
`langforge_generated`. This is why the example Makefiles run commands like:

```sh
go build -tags langforge_generated
go test -tags langforge_generated
```

The tag keeps a source-only checkout buildable before `generated` directories
exist. It is Go conditional compilation, not LangForge grammar syntax.

After building a standalone utility, point the examples at that binary:

```sh
make build
make -C examples/go/calc LANG_FORGE=../../../dist/lang-forge run
make -C examples/go/datakeeper LANG_FORGE=../../../dist/lang-forge run
make -C examples/go/draw LANG_FORGE=../../../dist/lang-forge run
make -C examples/go/vehicle-report LANG_FORGE=../../../dist/lang-forge run
make -C examples/csharp/calc LANG_FORGE=../../../dist/lang-forge run
make -C examples/csharp/datakeeper LANG_FORGE=../../../dist/lang-forge run
make -C examples/csharp/draw LANG_FORGE=../../../dist/lang-forge run
make -C examples/csharp/vehicle-report LANG_FORGE=../../../dist/lang-forge run
make -C examples/c/calc LANG_FORGE=../../../dist/lang-forge run
make -C examples/c/datakeeper LANG_FORGE=../../../dist/lang-forge run
make -C examples/c/draw LANG_FORGE=../../../dist/lang-forge run
make -C examples/c/vehicle-report LANG_FORGE=../../../dist/lang-forge run
```

To point an example Makefile at the Docker image instead, run from the example
directory and override `LANG_FORGE`:

```sh
cd examples/go/calc
LF_DOCKER="docker run --rm -u $(id -u):$(id -g) -v $(pwd):/workspace -w /workspace lang-forge:dev"
make LANG_FORGE="$LF_DOCKER" run
```

The same override works for the Go, C#, C, and C++ example families. See
[Invocation And Layout Patterns](invocation-and-layouts.md) for reusable
Makefile templates and multi-parser project organization.

## Recommended Learning Workflow

For a new grammar or experiment:

1. Write the smallest `.lf` file that validates.
2. Run `validate` after every grammar change.
3. Run `inspect --format text` when state counts or conflicts change.
4. Save `inspect --format json` when you want to compare table shape.
5. Generate into an ignored `generated` directory.
6. Prefer reducer-mode semantic code outside generated files. Use inline mode
   only when a target-specific generated reduction should call handwritten
   APIs directly.
7. Keep generated-dependent Go files behind the `langforge_generated` build
   tag when the generated package is not committed.
8. Add a small test before making the example more complex.

Generated Go files include source comments that point back to the `.lf`, `.l`,
or `.y` rule that produced key table entries. Inline Go semantic snippets also
use Go `//line` directives so target compiler errors can point back to the
grammar source instead of only naming generated `parser.go`.

This workflow keeps the learning loop short while preserving production habits:
deterministic output, repeatable examples, and clear separation between syntax
recognition and domain behavior.

## Exit Codes

| Code | Meaning |
|---:|---|
| 0 | Success |
| 2 | CLI usage error |
| 3 | Spec validation error |
| 4 | Grammar conflict |
| 5 | I/O or generation failure |

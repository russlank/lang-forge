# Invocation And Layout Patterns

Document id: `lang-forge-invocation-layouts-v1`

Status: `active`

Last updated: `2026-07-01`

Owner: `Project maintainers`

Scope: `Practical guide for invoking LangForge and organizing generated parser projects`

This guide shows how to run LangForge when the binary is not installed, how to
shape Makefiles around it, and how to organize projects that contain one or
more generated scanners and parsers.

The important habit is simple: treat LangForge as a command that can be
provided in several ways. Your project Makefile should not care whether that
command is a source checkout, a standalone binary, or a Docker image.

## Choosing How To Run LangForge

During LangForge development, running from source is convenient:

```sh
/usr/local/go/bin/go run ./cmd/lang-forge validate --spec examples/go/calc/calc.lf
```

After building a local binary, use the binary directly:

```sh
make build
./dist/lang-forge validate --spec examples/go/calc/calc.lf
```

If the binary is installed on `PATH`, use the short form:

```sh
lang-forge validate --spec grammar.lf
lang-forge generate --spec grammar.lf --target go --out generated
lang-forge generate --spec grammar.lf --target csharp --out Generated
```

If no binary is installed, the Docker image can be used as the command:

```sh
make docker-build
docker run --rm -v "$PWD:/workspace:ro" -w /workspace lang-forge:dev \
  validate --spec examples/go/calc/calc.lf
```

Use a read-only mount for validation and inspection. Use a writable mount when
generating files:

```sh
docker run --rm \
  -u "$(id -u):$(id -g)" \
  -v "$PWD:/workspace" \
  -w /workspace \
  lang-forge:dev \
  generate --spec examples/go/calc/calc.lf --target go --out examples/go/calc/generated
```

The `-u` option keeps generated files owned by the host user on Linux and WSL.
On platforms where that mapping is not useful, it can be omitted.

## Docker Wrapper Function

For repeated local work, a small shell function makes the image feel like an
installed CLI:

```sh
lf() {
  docker run --rm \
    -u "$(id -u):$(id -g)" \
    -v "$PWD:/workspace" \
    -w /workspace \
    lang-forge:dev "$@"
}

lf version
lf validate --spec examples/go/calc/calc.lf
lf inspect --spec examples/go/calc/calc.lf --format text
```

The function mounts the current directory as `/workspace`, so run it from the
project root, or from the directory that contains the grammar and output paths
you want to use.

Source references in generated files are based on the paths supplied to the
CLI. If you want comments such as `// Source: calc.lf:25:1`, run generation
from the grammar directory and pass `--spec calc.lf`. If you prefer repository
relative paths, run from the repository root and pass
`--spec examples/go/calc/calc.lf`.

## Reusable Makefile Shape

A small generated-on-demand project can use this pattern:

```makefile
GO ?= go
LANG_FORGE ?= lang-forge
LANG_FORGE_VERBOSITY ?= 1
LANG_FORGE_FLAGS ?= --verbosity $(LANG_FORGE_VERBOSITY)

SPEC := grammar.lf
LF_TARGET := go
GENERATED_DIR := generated
GENERATED_TAG := langforge_generated

.PHONY: validate generate test clean

validate:
	$(LANG_FORGE) validate --spec $(SPEC) $(LANG_FORGE_FLAGS)

generate: validate
	$(LANG_FORGE) generate --spec $(SPEC) --target $(LF_TARGET) --out $(GENERATED_DIR) $(LANG_FORGE_FLAGS)

test: generate
	$(GO) test -tags $(GENERATED_TAG) -count=1 ./...

clean:
	rm -rf $(GENERATED_DIR)
```

Use `LF_TARGET` or `LANGFORGE_TARGET` for LangForge generation. Avoid the
generic name `TARGET`; it is common in user shells and CI systems and should
not affect which language backend `lang-forge generate` uses.

The same Makefile can use different LangForge providers:

```sh
make LANG_FORGE=lang-forge generate
make LANG_FORGE=./dist/lang-forge generate
make LANG_FORGE='/usr/local/go/bin/go run ../../../cmd/lang-forge' generate
make LANG_FORGE_VERBOSITY=0 test
make LANG_FORGE_VERBOSITY=2 generate
```

The `LANG_FORGE_VERBOSITY` variable is optional, but useful in examples and
developer workflows. Level `1` shows major stages on stderr, level `2` adds
grammar and lexer decisions, and level `3` prints DFA/parser state rows. Set it
to `0` in quiet CI jobs or when you only want final command output.

When using Docker, define the command once and pass it through `LANG_FORGE`:

```sh
LF_DOCKER="docker run --rm -u $(id -u):$(id -g) -v $(pwd):/workspace -w /workspace lang-forge:dev"
make LANG_FORGE="$LF_DOCKER" generate
```

This works best when the Makefile is run from the project or example
directory. If you use `make -C` from a parent directory, set the container
working directory to the same logical directory as the Makefile:

```sh
REPO_ROOT="$(pwd)"
make -C examples/go/calc \
  LANG_FORGE="docker run --rm -u $(id -u):$(id -g) -v ${REPO_ROOT}:/workspace -w /workspace/examples/go/calc lang-forge:dev" \
  run
```

## Running The LangForge Examples With Docker

The example Makefiles default to running LangForge from source. To use the
Docker image instead, build the image from the repository root, then run the
example from its own directory:

```sh
make docker-build

cd examples/go/calc
LF_DOCKER="docker run --rm -u $(id -u):$(id -g) -v $(pwd):/workspace -w /workspace lang-forge:dev"
make LANG_FORGE="$LF_DOCKER" run
```

The same pattern works for `examples/go/datakeeper`, `examples/go/draw`, and
`examples/go/vehicle-report`.

## Single Grammar Project Layout

For a small DSL, keep the generated package and handwritten semantics separate:

```text
my-tool/
  Makefile
  grammar.lf
  cmd/my-tool/
    main.go
  internal/generated/myparser/
    generated by LangForge
  internal/mydsl/
    ast.go
    reducer.go
    evaluator.go
  testdata/
    valid-001.dsl
    invalid-001.dsl
```

Recommended rules:

- keep `grammar.lf` as the syntax source of truth;
- generate into one directory that can be deleted and recreated;
- keep reducers, AST nodes, interpreters, compilers, and adapters outside the
  generated directory;
- test the handwritten semantic layer through generated parser hooks;
- ignore generated output unless the project deliberately commits generated
  files for consumers that should not need LangForge.

## Multiple Parsers In One Go Program

A larger tool may include several small languages: a query language, a policy
language, a report template language, and an embedded expression grammar. Use
one spec and one generated package per language.

```text
my-platform/
  grammars/
    query/query.lf
    policy/policy.lf
    template/template.lf
  internal/generated/
    queryparser/
    policyparser/
    templateparser/
  internal/query/
    ast.go
    reducer.go
    planner.go
  internal/policy/
    ast.go
    reducer.go
    evaluator.go
  internal/template/
    ast.go
    reducer.go
    renderer.go
  cmd/platform/
    main.go
```

Each `.lf` file should use a distinct Go package name:

```text
%target go
%package queryparser
```

```text
%target go
%package policyparser
```

Then generate each parser into its own directory:

```makefile
LANG_FORGE ?= lang-forge
LANG_FORGE_VERBOSITY ?= 1
LANG_FORGE_FLAGS ?= --verbosity $(LANG_FORGE_VERBOSITY)

.PHONY: generate generate-query generate-policy generate-template

generate: generate-query generate-policy generate-template

generate-query:
	$(LANG_FORGE) generate --spec grammars/query/query.lf --target go --out internal/generated/queryparser $(LANG_FORGE_FLAGS)

generate-policy:
	$(LANG_FORGE) generate --spec grammars/policy/policy.lf --target go --out internal/generated/policyparser $(LANG_FORGE_FLAGS)

generate-template:
	$(LANG_FORGE) generate --spec grammars/template/template.lf --target go --out internal/generated/templateparser $(LANG_FORGE_FLAGS)
```

Generated Go packages are independent, so one program can import several of
them with aliases:

```go
queryAST, err := queryparser.ParseWithReducerFromSource(
	queryparser.NewScanner(querySource),
	queryparser.ReducerFunc(query.Reduce),
)
if err != nil {
	return err
}

policyAST, err := policyparser.ParseWithReducerFromSource(
	policyparser.NewScanner(policySource),
	policyparser.ReducerFunc(policy.Reduce),
)
if err != nil {
	return err
}
```

Use aliases when package names are similar, and keep reducer functions close to
the domain code they build. Shared business helpers should live in ordinary
packages imported by reducers, not inside generated output.

## Multiple Output Languages

The current implementation generates Go, C#, C, and C++ output. Keep generated
output target-specific so each language can use its normal file layout:

```text
grammars/
  calc/calc.lf
generated/
  go/calc/
  csharp/Calc/
  c/calc/
  cpp/calc/
```

The `.lf` file remains the source of truth. Generated directories are
target-specific products. Handwritten semantic code should be placed in the
target's normal source tree and referenced through explicit semantic imports or
reducer wiring.

For portable grammars, prefer short action labels:

```text
Expr : Expr Plus Term {go: add} {csharp: add} {c: add} {cpp: add}
     | Expr Minus Term {go: subtract} {csharp: subtract} {c: subtract} {cpp: subtract}
     ;
```

The labels become Go constants plus a reducer map, C# enum values plus a
`ReducerMap`, C enum values dispatched through a reducer function pointer, and
C++ `enum class` values dispatched through a generated `ReducerMap`. Inline
target code is useful, but it is intentionally less portable.

## Generated Output Policy

There are two common policies.

Generated-on-demand is best for examples, learning projects, and repositories
where every developer has LangForge available through source, binary, or
Docker:

```text
generated/
dist/
```

Commit neither directory. Regenerate before tests and demos.

Committed generated output is useful when downstream users should build a
program without installing LangForge:

```sh
make generate
git diff --exit-code
```

In that model, CI should fail when generated output is stale. Do not edit
generated files by hand; edit the `.lf` spec or handwritten reducer instead.

## More Complex Scenarios

Monorepo with many DSLs:

- place grammar sources under `grammars/<language>/<language>.lf`;
- generate into `internal/generated/<language>parser`;
- keep one testdata folder per DSL;
- add one aggregate `make generate` target and individual targets for focused
  work.

CLI with embedded snippets:

- start with one full-file grammar;
- add additional parser roots later when LangForge implements multiple roots;
- until then, model snippets as separate small `.lf` specs if they really need
  separate entry points.

CI with Docker but no installed LangForge:

- build or pull the image first;
- run validation with a read-only workspace mount;
- run generation with a writable workspace mount;
- run host-language tests after generation;
- if generated files are committed, run `git diff --exit-code` after
  generation.

Large semantic layer:

- keep action labels stable and short;
- dispatch with generated `SemanticAction*` constants and `ReducerMap`;
- put complex behavior in normal packages with normal unit tests;
- use inline actions only when compiler diagnostics pointing directly into the
  grammar are more valuable than backend portability.

Debugging generated code:

- use `inspect --format text` for a readable state summary;
- use `inspect --format json` when comparing table shape over time;
- read generated source comments to find which grammar rule produced a table
  entry;
- choose Docker working directories deliberately so generated source-reference
  paths match the level of detail you want.

# LangForge Example Projects

Use this reference when adding, running, or troubleshooting example projects.

## Standard Layout

```text
examples/<name>/
  Makefile
  README.md
  <name>.lf
  sample/input files
  cmd/<name>-demo/   # Go examples
  generated/   # generated, ignored
  Generated/   # generated, ignored for C#
  dist/        # binaries, logs, rendered output, ignored
```

Generated-dependent Go files should start with:

```go
//go:build langforge_generated
```

Command packages should also have a non-generated placeholder `main` guarded by
`//go:build !langforge_generated` so `go build ./...` works before generation.

## Makefile Contract

Example Makefiles should expose:

- `validate`: `$(LANG_FORGE) validate --spec <name>.lf`
- `generate`: validate and write target output under `generated` or `Generated`
- `build`: generate and build the demo
- `run`: build and run sample input, writing report/output under `dist`
- `test`: generate and run target-appropriate assertions
- `clean`: remove `generated` and `dist`

Default variables:

```make
GO ?= /usr/local/go/bin/go
LANG_FORGE ?= $(GO) run ../../../cmd/lang-forge
GENERATED_DIR := generated
DIST_DIR := dist
```

## Existing Examples

- `examples/go/{calc,datakeeper,draw,vehicle-report}`: full Go generated
  examples. Go builds use the `langforge_generated` tag.
- `examples/csharp/{calc,datakeeper,draw,vehicle-report}`: generated C# output
  under `Generated/` and handwritten C# reducers.
- `examples/c/{calc,datakeeper,draw,vehicle-report}`: generated C output with
  native compiler skip behavior through `CC`.
- `examples/cpp/{calc,datakeeper,draw,vehicle-report}`: generated C++17 output
  with native compiler skip behavior through `CXX`. The C++ DRAW example writes
  `dist/sample-cpp.png`.
- `examples/parser-algorithms`: source-only parser-table fixtures.

The root `Makefile` should list runnable example families in `examples-run`,
`examples-test`, and `examples-clean`. When adding an example, update both the
local files and the root targets.

## Artifact Policy

- Keep `examples/**/generated/`, `examples/**/Generated/`, and
  `examples/**/dist/` ignored.
- Do not commit regenerated recognizers unless the task explicitly calls for a
  golden fixture or bootstrapping artifact.
- Before final status, prefer `make -C examples/<name> clean` and rerun
  source-only root checks.
- Mention useful outputs in the final answer, such as `dist/*.log` or rendered
  PNG paths, even when those outputs are ignored.

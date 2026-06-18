# LangForge Example Projects

Use this reference when adding, running, or troubleshooting example projects.

## Standard Layout

```text
examples/<name>/
  Makefile
  README.md
  <name>.lf
  sample/input files
  cmd/<name>-demo/
  generated/   # generated, ignored
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
- `generate`: validate and write Go output under `generated`
- `build`: generate and build the demo with `-tags langforge_generated`
- `run`: build and run sample input, writing report/output under `dist`
- `test`: generate and run `go test -tags langforge_generated -count=1 ./...`
- `clean`: remove `generated` and `dist`

Default variables:

```make
GO ?= /usr/local/go/bin/go
LANG_FORGE ?= $(GO) run ../../cmd/lang-forge
GENERATED_DIR := generated
DIST_DIR := dist
TAGS := langforge_generated
```

## Existing Examples

- `examples/calc`: scanner/parser and reducer smoke demo for arithmetic
  expressions.
- `examples/datakeeper`: script compiler demo lowering to stack-machine code
  and mock execution logs.
- `examples/draw`: reducer-backed DRAW-language renderer demo producing a PNG
  and log.

Do not assume the root `Makefile` lists every example. Discover examples with
`find examples -maxdepth 2 -name Makefile -print` and inspect local targets.

## Artifact Policy

- Keep `examples/**/generated/` and `examples/**/dist/` ignored.
- Do not commit regenerated recognizers unless the task explicitly calls for a
  golden fixture or bootstrapping artifact.
- Before final status, prefer `make -C examples/<name> clean` and rerun
  source-only root checks.
- Mention useful outputs in the final answer, such as `dist/*.log` or rendered
  PNG paths, even when those outputs are ignored.

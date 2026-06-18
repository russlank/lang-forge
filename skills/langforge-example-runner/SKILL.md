---
name: langforge-example-runner
description: Generate, build, test, clean, or troubleshoot LangForge examples and generated parser/scanner output. Use when working under `examples/`, adding runnable demos, converting specs into full projects, running `make -C examples/...`, using `LANG_FORGE=../../dist/lang-forge`, managing `generated/` or `dist/` artifacts, or proving a `.lf` grammar with a runnable sample.
---

# LangForge Example Runner

## Overview

Run LangForge examples as reproducible source projects: spec in, generated Go
scanner/parser out, reducer or inline semantics wired, demo binary executed
against sample input, and generated output kept out of source control.

## Workflow

1. Read the example-local `Makefile`, `README.md`, `.lf` spec, and sample input.
2. Load `references/example-projects.md` before adding a new example, changing
   generated-output policy, or debugging a generated-dependent package.
3. Prefer the example-local targets over hand-written command sequences:

```sh
make -C examples/<name> validate
make -C examples/<name> generate
make -C examples/<name> build
make -C examples/<name> run
make -C examples/<name> test
make -C examples/<name> clean
```

4. Use source-run mode while developing:

```sh
make -C examples/<name> run
```

5. Verify standalone utility mode after `make build` when docs or release-like
   behavior matter:

```sh
make build
make -C examples/<name> LANG_FORGE=../../dist/lang-forge run
```

6. Keep generated imports behind `//go:build langforge_generated` when a clean
   checkout does not contain generated code.
7. Add a non-generated placeholder `main` for command packages so broad
   `go build ./...` stays healthy.
8. Clean generated and binary output before final status unless the task
   explicitly asks to inspect those files.

## Expected Evidence

For an example change, run the strongest applicable set:

```sh
make -C examples/<name> run
make -C examples/<name> test
make build
make -C examples/<name> LANG_FORGE=../../dist/lang-forge run
make -C examples/<name> clean
/usr/local/go/bin/go test -count=1 ./...
/usr/local/go/bin/go build ./...
```

Report the generated demo outputs, such as logs, PNGs, or token/parse reports,
without committing ignored artifacts.

---
name: langforge-example-runner
description: Generate, build, test, clean, or troubleshoot LangForge examples, templates, and generated parser/scanner output. Use when working under `examples/`, adding runnable demos, converting specs into full projects, running `make -C examples/...`, using `LANG_FORGE=../../../dist/lang-forge`, managing `generated/`, `Generated/`, `dist/`, `bin/`, or `obj/` artifacts, or proving a `.lf` grammar with a runnable sample.
---

# LangForge Example Runner

## Overview

Run LangForge examples as reproducible source projects: spec in, generated
scanner/parser output for the selected target, handwritten semantics wired,
demo binary executed against sample input, and generated output kept out of
source control. Current runnable families cover Go, C#, C, and C++; templates
under `examples/templates/{go,csharp,c,cpp}/mini-compiler` are the seed shape
for future project-bootstrap generation.

## Workflow

1. Read the example-local `Makefile`, `README.md`, `.lf` spec, and sample input.
2. Load `references/example-projects.md` before adding a new example, changing
   generated-output policy, or debugging a generated-dependent package.
3. Prefer the example-local targets over hand-written command sequences. Paths
   are language/scenario based, for example `examples/go/calc`,
   `examples/csharp/draw`, `examples/c/datakeeper`, or
   `examples/cpp/vehicle-report`:

```sh
make -C examples/<target>/<name> validate
make -C examples/<target>/<name> generate
make -C examples/<target>/<name> build
make -C examples/<target>/<name> run
make -C examples/<target>/<name> test
make -C examples/<target>/<name> clean
```

4. Use source-run mode while developing:

```sh
make -C examples/<target>/<name> run
```

5. Verify standalone utility mode after `make build` when docs or release-like
   behavior matter:

```sh
make build
make -C examples/<target>/<name> LANG_FORGE=../../../dist/lang-forge run
```

6. For Go examples, keep generated imports behind
   `//go:build langforge_generated` when a clean checkout does not contain
   generated code.
7. For Go command packages, add a non-generated placeholder `main` so broad
   `go build ./...` stays healthy.
8. Native C and C++ Makefiles validate/generate even when no compiler is
   installed; compile/run steps should print a skip message through `CC` or
   `CXX`.
9. For examples that use named RHS labels and semantic type declarations,
   verify the handwritten reducer boundary: Go and C# should prefer generated
   typed contexts; C and C++ should keep checked casts in small helper
   functions until typed-context parity lands.
10. Clean generated and binary output before final status unless the task
   explicitly asks to inspect those files.

## Expected Evidence

For an example change, run the strongest applicable set:

```sh
make -C examples/<target>/<name> run
make -C examples/<target>/<name> test
make build
make -C examples/<target>/<name> LANG_FORGE=../../../dist/lang-forge run
make -C examples/<target>/<name> clean
make examples-parity
make examples-testdata
make examples-templates
make examples-cleanliness
/usr/local/go/bin/go test -count=1 ./...
/usr/local/go/bin/go build ./...
```

Report the generated demo outputs, such as logs, PNGs, or token/parse reports,
without committing ignored artifacts.

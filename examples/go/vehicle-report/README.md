# Vehicle Report Demo

This example is inspired by a small Flex/Bison-style compiler homework that
parsed a `car = { ... }` description and printed XML-like output.

The LangForge version keeps the useful learning shape but makes it a clean
generated-on-demand project:

```text
vehicle.lf -> lang-forge generate -> generated parser reducer -> AST -> report
```

## Generated vs Handwritten Code

Only `generated` is produced by LangForge. The rest of this directory is
handwritten example code:

| Path | Role |
|---|---|
| `vehicle.lf` | Source grammar for scanner and parser generation |
| `generated/` | Recreated scanner/parser package, ignored by Git |
| `parser.go` | Handwritten adapter that calls `ParseWithReducer` and builds the AST |
| `ast.go` | Handwritten vehicle, feature, and repair model |
| `report.go` | Handwritten report/XML-like rendering |
| `cmd/vehicle-report-demo` | Handwritten command-line demo |

Action blocks in `vehicle.lf`, such as `{go: feature}` or
`{go: repair}`, are reducer labels. LangForge exposes generated action IDs and
rule values; the adapter maps those reductions into ordinary Go structs.

Files that import `generated` use the Go build tag
`//go:build langforge_generated`. The Makefile generates the package first and
then runs Go with `-tags langforge_generated`.

## Run The Demo

From this directory:

```sh
make run
```

The command validates `vehicle.lf`, generates the scanner/parser under
`generated`, builds `dist/vehicle-report-demo`, reads
[sample.vehicle](sample.vehicle), prints a report, and writes the same output
to `dist/vehicle-report-demo.log`.

Use a standalone LangForge binary like this:

```sh
make LANG_FORGE=../../../dist/lang-forge run
```

Run the generated-code tests:

```sh
make test
```

Remove generated and binary output:

```sh
make clean
```

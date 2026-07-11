# DataKeeper Script Compiler Demo

This example reconstructs the small DataKeeper scripting language from the
external Irony-based DataKeeperScripting compiler sources reviewed during
project development.

It is a LangForge usage example, not just a hand-written parser. The language
syntax lives in:

- [datakeeper.lf](datakeeper.lf)

The generated scanner/parser package is created on demand in:

- `generated`

The generated directory is ignored by Git. The semantic layer uses generated
typed reducer contexts to build the script model, then lowers it to a small
stack machine.
It does not touch a real database or DataKeeper service. The interesting path
is end to end:

```text
datakeeper.lf -> lang-forge generate -> script text -> generated parser reducer -> AST -> stack-machine code -> mock VM execution report
```

## Generated vs Handwritten Code

Only `generated` is produced by LangForge. The rest of this directory is
handwritten example code:

| Path | Role |
|---|---|
| `datakeeper.lf` | Source grammar for scanner and parser generation |
| `generated/` | Recreated scanner/parser package, ignored by Git |
| `model/` | Cycle-free AST model shared by generated typed contexts and handwritten code |
| `parser.go` | Handwritten adapter that calls `ParseWithReducerFromLexemeSource` and builds the AST |
| `compiler.go` | Handwritten lowering from AST to stack-machine instructions |
| `vm.go` | Handwritten mock execution engine |
| `cmd/datakeeper-demo` | Handwritten command-line demo |

Action blocks in `datakeeper.lf`, such as `{go: assign}` or
`{go: value.string}`, are reducer labels. LangForge exposes the label through
generated action IDs such as `SemanticActionAssign` and through the readable
string `Reduction.Action`. The adapter uses a generated `ReducerMap` keyed by
those IDs to decide what script node or value to create.

The grammar also labels RHS values, for example `parent=Value` and
`jobsTag=Value`, and declares `%semantic go type` entries for each important
nonterminal. LangForge uses that contract to generate contexts such as
`RunObjectsJobReduction`, so reducer code can read `ctx.Parent`, `ctx.Name`,
and `ctx.JobsTag` instead of counting parser-stack positions.

The AST types live in `model/` because generated Go code is in a child package.
Both the generated parser and this public example package can depend on
`model/`, avoiding an import cycle.

Files that import `generated` use the Go build tag
`//go:build langforge_generated`. The Makefile generates the package first and
then runs Go with `-tags langforge_generated`. This keeps a clean checkout
usable even before generated files exist.

For the same concept in the small calculator example, read
[../../../doc/generated-code-and-semantics.md](../../../doc/generated-code-and-semantics.md).

## Language Shape

The original Irony grammar supports:

```text
parameters Name, OtherName;

begin
  variable = value;
  replace(variable, oldValue, newValue);
  sqlrun(instanceGuid, sqlScript);
  addobject(parentGuid, objectXml);
  removeobject(parentGuid, objectName);
  runobjectsjob(parentGuid, objectName, jobsTag);
end
```

Values are:

- `#{...#}` strings, matching the original multi-line Irony string literal;
- quoted strings, added here for friendlier examples;
- integers, compiled as string values to match the old `IntConstNode`;
- references to variables or parameters.

Line comments `//...` and block comments `/*...*/` are supported.

## Stack Machine Mapping

The instruction order mirrors the old C# AST translation:

| Script form | Stack code shape |
|---|---|
| `x = value` | `PUSH_REF x`, compile `value`, `ASSIGN` |
| `replace(x, old, new)` | `PUSH_REF x`, compile `old`, compile `new`, `REPLACE_SUBSTR` |
| `sqlrun(a, b)` | compile `a`, compile `b`, `RUN_SQL` |
| `addobject(a, b)` | compile `a`, compile `b`, `ADD_OBJECT` |
| `removeobject(a, b)` | compile `a`, compile `b`, `REMOVE_OBJECT` |
| `runobjectsjob(a, b, c)` | compile `a`, compile `b`, compile `c`, `RUN_OBJECTS_JOB` |

Reference values compile as `PUSH_REF name`, `LOAD_REF`.

## Run The Demo

From this directory:

```sh
make run
```

This validates `datakeeper.lf`, generates the scanner/parser under `generated`,
builds `dist/datakeeper-demo`, runs [sample.dks](sample.dks), and writes the
same report to `dist/datakeeper-demo.log`.

The default Makefile runs LangForge from source. After a standalone LangForge
binary exists in the repository root, the same example can use it:

```sh
make LANG_FORGE=../../../dist/lang-forge run
```

Run the generated-code tests:

```sh
make test
```

The command fills omitted required parameters with deterministic demo values so
the sample is easy to run. The package tests cover strict missing-parameter
behavior through the VM API.

Remove generated and binary output:

```sh
make clean
```

# Calc LangForge Demo

This example is a small calculator grammar packaged as a runnable LangForge
project.

The source of truth is [calc.lf](calc.lf). The sample input is
[input.calc](input.calc). The grammar is kept in parity with the C#, C, and
C++ calculator specs, including decimal literals such as `4.5`.

From this directory:

```sh
make run
```

The target validates `calc.lf`, generates a Go scanner/parser under
`generated`, builds `dist/calc-demo`, tokenizes and parses `input.calc`,
evaluates the expression through rule-reduction actions, and writes the same
report to `dist/calc-demo.log`.

The grammar declares a reducer-mode semantic import for
`examples/go/calc/semantics`; the demo passes that handwritten package to the
generated parser with `ParseWithReducer`.

In reducer mode, action blocks such as `{go: add}` and `{go: subtract}` are
labels, not generated arithmetic code. LangForge stores the label on the
reduced rule and exposes both `Reduction.ActionID` and `Reduction.Action`.
The grammar also names RHS values (`left=Expr`, `right=Term`) and declares
`float64` results with `%semantic go type`. LangForge therefore generates
contexts such as `AddReduction` and adapters such as `TypedAdd`. The
handwritten reducer in [semantics/reducer.go](semantics/reducer.go) uses typed
fields (`ctx.Left`, `ctx.Right`) rather than positional indexes and casts.

The generated `ReducerMap` is keyed by constants such as
`SemanticActionAdd` and `SemanticActionSubtract`. Its coverage is validated
before `ParseWithReducer` starts parsing, so adding a grammar action without a
handler fails immediately. There is still no calculator-specific arithmetic
hard coded in LangForge.

Only the `generated` directory is produced by LangForge. The `semantics` and
`cmd` directories are ordinary source code that imports the generated package.

| Path | Role |
|---|---|
| `calc.lf` | Source grammar for scanner and parser generation |
| `generated/` | Recreated scanner/parser package, ignored by Git |
| `semantics/reducer.go` | Handwritten reducer that maps action labels to arithmetic |
| `cmd/calc-demo` | Handwritten command-line demo |

The generated directory also contains `langforge.actions.json`, a deterministic
record of action IDs, rules, RHS labels, and target semantic types.

The real demo files are guarded by the Go build tag
`//go:build langforge_generated` because they import `generated`. The Makefile
creates that package first, then builds with `-tags langforge_generated`.
Without that tag, Go compiles the small fallback file in `cmd/calc-demo`
instead, so a source-only checkout does not fail on a missing generated
package.

Read [../../../doc/generated-code-and-semantics.md](../../../doc/generated-code-and-semantics.md)
for the full explanation of reducer labels, inline mode, generated folders,
and Go build tags.

By default the Makefile runs LangForge from source. After a standalone
LangForge binary exists in the repository root, the same project can use it:

```sh
make LANG_FORGE=../../../dist/lang-forge run
```

Generated and binary output is ignored by Git. Remove it with:

```sh
make clean
```

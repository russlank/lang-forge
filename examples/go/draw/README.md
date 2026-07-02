# DRAW PNG Renderer Demo

This example modernizes the old [UCDT](https://github.com/russlank/UCDT)
`SAMPLES/DRAW` idea as a runnable LangForge project.

The language syntax lives in [draw.lf](draw.lf). The generated scanner/parser
package is created on demand under `generated`, and generated reduction hooks
build an AST before the renderer turns it into a PNG image with Go's standard
`image/png` package.

```text
draw.lf -> lang-forge generate -> generated parser reducer -> AST -> interpreter -> PNG
```

## Generated vs Handwritten Code

Only `generated` is produced by LangForge. The rest of this directory is
handwritten example code:

| Path | Role |
|---|---|
| `draw.lf` | Source grammar for scanner and parser generation |
| `generated/` | Recreated scanner/parser package, ignored by Git |
| `model/` | Dependency-only AST types shared by generated contexts and handwritten code |
| `parser_adapter.go` | Handwritten adapter that calls `ParseWithReducerFromSource` and builds the AST |
| `render.go` | Handwritten renderer that turns the AST into pixels |
| `cmd/draw-demo` | Handwritten command-line demo |

Action blocks in `draw.lf`, such as `{go: canvas}` or
`{go: figureRef.inline}`, are reducer labels. LangForge does not know how to
draw a canvas or figure. It recognizes the grammar rule and exposes generated
action IDs such as `SemanticActionCanvas`.

The grammar now labels semantic RHS values, for example
`width=Expr`, `height=Expr`, and `target=FigureReference`, and declares every
nonterminal result type with `%semantic go type`. LangForge therefore
generates typed contexts such as `CanvasReduction`, `RepdrawReduction`, and
`PrimitiveCircleReduction`. The adapter uses fields such as `ctx.Width`,
`ctx.Target`, and `ctx.Radius`; it contains no positional semantic indexes or
value casts.

`model/` keeps the AST independent from both the generated parser and the
handwritten renderer. This cycle-free dependency shape is useful for real
compilers whose generated contexts need to mention application AST types.
The generated `ReducerMap` is coverage-validated before parsing, and tests
also require every action in `langforge.actions.json` to be fully typed.

Files that import `generated` use the Go build tag
`//go:build langforge_generated`. The Makefile generates the package first and
then runs Go with `-tags langforge_generated`. This keeps a clean checkout
usable even before generated files exist.

For the same concept in the small calculator example, read
[../../../doc/generated-code-and-semantics.md](../../../doc/generated-code-and-semantics.md).

The language keeps the spirit of the Pascal sample:

- variables and arithmetic expressions;
- `sin`, `cos`, `tan`, `ln`, `sqrt`, `sqr`, and `exp` math functions;
- reusable figure blocks;
- `draw` and `repdraw`;
- `point`, `line`, `box`, and `circle` primitives.

This version also adds image-oriented commands:

```text
canvas 960,640;
background #101820;
stroke #F2AA4C;
fill none;
width 2;
```

Run the demo from this directory:

```sh
make run
```

The command validates `draw.lf`, generates the scanner/parser under `generated`,
builds `dist/draw-demo`, renders [sample.draw](sample.draw) to
`dist/sample.png`, and writes a report to `dist/draw-demo.log`.

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

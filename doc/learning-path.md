# Learning Path

Document id: `lang-forge-learning-path-v1`

Status: `active`

Last updated: `2026-07-01`

Owner: `Project maintainers`

Scope: `A guided path for learning compiler tooling through LangForge`

LangForge is meant to be useful software and readable compiler-learning
material. You should be able to move from a tiny calculator grammar to real
generated parser tables, semantic reducer hooks, DSL examples, and migration
fixtures without needing to reverse-engineer the whole repository at once.

## What You Can Learn Here

LangForge currently demonstrates:

- how Lex/Yacc-style specifications are structured;
- how regular expressions become lexical automata;
- how overlapping character classes become deterministic scanner alphabets;
- how longest-match and rule-priority tokenization works;
- how grammar rules become LR parser states;
- why SLR, LALR(1), IELR(1), and canonical LR(1) differ;
- how generated parsers and semantic reducers fit into a larger compiler or DSL
  tool;
- how to keep generated code deterministic and testable;
- how legacy compiler tools can be migrated without copying old assumptions.

## Suggested Order

### 1. Run The Smallest Thing

Start with the calculator:

```sh
/usr/local/go/bin/go run ./cmd/lang-forge validate --spec examples/go/calc/calc.lf
/usr/local/go/bin/go run ./cmd/lang-forge inspect --spec examples/go/calc/calc.lf --format text
make -C examples/go/calc run
```

Then open:

- [examples/go/calc/calc.lf](../examples/go/calc/calc.lf)
- [doc/generated-code-and-semantics.md](generated-code-and-semantics.md)
- [doc/handwritten-integration-guide.md](handwritten-integration-guide.md)
- [doc/specification.md](specification.md)
- [doc/usage.md](usage.md)

Goal: understand the boundary between lexer rules, parser rules, generated Go
parser output, and reducer-backed expression evaluation. In particular, learn
that `{go: add}` is a reducer label, not built-in arithmetic code. Then use
the handwritten integration guide to see which reducer, facade, library, and
test files a LangForge user normally writes beside the `.lf` file.

If a term is unfamiliar, keep [Glossary](glossary.md) open nearby.

### 2. Learn The Compiler Pipeline

Read [Compiler Pipeline](compiler-pipeline.md). It explains how source text
moves through these stages:

```text
.lf or .l/.y
  -> spec model
  -> lexer DFA
  -> grammar model
  -> parser table
  -> generated code
  -> reducer hooks or handwritten semantic layer
```

Goal: connect repository packages such as `internal/spec`, `internal/lex`, and
`internal/parse` to the compiler concepts they implement.

### 3. Inspect The Tables

Generate machine-readable inspection output:

```sh
/usr/local/go/bin/go run ./cmd/lang-forge inspect \
  --spec examples/go/calc/calc.lf \
  --format json > /tmp/calc.inspect.json
```

Look for:

- lexer states and accepting rules;
- parser `actions` and `gotos`;
- normalized grammar `rules`;
- parser `algorithm`.

Goal: see generated automata as data, not magic.

### 4. Compare LR Algorithms

Run the parser-algorithm fixtures:

```sh
make -C examples/parser-algorithms test
```

Then read [Parser Algorithms](parser-algorithms.md).

Goal: understand why a grammar can be LR(1) and LALR(1) but still fail under
SLR.

### 5. Recover From Syntax Errors

Run the recovery fixture:

```sh
make -C examples/go/parser-recovery run
```

Then read [Parser Error Recovery](parser-error-recovery.md).

Goal: understand the reserved `error` symbol, synchronization terminals,
expected-token diagnostics, cascade suppression, and the parser's progress
guarantee.

### 6. Study A Real DSL Flow

Run the DataKeeper example:

```sh
make -C examples/go/datakeeper run
```

Open:

- [examples/go/datakeeper/datakeeper.lf](../examples/go/datakeeper/datakeeper.lf)
- [examples/go/datakeeper/compiler.go](../examples/go/datakeeper/compiler.go)
- [examples/go/datakeeper/vm.go](../examples/go/datakeeper/vm.go)

Goal: see a generated parser reducer build an AST, lower it to stack-machine
instructions, and run a mock execution.

### 7. Study A Visual DSL

Run the DRAW renderer:

```sh
make -C examples/go/draw run
```

Open:

- [examples/go/draw/draw.lf](../examples/go/draw/draw.lf)
- [examples/go/draw/parser_adapter.go](../examples/go/draw/parser_adapter.go)
- [examples/go/draw/render.go](../examples/go/draw/render.go)

Goal: see a generated parser reducer build a visual DSL AST that becomes an
interpreted model and visible output.

### 8. Study A Migration-Shaped DSL

Run the vehicle report example:

```sh
make -C examples/go/vehicle-report run
```

Open:

- [examples/go/vehicle-report/vehicle.lf](../examples/go/vehicle-report/vehicle.lf)
- [examples/go/vehicle-report/parser.go](../examples/go/vehicle-report/parser.go)
- [examples/go/vehicle-report/report.go](../examples/go/vehicle-report/report.go)

Goal: see how a Flex/Bison-style exercise language can become a modern
LangForge `.lf` spec with generated parser reductions, AST construction, and
report output.

### 9. Explore Legacy Inspiration

Read [UCDT Legacy Inspiration](ucdt-legacy-inspiration.md), then validate a split
fixture:

```sh
/usr/local/go/bin/go run ./cmd/lang-forge validate \
  --lex testdata/ucdt/draw/draw.l \
  --yacc testdata/ucdt/draw/draw.y
```

Goal: understand which old Lex/Yacc ideas inspired LangForge and how old
fixtures can be used without turning them into a compatibility promise.

## Concept Map

| Concept | Public doc | Code to read | Example |
|---|---|---|---|
| Vocabulary | [Glossary](glossary.md) | names across packages | all examples |
| Spec syntax | [Specification](specification.md) | `internal/spec` | `examples/go/calc/calc.lf` |
| Lexer automata | [Compiler Pipeline](compiler-pipeline.md) | `internal/lex` | calc lexer rules |
| Parser tables | [Parser Algorithms](parser-algorithms.md) | `internal/parse` | parser algorithm fixtures |
| Generated Go | [Usage](usage.md) | `internal/codegen/golang` | `examples/go/calc` |
| Semantic layer | [Examples](examples.md) | generated reducers and example-local packages | calc, DataKeeper, DRAW, and vehicle report |
| Legacy inspiration | [UCDT Legacy Inspiration](ucdt-legacy-inspiration.md) | `internal/spec` split parsing | `testdata/ucdt` |

## Exercises

Try these in order:

1. Add a modulo operator to `examples/go/calc/calc.lf`.
2. Add a keyword token before an identifier token in a small test grammar and
   observe longest-match plus rule-priority behavior.
3. Change the parser algorithm fixture from `%type lalr` to `%type slr` and
   inspect the conflict.
4. Add a harmless comment syntax to the DataKeeper lexer and send it to a
   hidden channel.
5. Add a new DRAW command that maps cleanly to one new AST node and renderer
   operation.
6. Add one vehicle feature field to `examples/go/vehicle-report/sample.vehicle`
   and verify the report changes.
7. Save `inspect --format json` output before and after a grammar change and
   compare state counts.

## Learning-Friendly Coding Practices

LangForge code should prefer:

- small packages with clear compiler-stage ownership;
- named concepts that match compiler literature where practical;
- deterministic output so examples are repeatable;
- focused tests that show the reason a behavior exists;
- public docs that explain how to use a feature before explaining internals;
- comments near non-obvious algorithms, not comment noise around simple code.

When adding a feature, try to leave three breadcrumbs:

1. A small example or fixture.
2. A test that protects the edge case.
3. A short doc note explaining when a user should care.

## Best Practices For Users

- Keep `.lf` specs as the source of truth.
- Validate early and often.
- Use LALR(1) by default.
- Use IELR(1) when LALR reports a false merge conflict.
- Use canonical LR(1) to diagnose deeper conflicts.
- Keep generated output in ignored `generated` directories unless you are
  intentionally creating golden fixtures.
- Use target-tagged reducer hooks for rule-local semantics, and keep larger
  domain behavior in ordinary target-language code. Use `%semantic <target>
  import` to record handwritten semantic dependencies, and reserve inline mode
  for target-specific generated reductions that truly need to call APIs
  directly.
- Use `inspect --format json` to reason about state counts, conflicts, and
  table shape.

## Quality And Performance Expectations

Learning material should not make the tool slow or sloppy. LangForge aims for:

- deterministic generation without timestamp churn;
- compact table-driven scanner and parser runtimes;
- reentrant generated APIs;
- validation that rejects non-progressing scanners and malformed grammars early;
- tests for edge cases before examples rely on them;
- generated code that reads like normal target-language code.

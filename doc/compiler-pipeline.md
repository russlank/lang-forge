# Compiler Pipeline

Document id: `lang-forge-compiler-pipeline-v1`
Status: `active`
Last updated: `2026-06-18`
Owner: `Project maintainers`
Scope: `Plain-language explanation of the LangForge compiler-tooling pipeline`

This guide explains what happens between a grammar file and generated code.
It is intentionally practical: each stage names the concept, the repository
package, the useful commands, and the quality checks that keep the stage
reliable.

For terminology, keep [Glossary](glossary.md) nearby.

## Big Picture

```text
source specification
  -> parsed spec model
  -> lexer regex ASTs
  -> NFA
  -> DFA
  -> minimized DFA
  -> normalized grammar
  -> LR parser table
  -> generated target code
  -> reducer hooks or handwritten semantic layer
```

LangForge separates the language-independent compiler algorithms from target
code generation. This keeps scanner and parser behavior consistent across the
current Go, C#, and C backends.

## Stage 1: Source Specification

Package:

- `internal/spec`

Inputs:

- combined `.lf` specs;
- legacy split `.l` and `.y` inputs.

The parser builds one shared model with:

- lexer definitions;
- lexer rules and actions;
- token declarations;
- parser rules;
- start symbol;
- parser algorithm selection.

Important validation:

- token names must be identifiers;
- rule names must be identifiers;
- token and nonterminal roles must not collide;
- legacy Pascal action blocks are not executed.

Useful commands:

```sh
/usr/local/go/bin/go run ./cmd/lang-forge validate --spec examples/go/calc/calc.lf
/usr/local/go/bin/go run ./cmd/lang-forge validate \
  --lex testdata/ucdt/calc/calc.l \
  --yacc testdata/ucdt/calc/calc.y
```

Learning note: the spec parser is deliberately small and direct. It is a good
entry point before reading the automata code.

## Stage 2: Regex AST

Package:

- `internal/lex`

Lexer rules start as regular-expression text:

```text
DIGIT = [0-9];
NUMBER = DIGIT+;

NUMBER => token(Number);
[1-32]+ => skip;
```

LangForge parses them into an expression tree with nodes such as:

- set;
- concatenation;
- alternation;
- star;
- plus;
- optional;
- named reference.

Important validation:

- named definitions must exist;
- recursive definitions are rejected;
- complete lexer rules must not match the empty string;
- ranges must stay inside the active Unicode scalar scanner domain.

Learning note: regex parsing is where concrete syntax starts turning into a
compiler-friendly tree.

## Stage 3: NFA Construction

Package:

- `internal/lex`

An NFA is a graph that may have many possible active states at once. It is easy
to build from regex structure:

```text
literal "a"     -> one transition on "a"
A B             -> connect A's end to B's start
A | B           -> branch into A or B
A*              -> loop around A and allow empty
A+              -> A followed by A*
A?              -> A or empty
```

LangForge builds one NFA per lexer rule and combines them under a shared start
state. Accepting NFA states remember the lexer rule index so priority is
preserved.

Learning note: NFAs are flexible but not ideal for generated scanner code
because runtime matching would have to track sets of states.

## Stage 4: DFA Construction

Package:

- `internal/lex`

A DFA has exactly one active state at a time. LangForge builds it with subset
construction:

```text
startDFA = epsilonClosure({nfaStart})

for each unprocessed DFA state:
    for each alphabet class:
        nextNFASet = move(currentNFASet, alphabetClass)
        nextDFA = epsilonClosure(nextNFASet)
        add transition currentDFA --alphabetClass--> nextDFA
```

Before this, overlapping character ranges are partitioned into deterministic
alphabet classes. For example, `[A-Z]` and `[A-F]` must be split so every input
symbol belongs to one stable transition class. In UTF-8 mode, that input symbol
is a Unicode scalar value while token spans still preserve byte offsets.

Runtime matching uses:

1. longest match;
2. then earliest lexer rule.

Learning note: this is the heart of Lex-style scanning. It explains why rule
order matters for equal-length matches, but longer matches win first.

## Stage 5: DFA Minimization

Package:

- `internal/lex`

Many DFA states can be equivalent. Minimization merges states that have:

- the same accepting behavior;
- the same transitions to equivalent groups.

LangForge preserves accepting-rule identity while minimizing so token priority
does not change.

Quality goal: generated scanners should be compact without becoming hard to
reason about.

## Stage 6: Grammar Normalization

Package:

- `internal/parse`

Parser rules become numbered productions:

```text
Expr : Expr Plus Term
     | Term
     ;
```

becomes:

```text
1) Expr -> Expr Plus Term
2) Expr -> Term
```

LangForge also adds an internal start production:

```text
0) S' -> S
```

The grammar package computes:

- nullable nonterminals;
- FIRST sets;
- FOLLOW sets.

Learning note: these sets are the bridge between grammar structure and parser
table construction.

## Stage 7: LR Parser Tables

Package:

- `internal/parse`
- `internal/parseralgo`

LangForge supports:

- SLR;
- LALR(1);
- IELR(1);
- canonical LR(1).

Each parser state contains items, which are productions with a dot position:

```text
Expr -> Expr Plus . Term
```

The table has:

- `actions[state][terminal]` for shift, reduce, and accept;
- `gotos[state][nonterminal]` for moving after reductions;
- conflicts when two actions compete for the same state and lookahead.

Read [Parser Algorithms](parser-algorithms.md) for detailed pseudo-code, the
LR(1)-not-SLR example, and the mysterious LALR conflict example used to
exercise IELR.

Quality goal: conflicts are never hidden. `validate` fails on conflicts, while
`inspect` exposes the table and conflict details.

## Stage 8: Code Generation

Package:

- `internal/codegen/golang`

Current Go generation writes:

- `tokens.go`;
- `scanner.go`;
- `parser.go`;
- `langforge.tables.json`;
- `langforge.manifest.json`.

The generated parser is a table-driven recognizer with an optional semantic
reducer. Calling `Parse` validates the token stream. Calling `ParseValue` or
`ParseWithReducer` also carries a semantic value stack and dispatches
target-tagged rule actions to user code.

Current C# generation writes:

- `Tokens.g.cs`;
- `Scanner.g.cs`;
- `Parser.g.cs`;
- `langforge.tables.json`;
- `langforge.manifest.json`.

The generated C# parser mirrors the reducer-first Go model with
`SemanticAction` enum values, `Reduction` records, `IReducer`, and
`ReducerMap`.

Quality goals:

- generated Go is `gofmt` clean;
- generated APIs have doc comments;
- output is deterministic;
- manifests make generated output auditable.

## Stage 9: Semantic Layer

Examples:

- `examples/go/calc`
- `examples/go/datakeeper`
- `examples/go/draw`
- `examples/go/vehicle-report`

Semantic work can use generated reduction hooks or a handwritten layer around
the generated recognizer:

```text
generated scanner/parser
  -> reducer callback for target-tagged actions
  -> AST, command model, compiler, interpreter, renderer, or VM
```

For Go output, shifted terminals arrive as generated `Lexeme` values and
reduced nonterminals arrive as the values previously returned by the reducer.
The reducer receives:

```text
rule id, left-hand side, right-hand side symbols, action ID, action text, RHS values
```

Calc, DataKeeper, DRAW, and vehicle report all use this reducer-backed path.

If you are new to this pattern, read
[Generated Code And Semantics](generated-code-and-semantics.md) before digging
into generated parser tables. It explains why `{go: add}` is a label, where
the real behavior lives, how generated action IDs and reducer maps work, and
why example files use Go build tags.

Specs can declare target-specific semantic dependencies:

```text
%semantic go import calcsem "example.com/project/calcsem"
```

In reducer mode, includes are recorded in generated metadata so the
handwritten reducer layer and tooling can see the intended dependencies. In Go
inline mode, includes become imports in `parser.go`, and target action blocks
are emitted into a generated `reduceInline` switch. Inline mode is powerful but
target-specific; reducer mode is the portable default.

This keeps generated syntax recognition separate from domain behavior while
still making rule reductions observable.

Learning note: this separation is a good compiler engineering practice. It
makes syntax changes, semantic checks, and runtime behavior easier to test
independently.

## How To Read The Code

Read in this order:

1. `examples/go/calc/calc.lf`
2. `internal/spec`
3. `internal/lex/regex.go`
4. `internal/lex/automata.go`
5. `internal/parse/grammar.go`
6. `internal/parse/slr.go`
7. `internal/parse/lr1.go`
8. `internal/codegen/golang`
9. `examples/go/datakeeper`, `examples/go/draw`, or `examples/go/vehicle-report`

At each stage, run one focused test package:

```sh
/usr/local/go/bin/go test -count=1 ./internal/spec
/usr/local/go/bin/go test -count=1 ./internal/lex
/usr/local/go/bin/go test -count=1 ./internal/parse
/usr/local/go/bin/go test -count=1 ./internal/codegen/golang
```

## Best-Practice Checklist

For grammar authors:

- prefer combined `.lf` files for new work;
- keep token and nonterminal names distinct;
- place keyword rules before generic identifier rules;
- keep lexer rules non-nullable;
- encode precedence through grammar layers for now;
- use LALR by default;
- switch to IELR(1) when LALR reports a false merge conflict;
- switch to canonical LR(1) for deep conflict diagnosis;
- inspect JSON when table behavior is surprising.

For contributors:

- keep compiler-stage ownership clear;
- add tests for edge cases before relying on them in examples;
- preserve deterministic generation;
- avoid target-specific assumptions in core packages;
- document public behavior in public docs;
- keep private planning notes out of public-facing links.

For generated output:

- do not write timestamps into deterministic artifacts;
- keep runtimes reentrant;
- prefer compact tables and simple lookup paths;
- expose enough metadata for debugging;
- keep generated code boring, stable, and easy to delete/regenerate.

# LangForge Specification Format

Document id: `lang-forge-specification-v1`

Status: `active`

Last updated: `2026-07-01`

Owner: `Project maintainers`

Scope: `Current combined .lf format and supported split-file import syntax`

## Combined `.lf`

A combined spec has directives, a lexer section, and a parser section:

```text
%target go
%package calc
%start S
%token Number Plus

%% lexer
DIGIT = [0-9];
NUMBER = DIGIT+;

NUMBER => token(Number);
"+"    => token(Plus);
[1-32]+ => skip;

%% parser
S : Expr ;
Expr : Expr Plus Term | Term ;
```

For a guided tour of how this file turns into scanner and parser tables, see
[Compiler Pipeline](compiler-pipeline.md).

## Directives

| Directive | Meaning |
|---|---|
| `%target go` / `%target csharp` / `%target c` / `%target cpp` | Preferred generation target. The CLI still requires `--target` today. |
| `%package name` | Package/namespace/prefix hint for generated code. Go expects a package identifier; C# expects a namespace; C derives a public symbol prefix; C++ expects a namespace. |
| `%scanner utf8` | Selects checked UTF-8 scanner input. This is the default. |
| `%scanner encoding=utf8 invalid=error newline=lf` | Structured scanner settings for future extension. |
| `%semantic go mode reducer` | Treat Go parser actions as reducer labels. This is the default. |
| `%semantic go mode inline` | Treat Go parser actions as Go statements emitted into generated reduction code. |
| `%semantic go import alias "module/path"` | Declare a target-specific handwritten semantic dependency. Go generation imports it for inline actions or when a typed reducer context references the alias; otherwise reducer mode records it as metadata. |
| `%semantic go type Expr float64` | Declare the target-language semantic result type of a nonterminal. Replace `go` and `float64` with the target and its native type. Terminals remain generated `Lexeme` values. |
| `%start Symbol` | Grammar start symbol. |
| `%token A B C` | Declares parser terminals. |
| `%alias Ident "identifier"` | Gives a terminal a human-readable name in parser diagnostics. |
| `%group operator Plus Minus Star Slash` | Groups two or more simultaneously expected terminals under one diagnostic concept. |
| `%hide-expected Comma Semi` | Omits low-value terminals from expected-token reports without changing parser behavior. |
| `%type slr` | Build SLR tables. Useful for small/simple grammars and diagnostics. |
| `%type lalr` | Build LALR(1) tables. This is the default when `%type` is omitted. |
| `%type ielr` | Build correctness-first IELR(1) tables with merge/split reporting. Useful when LALR introduces a false merge conflict that canonical LR(1) avoids. |
| `%type canonical` | Build canonical LR(1) tables with full lookahead state separation. |

See [Parser Algorithms](parser-algorithms.md) for how LR(0), SLR, LALR(1),
IELR(1), and canonical LR(1) are implemented and when to choose each mode.

## Lexer Syntax

Definitions:

```text
NAME = regex;
```

Rules:

```text
regex => token(TokenName);
regex => skip;
regex => channel(ChannelName);
```

Supported regex features:

- quoted literals: `"+"`, `"while"`;
- character classes: `[A-Z]`, `[0-9]`, `[1-32]`;
- Unicode scalar escapes: `\uXXXX`, `\UXXXXXXXX`, and `\u{...}`;
- selected Unicode properties: `\p{L}`, `\P{Number}`, scripts, and Go
  `unicode` property names;
- grouping: `( ... )`;
- alternation: `a|b`;
- implicit concatenation;
- repetition: `*`, `+`, `?`;
- named definition references.

The scanner defaults to checked UTF-8. Regexes operate on Unicode scalar
values; surrogate code points and values above `U+10FFFF` are rejected. See
[Scanner Encoding Architecture](encoding.md).

Lexer rules must consume at least one scanner symbol. Patterns such as `""`,
`"a"*`, or `"b"?` are rejected because a generated scanner cannot make
progress after an empty match.

## Parser Syntax

Productions use Yacc-like rules:

```text
Expr : Expr Plus Term
     | Term
     ;
```

Empty productions may be written as an empty alternative, `e`, `ε`, or
`%empty`.

RHS values can be given stable names:

```text
Expr : left=Expr Plus right=Term {go: add}
```

Labels must be identifiers and unique inside one alternative. They are
preserved in table JSON and `langforge.actions.json`. Generated Go reductions
also expose `ctx.ValueFor("left")`; when semantic result types are complete,
LangForge generates a typed context with fields such as `Left` and `Right`.

Declare nonterminal result types for each generated target:

```text
%semantic go type Expr float64
%semantic csharp type Expr double
%semantic c type Expr double
%semantic cpp type Expr double
```

The declaration describes the value returned when that nonterminal reduces.
Tokens are deliberately not declared this way: shifted terminals are generated
scanner `Lexeme` values preserving token text and source positions.

### Error Recovery

The reserved `error` symbol adds grammar-directed synchronization:

```text
Statement : Ident Assign Number Semi
          | error Semi
          ;
```

`error` is parser-only and must be followed by a synchronization terminal in
the same alternative. It cannot be declared as a token, emitted by the
scanner, used as a rule name, repeated in one alternative, or given an RHS
label. Generated parsers collect source-rich diagnostics and discard tokens
only until the synchronization production can continue.

See [Parser Error Recovery](parser-error-recovery.md) for the runtime
algorithm, reporting directives, target APIs, progress guarantee, and runnable
example.

Target-tagged semantic action blocks attach a reduction hook to a production
alternative:

```text
Expr : Expr Plus Term {go:
    add
}
```

The target tag can also be `csharp`, `c`, or `cpp` for generated reducer hooks:

```text
Expr : Expr Plus Term {csharp: add}
Expr : Expr Plus Term {c: add}
Expr : Expr Plus Term {cpp: add}
```

For a beginner-friendly explanation of how this hook becomes runtime behavior,
read [Generated Code And Semantics](generated-code-and-semantics.md).

For generated Go output, the `go` action text is passed to a user-supplied
`Reducer` when that production reduces. Shifted terminal values are generated
`Lexeme` objects; reduced nonterminals are whatever values the reducer returned
for earlier reductions.

```go
reducers := parser.ReducerMap{
	parser.SemanticActionAdd: parser.TypedAdd(
		func(ctx parser.AddReduction) (float64, error) {
			return ctx.Left + ctx.Right, nil
		},
	),
}
value, err := parser.ParseWithReducerFromLexemeSource(parser.NewScanner(source), reducers)
```

Typed adapters are generated by the Go, C#, C, and C++ backends when every rule
using an action label has the same named field types and return type. Boxed
`Reduction.Values` remains available for gradual migration and unusual cases.
C emits typed argument contexts in `parser_typed.h` while preserving `void *`
returns for explicit ownership. C++ emits `parser_typed.hpp` adapters that
return native values and box them back into `std::any`.

When a C++ nonterminal is declared as `std::nullptr_t`, reducer code should
return `nullptr` for that action. In a boxed-only reducer path, `return {};`
creates an empty `std::any` and can be tolerated only when nobody reads that
value. In a boxed reducer adapted through `typed_reducer_map_from_boxed`, the
adapter validates with `std::any_cast<std::nullptr_t>`, so an empty `std::any`
is rejected and `return nullptr;` is required. Direct typed reducers should
also return `nullptr` because their declared result type is `std::nullptr_t`.

`ReducerMap.ValidateCoverage` checks that every grammar action has one handler
and that the map contains no unknown generated action ID. The standard Go, C#,
and C++ map-based parse paths perform this check before parsing. C typed
reducers validate required handler pointers before parsing.

Use short action names such as `add`, `program.withParameters`, or
`value.string` in reducer mode. Put substantial behavior in ordinary
target-language code and pass it with `ParseWithReducer`. These action names
are not built-in commands. They are labels that the generated parser copies
into `Reduction.Action` for diagnostics and into generated action IDs such as
`Reduction.ActionID` / `SemanticActionAdd` for dispatch.

Action labels should be portable grammar names, not target-language
identifiers. Prefer the same label text across Go, C#, C, and C++ specs, for
example `runObjectsJob` or `feature.tail.more`. Backends preserve that original
text in `Reduction.Action`, lookup APIs, diagnostics, and
`langforge.actions.json`, then derive target-safe identifiers from it:

| Label | C identifier | C++ identifier |
|---|---|---|
| `runObjectsJob` | `PREFIX_ACTION_RUN_OBJECTS_JOB` | `SemanticAction::RunObjectsJob` |
| `feature.tail.more` | `PREFIX_ACTION_FEATURE_TAIL_MORE` | `SemanticAction::FeatureTailMore` |

C generated typed handler names use snake case such as
`prefix_run_objects_job_handler`. C++ typed adapter functions use snake case
such as `typed_run_objects_job`, while enum values remain PascalCase. This
keeps the grammar target-neutral and the generated code idiomatic for each
language.

When a Go project needs generated reductions to call handwritten libraries
directly, opt into inline mode:

```text
%semantic go mode inline
%semantic go import sem "example.com/project/semantics"

Expr : Expr Plus Term {go:
    return sem.Add(ctx.Values[0], ctx.Values[2])
}
```

Inline Go action text is emitted as statements inside a generated
`reduceInline(ctx Reduction) (Value, error)` switch. It can refer to `ctx`,
`Value`, `Lexeme`, and any packages declared with `%semantic go import`.
Reducer mode remains the default because it avoids target import cycles and
keeps generated files independent from handwritten semantics.

Generated Go, C#, C, and C++ output includes source comments for parser table
entries that come from grammar rules. The generated comments include the
normalized grammar alternative and, when available, the original source
filename, line, and column. Generated Go scanner rule tables also carry source
comments. Inline Go action text additionally uses Go `//line` directives so
compiler diagnostics can point back to the grammar source. Reducer-mode labels
remain metadata, so they use source comments and structured spans rather than
compiler line directives.

Grammar symbols must have one role. A name cannot be both a `%token` and a
grammar rule name, because parser tables need terminals and nonterminals to map
to distinct action/goto entries.

For Go generation, `%package` must be a valid non-keyword Go package identifier
when it is set explicitly. For C# generation, `%package` is a dotted namespace.
For C++ generation, `%package` is a namespace written with `::` or dotted
separators, such as `LangForge::Examples::Calc::Generated`. For C generation,
`%package` becomes the public symbol prefix.

## Authoring Style For Readable Grammars

Prefer grammar files that teach their own structure:

- keep equivalent grammars in different target-language examples visually
  parallel, so parity reviews can focus on real target differences;
- group directives at the top in this order when practical: `%target`,
  `%package`, `%semantic ... mode`, `%semantic ... import`, `%semantic ... type`,
  parser options, `%start`, then `%token`;
- declare all visible tokens near the top;
- name tokens after their grammar role, not only their spelling;
- keep lexer definitions for repeated character classes;
- place more specific lexer rules before more general ones;
- align lexer action arrows within a local rule block when the padding stays
  readable;
- keep parser nonterminals layered by precedence or language concept;
- write one parser alternative per line for nontrivial productions;
- indent reduction action labels under the alternative they belong to;
- keep named right-hand-side labels in the same positions across target
  variants;
- use `%empty` for intentional empty alternatives;
- keep semantic action labels small and target-specific;
- put substantial semantic behavior in ordinary target-language reducer code.

For example, keep target variants shaped like this and let only the target
language, package, semantic types, imports, and action label prefixes differ:

```lf
%target go
%package calc
%semantic go mode reducer
%semantic go type Expr float64
%start Expr
%token Number Plus Minus

%% lexer
DIGIT = [0-9];
NUMBER = DIGIT+;

NUMBER  => token(Number);
"+"     => token(Plus);
"-"     => token(Minus);
[1-32]+ => skip;

%% parser
Expr : left=Expr Plus right=Term
         {go: add}
     | left=Expr Minus right=Term
         {go: subtract}
     | value=Term
         {go: pass}
     ;
```

Tiny wrapper productions can stay on one line when that makes the surrounding
grammar easier to scan:

```lf
S : value=Expr {go: start} ;
```

Readable grammar files are easier to debug, easier to port across backends, and
better learning material for the next person who opens the project.

## Legacy `.l` and `.y`

LangForge accepts split inputs:

```sh
lang-forge validate --lex lexer.l --yacc parser.y
```

The current migration support is intentionally narrow but useful:

- Lex files use `definitions %% rules %%`.
- Yacc files use `declarations %% rules %%`.
- Pascal/[UCDT](https://github.com/russlank/UCDT) `#{...#}` action blocks are
  stripped from grammar rules for table construction.
- Lex action blocks containing `YACC_Name` infer `token(Name)`.
- `LEX_Skip` or `AReturn := False` infer `skip`.
- Quoted `%%`, escaped punctuation, escaped colons, block-comment delimiters
  inside regex literals, and selected byte-oriented ranges are handled for
  curated source-only fixtures.

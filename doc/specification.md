# LangForge Specification Format

Document id: `lang-forge-specification-v1`
Status: `active`
Last updated: `2026-06-25`
Owner: `Project maintainers`
Scope: `Current combined .lf format and supported legacy migration syntax`

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
| `%type slr` | Build SLR tables. Useful for small/simple grammars and diagnostics. |
| `%type lalr` | Build LALR(1) tables. This is the default when `%type` is omitted. |
| `%type ielr` | Build conservative IELR(1) tables. Useful when LALR introduces a false merge conflict that canonical LR(1) avoids. |
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
value, err := parser.ParseWithReducer(tokens, parser.ReducerMap{
	parser.SemanticActionAdd: parser.TypedAdd(
		func(ctx parser.AddReduction) (float64, error) {
			return ctx.Left + ctx.Right, nil
		},
	),
})
```

Typed adapters are currently generated by the Go backend when every rule using
an action label has the same named field types and return type. Boxed
`Reduction.Values` remains available for gradual migration and unusual cases.
The C#, C, and C++ backends already emit the same labels and types in the
action manifest; idiomatic typed adapters for those targets are planned next.

`ReducerMap.ValidateCoverage` checks that every grammar action has one handler
and that the map contains no unknown generated action ID. The standard Go
`ParseWithReducer` path performs this check before parsing.

Use short action names such as `add`, `program.withParameters`, or
`value.string` in reducer mode. Put substantial behavior in ordinary
target-language code and pass it with `ParseWithReducer`. These action names
are not built-in commands. They are labels that the generated parser copies
into `Reduction.Action` for diagnostics and into generated action IDs such as
`Reduction.ActionID` / `SemanticActionAdd` for dispatch.

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

Generated Go output includes source comments for scanner and parser table
entries that come from grammar rules. Inline Go action text additionally uses
Go `//line` directives so compiler diagnostics can point back to the grammar
source. Reducer-mode labels remain metadata, so they use source comments and
structured spans rather than compiler line directives.

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

- declare all visible tokens near the top;
- name tokens after their grammar role, not only their spelling;
- keep lexer definitions for repeated character classes;
- place more specific lexer rules before more general ones;
- keep parser nonterminals layered by precedence or language concept;
- use `%empty` for intentional empty alternatives;
- keep action hooks small and target-tagged;
- put substantial semantic behavior in ordinary target-language reducer code.

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
  inside regex literals, and selected legacy byte ranges are handled for
  curated source-only fixtures.

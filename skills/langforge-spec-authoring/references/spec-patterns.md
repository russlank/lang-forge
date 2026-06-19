# LangForge Spec Patterns

Use this reference for non-trivial `.lf` authoring, legacy migration, and
validation debugging.

## Combined Spec Skeleton

```text
%target go
%package packagename
%semantic go mode reducer
%start StartSymbol
%token TokenA TokenB TokenC

%% lexer
IDENT = [A-Za-z_] [A-Za-z0-9_]*;
NUMBER = [0-9]+;

"literal" => token(TokenA);
IDENT     => token(TokenB);
[1-32]+   => skip;

%% parser
StartSymbol : Rule ;
Rule : TokenA TokenB
     | %empty
     ;
```

## Lexer Patterns

- Use definitions for repeated character classes and token families.
- Use quoted literals for punctuation and keywords.
- Put keywords before generic `IDENT` rules.
- Use `[1-32]+ => skip;` for ASCII whitespace when exact whitespace channels
  are not needed.
- Use `channel(Name)` only when hidden-channel tokens must be preserved for a
  caller; generated parsers expect visible grammar tokens.
- Current generated Go, C#, C, and C++ scanners use checked UTF-8 and Unicode
  scalar ranges. Keep non-UTF-8 encoding assumptions explicit.
- Avoid nullable expressions: `""`, `X*`, and `X?` as whole rules are invalid.

## Parser Patterns

- Encode precedence by grammar layering:
  `Expr -> Expr Plus Term | Term`, `Term -> Term Mul Factor | Factor`.
- Use `%empty` for intentional empty productions.
- Keep terminals declared with `%token`; every nonterminal should have a rule.
- Keep token and nonterminal names disjoint.
- Use `%type slr`, `%type lalr`, `%type ielr`, or `%type canonical` only when
  selecting a parser algorithm intentionally. LALR is the default.
- Treat conflicts as design work, not warnings to ignore. Use `inspect` to
  review states and lookaheads.
- Use target-tagged reducer labels such as `{go: add}`, `{csharp: add}`,
  `{c: add}`, or `{cpp: add}` for runnable examples. C++ reducers normally use
  generated `SemanticAction` values with `ReducerMap`.

## Legacy Split Inputs

Validate split inputs with:

```sh
/usr/local/go/bin/go run ./cmd/lang-forge validate --lex file.l --yacc file.y
```

Current split-input support:

- Lex files use `definitions %% rules %%`.
- Yacc files use `declarations %% rules %%`.
- [UCDT](https://github.com/russlank/UCDT)/Pascal `#{...#}` action blocks are
  stripped for table construction.
- Lex actions containing `YACC_Name` infer `token(Name)`.
- `LEX_Skip` or `AReturn := False` infer `skip`.
- Quoted `%%`, escaped punctuation, escaped colons, and block-comment
  delimiters inside regex literals are supported for curated source-only
  fixtures.

When converting legacy samples to `.lf`, keep the original fixture under
`testdata/ucdt` when useful, then create a modern combined spec under
`examples/<name>/<name>.lf` if it should become runnable.
Do not treat UCDT behavior as a compatibility contract.

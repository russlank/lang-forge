# UCDT Reference

Document id: `lang-forge-ucdt-reference-v1`

Status: `active`

Last updated: `2026-06-19`

Owner: `Project maintainers`

Scope: `How the Pascal UCDT project is used as a reference for LangForge`

LangForge acknowledges the Pascal
[UCDT](https://github.com/russlank/UCDT) project as an important reference,
especially its practical Lex/Yacc implementation and sample languages.

New LangForge work should use the combined `.lf` format, target-neutral
scanner/parser models, and generated output designed for Go, C#, C, C++, and
future backends. UCDT-derived files under `testdata/ucdt` are source-only
regression and learning fixtures.

## Reference Areas

| UCDT idea | LangForge direction |
|---|---|
| Separate Lex and Yacc tools | A single `lang-forge` utility with lexer, parser, inspection, and generation commands. |
| Split `.l` and `.y` inputs | Optional import convenience; combined `.lf` is the primary format. |
| Regex to NFA/DFA pipeline | Target-neutral lexer engine with deterministic generation and inspection output. |
| Longest-match plus rule-priority scanning | Preserved as the default scanner resolution model. |
| Parser table construction | LR(0) internals plus SLR, LALR(1), IELR(1), and canonical LR(1) modes. |
| Pascal skeleton output | Replaced with modern target backends and generated manifests. |
| Pascal semantic actions | Replaced by target-specific reducer hooks by default, with explicit Go inline action mode for advanced target-specific snippets. |
| Byte-oriented character sets | Replaced by the encoding-aware scanner direction, with UTF-8 first. |
| Sample languages such as calc and DRAW | Rebuilt as modern `.lf` runnable examples where useful. |

## Fixture Role

| Fixture group | Input | Current result |
|---|---|---|
| UCDT calc | `testdata/ucdt/calc/calc.l`, `testdata/ucdt/calc/calc.y` | Validates with `11` lexer states, `19` parser states, `10` grammar rules. |
| UCDT DRAW | `testdata/ucdt/draw/draw.l`, `testdata/ucdt/draw/draw.y` | Validates with `53` lexer states, `78` parser states, `31` grammar rules. |
| UCDT Lex meta | `testdata/ucdt/metas/lex.l`, `testdata/ucdt/metas/lex.y` | Validates with `31` lexer states, `40` parser states, `23` grammar rules. |
| UCDT Yacc meta | `testdata/ucdt/metas/yacc.l`, `testdata/ucdt/metas/yacc.y` | Validates with `18` lexer states, `28` parser states, `17` grammar rules. |

The fixture corpus is intentionally source-only. Generated Pascal output,
executables, graphics runtime files, and other binary artifacts are excluded.

Fixture validation is useful because it catches regressions in split input
parsing, regex handling, LR table construction, and sample translation.

## Current Direction

The preferred path for new examples and users is:

```text
.lf spec -> target-neutral lexer/parser tables -> generated Go/C#/C/C++
```

LangForge is free to choose clearer syntax, UTF-8 source handling, stronger
diagnostics, and cleaner generated APIs.

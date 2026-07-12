---
name: langforge-spec-authoring
description: Create, edit, validate, or migrate LangForge grammar specifications. Use when working with `.lf` files, split `.l`/`.y` Lex/Yacc-style inputs, lexer rules, parser productions, `%token`/`%start`/`%type` directives, named RHS labels, semantic type declarations, parser recovery, parser conflicts, regex validation errors, or grammar examples that should be accepted by `lang-forge validate`, `inspect`, or `generate`.
---

# LangForge Spec Authoring

## Overview

Author LangForge specifications as source-of-truth compiler inputs. Keep lexer
and parser changes small, validate early, and explain grammar choices when they
affect ambiguity, token priority, split-file import behavior, or scanner encoding.

## Workflow

1. Read the relevant existing spec and nearby tests/examples before editing.
2. Load `references/spec-patterns.md` when writing non-trivial lexer/parser
   syntax, migrating split `.l`/`.y`, or debugging validation errors.
3. Prefer combined `.lf` specs for new examples and tools.
4. Remember the current scanner defaults to checked UTF-8 for in-process and
   generated Go, C#, C, and C++ output. Additional non-UTF-8 source encoding
   adapters are future work.
5. Encode precedence through grammar structure because precedence declarations
   are not implemented yet. Use target-specific semantic action labels as reducer hooks
   by default; add named RHS labels such as `left=Expr`; declare
   `%semantic <target> type Nonterminal Type` when reducers should have a
   typed contract; add `%semantic <target> import` for handwritten semantic
   dependencies; and use `%semantic go mode inline` only for intentional
   target-specific generated Go snippets. Supported generation targets are Go,
   C#, C, and C++ (`go`, `csharp`, `c`, and `cpp`/`c++`).
6. Validate with the source runner first:

```sh
/usr/local/go/bin/go run ./cmd/lang-forge validate --spec path/to/spec.lf
```

7. Inspect table shape when a grammar change is surprising:

```sh
/usr/local/go/bin/go run ./cmd/lang-forge inspect --spec path/to/spec.lf --format text
/usr/local/go/bin/go run ./cmd/lang-forge inspect --spec path/to/spec.lf --format json
```

8. If generating target output, write it to the example/tool-local generated
   directory (`generated/` for Go, C, and C++; `Generated/` for C#) and keep
   generated output out of committed source unless the task explicitly asks for
   a golden fixture. Generated outputs use conventional filenames: Go `.go`,
   C# `*.g.cs`, C `.h`/`.c`, and C++ `.hpp`/`.cpp`.
9. Use generated `langforge.actions.json` as the cross-target semantic
   contract. Go, C#, C, and C++ can generate typed reducer contexts/adapters;
   Go, C#, and C++ validate reducer-map coverage, and C validates required
   typed handler pointers.

## Rules Of Thumb

- Order lexer rules from most specific to least specific; longest match wins
  first, then rule priority.
- Reject or rewrite rules that can match empty input.
- Keep token names and nonterminal names disjoint.
- Use `%type slr`, `%type lalr`, `%type ielr`, or `%type canonical` only when
  selecting a parser algorithm intentionally. LALR is the default.
- Use the reserved `error` symbol only for parser recovery productions and keep
  expected-token aliases/groups in the grammar when diagnostics need friendly
  names.
- Use split `.l`/`.y` inputs only for source-only fixtures or import
  experiments; UCDT is a reference, not a generated-output target.

## Validation

For a spec-only change, run at least:

```sh
/usr/local/go/bin/go run ./cmd/lang-forge validate --spec path/to/spec.lf
/usr/local/go/bin/go test -count=1 ./...
```

For a migration fixture, run:

```sh
/usr/local/go/bin/go run ./cmd/lang-forge validate --lex path/to/file.l --yacc path/to/file.y
```

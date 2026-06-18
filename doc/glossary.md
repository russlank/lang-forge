# Glossary

Document id: `lang-forge-glossary-v1`
Status: `active`
Last updated: `2026-06-18`
Owner: `Project maintainers`
Scope: `Common compiler terms used by LangForge documentation and code`

This glossary uses LangForge's current implementation vocabulary. It is not a
complete compiler textbook, but it should make the docs and code easier to
read.

| Term | Meaning in LangForge |
|---|---|
| Action | A parser-table operation: shift, reduce, accept, or error. Lexer actions are separate and currently include token, skip, channel, or raw legacy action. |
| Alphabet class | A deterministic scanner-symbol range partition used by the DFA. In UTF-8 mode this is a sparse partition over Unicode scalar ranges. |
| AST | Abstract syntax tree. Examples build ASTs or command models either through generated reducer hooks or handwritten semantic layers. |
| Backend | A target-language code generator. The current implemented backend is Go. |
| Build tag | A Go conditional-compilation marker such as `//go:build langforge_generated`. The examples use it so generated-dependent files compile only after the Makefile has created `generated/`. |
| Canonical LR(1) | The most precise implemented LR parser-table algorithm. It keeps full lookahead-specific states. |
| Channel | A lexer output lane. Hidden channels can preserve whitespace or comments while keeping them away from the parser. |
| Conflict | A parser table cell that wants two incompatible actions, such as shift/reduce or reduce/reduce. |
| DFA | Deterministic finite automaton. The generated scanner uses one active DFA state at a time. |
| EOF | End of input. LangForge uses internal parser terminal `$` and generated Go token `TokenEOF`. |
| FIRST set | For a symbol or sequence, the terminals that can appear first in strings derived from it. |
| FOLLOW set | For a nonterminal, the terminals that can appear immediately after it in some derivation. |
| Generated directory | The local `generated/` folder where example Makefiles write reproducible scanner/parser output. It is ignored by Git. |
| Generated recognizer | Generated scanner/parser code that validates token streams against a grammar. Generated Go parsers can also run reducer callbacks for target-tagged grammar actions. |
| Grammar | The parser part of a specification: nonterminals, terminals, start symbol, and productions. |
| IR | Intermediate representation. A target-neutral model used or planned between parsing and code generation. |
| Item | A grammar production with a dot marking progress, such as `Expr -> Expr Plus . Term`. |
| Inline mode | Go semantic mode where target action text is emitted into generated `parser.go`. It is explicit and target-specific. |
| LALR(1) | The default parser algorithm. It builds canonical LR(1) items and merges states with the same LR(0) core. |
| IELR(1) | A deterministic LR(1) parser algorithm that keeps LALR-style merges only when they preserve canonical-LR behavior. |
| Lexeme | The slice of source text matched by a token, such as `"123"` for token `Number`. |
| Lexer | The scanner stage that converts source text into tokens. |
| Longest match | Scanner rule: choose the longest token text possible before applying rule priority. |
| LR(0) core | A set of LR items without lookahead. LALR merges states with equal LR(0) cores. |
| NFA | Nondeterministic finite automaton. LangForge builds NFAs from regexes before converting them to DFAs. |
| Nonterminal | A grammar symbol defined by parser rules, such as `Expr` or `Statement`. |
| Nullable | A nonterminal or expression that can derive or match the empty string. Whole lexer rules must not be nullable. |
| Parser | The stage that checks whether a token stream matches grammar productions. |
| Production | One grammar alternative, such as `Expr -> Expr Plus Term`. |
| Reduce | Parser action that replaces a matched right-hand side with its left-hand nonterminal. |
| Reducer mode | Default Go semantic mode where target action text is treated as a label and passed to user code as both `Reduction.ActionID` and `Reduction.Action`. |
| Reentrant | Safe to use through independent scanner/parser instances without global mutable parse state. |
| Rule priority | Scanner tie-breaker after longest match: earlier rules win equal-length matches. |
| Semantic action | Target-tagged text associated with a production alternative. Generated Go exposes reducer-mode labels through generated `SemanticAction` constants plus the original string, or emits inline-mode code into generated reduction code. |
| Shift | Parser action that consumes one input token and moves to another state. |
| SLR(1) | A compact LR parser algorithm using LR(0) states plus FOLLOW-set reductions. Useful for simple grammars and teaching. |
| Spec | A `.lf` file or split `.l`/`.y` pair that defines lexer and parser behavior. |
| Terminal | A parser-visible token name, such as `Number`, `Plus`, or `Identifier`. |
| Token | A symbolic category emitted by the lexer and consumed by the parser. |

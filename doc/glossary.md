# Glossary

Document id: `lang-forge-glossary-v1`

Status: `active`

Last updated: `2026-07-12`

Owner: `Project maintainers`

Scope: `Common compiler terms used by LangForge documentation and code`

This glossary uses LangForge's current implementation vocabulary. It is not a
complete compiler textbook, but it should make the docs and code easier to
read.

| Term | Meaning in LangForge |
|---|---|
| Action manifest | The generated `langforge.actions.json` file. It records semantic action labels, grammar rules, RHS labels, typed/untyped status, and source spans so handwritten reducers and parity checks can verify their contract with the grammar. |
| Alphabet class | A deterministic scanner-symbol range partition used by the DFA. In UTF-8 mode this is a sparse partition over Unicode scalar ranges. |
| AST | Abstract syntax tree. Examples build ASTs or command models either through generated reducer hooks or handwritten semantic layers. |
| Backend | A target-language code generator. Current implemented backends emit Go, C#, C, and C++. |
| Boxed semantic value | A semantic value carried through a target's general-purpose container, such as Go `any`, C# `object?`, C tagged values, or C++ `std::any`. Boxed values are flexible but need runtime checks. |
| Build tag | A Go conditional-compilation marker such as `//go:build langforge_generated`. The examples use it so generated-dependent files compile only after the Makefile has created `generated/`. |
| Canonical LR(1) | The most precise implemented LR parser-table algorithm. It keeps full lookahead-specific states. |
| Channel | A lexer output lane. Hidden channels can preserve whitespace or comments while keeping them away from the parser. |
| Conflict | A parser table cell that wants two incompatible actions, such as shift/reduce or reduce/reduce. |
| DFA | Deterministic finite automaton. The generated scanner uses one active DFA state at a time. |
| EOF | End of input. LangForge uses internal parser terminal `$` plus target-specific generated EOF token names such as Go `TokenEOF`, C# `Token.EOF`, C `*_TOKEN_EOF`, and C++ `Token::EOF_`. |
| FIRST set | For a symbol or sequence, the terminals that can appear first in strings derived from it. |
| FOLLOW set | For a nonterminal, the terminals that can appear immediately after it in some derivation. |
| Generated directory | The local `generated/` folder where example Makefiles write reproducible scanner/parser output. It is ignored by Git. |
| Generated recognizer | Generated scanner/parser code that recognizes whether lexemes match the grammar. Recognizers can also invoke reducers for semantic actions when the caller asks for a semantic value. |
| Grammar | The parser part of a specification: nonterminals, terminals, start symbol, and productions. |
| IR | Intermediate representation. A target-neutral model used or planned between parsing and code generation. |
| Item | A grammar production with a dot marking progress, such as `Expr -> Expr Plus . Term`. |
| Inline mode | Go semantic mode where target action text is emitted into generated `parser.go`. It is explicit and target-specific. |
| LALR(1) | The default parser algorithm. It builds canonical LR(1) items and merges states with the same LR(0) core. |
| IELR(1) | A deterministic LR(1) parser algorithm that keeps LALR-style merges only when they preserve canonical-LR behavior. |
| Lexeme | One scanner output item: token kind, matched source text, channel, byte/character span, and line/column span. For example, source text `"123"` may produce a `Number` lexeme. |
| Lexeme source | A synchronous pull abstraction that returns one lexeme at a time to the parser. Generated scanners implement this role, and token collections can be adapted to it. |
| Lexer | The scanner stage that converts source text into tokens. |
| Longest match | Scanner rule: choose the longest token text possible before applying rule priority. |
| LR(0) core | A set of LR items without lookahead. LALR merges states with equal LR(0) cores. |
| NFA | Nondeterministic finite automaton. LangForge builds NFAs from regexes before converting them to DFAs. |
| Nonterminal | A grammar symbol defined by parser rules, such as `Expr` or `Statement`. |
| Nullable | A nonterminal or expression that can derive or match the empty string. Whole lexer rules must not be nullable. |
| Parser | The stage that checks whether a lexeme stream matches grammar productions, runs reductions, and optionally reports recovery diagnostics. |
| Parser action | A parser-table operation: shift, reduce, accept, or error. This is different from a semantic action label such as `{go: add}`. |
| Parser facade | Handwritten application code that hides generated scanner/parser details behind a stable domain API, such as `Parse(source) -> ParseResult<ProgramNode>`. |
| Pre-tokenized parsing | Parsing an already materialized token/lexeme collection. This is useful for tests, token reports, and debugging, but it stores the full token stream before parsing. |
| Production | One grammar alternative, such as `Expr -> Expr Plus Term`. |
| Reduce | Parser action that replaces a matched right-hand side with its left-hand nonterminal. |
| Reducer | Handwritten semantic code called when the parser reduces a grammar rule with a semantic action label. Reducers build AST nodes, values, commands, or diagnostics policy results. |
| Reducer mode | Semantic mode where target action text is treated as a label and passed to handwritten reducer code through a generated action ID, original label, RHS labels, and semantic values. |
| Reduction | One reducer invocation: the grammar rule, semantic action, RHS labels, and semantic values produced by the parser for a completed rule. |
| Reentrant | Safe to use through independent scanner/parser instances without global mutable parse state. |
| Rule priority | Scanner tie-breaker after longest match: earlier rules win equal-length matches. |
| Semantic action | Target-tagged label or code associated with a production alternative, such as `{go: add}` or `{csharp: statement.print}`. In reducer mode the label becomes the contract between the grammar and handwritten reducer. |
| Shift | Parser action that consumes one input lexeme and moves to another state. |
| SLR(1) | A compact LR parser algorithm using LR(0) states plus FOLLOW-set reductions. Useful for simple grammars and teaching. |
| Spec | A `.lf` file or split `.l`/`.y` pair that defines lexer and parser behavior. |
| Source-based parsing | Parsing directly from a scanner or lexeme source. The parser pulls lexemes lazily on demand; this is synchronous streaming, not async producer/consumer machinery. |
| Source text | The original input characters read by the scanner from a string, reader, stream, callback, file, or other source. |
| Terminal | A parser-visible token name, such as `Number`, `Plus`, or `Identifier`. |
| Token | A symbolic category emitted by the lexer and consumed by the parser. |
| Token collection | A materialized slice/list/vector/array of lexemes produced before parsing. LangForge docs use “token collection” for tests, debugging, token inspection, and simple examples; “lexeme source” names the lazy parser input. |
| Typed reducer context | A generated, target-specific reducer argument whose fields are named from RHS labels and whose values use declared semantic types. Typed contexts are the preferred production shape when semantic types are known. |

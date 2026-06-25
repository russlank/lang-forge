# LangForge Tool Improvement Roadmap

Document id: `lang-forge-tool-improvement-roadmap-v1`
Status: `active`
Last updated: `2026-06-25`
Owner: `Project maintainers`
Scope: `Forward-looking public roadmap for LangForge usability, diagnostics, generated APIs, editor tooling, and production-readiness`

**Document purpose:** Capture suggested improvements to LangForge as a parser/scanner/compiler tooling project.  
**Audience:** LangForge maintainers, contributors, and AI coding agents working on future development.  
**Scope:** Core generator features, grammar ergonomics, typed semantic values, diagnostics, IDE/editor support, runtime maturity, and production-readiness.  
**Date:** 2026-06-25

---

This roadmap is directional rather than a commitment list. The implementation
backlog remains the source of truth for accepted work, while this document
explains the larger improvement themes and helps contributors understand which
ideas should be shaped into backlog items or ADRs next.

## 1. Current Position

LangForge already has a strong foundation as a modern Lex/Yacc-style parser-generator tool. Its current strengths include:

- combined `.lf` specifications;
- scanner generation;
- parser-table construction;
- LR-family algorithms such as SLR, LALR(1), IELR(1), and canonical LR(1);
- conflict reporting;
- Go, C#, C, and C++ output;
- reducer-based semantic hooks;
- deterministic manifests;
- named RHS labels and target-specific nonterminal type declarations;
- generated Go typed reducer contexts and reducer coverage validation;
- deterministic cross-target semantic action manifests;
- generated example projects.

The examples show the intended model well:

```text
.lf grammar
  -> generated scanner/parser
  -> handwritten reducer
  -> AST or semantic model
  -> compiler/interpreter/renderer/report
```

The next stage should focus on making LangForge easier, safer, more diagnosable, and more useful for both beginners and production users.

---

## 2. Strategic Direction

The most valuable long-term direction is:

```text
less runtime casting
better diagnostics
more grammar tooling
cleaner generated APIs
better editor integration
stronger production/runtime story
```

LangForge should continue to serve three levels of users:

1. **Beginners** building small DSLs and learning compiler construction.
2. **Application developers** embedding parsers in real products.
3. **Advanced users** building compilers, language servers, transpilers, validators, or editor tooling.

---

## 3. Highest-Value Addition: Typed Semantic Values

### Problem

The current reducer model naturally exposes boxed values:

| Target | Current semantic value style |
|---|---|
| Go | `any` |
| C# | `object?` |
| C | `void*` |
| C++ | `std::any` |

This gives flexibility, but it pushes casts into user code:

```csharp
SemanticAction.Add => (double)ctx.Values[0]! + (double)ctx.Values[2]!
```

```cpp
return std::any_cast<double>(ctx.values.at(index));
```

```c
return calc_number(demo, error, calc_value_as_number(ctx->values[0]) + calc_value_as_number(ctx->values[2]));
```

This creates several problems:

- reducer code becomes fragile after grammar changes;
- type errors appear at runtime;
- examples teach scattered casts;
- diagnostics vary by target;
- code becomes harder to audit.

### Implemented Foundation

LangForge now accepts target-specific nonterminal result types:

```lf
%semantic go type Expr float64
%semantic csharp type Expr double
%semantic c type Expr double
%semantic cpp type Expr double
```

Terminals remain generated `Lexeme` values so scanner text and source spans are
not discarded. The Go backend validates type syntax and generates contexts:

```go
parser.TypedAdd(func(ctx parser.AddReduction) (float64, error) {
	return ctx.Left + ctx.Right, nil
})
```

Named RHS labels are implemented in the grammar:

```lf
Expr : left=Expr Plus right=Term {go: add}
```

All backends emit the labels and declared types in
`langforge.actions.json`. C#, C, and C++ typed context APIs remain the next
backend-parity step.

### Acceptance Criteria

- Generated reducers can avoid direct `ctx.Values[n]` access.
- Type mismatches produce clear compile-time or high-quality runtime errors.
- Examples use typed accessors consistently.
- Backward compatibility with boxed reducer mode is preserved.

---

## 4. Generate CST, Parse Trees, Visitors, and Listeners

### Problem

Reducer mode is compact and good for examples, but many users expect generated parse trees that can be walked independently from semantic actions.

### Recommendation

Add optional parse-tree generation.

Possible options:

```lf
%tree cst
%visitor true
%listener true
```

Generated output could include:

```text
ParseTree
NodeKind
Visit(node)
Walk(listener)
```

### Benefits

- Beginners can inspect parse results.
- Tool builders can use concrete syntax trees.
- Formatters and refactoring tools can preserve source structure.
- Compiler authors can explicitly lower CST to AST.
- Reducer mode remains available for compact semantic evaluation.

### Acceptance Criteria

- Users can choose reducer-only or CST mode.
- CST nodes include source spans.
- Generated visitors/listeners are idiomatic per target.
- Parse-tree generation is optional to avoid overhead for production parsers.

---

## 5. Structured Diagnostics and Error Recovery

### Problem

Production parsers need useful errors, not just parse failure.

### Recommendation

Introduce a target-neutral diagnostic model.

Example JSON shape:

```json
{
  "severity": "error",
  "code": "LF_PARSE_001",
  "message": "expected expression after '+'",
  "file": "input.calc",
  "range": {
    "start": { "line": 1, "column": 4 },
    "end": { "line": 1, "column": 5 }
  },
  "expected": ["Number", "LParen", "Minus"]
}
```

Generated APIs should expose:

```text
Diagnostic
SourceRange
ExpectedToken
RecoveryAction
```

### Error Recovery

Add grammar-level recovery support.

Current status: the first conservative recovery level is implemented. The
reserved `error` symbol, explicit synchronization productions, aliases,
groups, hidden expected tokens, structured cross-target results, and
non-looping tests are available. Phrase/editor recovery heuristics and
lookahead-correction refinement remain future work.

Possible syntax:

```lf
Statement : error Semi {go: recover.statement}
```

Possible recovery modes:

```lf
%recovery panic
%recovery phrase
%recovery editor
```

### Recovery Levels

| Mode | Purpose |
|---|---|
| `panic` | Skip tokens until synchronizing token. |
| `phrase` | Recover inside a known grammar phrase. |
| `editor` | Try to continue and produce partial trees for IDE use. |

### Acceptance Criteria

- Parser can return multiple diagnostics.
- Parser can optionally continue after syntax errors.
- Recovery behavior is grammar-controllable.
- Error messages include source spans and expected tokens.

---

## 6. Add an Incremental / Editor-Facing Mode

### Problem

Compiler parsers can fail fast. IDE parsers must handle incomplete and invalid input continuously.

### Recommendation

Add separate generation modes:

```sh
lang-forge generate --target go --mode compiler
lang-forge generate --target go --mode editor
```

### Compiler Mode

Optimized for:

- strict parsing;
- fast failure;
- AST/reducer output;
- production compilation.

### Editor Mode

Optimized for:

- partial parse results;
- multiple diagnostics;
- syntax-error recovery;
- hidden-token/trivia preservation;
- stable node ranges;
- future incremental reparsing.

### First Milestone

Before full incremental parsing, implement:

- tolerant parse;
- partial CST;
- stable source spans;
- multiple diagnostics;
- expected-token information.

### Later Milestone

Add incremental parsing support:

```text
previous tree + text edit -> updated tree
```

### Acceptance Criteria

- Compiler mode remains fast and strict.
- Editor mode can produce useful output for invalid input.
- Future LSP/editor features can build on editor mode.

---

## 7. Improve Grammar Ergonomics

### 7.1 Grammar Imports and Modules

Add:

```lf
%import common.tokens
%import expressions.grammar
```

Useful for:

- shared expression grammars;
- shared lexer definitions;
- large language specifications;
- reusable token sets.

### 7.2 EBNF Sugar

Allow concise grammar forms:

```lf
Arguments : Expr (Comma Expr)* ;
Block : LBrace Statement* RBrace ;
ParameterList : Ident (Comma Ident)* ;
```

Internally lower to LR-compatible productions.

### 7.3 Named RHS Labels

Add:

```lf
FunctionDecl :
    Func name=Ident LParen params=ParameterList RParen body=Block
    {go: functionDecl}
;
```

This improves:

- reducers;
- diagnostics;
- generated CST/AST names;
- readability.

### 7.4 Precedence and Associativity

Support, if not already available:

```lf
%left Plus Minus
%left Star Slash
%right UnaryMinus
```

This is expected by users familiar with yacc/bison-like tools.

### 7.5 Lexer Modes / States

Add lexer modes:

```lf
%mode default
%mode string
%mode interpolation
```

Use cases:

- strings;
- multiline comments;
- template languages;
- XML/HTML-like syntax;
- interpolated expressions.

### 7.6 Unicode Categories

Support readable Unicode categories:

```lf
LETTER = \p{L};
DIGIT = \p{Nd};
IDENT = LETTER (LETTER | DIGIT | "_")*;
```

This would make Unicode-aware lexers much easier to write.

---

## 8. Add Optional GLR or Generalized Parsing

### Problem

Some real grammars are hard to express as deterministic LR without awkward refactoring.

### Recommendation

Keep LALR/IELR/canonical LR as the default, but add an advanced generalized parser mode later.

Possible syntax:

```lf
%parser glr
%ambiguity report
%ambiguity keep
%ambiguity resolve preferShift
```

### Use Cases

- ambiguous DSLs;
- legacy grammars;
- natural-language-like inputs;
- complex expression syntaxes;
- IDE parsing;
- grammars where ambiguity should be inspected rather than rejected.

### Roadmap

1. Improve conflict explanation first.
2. Add ambiguity reporting.
3. Add optional GLR mode.
4. Add ambiguity-resolution hooks.

### Acceptance Criteria

- Deterministic LR remains the default.
- GLR is clearly documented as advanced.
- Ambiguities are inspectable and controllable.

---

## 9. Conflict Diagnosis and Grammar Explainability

### Problem

Conflict reports are necessary, but users also need to understand why conflicts happen.

### Recommendation

Add commands:

```sh
lang-forge explain --spec grammar.lf --conflicts
lang-forge explain --spec grammar.lf --state 42
lang-forge visualize --spec grammar.lf --format html
```

### Useful Outputs

- minimal token sequence that reaches the conflict;
- competing parse paths;
- shift/reduce or reduce/reduce explanation;
- FIRST/FOLLOW sets;
- state-machine graph;
- lookahead propagation;
- LALR merge explanation;
- suggestions such as “try IELR” or “try canonical LR”.

### Acceptance Criteria

- Users can debug conflicts without reading raw table dumps.
- Reports are useful for both beginners and experts.
- CI can produce conflict reports as artifacts.

---

## 10. Add Grammar Formatter and Linter

### Recommendation

Add:

```sh
lang-forge fmt grammar.lf
lang-forge lint grammar.lf
```

### Lint Rules

Potential lint warnings:

- unused tokens;
- unused lexer macros;
- unreachable parser rules;
- tokens shadowed by earlier lexer rules;
- duplicate semantic action labels;
- nullable cycles;
- empty productions that cause conflicts;
- rules that only forward values unnecessarily;
- inconsistent action naming;
- token/nonterminal naming convention issues;
- hidden ambiguity risks.

### Acceptance Criteria

- `fmt` produces stable output.
- `lint` can run in CI.
- Warnings have stable codes.
- Users can suppress intentional warnings.

---

## 11. Add Project Scaffolding

### Recommendation

Add:

```sh
lang-forge init calc --target go
lang-forge init mini-compiler --target csharp
lang-forge init dsl --target cpp --template compiler
```

### Templates

Suggested templates:

```text
calc
expression
config-file
mini-compiler
tree-walker
repl
language-server
```

### Generated Project Structure

A new project should include:

```text
grammar.lf
Makefile or build script
README.md
sample input
parser adapter
typed reducer helpers
AST
tests
```

### Acceptance Criteria

- New users can start a working parser project in one command.
- Generated starter projects follow best practices.
- Templates are kept in sync with examples.

---

## 12. Add an LSP or VS Code Extension for `.lf`

### Recommendation

Add editor support for grammar authoring.

Useful features:

- syntax highlighting;
- format-on-save;
- token/rule navigation;
- go to action implementation;
- inline conflict diagnostics;
- preview generated tokens/rules;
- preview parse tree for sample input;
- FIRST/FOLLOW/state inspection;
- warnings for unused tokens/rules.

### Acceptance Criteria

- `.lf` files become comfortable to edit.
- Conflicts and lint warnings show inline.
- The extension can call the CLI for validation.

---

## 13. Add a Parse Playground

### Recommendation

Add a local playground:

```sh
lang-forge playground --spec grammar.lf
```

### Features

- paste sample input;
- show token stream;
- show parse tree;
- show reductions;
- show diagnostics;
- show generated action IDs;
- compare LALR vs IELR vs canonical behavior;
- export minimal conflict reproductions.

### Acceptance Criteria

- Useful for demos, docs, and debugging.
- Can run locally without external services.
- Can produce shareable reports.

---

## 14. Add Performance and Size Reports

### Recommendation

Add:

```sh
lang-forge inspect --spec grammar.lf --metrics
lang-forge bench --spec grammar.lf --input sample.txt
```

### Metrics

Report:

```text
lexer states
parser states
action table size
goto table size
conflicts
nullable rules
generation time
parse throughput
allocations
largest DFA state
largest parser state
```

### Acceptance Criteria

- Users can evaluate parser size and performance.
- CI can track generated parser growth.
- Benchmarks are reproducible.

---

## 15. Improve Runtime Packaging and Versioning

### Problem

Multi-target generators need consistent runtime and generated-code versioning.

### Recommendation

Track:

```text
lang-forge CLI version
generated manifest version
runtime API version
target backend version
spec hash
```

Example manifest:

```json
{
  "langforgeVersion": "x.y.z",
  "target": "go",
  "backendVersion": "x.y.z",
  "runtimeApi": "v1",
  "specHash": "...",
  "generatedAt": null,
  "deterministic": true
}
```

Add:

```sh
lang-forge verify-generated
```

### Acceptance Criteria

- Stale generated code can be detected.
- Tool/runtime mismatches are clear.
- CI can verify generated output.

---

## 16. Add Optional Shared Runtime Packages

### Recommendation

Support two runtime modes:

```sh
--runtime embedded
--runtime package
```

### Embedded Runtime

Benefits:

- no external dependency;
- simple examples;
- easy vendoring.

### Package Runtime

Benefits:

- smaller generated code;
- shared diagnostics;
- easier bug fixes;
- stable target APIs.

Possible packages:

```text
github.com/digixoil/langforge/runtime/go
LangForge.Runtime.CSharp
langforge_runtime_c
langforge_runtime_cpp
```

### Acceptance Criteria

- Existing embedded-output behavior remains available.
- Package runtime is optional.
- Generated manifests record runtime mode.

---

## 17. Better Source Span and Trivia Support

### Recommendation

Generated tokens and parse nodes should optionally track:

```text
byte start/end
rune start/end
line/column start/end
file name
leading trivia
trailing trivia
hidden tokens
```

Add options:

```lf
%trivia preserve
%trivia discard
```

### Use Cases

- formatters;
- refactoring tools;
- language servers;
- documentation generators;
- code generators that preserve comments.

### Acceptance Criteria

- Compiler-focused users can discard trivia.
- Editor/tooling users can preserve trivia.
- CST nodes can carry full source ranges.

---

## 18. Semantic Action Validation

### Recommendation

Generate an action manifest and validate reducer coverage.

Status: implemented across Go, C#, C, and C++ generation as deterministic
`langforge.actions.json`. Go `ReducerMap` additionally exposes
`ValidateCoverage`, and `ParseWithReducer` performs the check automatically.

Example manifest:

```json
{
  "actions": [
    {
      "id": 1,
      "name": "add",
      "typed": true,
      "rules": [
        {
          "id": 2,
          "lhs": "Expr",
          "returnType": "float64",
          "rhs": [
            {"position": 1, "symbol": "Expr", "label": "left", "type": "float64"},
            {"position": 2, "symbol": "Plus", "type": "Lexeme"},
            {"position": 3, "symbol": "Term", "label": "right", "type": "float64"}
          ]
        }
      ]
    }
  ]
}
```

Checks:

- action label declared but reducer missing;
- reducer implements action not present in grammar;
- rule has no action and default reduce is ambiguous;
- action naming inconsistent;
- action return type mismatch when typed semantics exist.

### Acceptance Criteria

- Users can test reducer coverage.
- Missing semantic actions are easy to find.
- Large grammars become easier to maintain.

---

## 19. Optional AST Generation

### Recommendation

Add optional AST generation for simple grammars.

Possible syntax:

```lf
%ast generate
```

or:

```lf
Expr :
    left=Expr Plus right=Term {node: BinaryExpr}
  | Number                  {node: NumberExpr}
;
```

### Benefits

- beginners can get started quickly;
- DSL authors can avoid boilerplate;
- generated visitors can operate on AST nodes;
- examples become easier to explain.

### Caution

Generated AST should remain optional. Serious compilers often need handwritten ASTs.

### Acceptance Criteria

- AST generation is opt-in.
- Generated AST nodes are idiomatic per target.
- Users can still use manual reducers.

---

## 20. Language-Server-Oriented Output

### Recommendation

Emit optional metadata useful for editor tooling:

```text
tokens.json
grammar.json
nodes.json
highlights.scm
folds.scm
indents.scm
symbols.json
```

### Use Cases

- syntax highlighting;
- folding;
- symbol extraction;
- outline views;
- language-server indexing;
- grammar documentation.

### Acceptance Criteria

- Metadata output is deterministic.
- Editor tooling can consume it without parsing generated code.
- Metadata aligns with generated parser behavior.

---

## 21. Strict Mode, Teaching Mode, and Production Mode

### Teaching Mode

```lf
%mode teaching
```

Prioritizes:

- readable generated code;
- verbose comments;
- beginner diagnostics;
- examples in output.

### Production Mode

```lf
%mode production
```

Prioritizes:

- compact tables;
- fewer comments;
- fewer allocations;
- performance;
- stable API.

### IDE Mode

```lf
%mode ide
```

Prioritizes:

- recovery;
- partial trees;
- trivia;
- source spans;
- incremental parsing support.

### Acceptance Criteria

- Users can choose output style by purpose.
- Defaults remain simple.
- Mode differences are documented.

---

## 22. Suggested Development Roadmap

### Phase 1: Polish Current Strengths

1. typed reducer helpers;
2. named RHS labels;
3. structured diagnostics;
4. generated action manifest;
5. `lang-forge fmt`;
6. `lang-forge lint`;
7. clean reusable examples.

### Phase 2: Improve Developer Workflow

1. `lang-forge init`;
2. watch mode;
3. generated-code verification;
4. grammar visualization;
5. conflict explainer;
6. performance metrics;
7. reusable runtime option.

### Phase 3: Improve Generated Structure

1. CST generation;
2. visitor/listener generation;
3. optional AST generation;
4. source-span/trivia preservation;
5. semantic action coverage checks.

### Phase 4: Editor and Tooling Support

1. `.lf` language server;
2. VS Code extension;
3. parse playground;
4. tolerant parse mode;
5. multiple-error reporting;
6. editor metadata output.

### Phase 5: Advanced Parsing

1. ambiguity reporting;
2. optional GLR mode;
3. ambiguity-resolution hooks;
4. incremental parsing;
5. grammar imports/modules at scale.

---

## 23. Top 10 Recommendations

If only ten improvements are chosen, prioritize:

1. **Typed semantic values or typed reducer accessors.**
2. **Named RHS symbols** instead of positional `ctx.Values[n]`.
3. **Structured diagnostics** with source ranges and expected tokens.
4. **`lang-forge lint`** for grammar quality.
5. **`lang-forge fmt`** for grammar consistency.
6. **Conflict explainer** with state, lookahead, and minimal repro.
7. **Project templates via `lang-forge init`.**
8. **CST plus visitor/listener generation.**
9. **Source-only reusable examples with golden tests.**
10. **IDE/editor roadmap: recovery, partial trees, metadata, and incremental parsing.**

---

## 24. Recommended First Implementation Tasks

### Task 1: Named RHS Labels

Add grammar support for:

```lf
Expr : left=Expr Plus right=Term {go: add}
```

Generate metadata exposing RHS labels.

### Task 2: Typed Accessor Helpers

Generate or document target-specific helpers:

```text
Arg<T>
LexemeArg
TextArg
NodeArg
```

### Task 3: Structured Diagnostics

Introduce a target-neutral diagnostic shape.

### Task 4: Grammar Linter

Add warnings for unused tokens, unreachable rules, shadowed lexer rules, duplicate actions, and nullable cycles.

### Task 5: Conflict Explainer

Produce human-readable conflict reports and minimal examples.

### Task 6: Template Scaffolding

Add:

```sh
lang-forge init mini-compiler --target go
```

Then mirror it across C#, C, and C++.

---

## 25. Final Direction

LangForge should not try to clone any single existing parser generator. Its opportunity is to combine:

```text
Lex/Yacc-style determinism
modern multi-target code generation
clear reducer semantics
strong diagnostics
typed generated APIs
clean templates
editor-aware parsing
```

The core is already promising. The next major improvement is to make the tool feel safer and more guided:

```text
grammar authors get linting and explanations
application developers get typed APIs and templates
compiler authors get diagnostics and performance reports
IDE/tooling authors get CSTs, spans, recovery, and metadata
```

That would make LangForge useful across the full range of needs: learning, DSL embedding, production parsing, compiler construction, and language tooling.

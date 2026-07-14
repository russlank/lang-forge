# Parser Algorithms

Document id: `lang-forge-parser-algorithms-v1`

Status: `active`

Last updated: `2026-07-11`

Owner: `Project maintainers`

Scope: `Implemented LR parser-table algorithms, automata shape, examples, and usage guidance`

LangForge currently implements the classic deterministic LR family needed by
Lex/Yacc-style tools:

| Algorithm | User selector | Implementation role | Best use |
|---|---|---|---|
| LR(0) item automaton | Not exposed directly | Core item-set construction used by SLR and as the core shape for LALR merging | Understanding states, shifts, gotos, and conflicts |
| SLR(1) | `%type slr` | LR(0) states plus FOLLOW-set reductions | Compatibility checks, small grammars, and teaching/debugging |
| LALR(1) | `%type lalr` or omitted | Canonical LR(1) states merged by identical LR(0) core | Default production choice, close to traditional Yacc behavior |
| IELR(1) | `%type ielr` | Canonical LR(1) states merged only when the merge remains deterministic | LR(1) precision with tables usually close to LALR size |
| Canonical LR(1) | `%type canonical` | Full LR(1) states with separate lookahead-specific items | Deep conflict diagnosis and maximum LR(1) precision |

LALR(1) is the default because it accepts many practical grammars that SLR
rejects while keeping tables much smaller than full canonical LR(1). IELR(1)
is the stronger deterministic mode to try when LALR reports a conflict that
canonical LR(1) can avoid.

For the broader stage-by-stage compiler flow, read
[Compiler Pipeline](compiler-pipeline.md). For visual diagrams of scanner and
parser driving tables before the LR details, read
[Automata And Driving Tables](automata-and-tables.md). For a
beginner-oriented sequence of examples, read [Learning Path](learning-path.md).

## Grammar Shape

LangForge normalizes parser rules into numbered productions and adds an
internal start rule:

```text
0) S' -> S
1) S  -> L Eq R
2) S  -> R
3) L  -> Star R
4) L  -> ID
5) R  -> L
```

The generated parser table has:

- `states`: parser automaton states;
- `actions`: terminal transitions: `shift`, `reduce`, `accept`, or error;
- `gotos`: nonterminal transitions after reductions;
- `conflicts`: recorded shift/reduce or reduce/reduce conflicts;
- `rules`: normalized production list.

`inspect --format json` exposes this shape. LR(1)-based modes also include
`lr1Items`, which carry lookahead terminals.

## Item Notation

An LR item is a production with a dot marking how much of the right-hand side
has already been recognized.

```text
L -> Star . R
```

This means the parser has seen `Star` and now expects an `R`.

An LR(1) item also carries one lookahead terminal:

```text
L -> Star . R, Eq
```

This means the item is relevant when the future lookahead is `Eq`. That small
piece of context is what lets LR(1), IELR(1), and LALR(1) avoid some SLR
conflicts.

## Shared Table-Building Flow

All implemented parser modes start from the same normalized grammar model:

```text
source spec
  -> declarations and rules
  -> terminals and nonterminals
  -> nullable, FIRST, FOLLOW
  -> LR item automata
  -> action/goto tables
  -> conflict recording
```

The generated Go parser runtime is table-driven:

```text
stack = [state 0]
lookahead = next visible token

loop:
    state = top(stack)
    action = actions[state][lookahead]

    if action is shift:
        push action.state
        lookahead = next token

    if action is reduce by A -> beta:
        pop len(beta) states
        state = top(stack)
        push gotos[state][A]

    if action is accept:
        return success

    otherwise:
        return parse error
```

Semantic actions are target-specific reduction hooks in the source model.
Generated Go parsers can dispatch those hooks to a reducer callback while still
supporting recognizer-only parsing through `Parse`.

## LR(0)

LR(0) uses items without lookahead. It answers only one question:

> Based on what has already been shifted, where can the dot move next?

Core operations:

```text
closure(items):
    repeat until no new item is added:
        for each item A -> alpha . B beta:
            if B is a nonterminal:
                add B -> . gamma for every production B -> gamma

goto(items, symbol):
    moved = {}
    for each item A -> alpha . symbol beta:
        add A -> alpha symbol . beta to moved
    return closure(moved)

canonicalLR0(grammar):
    start = closure({ S' -> . S })
    states = [start]
    for each discovered state:
        for each terminal or nonterminal symbol:
            next = goto(state, symbol)
            if next is not empty:
                add or reuse next state
                record transition state --symbol--> next
```

LangForge does not expose `%type lr0` because raw LR(0) reductions are too
coarse for most useful programming-language grammars. LR(0) remains important:
it is the core automaton behind SLR and the core identity used when merging
canonical states into LALR states.

## SLR(1)

SLR uses LR(0) states, then limits reduce actions with FOLLOW sets.

```text
buildSLR(grammar):
    states, transitions = canonicalLR0(grammar)
    follow = FOLLOW(grammar)

    for each transition state --terminal--> to:
        action[state][terminal] = shift to

    for each transition state --nonterminal--> to:
        goto[state][nonterminal] = to

    for each complete item A -> alpha . in state:
        if A is S':
            action[state][$] = accept
        else:
            for each terminal t in FOLLOW(A):
                action[state][t] = reduce A -> alpha
```

SLR is compact and easy to reason about, but FOLLOW sets are global. A reduce
can be enabled on a terminal that is legal after the nonterminal somewhere else
in the grammar, even when it is not legal in this specific state. That can
create conflicts that a real LR(1) context avoids.

Use `%type slr` when:

- comparing behavior with simpler Yacc-like grammars;
- teaching or debugging the basic automaton;
- you want the smallest conceptual table and the grammar validates cleanly.

Do not force SLR for a real DSL just because it is smaller. If SLR reports a
conflict but LALR does not, prefer LALR unless you intentionally want to
simplify the grammar.

## Canonical LR(1)

Canonical LR(1) carries lookahead on every item. Its closure computes
lookaheads from `FIRST(beta lookahead)` for items of the form:

```text
A -> alpha . B beta, lookahead
```

For every `B -> gamma`, closure adds:

```text
B -> . gamma, t
```

for each `t` in `FIRST(beta lookahead)`.

Pseudo-code:

```text
closureLR1(items):
    repeat until no new item is added:
        for each item A -> alpha . B beta, la:
            if B is a nonterminal:
                lookaheads = FIRST(beta followed by la)
                for each production B -> gamma:
                    for each t in lookaheads:
                        add B -> . gamma, t

gotoLR1(items, symbol):
    moved = {}
    for each item A -> alpha . symbol beta, la:
        add A -> alpha symbol . beta, la to moved
    return closureLR1(moved)

canonicalLR1(grammar):
    start = closureLR1({ S' -> . S, $ })
    discover states with gotoLR1 just like LR(0)
```

Table construction then reduces only on each item's own lookahead:

```text
for each complete item A -> alpha ., la:
    if A is S':
        action[state][$] = accept
    else:
        action[state][la] = reduce A -> alpha
```

Canonical LR(1) is the most precise implemented mode. It can create many more
states than LALR because two states with the same LR(0) core but different
lookahead sets stay separate.

Use `%type canonical` when:

- diagnosing a conflict and you want to know whether the grammar is truly not
  LR(1), or only suffering from SLR/LALR table approximation;
- validating tricky grammar changes before deciding whether LALR is safe;
- producing inspection JSON where exact lookahead separation matters more than
  table size.

## LALR(1)

LALR starts from canonical LR(1), then merges states that have the same LR(0)
core. During the merge, lookaheads are unioned.

```text
buildLALR(grammar):
    canonicalStates, canonicalTransitions = canonicalLR1(grammar)

    for each canonical state:
        key = sorted LR(0) core items
        mergedState[key] += all LR(1) items from canonical state

    for each canonical transition oldFrom --symbol--> oldTo:
        from = mergedStateOf(oldFrom)
        to = mergedStateOf(oldTo)
        transition[from][symbol] = to

    build LR(1)-style action/goto tables from merged states
```

This keeps most of the useful LR(1) context while producing state counts close
to compact Yacc-style table sizes. It can introduce reduce/reduce conflicts that
canonical LR(1) would avoid if merged states union incompatible lookaheads, but
that is rare for ordinary language grammars and is exactly why `%type ielr`
and `%type canonical` remain available for diagnosis.

Use LALR by default for LangForge projects.

## IELR(1)

IELR sits between LALR and canonical LR(1). It has the same user-facing goal as
GNU Bison's IELR mode: keep canonical-LR language precision while merging
states whenever the merge is still safe. Bison documents IELR as a way to
eliminate LALR's "mysterious conflicts" without always paying the full
canonical-LR state count:

- <https://www.gnu.org/software/bison/manual/html_node/LR-Table-Construction.html>
- <https://www.gnu.org/software/bison/manual/html_node/Mysterious-Conflicts.html>

LangForge's IELR implementation starts from canonical LR(1), groups states by
LR(0) core like LALR, and accepts a merged state only when the merged state has
deterministic actions and transitions. If the merge would create a
shift/reduce or reduce/reduce conflict, LangForge splits that group back
toward canonical LR(1), preserving compatible subgroups when it can do so
without introducing an action conflict.

Pseudo-code:

```text
buildIELR(grammar):
    canonicalStates, canonicalTransitions = canonicalLR1(grammar)
    partitions = group canonical states by LR(0) core

    repeat:
        for each partition:
            mergedItems = union LR(1) items from member states
            shiftTerminals = terminal transitions from member states
            if mergedItems plus shiftTerminals creates an action conflict:
                split the partition into deterministic compatible subgroups

        refine partitions until every member has the same transition signature
        when mapped through the current partitions

    build LR(1)-style action/goto tables from the remaining partitions
    attach an IELR report with LALR/IELR/canonical counts and merge decisions
```

This is still intentionally correctness-first rather than a verbatim copy of
Bison's full IELR implementation. It may keep more states than Bison's most
optimized tables, but it should never need more states than canonical LR(1),
and it keeps LALR-sized tables when LALR core merges are already safe.

`inspect` explains what happened:

```text
Parser algorithm: ielr
Parser states: 20
IELR state counts: LALR=19, IELR=20, canonical=21
IELR merges: accepted=1, rejected=1
  accepted core Type -> ID • from canonical states [15,19]
  rejected core Type -> ID •; Name -> ID • from canonical states [2,9]: action-conflict -> [2] [9] (1 candidate conflict(s))
```

The same details are available in JSON under `parseTable.ielr`.

Use `%type ielr` when:

- LALR reports a conflict, but canonical LR(1) validates;
- you want deterministic LR(1) recognition without jumping straight to the
  full canonical table;
- you are documenting a grammar that should be LR(1) but is affected by a
  LALR merge artifact.

## LR(1)-Not-SLR Example

This classic grammar is accepted by canonical LR(1) and LALR(1), but rejected
by SLR:

```text
%type lalr
%token ID Star Eq
%start S

%% lexer
"id" => token(ID);
"*"  => token(Star);
"="  => token(Eq);
[1-32]+ => skip;

%% parser
S : L Eq R
  | R
  ;
L : Star R
  | ID
  ;
R : L
  ;
```

Why SLR conflicts:

- `R -> L .` is a complete item, so SLR reduces on every token in `FOLLOW(R)`.
- `FOLLOW(R)` includes `Eq` because of `L -> Star R` and `S -> L Eq R`.
- In the state that also has `S -> L . Eq R`, the parser can shift `Eq`.
- SLR therefore sees a shift/reduce conflict on `Eq`.

Why LR(1) works:

- In the state with `S -> L . Eq R`, the complete `R -> L .` item is not valid
  with lookahead `Eq`.
- The LR(1) item lookahead restricts the reduce to the actual local contexts.
- LALR keeps enough of that context for this grammar after merging cores.

The same grammar is checked in under
[examples/parser-algorithms](../examples/parser-algorithms). Try the four
implemented modes:

```sh
go run ./cmd/lang-forge validate \
  --spec examples/parser-algorithms/lr1-not-slr-lalr.lf
go run ./cmd/lang-forge validate \
  --spec examples/parser-algorithms/lr1-not-slr-ielr.lf
go run ./cmd/lang-forge validate \
  --spec examples/parser-algorithms/lr1-not-slr-canonical.lf
go run ./cmd/lang-forge validate \
  --spec examples/parser-algorithms/lr1-not-slr-slr.lf
```

The LALR, IELR, and canonical fixtures validate. The SLR fixture returns a
parser conflict exit, which is the expected demonstration.

For inspection:

```sh
go run ./cmd/lang-forge inspect \
  --spec examples/parser-algorithms/lr1-not-slr-canonical.lf \
  --format json > /tmp/lr1-not-slr.inspect.json
```

Look for:

- `parseTable.algorithm`;
- `parseTable.states[].items` for LR(0) cores;
- `parseTable.states[].lr1Items` for LALR, IELR, and canonical lookaheads;
- `parseTable.actions` and `parseTable.gotos`;
- `parseTable.conflicts` when validation reports a conflict.

## Mysterious LALR Conflict Example

This grammar is deterministic LR(1), but LALR merges two states with the same
LR(0) core and different lookahead meaning:

```text
%type ielr
%token ID Colon Comma
%start Def

%% lexer
"id" => token(ID);
":"  => token(Colon);
","  => token(Comma);
[1-32]+ => skip;

%% parser
Def : ParamSpec ReturnSpec Comma
  ;
ParamSpec : Type
  | NameList Colon Type
  ;
ReturnSpec : Type
  | Name Colon Type
  ;
Type : ID
  ;
Name : ID
  ;
NameList : Name
  | Name Comma NameList
  ;
```

The important pair is:

```text
Type : ID ;
Name : ID ;
```

Depending on where `ID` appears, the parser may need to reduce it as `Type` or
as `Name`. Canonical LR(1) keeps those contexts separate. LALR unions the
lookaheads after merging the shared LR(0) core and reports a reduce/reduce
conflict. IELR detects that the merge is unsafe and keeps enough separation to
validate the grammar.

Try it:

```sh
go run ./cmd/lang-forge validate \
  --spec examples/parser-algorithms/mysterious-conflict-lalr.lf
go run ./cmd/lang-forge validate \
  --spec examples/parser-algorithms/mysterious-conflict-ielr.lf
go run ./cmd/lang-forge validate \
  --spec examples/parser-algorithms/mysterious-conflict-canonical.lf
```

Expected behavior:

- LALR reports the expected reduce/reduce conflict.
- IELR validates with fewer states than canonical LR(1) for this fixture and
  reports the accepted/rejected LR(0)-core merges in `inspect`.
- Canonical LR(1) validates and remains the precision baseline.

## Choosing an Algorithm

Start here:

1. Omit `%type` and use the default LALR(1).
2. If you see a conflict, run `inspect --format text` for a quick state
   summary.
3. Switch to `%type ielr`.
4. If IELR succeeds, the grammar is LR(1) and the LALR conflict came from an
   unsafe merge. Keep IELR when the table size is acceptable.
5. If IELR still conflicts, switch temporarily to `%type canonical`.
6. If canonical also conflicts, the grammar is ambiguous or not LR(1) in its
   current form. If canonical succeeds and IELR does not, treat that as an
   implementation gap to report and minimize.
7. Use `%type slr` only when SLR comparison or simplicity is the goal.

Practical rule:

| Situation | Recommended mode |
|---|---|
| New DSL or compiler project | Omit `%type` and use default LALR |
| LALR conflict but grammar should be LR(1) | Use `%type ielr`; compare with `%type canonical` if needed |
| Deep conflict diagnosis | Temporarily use `%type canonical` and inspect JSON |
| Teaching or very small grammars | Try `%type slr` |
| Matching Yacc-like expectations | Use LALR first, SLR only for specific SLR comparison checks |
| Table-size investigation | Compare LALR, IELR, and canonical state counts |

## Best Use of LangForge

For new work:

1. Keep one combined `.lf` file as the syntax source of truth.
2. Put lexer rules from most specific to least specific.
3. Encode precedence through grammar layers until precedence declarations are
   implemented.
4. Keep token names and nonterminal names disjoint.
5. Validate before generating.
6. Inspect JSON when state shape or conflicts are surprising.
7. Switch from LALR to IELR when the grammar is LR(1) but LALR reports a merge
   artifact conflict.
8. Generate into a local `generated` directory and keep it ignored unless you
   intentionally create golden fixtures.
9. Use generated reducer hooks for rule-based semantics, or wrap the generated
   recognizer with a handwritten semantic layer when that is clearer.

Recommended loop:

```sh
go run ./cmd/lang-forge validate --spec grammar.lf
go run ./cmd/lang-forge inspect --spec grammar.lf --format text
go run ./cmd/lang-forge generate --spec grammar.lf --target go --out generated
go test ./...
```

For real projects, package that loop in a Makefile. The included examples show
the intended generated-on-demand pattern:

```sh
make -C examples/go/calc test
make -C examples/go/datakeeper test
make -C examples/go/draw test
make -C examples/go/vehicle-report test
make -C examples/parser-algorithms test
```

For split-file fixture work:

1. Keep raw `.l` and `.y` fixtures under `testdata/ucdt` or a similar
   source-only fixture directory.
2. Validate split inputs with `--lex` and `--yacc`.
3. Treat source fixtures as regression input, not as a promise to preserve every
   source-tool quirk.
4. Create a modern `.lf` example only after the split fixture is understood.

## Current Limits

- LR(0) is not selectable as a generated parser mode.
- Precedence and associativity declarations are not implemented yet, so encode
  precedence in the grammar.
- Generated parsers dispatch target-specific semantic action labels to reducer
  callbacks by default. Go also supports explicit inline action mode for
  target-specific library calls. Generated AST helpers and richer debug
  tracing remain future work.
- Conflict diagnostics record state, symbol, competing actions, involved
  reduce rules, source spans, expanded item displays, and item cores. They do
  not yet generate minimal counterexample strings.
- IELR is correctness-first and now reports accepted and rejected core merges.
  Future research can still compare LangForge's table sizes against Bison's
  most optimized IELR construction on a broader grammar corpus.
- LALR is the default today; future work may add a clearer parser directive
  while preserving `%type` support.

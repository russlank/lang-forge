# LangForge Tracking And Review

Use this reference for hardening reviews, private project-memory updates, and
handoffs.

## Review Focus

Prioritize:

- Parser/lexer correctness, especially empty matches, rule priority,
  ambiguity, explicit EOF handling, and token/nonterminal role collisions.
- Generated-code determinism and package-name validation.
- Example reproducibility from a clean checkout.
- Correct separation between generated parser reducer hooks, recognizer-only
  parsing, inline target-specific actions, and handwritten semantic layers.
- Named RHS labels, target-specific semantic type declarations, generated
  action manifests, Go typed contexts, and checked boxed-helper boundaries for
  C#/C/C++.
- Parser recovery contracts: reserved `error` productions, expected-token
  diagnostics, partial/recovering APIs, and non-looping progress guarantees.
- Test coverage for malformed input, edge grammar constructs, and clean
  artifact policy.
- Cross-target example parity for Go, C#, C, and C++, especially when a scenario
  exists in one family but not another.

## Private Project Memory

When behavior or scope changes and private tracking documents are available,
update the smallest relevant set:

- backlog or implementation queue: states, acceptance criteria, and evidence;
- wishlist/roadmap/plan links when a new idea graduates from theme to tracked
  work;
- baseline or current solution snapshot: current implementation state and
  verification table;
- handoff notes: what changed, evidence, and next actions;
- review notes: substantial findings and hardening notes.

Keep dates concrete. The active workspace date is provided in the environment
context; use that rather than stale dates copied from previous docs.

## Verification Matrix

Use these commands as evidence candidates:

```sh
/usr/local/go/bin/go fmt ./...
/usr/local/go/bin/go test -count=1 ./...
/usr/local/go/bin/go test -cover ./...
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go build ./...
/usr/local/go/bin/go build -trimpath -o dist/lang-forge ./cmd/lang-forge
make examples-test
make examples-run
make examples-clean
make examples-cleanliness
git diff --check
```

For example workflows, also test standalone mode:

```sh
make build
make -C examples/<target>/<name> LANG_FORGE=../../../dist/lang-forge run
```

For native example changes, prefer strict warning checks when compilers are
available:

```sh
make -C examples/c/<name> test CC=gcc CFLAGS='-std=c11 -Wall -Wextra -Werror -O2'
make -C examples/cpp/<name> test CXX=g++ CXXFLAGS='-std=c++17 -Wall -Wextra -Werror -O2'
make -C examples/cpp/<name> test CXX=clang++ CXXFLAGS='-std=c++17 -Wall -Wextra -Werror -O2'
```

## Final Handoff Notes

Summarize:

- Changed files and why they matter.
- Verification commands and results.
- Any generated or ignored output intentionally left or cleaned.
- Remaining gaps and the next useful implementation step.

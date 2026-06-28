---
name: langforge-project-steward
description: Review, harden, document, and track LangForge implementation work. Use when doing code review, bug/edge-case analysis, SOLID/refactoring checks, test coverage improvement, generated-code policy changes, repo-local skill updates, private tracking alignment when available, documentation updates, or preparing implementation context for the next agent.
---

# LangForge Project Steward

## Overview

Keep LangForge changes coherent across code, tests, docs, examples, and project
memory. Treat generated outputs, parser/lexer edge cases, and stale handoff
notes as first-class review surfaces when private tracking documents are
available.

## Workflow

1. Start with `git status --short` and avoid reverting unrelated user changes.
2. Read the affected code and nearby tests before editing.
3. Load `references/tracking-and-review.md` when aligning private tracking
   notes, updating implementation context, or performing a hardening review.
4. For reviews, lead with findings grounded in file/line references; then list
   residual test gaps or assumptions.
5. For implementation, keep edits narrow and follow existing packages:
   `internal/spec`, `internal/lex`, `internal/parse`, `internal/codegen`, and
   example-local semantic layers.
6. Keep one source of truth for repeated concepts such as parser algorithms,
   generated-output policy, semantic action contracts, bootstrap templates, and
   example workflows.
7. Update user-facing docs and private project memory, when present, if
   behavior, workflows, or verification evidence changes.

## Validation Ladder

Choose the smallest set that covers the blast radius, then broaden when touching
shared compiler behavior or example workflows:

```sh
/usr/local/go/bin/go fmt ./...
/usr/local/go/bin/go test -count=1 ./...
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go build ./...
make examples-test
make examples-run
git diff --check
```

For generated code or example policy changes, also prove a clean source tree by
cleaning example outputs and rerunning root tests/builds.

## Documentation Surfaces

Update these when relevant:

- `README.md`, `doc/usage.md`, `doc/examples.md`, `doc/specification.md`,
  `doc/architecture.md`, `doc/generated-code-and-semantics.md`,
  `doc/tool-improvement-roadmap.md`, and `doc/example-template-guide.md`.
- Repo-local skills under `skills/` when workflows or current capabilities
  change.
- Private project memory and handoff notes, when available in the workspace.
- Review notes for substantial findings, when the repository keeps them.

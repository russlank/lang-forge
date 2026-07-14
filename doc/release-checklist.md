# Release Checklist

Document id: `lang-forge-release-checklist-v1`

Status: `active`

Last updated: `2026-07-14`

Owner: `Project maintainers`

Scope: `Repository-local checks to run before a release candidate is tagged`

This checklist is for preparing repository contents before a release candidate.
It does not create tags, publish releases, push Docker images, or modify remote
repositories.

## Local Verification

Start from a clean source tree and run:

```sh
make clean
make ci
make examples-benchmarks
make examples-benchmarks-report BENCH_COUNT=5 BENCH_TIME=1s
make dist VERSION=0.1.0-rc.1
make docker-build
make docker-smoke
```

Use a specific Go executable only when `go` is not on `PATH`:

```sh
make GO=/path/to/go ci
```

The local verification phase should leave release artifacts only under
`dist/`, benchmark reports under `dist/benchmarks/`, and ignored generated
example output. Run `make clean` after inspection when you want to return to a
source-only tree.

## Artifact Expectations

`make dist VERSION=0.1.0-rc.1` should create:

```text
dist/lang-forge-linux-amd64
dist/lang-forge-linux-arm64
dist/lang-forge-darwin-arm64
dist/lang-forge-darwin-amd64
dist/lang-forge-windows-amd64.exe
dist/install-lang-forge.sh
dist/SHA256SUMS
```

`dist/install-lang-forge.sh` should match
[../scripts/install-lang-forge.sh](../scripts/install-lang-forge.sh). The
checksums file should include every binary artifact and the installer script.

## CI Expectations

The GitHub workflow in [../.github/workflows/ci.yml](../.github/workflows/ci.yml)
builds and tests the project, creates local release artifacts, and smoke-tests a
local Docker image. It does not publish Docker images.

The Woodpecker pipeline in [../.woodpecker.yml](../.woodpecker.yml) runs the
example test suite with Go, .NET, GCC, and G++ available. Tag-triggered
publishing behavior is a maintainer-operated step and is not part of this local
checklist.

## Maintainer Tag And Publish Phase

Only after local verification and CI are satisfactory should the maintainer
perform tag and publishing work. That phase is intentionally separate from this
repository-content checklist.

Do not do any of the following as part of local verification:

- create or delete Git tags;
- push tags or branches;
- delete GitHub or Gitea releases;
- publish GitHub or Gitea releases;
- push Docker images;
- force-push anything.


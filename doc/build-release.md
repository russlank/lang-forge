# Build, Pipeline, And Docker

Document id: `lang-forge-build-release-v1`
Status: `active`
Last updated: `2026-06-20`
Owner: `Project maintainers`
Scope: `Local build targets, CI pipelines, release artifacts, Docker image, and licensing`

This guide explains how LangForge is built locally, checked in CI, packaged for
release, and built as a container image.

## Toolchain Requirements

Core build targets require Go `1.26.4` or a compatible newer Go toolchain,
`make`, Git, and a POSIX-like shell. The full CI target also requires GCC or
another C11 compiler because `make test-race` uses CGO and the C examples
compile generated C output. The C# examples require the .NET `10.0` SDK because
the projects target `net10.0`. The C++ examples require a C++17 compiler such
as `g++` or `clang++`.

See [Requirements](requirements.md) for the complete local and CI toolchain
matrix.

## License

LangForge is licensed under the MIT license.

- License file: [../LICENSE](../LICENSE)
- Copyright holder: Russlan Kafri

Every source distribution should include the `LICENSE` file.

## Local Build Targets

The root [Makefile](../Makefile) is the canonical local entry point.

Common targets:

```sh
make fmt
make fmt-check
make vet
make test
make test-race
make build
make ci
```

`make build` writes the local CLI binary to:

```text
dist/lang-forge
```

The binary includes linker-injected metadata:

```text
version, commit, branch, build date
```

You can override these values:

```sh
make build VERSION=0.1.0 COMMIT="$(git rev-parse --short=12 HEAD)" BRANCH=main
```

## Example Targets

Runnable examples regenerate ignored generated code before testing or running:

```sh
make examples-test
make examples-run
make examples-clean
```

The example targets cover:

- calc generated scanner/parser and semantic reducer;
- DataKeeper script compiler demo;
- DRAW PNG renderer demo;
- vehicle report parser/report demo;
- parser algorithm fixtures.

## Release Artifacts

Build cross-platform release artifacts:

```sh
make dist VERSION=0.1.0
```

This creates:

```text
dist/lang-forge-linux-amd64
dist/lang-forge-linux-arm64
dist/lang-forge-darwin-arm64
dist/lang-forge-darwin-amd64
dist/lang-forge-windows-amd64.exe
dist/SHA256SUMS
```

The current release set is intentionally CLI-focused. Generated C# projects
build with the local .NET SDK from source/output files; future C or
language-specific runtime packages can be added without changing these base
artifact names.

## Docker Image

Build the local image:

```sh
make docker-build
```

Run smoke checks:

```sh
make docker-smoke
```

The smoke target checks:

```text
lang-forge version
lang-forge validate --spec examples/go/calc/calc.lf
```

The container image uses a multi-stage build:

- `golang:1.26.4-alpine` builds a static Linux binary;
- `alpine:3.20` runs the final CLI image.

The image entrypoint is:

```text
/usr/local/bin/lang-forge
```

So these forms are equivalent:

```sh
docker run --rm lang-forge:dev version
docker run --rm -v "$PWD:/workspace:ro" -w /workspace lang-forge:dev validate --spec examples/go/calc/calc.lf
```

The image can also be used when `lang-forge` is not installed locally. Use a
read-only mount for commands that only read the source tree:

```sh
docker run --rm -v "$PWD:/workspace:ro" -w /workspace lang-forge:dev \
  inspect --spec examples/go/calc/calc.lf --format text
```

Use a writable mount for generation, and map the container user to the host
user on Linux/WSL so generated files are not owned by root:

```sh
docker run --rm \
  -u "$(id -u):$(id -g)" \
  -v "$PWD:/workspace" \
  -w /workspace \
  lang-forge:dev \
  generate --spec examples/go/calc/calc.lf --target go --out examples/go/calc/generated
```

Project Makefiles should accept `LANG_FORGE` as the command to run. That lets
the same targets use a source checkout, standalone binary, installed binary,
or Docker image. The full pattern is documented in
[Invocation And Layout Patterns](invocation-and-layouts.md).

Compose smoke validation is available with:

```sh
docker compose -f docker-compose.smoke.yml run --rm lang-forge
```

## Image Tags

The helper script [../scripts/build-image-tags.sh](../scripts/build-image-tags.sh)
generates Docker tags for registry pipelines.

Examples:

```sh
CI_COMMIT_TAG=v1.2.3 ./scripts/build-image-tags.sh
```

Produces:

```text
1.2.3
1
1.2
latest
```

For branch builds, it emits a `sha-...` tag and a sanitized branch tag.

## GitHub Actions

The GitHub workflow is:

- [.github/workflows/ci.yml](../.github/workflows/ci.yml)

It runs:

- gofmt check;
- `go vet`;
- race tests;
- generated example tests;
- optional `govulncheck`;
- release artifact build;
- local Docker build and smoke test;
- GitHub release publishing on `v*` tags.

The workflow intentionally does not publish Docker images for now. It does not
log in to a registry, does not call `docker push`, and does not request package
write permissions. The local image is built only to smoke-test the Dockerfile.
GitHub releases attach CLI binaries and `SHA256SUMS`.

## Woodpecker

The Woodpecker pipeline is:

- [../.woodpecker.yml](../.woodpecker.yml)

It runs the same core gates and, on `v*` tags, can:

- build release artifacts;
- generate Docker tags;
- build and push multi-architecture container images;
- publish a Gitea release with attached binaries and checksums.

The Woodpecker `test` step uses the Go Alpine image and installs the extra
toolchains required by the full example suite:

```sh
apk add --no-cache gcc g++ musl-dev make git dotnet10-sdk
```

`gcc`/`musl-dev` cover Go race tests and C examples. `g++` covers the C++17
examples. `dotnet10-sdk` covers the C# examples, which target `net10.0`. `git`
is required by `make examples-cleanliness`, which checks that generated and
build artifacts are not tracked as source.

Required secrets for release publishing:

```text
digixoil_registry_username
digixoil_registry_password
gitea_api_key
```

## Clean Outputs

Generated local outputs are ignored by Git. Clean them with:

```sh
make clean
```

This removes:

- `dist`;
- `.tags`;
- generated/dist output under runnable examples.

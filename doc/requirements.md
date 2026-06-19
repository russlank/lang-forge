# Requirements

Document id: `lang-forge-requirements-v1`
Status: `active`
Last updated: `2026-06-19`
Owner: `Project maintainers`
Scope: `Toolchain requirements for building LangForge and running examples`

LangForge itself is a Go command-line tool. The runnable examples also exercise
generated Go, C#, C, and C++ output, so the full example suite needs the target
language toolchains listed below.

## Core Tooling

Required for the main CLI:

- Go `1.26.4` or a compatible newer Go toolchain.
- GNU Make or a compatible `make`.
- A POSIX-like shell for the provided Makefile and script targets.

The Makefile defaults to `/usr/local/go/bin/go` because that is the toolchain
location in the current development workspace. Override it when Go is on your
`PATH` or installed elsewhere:

```sh
make GO=go test
make GO=/path/to/go build
```

## Full Local CI

`make ci` runs formatting, vetting, race tests, a CLI build, and all runnable
example tests. It therefore needs:

- the core Go/Make/shell tooling;
- a C compiler for Go's race detector and C examples;
- the .NET `10.0` SDK for C# examples;
- a C++17 compiler for C++ examples.

Use these checks to confirm the expected tools are available:

```sh
go version
dotnet --version
gcc --version
g++ --version
make --version
```

## C# Examples

The C# examples target `net10.0` and require the .NET `10.0` SDK. The generated
files use `.g.cs` filenames and are built by SDK-style projects.

Run:

```sh
make -C examples/csharp/calc test
make -C examples/csharp/datakeeper test
make -C examples/csharp/draw test
make -C examples/csharp/vehicle-report test
```

## C Examples

The C examples require a C11-capable compiler. GCC is the verified default in
the current workspace; Clang or another compatible compiler can be selected
with `CC`.

Run with GCC:

```sh
make -C examples/c/calc test CC=gcc
make -C examples/c/datakeeper test CC=gcc
make -C examples/c/draw test CC=gcc
make -C examples/c/vehicle-report test CC=gcc
```

The C DRAW example links the math library through `LDLIBS=-lm` by default.
Override `CFLAGS`, `CC`, or `LDLIBS` when a platform needs different compiler
or linker options.

The C example Makefiles still validate and generate when no C compiler is
available. Build and run steps print a skip message if `CC` cannot be found.

## C++ Examples

The C++ examples require a C++17-capable compiler. GCC `g++` is the default
selected by the root Makefile; Clang or another compatible compiler can be
selected with `CXX`.

Run with GCC:

```sh
make -C examples/cpp/calc test CXX=g++
make -C examples/cpp/datakeeper test CXX=g++
make -C examples/cpp/draw test CXX=g++
make -C examples/cpp/vehicle-report test CXX=g++
```

Run with Clang:

```sh
make -C examples/cpp/calc test CXX=clang++
make -C examples/cpp/datakeeper test CXX=clang++
make -C examples/cpp/draw test CXX=clang++
make -C examples/cpp/vehicle-report test CXX=clang++
```

The C++ example Makefiles still validate and generate when no C++ compiler is
available. Build and run steps print a skip message if `CXX` cannot be found.

## Docker

Docker is optional. It is useful for:

- building a local smoke-test image with `make docker-build`;
- running `make docker-smoke`;
- invoking LangForge without installing a binary.

Docker Compose is optional and only needed for the compose smoke target:

```sh
docker compose -f docker-compose.smoke.yml run --rm lang-forge
```

## GitHub Actions

The GitHub workflow provisions:

- Go `1.26.4`;
- .NET `10.0.x`;
- native build tools through `build-essential`.

It builds CLI release artifacts and a local Docker image for smoke testing. It
does not log in to a registry and does not publish Docker images. For now,
GitHub releases contain CLI binaries and checksums only.

## Woodpecker

The Woodpecker `test` step runs `make examples-test`, so it must include all
example toolchains in addition to Go. The pipeline installs:

```sh
apk add --no-cache gcc g++ musl-dev make dotnet10-sdk
```

If the `dotnet10-sdk` package is missing from the runner's Alpine repositories,
the C# examples fail at `dotnet run` with `No such file or directory`. In that
case, update the runner image or Alpine repository set so .NET `10.0` SDK
packages are available.

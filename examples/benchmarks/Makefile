GO ?= /usr/local/go/bin/go
DOTNET ?= dotnet
CC ?= gcc
CXX ?= g++

.PHONY: bench benchmarks go csharp c cpp smoke clean

bench benchmarks go:
	$(MAKE) -C go GO=$(GO) bench

smoke:
	$(MAKE) -C go GO=$(GO) smoke

csharp:
	@echo "C# benchmark harness is planned; see examples/benchmarks/README.md"

c:
	@echo "C benchmark harness is planned; see examples/benchmarks/README.md"

cpp:
	@echo "C++ benchmark harness is planned; see examples/benchmarks/README.md"

clean:
	$(MAKE) -C go clean

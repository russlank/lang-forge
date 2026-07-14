GO ?= go
DOTNET ?= dotnet
CC ?= gcc
CXX ?= g++

BENCH_TIME ?= 1s
BENCH_COUNT ?= 1
BENCH_PATTERN ?= .
REPORT_DIR ?= $(abspath ../../dist/benchmarks)

CSHARP_BENCH_CONFIGURATION ?= Release
CSHARP_BENCH_FILTER ?= *CalcParse*
CSHARP_BENCH_JOB ?= short
CSHARP_BENCH_ARTIFACTS ?= $(abspath ../../dist/benchmarks/csharp)

.PHONY: prepare prepare-verbose bench benchmarks verbose go csharp csharp-required c cpp report profile compare smoke clean

prepare:
	@$(MAKE) --no-print-directory -C go GO=$(GO) prepare
	@if command -v $(DOTNET) >/dev/null 2>&1; then \
		$(MAKE) --no-print-directory -C csharp GO=$(GO) DOTNET=$(DOTNET) prepare; \
	else \
		echo "C# benchmark prepare skipped: dotnet not found"; \
	fi

prepare-verbose:
	$(MAKE) -C go GO=$(GO) prepare-verbose
	$(MAKE) -C csharp GO=$(GO) DOTNET=$(DOTNET) prepare-verbose

bench benchmarks: go csharp

verbose:
	$(MAKE) -C go GO=$(GO) BENCH_TIME=$(BENCH_TIME) BENCH_COUNT=$(BENCH_COUNT) BENCH_PATTERN='$(BENCH_PATTERN)' bench-verbose
	$(MAKE) -C csharp GO=$(GO) DOTNET=$(DOTNET) CSHARP_BENCH_CONFIGURATION=$(CSHARP_BENCH_CONFIGURATION) CSHARP_BENCH_FILTER='$(CSHARP_BENCH_FILTER)' CSHARP_BENCH_JOB='$(CSHARP_BENCH_JOB)' CSHARP_BENCH_ARTIFACTS='$(CSHARP_BENCH_ARTIFACTS)' bench-verbose

go:
	@$(MAKE) --no-print-directory -C go GO=$(GO) BENCH_TIME=$(BENCH_TIME) BENCH_COUNT=$(BENCH_COUNT) BENCH_PATTERN='$(BENCH_PATTERN)' bench

csharp:
	@if command -v $(DOTNET) >/dev/null 2>&1; then \
		$(MAKE) --no-print-directory -C csharp GO=$(GO) DOTNET=$(DOTNET) CSHARP_BENCH_CONFIGURATION=$(CSHARP_BENCH_CONFIGURATION) CSHARP_BENCH_FILTER='$(CSHARP_BENCH_FILTER)' CSHARP_BENCH_JOB='$(CSHARP_BENCH_JOB)' CSHARP_BENCH_ARTIFACTS='$(CSHARP_BENCH_ARTIFACTS)' bench; \
	else \
		echo "C# benchmarks skipped: dotnet not found. Run examples-benchmarks-csharp after installing .NET."; \
	fi

csharp-required:
	@$(MAKE) --no-print-directory -C csharp GO=$(GO) DOTNET=$(DOTNET) CSHARP_BENCH_CONFIGURATION=$(CSHARP_BENCH_CONFIGURATION) CSHARP_BENCH_FILTER='$(CSHARP_BENCH_FILTER)' CSHARP_BENCH_JOB='$(CSHARP_BENCH_JOB)' CSHARP_BENCH_ARTIFACTS='$(CSHARP_BENCH_ARTIFACTS)' bench

c:
	@echo "C benchmark harness is planned; see examples/benchmarks/README.md"

cpp:
	@echo "C++ benchmark harness is planned; see examples/benchmarks/README.md"

report:
	@mkdir -p $(REPORT_DIR)
	@$(MAKE) --no-print-directory -C go GO=$(GO) BENCH_TIME=$(BENCH_TIME) BENCH_COUNT=$(BENCH_COUNT) BENCH_PATTERN='$(BENCH_PATTERN)' REPORT_DIR='$(REPORT_DIR)' report
	@if command -v $(DOTNET) >/dev/null 2>&1; then \
		$(MAKE) --no-print-directory -C csharp GO=$(GO) DOTNET=$(DOTNET) CSHARP_BENCH_CONFIGURATION=$(CSHARP_BENCH_CONFIGURATION) CSHARP_BENCH_FILTER='$(CSHARP_BENCH_FILTER)' CSHARP_BENCH_JOB='$(CSHARP_BENCH_JOB)' CSHARP_BENCH_ARTIFACTS='$(CSHARP_BENCH_ARTIFACTS)' report; \
	else \
		echo "C# benchmark report skipped: dotnet not found"; \
	fi
	@$(GO) run ./tools/artifact-report --repo ../.. --out $(REPORT_DIR)

profile:
	@$(MAKE) --no-print-directory -C go GO=$(GO) BENCH_TIME=$(BENCH_TIME) REPORT_DIR='$(REPORT_DIR)' profile

compare:
	@if ! command -v benchstat >/dev/null 2>&1; then \
		echo "benchstat not found."; \
		echo "Install with:"; \
		echo "  go install golang.org/x/perf/cmd/benchstat@latest"; \
		exit 0; \
	fi; \
	if [ -z "$(BEFORE)" ] || [ -z "$(AFTER)" ]; then \
		echo "Usage: make examples-benchmarks-compare BEFORE=dist/benchmarks/before.txt AFTER=dist/benchmarks/after.txt"; \
		exit 2; \
	fi; \
	benchstat "$(BEFORE)" "$(AFTER)"

smoke:
	@$(MAKE) --no-print-directory -C go GO=$(GO) smoke

clean:
	@$(MAKE) --no-print-directory -C go clean
	@$(MAKE) --no-print-directory -C csharp clean

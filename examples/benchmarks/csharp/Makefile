GO ?= go
DOTNET ?= dotnet
LANG_FORGE_VERBOSITY ?= 0
PROJECT := LangForge.Examples.Benchmarks.CSharp.csproj
CSHARP_BENCH_CONFIGURATION ?= Release
CSHARP_BENCH_FILTER ?= *CalcParse*
CSHARP_BENCH_JOB ?= short
CSHARP_BENCH_ARTIFACTS ?= $(abspath ../../../dist/benchmarks/csharp)
REPORT_DIR ?= $(abspath ../../../dist/benchmarks)
REPO_ROOT ?= $(abspath ../../..)

.PHONY: prepare prepare-verbose bench bench-verbose report print-summary test clean

prepare:
	@printf 'Preparing C# benchmark generated examples...\n'
	@$(MAKE) -s -C ../../csharp/calc GO=$(GO) DOTNET=$(DOTNET) LANG_FORGE_VERBOSITY=$(LANG_FORGE_VERBOSITY) generate >/dev/null
	@$(MAKE) -s -C ../../csharp/draw GO=$(GO) DOTNET=$(DOTNET) LANG_FORGE_VERBOSITY=$(LANG_FORGE_VERBOSITY) generate >/dev/null
	@$(MAKE) -s -C ../../csharp/parser-recovery GO=$(GO) DOTNET=$(DOTNET) LANG_FORGE_VERBOSITY=$(LANG_FORGE_VERBOSITY) generate >/dev/null

prepare-verbose:
	$(MAKE) -C ../../csharp/calc GO=$(GO) DOTNET=$(DOTNET) LANG_FORGE_VERBOSITY=1 generate
	$(MAKE) -C ../../csharp/draw GO=$(GO) DOTNET=$(DOTNET) LANG_FORGE_VERBOSITY=1 generate
	$(MAKE) -C ../../csharp/parser-recovery GO=$(GO) DOTNET=$(DOTNET) LANG_FORGE_VERBOSITY=1 generate

bench: prepare
	@mkdir -p $(REPORT_DIR) $(CSHARP_BENCH_ARTIFACTS)
	@echo "Running C# BenchmarkDotNet benchmarks (filter: $(CSHARP_BENCH_FILTER), job: $(CSHARP_BENCH_JOB))..."
	@rm -rf $(CSHARP_BENCH_ARTIFACTS)/results
	@$(DOTNET) run -c $(CSHARP_BENCH_CONFIGURATION) --project $(PROJECT) -- --filter '$(CSHARP_BENCH_FILTER)' --artifacts '$(CSHARP_BENCH_ARTIFACTS)' --lf-job '$(CSHARP_BENCH_JOB)' > $(REPORT_DIR)/csharp-benchmarks.txt || { echo "C# BenchmarkDotNet failed; see $(REPORT_DIR)/csharp-benchmarks.txt"; exit 1; }
	@$(MAKE) --no-print-directory -s print-summary

bench-verbose: prepare-verbose
	rm -rf $(CSHARP_BENCH_ARTIFACTS)/results
	$(DOTNET) run -c $(CSHARP_BENCH_CONFIGURATION) --project $(PROJECT) -- --filter '$(CSHARP_BENCH_FILTER)' --artifacts '$(CSHARP_BENCH_ARTIFACTS)' --lf-job '$(CSHARP_BENCH_JOB)'

report: prepare
	@mkdir -p $(REPORT_DIR)
	@echo "Running C# BenchmarkDotNet report (filter: $(CSHARP_BENCH_FILTER), job: $(CSHARP_BENCH_JOB))..."
	@rm -rf $(CSHARP_BENCH_ARTIFACTS)/results
	@$(DOTNET) run -c $(CSHARP_BENCH_CONFIGURATION) --project $(PROJECT) -- --filter '$(CSHARP_BENCH_FILTER)' --artifacts '$(CSHARP_BENCH_ARTIFACTS)' --lf-job '$(CSHARP_BENCH_JOB)' > $(REPORT_DIR)/csharp-benchmarks.txt || { echo "C# BenchmarkDotNet failed; see $(REPORT_DIR)/csharp-benchmarks.txt"; exit 1; }
	@$(MAKE) --no-print-directory -s print-summary

print-summary:
	@$(GO) run ../tools/csharp-summary --repo '$(REPO_ROOT)' --artifacts '$(CSHARP_BENCH_ARTIFACTS)' --out '$(REPORT_DIR)/csharp-benchmarks-summary.md' --job '$(CSHARP_BENCH_JOB)' --filter '$(CSHARP_BENCH_FILTER)'
	@sed -n '1,160p' $(REPORT_DIR)/csharp-benchmarks-summary.md

test: prepare
	$(DOTNET) build -c $(CSHARP_BENCH_CONFIGURATION) $(PROJECT)

clean:
	rm -rf bin obj

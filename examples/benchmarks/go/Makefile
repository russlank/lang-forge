GO ?= /usr/local/go/bin/go
BENCH_TIME ?= 1s
BENCH_COUNT ?= 1
BENCH_PATTERN ?= .
BENCH ?=
BENCHTIME ?=
COUNT ?=
TAGS ?= langforge_generated
LANG_FORGE_VERBOSITY ?= 0
REPORT_DIR ?= $(abspath ../../../dist/benchmarks)
PROFILE_TIME ?= 5s
PROFILE_DIR ?= $(REPORT_DIR)/profiles/go

ifneq ($(strip $(BENCH)),)
override BENCH_PATTERN := $(BENCH)
endif
ifneq ($(strip $(BENCHTIME)),)
override BENCH_TIME := $(BENCHTIME)
endif
ifneq ($(strip $(COUNT)),)
override BENCH_COUNT := $(COUNT)
endif

.PHONY: prepare prepare-verbose bench bench-verbose smoke test report profile clean

prepare:
	@printf 'Preparing Go benchmark generated examples...\n'
	@$(MAKE) -s -C ../../go/calc GO=$(GO) LANG_FORGE_VERBOSITY=$(LANG_FORGE_VERBOSITY) generate >/dev/null
	@$(MAKE) -s -C ../../go/draw GO=$(GO) LANG_FORGE_VERBOSITY=$(LANG_FORGE_VERBOSITY) generate >/dev/null
	@$(MAKE) -s -C ../../go/parser-recovery GO=$(GO) LANG_FORGE_VERBOSITY=$(LANG_FORGE_VERBOSITY) generate >/dev/null

prepare-verbose:
	$(MAKE) -C ../../go/calc GO=$(GO) LANG_FORGE_VERBOSITY=1 generate
	$(MAKE) -C ../../go/draw GO=$(GO) LANG_FORGE_VERBOSITY=1 generate
	$(MAKE) -C ../../go/parser-recovery GO=$(GO) LANG_FORGE_VERBOSITY=1 generate

bench: prepare
	@mkdir -p $(REPORT_DIR)
	@echo "Running Go benchmarks (pattern: $(BENCH_PATTERN), time: $(BENCH_TIME), count: $(BENCH_COUNT))..."
	@$(GO) test -tags $(TAGS) -run '^$$' -bench '$(BENCH_PATTERN)' -benchmem -benchtime $(BENCH_TIME) -count $(BENCH_COUNT) ./... > $(REPORT_DIR)/go-benchmarks.txt || { cat $(REPORT_DIR)/go-benchmarks.txt; exit 1; }
	@$(GO) run ../tools/go-bench-summary --input $(REPORT_DIR)/go-benchmarks.txt --out $(REPORT_DIR)/go-benchmarks-summary.md --go $(GO) --command "$(GO) test -tags $(TAGS) -run '^$$' -bench '$(BENCH_PATTERN)' -benchmem -benchtime $(BENCH_TIME) -count $(BENCH_COUNT) ./..."
	@sed -n '1,160p' $(REPORT_DIR)/go-benchmarks-summary.md

bench-verbose: prepare-verbose
	$(GO) test -tags $(TAGS) -run '^$$' -bench '$(BENCH_PATTERN)' -benchmem -benchtime $(BENCH_TIME) -count $(BENCH_COUNT) ./...

smoke: BENCH_TIME=100ms
smoke: BENCH_COUNT=1
smoke: bench

test: prepare
	@$(GO) test -tags $(TAGS) -run TestBenchmarkFixtures -count=1 ./...

report: prepare
	@mkdir -p $(REPORT_DIR)
	@echo "Running Go benchmark report (pattern: $(BENCH_PATTERN), time: $(BENCH_TIME), count: $(BENCH_COUNT))..."
	@$(GO) test -tags $(TAGS) -run '^$$' -bench '$(BENCH_PATTERN)' -benchmem -benchtime $(BENCH_TIME) -count $(BENCH_COUNT) ./... > $(REPORT_DIR)/go-benchmarks.txt || { cat $(REPORT_DIR)/go-benchmarks.txt; exit 1; }
	@$(GO) run ../tools/go-bench-summary --input $(REPORT_DIR)/go-benchmarks.txt --out $(REPORT_DIR)/go-benchmarks-summary.md --go $(GO) --command "$(GO) test -tags $(TAGS) -run '^$$' -bench '$(BENCH_PATTERN)' -benchmem -benchtime $(BENCH_TIME) -count $(BENCH_COUNT) ./..."
	@sed -n '1,160p' $(REPORT_DIR)/go-benchmarks-summary.md

profile: prepare
	@mkdir -p $(PROFILE_DIR)
	@$(GO) test -tags $(TAGS) -run '^$$' -bench '^BenchmarkCalcParse/ParseFromStringScanner/TypedReducer$$' -benchmem -benchtime $(PROFILE_TIME) -count 1 -cpuprofile $(PROFILE_DIR)/calc-source-typed.cpu.pprof -memprofile $(PROFILE_DIR)/calc-source-typed.mem.pprof ./...
	@$(GO) test -tags $(TAGS) -run '^$$' -bench '^BenchmarkDrawParse/ParseFromStringScanner/BuildAST$$' -benchmem -benchtime $(PROFILE_TIME) -count 1 -cpuprofile $(PROFILE_DIR)/draw-large.cpu.pprof -memprofile $(PROFILE_DIR)/draw-large.mem.pprof ./...
	@echo "Go profiles written under $(PROFILE_DIR)"

clean:
	rm -rf dist

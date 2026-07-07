GO ?= /usr/local/go/bin/go
BENCH ?= .
BENCHTIME ?= 1s
COUNT ?= 1
TAGS ?= langforge_generated

.PHONY: generate bench smoke test clean

generate:
	$(MAKE) -C ../../go/calc GO=$(GO) generate
	$(MAKE) -C ../../go/draw GO=$(GO) generate
	$(MAKE) -C ../../go/parser-recovery GO=$(GO) generate

bench: generate
	$(GO) test -tags $(TAGS) -run '^$$' -bench '$(BENCH)' -benchmem -benchtime $(BENCHTIME) -count $(COUNT) ./...

smoke: BENCHTIME=100ms
smoke: COUNT=1
smoke: bench

test: generate
	$(GO) test -tags $(TAGS) -run TestBenchmarkFixtures -count=1 ./...

clean:
	rm -rf dist

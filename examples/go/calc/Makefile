GO ?= /usr/local/go/bin/go
LANG_FORGE ?= $(GO) run ../../../cmd/lang-forge
GENERATED_DIR := generated
DIST_DIR := dist
TAGS := langforge_generated
INPUT ?= input.calc
LOG ?= $(DIST_DIR)/calc-demo.log

.PHONY: validate generate build run test clean

validate:
	$(LANG_FORGE) validate --spec calc.lf

generate: validate
	$(LANG_FORGE) generate --spec calc.lf --target go --out $(GENERATED_DIR)

build: generate
	mkdir -p $(DIST_DIR)
	$(GO) build -tags $(TAGS) -trimpath -o $(DIST_DIR)/calc-demo ./cmd/calc-demo

run: build
	./$(DIST_DIR)/calc-demo --input $(INPUT) --log $(LOG)

test: generate
	$(GO) test -tags $(TAGS) -count=1 ./...

clean:
	rm -rf $(GENERATED_DIR) $(DIST_DIR)

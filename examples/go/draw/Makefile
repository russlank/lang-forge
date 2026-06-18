GO ?= /usr/local/go/bin/go
LANG_FORGE ?= $(GO) run ../../../cmd/lang-forge
GENERATED_DIR := generated
DIST_DIR := dist
TAGS := langforge_generated
INPUT ?= sample.draw
OUTPUT ?= $(DIST_DIR)/sample.png
LOG ?= $(DIST_DIR)/draw-demo.log

.PHONY: validate generate build run test clean

validate:
	$(LANG_FORGE) validate --spec draw.lf

generate: validate
	$(LANG_FORGE) generate --spec draw.lf --target go --out $(GENERATED_DIR)

build: generate
	mkdir -p $(DIST_DIR)
	$(GO) build -tags $(TAGS) -trimpath -o $(DIST_DIR)/draw-demo ./cmd/draw-demo

run: build
	./$(DIST_DIR)/draw-demo --input $(INPUT) --output $(OUTPUT) --log $(LOG)

test: generate
	$(GO) test -tags $(TAGS) -count=1 ./...

clean:
	rm -rf $(GENERATED_DIR) $(DIST_DIR)

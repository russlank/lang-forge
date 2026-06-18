GO ?= /usr/local/go/bin/go
LANG_FORGE ?= $(GO) run ../../../cmd/lang-forge
GENERATED_DIR := generated
DIST_DIR := dist
TAGS := langforge_generated
INPUT ?= sample.dks
LOG ?= $(DIST_DIR)/datakeeper-demo.log
PARAMS ?= --param InstanceGuid=sql-instance-01 --param ParentGuid=farm-root --param ObjectName=daily-backup --param JobsTag=maintenance

.PHONY: validate generate build run test clean

validate:
	$(LANG_FORGE) validate --spec datakeeper.lf

generate: validate
	$(LANG_FORGE) generate --spec datakeeper.lf --target go --out $(GENERATED_DIR)

build: generate
	mkdir -p $(DIST_DIR)
	$(GO) build -tags $(TAGS) -trimpath -o $(DIST_DIR)/datakeeper-demo ./cmd/datakeeper-demo

run: build
	./$(DIST_DIR)/datakeeper-demo --script $(INPUT) $(PARAMS) --log $(LOG)

test: generate
	$(GO) test -tags $(TAGS) -count=1 ./...

clean:
	rm -rf $(GENERATED_DIR) $(DIST_DIR)

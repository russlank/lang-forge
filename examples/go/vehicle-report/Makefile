GO ?= /usr/local/go/bin/go
LANG_FORGE ?= $(GO) run ../../../cmd/lang-forge
GENERATED_DIR := generated
DIST_DIR := dist
TAGS := langforge_generated
INPUT ?= sample.vehicle
LOG ?= $(DIST_DIR)/vehicle-report-demo.log

.PHONY: validate generate build run test clean

validate:
	$(LANG_FORGE) validate --spec vehicle.lf

generate: validate
	$(LANG_FORGE) generate --spec vehicle.lf --target go --out $(GENERATED_DIR)

build: generate
	mkdir -p $(DIST_DIR)
	$(GO) build -tags $(TAGS) -trimpath -o $(DIST_DIR)/vehicle-report-demo ./cmd/vehicle-report-demo

run: build
	./$(DIST_DIR)/vehicle-report-demo --input $(INPUT) --log $(LOG)

test: generate
	$(GO) test -tags $(TAGS) -count=1 ./...

clean:
	rm -rf $(GENERATED_DIR) $(DIST_DIR)

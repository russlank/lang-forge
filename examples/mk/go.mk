# Shared targets for Go LangForge examples.

LF_TARGET := go
GENERATED_DIR ?= generated
DIST_DIR ?= dist
TAGS ?= langforge_generated
GO_BUILD_PKG ?= ./cmd/$(APP_NAME)
BIN ?= $(DIST_DIR)/$(APP_NAME)
RUN_ARGS ?=

include $(REPO_ROOT)/examples/mk/langforge.mk

.PHONY: build run test clean

build: generate
	mkdir -p $(DIST_DIR)
	$(GO) build -tags $(TAGS) -trimpath -o $(BIN) $(GO_BUILD_PKG)

run: build
	./$(BIN) $(RUN_ARGS)

test: generate
	$(GO) test -tags $(TAGS) -count=1 ./...

clean:
	rm -rf $(GENERATED_DIR) $(DIST_DIR)

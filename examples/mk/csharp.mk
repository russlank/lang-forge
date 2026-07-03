# Shared targets for C# LangForge examples.

LF_TARGET := csharp
GENERATED_DIR ?= Generated
DIST_DIR ?= dist
DOTNET ?= dotnet
RUN_ARGS ?=
TEST_ARGS ?= --assert $(INPUT) --log $(LOG)

include $(REPO_ROOT)/examples/mk/langforge.mk

.PHONY: build run test clean

build: generate
	$(DOTNET) build $(PROJECT)

run: generate
	$(DOTNET) run --project $(PROJECT) -- $(RUN_ARGS)

test: generate
	$(DOTNET) run --project $(PROJECT) -- $(TEST_ARGS)

clean:
	rm -rf $(GENERATED_DIR) $(DIST_DIR) bin obj

# Shared LangForge example generation targets.
#
# Example Makefiles set REPO_ROOT, SPEC, TARGET, and GENERATED_DIR before
# including this fragment. The fragment keeps validation and generation command
# lines identical across runnable demos and copyable templates.

REPO_ROOT ?= ../../..
GO ?= /usr/local/go/bin/go
LANG_FORGE ?= $(GO) run $(REPO_ROOT)/cmd/lang-forge
LANG_FORGE_VERBOSITY ?= 1
LANG_FORGE_FLAGS ?= --verbosity $(LANG_FORGE_VERBOSITY)
SPEC ?= example.lf
GENERATED_DIR ?= generated

.PHONY: validate generate

validate:
	$(LANG_FORGE) validate --spec $(SPEC) $(LANG_FORGE_FLAGS)

generate: validate
	$(LANG_FORGE) generate --spec $(SPEC) --target $(TARGET) --out $(GENERATED_DIR) $(LANG_FORGE_FLAGS)

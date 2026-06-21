# Shared LangForge example generation targets.
#
# Example Makefiles set REPO_ROOT, SPEC, TARGET, and GENERATED_DIR before
# including this fragment. The fragment keeps validation and generation command
# lines identical across runnable demos and copyable templates.

REPO_ROOT ?= ../../..
GO ?= /usr/local/go/bin/go
LANG_FORGE ?= $(GO) run $(REPO_ROOT)/cmd/lang-forge
SPEC ?= example.lf
GENERATED_DIR ?= generated

.PHONY: validate generate

validate:
	$(LANG_FORGE) validate --spec $(SPEC)

generate: validate
	$(LANG_FORGE) generate --spec $(SPEC) --target $(TARGET) --out $(GENERATED_DIR)

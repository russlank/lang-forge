GO ?= /usr/local/go/bin/go
LANG_FORGE ?= $(GO) run ../../cmd/lang-forge
LANG_FORGE_VERBOSITY ?= 1
LANG_FORGE_FLAGS ?= --verbosity $(LANG_FORGE_VERBOSITY)

.PHONY: test

test:
	$(LANG_FORGE) validate --spec lr1-not-slr-lalr.lf $(LANG_FORGE_FLAGS)
	$(LANG_FORGE) validate --spec lr1-not-slr-ielr.lf $(LANG_FORGE_FLAGS)
	$(LANG_FORGE) validate --spec lr1-not-slr-canonical.lf $(LANG_FORGE_FLAGS)
	@set +e; \
	$(LANG_FORGE) validate --spec lr1-not-slr-slr.lf $(LANG_FORGE_FLAGS) >/tmp/langforge-lr1-not-slr.out 2>/tmp/langforge-lr1-not-slr.err; \
	code=$$?; \
	if [ $$code -eq 0 ]; then \
		cat /tmp/langforge-lr1-not-slr.out; \
		cat /tmp/langforge-lr1-not-slr.err; \
		echo "expected SLR fixture to report a parser conflict"; \
		exit 1; \
	fi; \
	if ! grep -q "conflict" /tmp/langforge-lr1-not-slr.err; then \
		cat /tmp/langforge-lr1-not-slr.out; \
		cat /tmp/langforge-lr1-not-slr.err; \
		echo "expected SLR fixture stderr to mention conflict, got exit $$code"; \
		exit 1; \
	fi; \
	echo "ok: SLR fixture reports expected conflict"
	$(LANG_FORGE) validate --spec mysterious-conflict-ielr.lf $(LANG_FORGE_FLAGS)
	$(LANG_FORGE) validate --spec mysterious-conflict-canonical.lf $(LANG_FORGE_FLAGS)
	@set +e; \
	$(LANG_FORGE) validate --spec mysterious-conflict-lalr.lf $(LANG_FORGE_FLAGS) >/tmp/langforge-mysterious-lalr.out 2>/tmp/langforge-mysterious-lalr.err; \
	code=$$?; \
	if [ $$code -eq 0 ]; then \
		cat /tmp/langforge-mysterious-lalr.out; \
		cat /tmp/langforge-mysterious-lalr.err; \
		echo "expected LALR mysterious-conflict fixture to report a parser conflict"; \
		exit 1; \
	fi; \
	if ! grep -q "conflict" /tmp/langforge-mysterious-lalr.err; then \
		cat /tmp/langforge-mysterious-lalr.out; \
		cat /tmp/langforge-mysterious-lalr.err; \
		echo "expected LALR mysterious-conflict stderr to mention conflict, got exit $$code"; \
		exit 1; \
	fi; \
	echo "ok: LALR mysterious-conflict fixture reports expected conflict"

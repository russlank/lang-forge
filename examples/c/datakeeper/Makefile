GO ?= /usr/local/go/bin/go
CC ?= cc
CFLAGS ?= -std=c11 -Wall -Wextra -O2
LANG_FORGE ?= $(GO) run ../../../cmd/lang-forge
GENERATED_DIR := generated
DIST_DIR := dist
BIN := $(DIST_DIR)/datakeeper-c-demo
INPUT ?= sample.dks
LOG ?= $(DIST_DIR)/datakeeper-c-demo.log

.PHONY: validate generate build run test clean

validate:
	$(LANG_FORGE) validate --spec datakeeper.lf

generate: validate
	$(LANG_FORGE) generate --spec datakeeper.lf --target c --out $(GENERATED_DIR)

build: generate
	@if ! command -v "$(CC)" >/dev/null 2>&1; then \
		echo "skip: C compiler '$(CC)' not found"; \
	else \
		mkdir -p $(DIST_DIR); \
		$(CC) $(CFLAGS) -I$(GENERATED_DIR) -I../common main.c ../common/demo.c $(GENERATED_DIR)/scanner.c $(GENERATED_DIR)/parser.c -o $(BIN); \
	fi

run: build
	@if ! command -v "$(CC)" >/dev/null 2>&1; then \
		echo "skip: C compiler '$(CC)' not found"; \
	else \
		./$(BIN) $(INPUT) --log $(LOG); \
	fi

test: build
	@if ! command -v "$(CC)" >/dev/null 2>&1; then \
		echo "skip: C compiler '$(CC)' not found"; \
	else \
		./$(BIN) --assert $(INPUT) --log $(LOG); \
	fi

clean:
	rm -rf $(GENERATED_DIR) $(DIST_DIR)

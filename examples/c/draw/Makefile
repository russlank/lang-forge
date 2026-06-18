GO ?= /usr/local/go/bin/go
CC ?= cc
CFLAGS ?= -std=c11 -Wall -Wextra -O2
LDLIBS ?= -lm
LANG_FORGE ?= $(GO) run ../../../cmd/lang-forge
GENERATED_DIR := generated
DIST_DIR := dist
BIN := $(DIST_DIR)/draw-c-demo
INPUT ?= sample.draw
OUTPUT ?= $(DIST_DIR)/sample-c.png
LOG ?= $(DIST_DIR)/draw-c-demo.log

.PHONY: validate generate build run test clean

validate:
	$(LANG_FORGE) validate --spec draw.lf

generate: validate
	$(LANG_FORGE) generate --spec draw.lf --target c --out $(GENERATED_DIR)

build: generate
	@if ! command -v "$(CC)" >/dev/null 2>&1; then \
		echo "skip: C compiler '$(CC)' not found"; \
	else \
		mkdir -p $(DIST_DIR); \
		$(CC) $(CFLAGS) -I$(GENERATED_DIR) -I../common main.c ../common/demo.c $(GENERATED_DIR)/scanner.c $(GENERATED_DIR)/parser.c -o $(BIN) $(LDLIBS); \
	fi

run: build
	@if ! command -v "$(CC)" >/dev/null 2>&1; then \
		echo "skip: C compiler '$(CC)' not found"; \
	else \
		./$(BIN) $(INPUT) --output $(OUTPUT) --log $(LOG); \
	fi

test: build
	@if ! command -v "$(CC)" >/dev/null 2>&1; then \
		echo "skip: C compiler '$(CC)' not found"; \
	else \
		./$(BIN) --assert $(INPUT) --output $(OUTPUT) --log $(LOG); \
	fi

clean:
	rm -rf $(GENERATED_DIR) $(DIST_DIR)

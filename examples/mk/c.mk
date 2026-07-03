# Shared targets for C LangForge examples.

LF_TARGET := c
GENERATED_DIR ?= generated
DIST_DIR ?= dist
CC ?= cc
CFLAGS ?= -std=c11 -Wall -Wextra -O2
INCLUDES ?= -I$(GENERATED_DIR) -I../common
SOURCES ?= main.c ../common/demo.c $(GENERATED_DIR)/scanner.c $(GENERATED_DIR)/parser.c
LDLIBS ?=
RUN_ARGS ?= $(INPUT) --log $(LOG)
TEST_ARGS ?= --assert $(INPUT) --log $(LOG)

include $(REPO_ROOT)/examples/mk/langforge.mk

.PHONY: build run test clean

build: generate
	@if ! command -v "$(CC)" >/dev/null 2>&1; then \
		echo "skip: C compiler '$(CC)' not found"; \
	else \
		mkdir -p $(DIST_DIR); \
		$(CC) $(CFLAGS) $(INCLUDES) $(SOURCES) -o $(BIN) $(LDLIBS); \
	fi

run: build
	@if ! command -v "$(CC)" >/dev/null 2>&1; then \
		echo "skip: C compiler '$(CC)' not found"; \
	else \
		./$(BIN) $(RUN_ARGS); \
	fi

test: build
	@if ! command -v "$(CC)" >/dev/null 2>&1; then \
		echo "skip: C compiler '$(CC)' not found"; \
	else \
		./$(BIN) $(TEST_ARGS); \
	fi

clean:
	rm -rf $(GENERATED_DIR) $(DIST_DIR)

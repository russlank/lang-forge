REPO_ROOT ?= ../../../..
SPEC := grammar.lf
LF_TARGET := c
GENERATED_DIR ?= generated
DIST_DIR ?= dist
BIN = $(DIST_DIR)/library-c-demo
TEST_BIN = $(DIST_DIR)/library-c-tests
INPUT ?= input.dsl
LOG ?= $(DIST_DIR)/library-c.log
COMMON_SOURCES = ast.c diagnostics.c semantics.c parser_facade.c $(GENERATED_DIR)/scanner.c $(GENERATED_DIR)/parser.c
SOURCES = main.c $(COMMON_SOURCES)
TEST_SOURCES = tests.c $(COMMON_SOURCES)
INCLUDES = -I. -I$(GENERATED_DIR)
CFLAGS ?= -std=c11 -Wall -Wextra -O2
CC ?= cc
RUN_ARGS ?= $(INPUT) --log $(LOG)

include ../../../mk/langforge.mk

.PHONY: build test-build run test clean

build: generate
	@if ! command -v "$(CC)" >/dev/null 2>&1; then \
		echo "skip: C compiler '$(CC)' not found"; \
	else \
		mkdir -p $(DIST_DIR); \
		$(CC) $(CFLAGS) $(INCLUDES) $(SOURCES) -o $(BIN); \
	fi

test-build: generate
	@if ! command -v "$(CC)" >/dev/null 2>&1; then \
		echo "skip: C compiler '$(CC)' not found"; \
	else \
		mkdir -p $(DIST_DIR); \
		$(CC) $(CFLAGS) $(INCLUDES) $(TEST_SOURCES) -o $(TEST_BIN); \
	fi

run: build
	@if ! command -v "$(CC)" >/dev/null 2>&1; then \
		echo "skip: C compiler '$(CC)' not found"; \
	else \
		./$(BIN) $(RUN_ARGS); \
	fi

test: test-build
	@if ! command -v "$(CC)" >/dev/null 2>&1; then \
		echo "skip: C compiler '$(CC)' not found"; \
	else \
		./$(TEST_BIN); \
	fi

clean:
	rm -rf $(GENERATED_DIR) $(DIST_DIR)

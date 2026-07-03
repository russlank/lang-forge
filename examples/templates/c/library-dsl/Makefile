REPO_ROOT ?= ../../../..
SPEC := grammar.lf
BIN = $(DIST_DIR)/library-c-demo
INPUT ?= input.dsl
LOG ?= $(DIST_DIR)/library-c.log
SOURCES = main.c ast.c diagnostics.c semantics.c parser_facade.c $(GENERATED_DIR)/scanner.c $(GENERATED_DIR)/parser.c
INCLUDES = -I. -I$(GENERATED_DIR)

include ../../../mk/c.mk

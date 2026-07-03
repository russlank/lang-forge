REPO_ROOT ?= ../../../..
SPEC := grammar.lf
GENERATED_DIR = src/generated
BIN = $(DIST_DIR)/library-cpp-demo
INPUT ?= input.dsl
LOG ?= $(DIST_DIR)/library-cpp.log
INCLUDES = -Iinclude -Isrc
SOURCES = src/main.cpp src/ast.cpp src/diagnostics.cpp src/semantics.cpp src/parser_facade.cpp $(GENERATED_DIR)/scanner.cpp $(GENERATED_DIR)/parser.cpp

include ../../../mk/cpp.mk

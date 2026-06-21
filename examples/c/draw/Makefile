REPO_ROOT ?= ../../..
SPEC := draw.lf
BIN = $(DIST_DIR)/draw-c-demo
INPUT ?= ../../testdata/draw/valid/basic.draw
OUTPUT ?= $(DIST_DIR)/sample-c.png
LOG ?= $(DIST_DIR)/draw-c-demo.log
SOURCES = main.c ast.c parser_adapter.c renderer.c report.c ../common/demo.c $(GENERATED_DIR)/scanner.c $(GENERATED_DIR)/parser.c
INCLUDES = -I. -I$(GENERATED_DIR) -I../common
LDLIBS = -lm
RUN_ARGS = $(INPUT) --output $(OUTPUT) --log $(LOG)
TEST_ARGS = --assert $(INPUT) --output $(OUTPUT) --log $(LOG)

include ../../mk/c.mk

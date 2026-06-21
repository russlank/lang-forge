REPO_ROOT ?= ../../..
SPEC := draw.lf
BIN = $(DIST_DIR)/draw-cpp-demo
INPUT ?= ../../testdata/draw/valid/basic.draw
OUTPUT ?= $(DIST_DIR)/sample-cpp.png
LOG ?= $(DIST_DIR)/draw-cpp-demo.log
SOURCES = main.cpp io.cpp parser_adapter.cpp renderer.cpp png_writer.cpp report.cpp $(GENERATED_DIR)/scanner.cpp $(GENERATED_DIR)/parser.cpp
INCLUDES = -I. -I$(GENERATED_DIR)
RUN_ARGS = $(INPUT) --output $(OUTPUT) --log $(LOG)
TEST_ARGS = --assert $(INPUT) --output $(OUTPUT) --log $(LOG)

include ../../mk/cpp.mk

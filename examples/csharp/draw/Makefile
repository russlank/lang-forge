REPO_ROOT ?= ../../..
SPEC := draw.lf
PROJECT := DrawCSharp.csproj
INPUT ?= ../../testdata/draw/valid/basic.draw
OUTPUT ?= $(DIST_DIR)/sample-csharp.png
LOG ?= $(DIST_DIR)/draw-csharp-demo.log
RUN_ARGS = $(INPUT) --output $(OUTPUT) --log $(LOG)
TEST_ARGS = --assert $(INPUT) --output $(OUTPUT) --log $(LOG)

include ../../mk/csharp.mk

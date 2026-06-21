REPO_ROOT ?= ../../..
SPEC := draw.lf
APP_NAME := draw-demo
BIN = $(DIST_DIR)/draw-demo
INPUT ?= ../../testdata/draw/valid/basic.draw
OUTPUT ?= $(DIST_DIR)/sample.png
LOG ?= $(DIST_DIR)/draw-demo.log
RUN_ARGS = --input $(INPUT) --output $(OUTPUT) --log $(LOG)

include ../../mk/go.mk

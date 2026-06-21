REPO_ROOT ?= ../../../..
SPEC := mini.lf
BIN = $(DIST_DIR)/mini-c-demo
INPUT ?= input.mini
LOG ?= $(DIST_DIR)/mini-c.log
SOURCES = main.c $(GENERATED_DIR)/scanner.c $(GENERATED_DIR)/parser.c
INCLUDES = -I$(GENERATED_DIR)

include ../../../mk/c.mk

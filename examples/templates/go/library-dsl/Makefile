REPO_ROOT ?= ../../../..
SPEC := grammar.lf
APP_NAME := library-dsl
GO_BUILD_PKG = ./cmd/library-dsl
INPUT ?= input.dsl
LOG ?= $(DIST_DIR)/library-go.log
RUN_ARGS = --input $(INPUT) --log $(LOG)

include ../../../mk/go.mk

REPO_ROOT ?= ../../../..
SPEC := mini.lf
APP_NAME := mini-compiler
GO_BUILD_PKG = ./cmd/mini-compiler
INPUT ?= input.mini
LOG ?= $(DIST_DIR)/mini-go.log
RUN_ARGS = --input $(INPUT) --log $(LOG)

include ../../../mk/go.mk

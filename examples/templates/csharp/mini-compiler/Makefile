REPO_ROOT ?= ../../../..
SPEC := mini.lf
PROJECT := MiniCompilerCSharp.csproj
INPUT ?= input.mini
LOG ?= $(DIST_DIR)/mini-csharp.log
RUN_ARGS = $(INPUT) --log $(LOG)
TEST_ARGS = --assert $(INPUT) --log $(LOG)

include ../../../mk/csharp.mk

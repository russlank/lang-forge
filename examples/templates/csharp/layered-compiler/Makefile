REPO_ROOT ?= ../../../..
SPEC := grammar.lf
PROJECT := LayeredCompilerCSharp.csproj
INPUT ?= input.mini
LOG ?= $(DIST_DIR)/layered-csharp.log
RUN_ARGS = $(INPUT) --log $(LOG)
TEST_ARGS = --assert $(INPUT) --log $(LOG)

include ../../../mk/csharp.mk

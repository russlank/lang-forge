REPO_ROOT ?= ../../../..
SPEC := grammar.lf
PROJECT := LibraryDslCSharp.csproj
INPUT ?= input.dsl
LOG ?= $(DIST_DIR)/library-csharp.log
RUN_ARGS = $(INPUT) --log $(LOG)
TEST_ARGS = --assert $(INPUT) --log $(LOG)

include ../../../mk/csharp.mk

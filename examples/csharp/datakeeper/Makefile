REPO_ROOT ?= ../../..
SPEC := datakeeper.lf
PROJECT := DataKeeperCSharp.csproj
INPUT ?= ../../testdata/datakeeper/valid/basic.dks
LOG ?= $(DIST_DIR)/datakeeper-csharp-demo.log
RUN_ARGS = $(INPUT) --log $(LOG)
TEST_ARGS = --assert $(INPUT) --log $(LOG)

include ../../mk/csharp.mk

REPO_ROOT ?= ../../..
SPEC := datakeeper.lf
APP_NAME := datakeeper-demo
BIN = $(DIST_DIR)/datakeeper-demo
INPUT ?= ../../testdata/datakeeper/valid/basic.dks
LOG ?= $(DIST_DIR)/datakeeper-demo.log
PARAMS ?= --param InstanceGuid=sql-instance-01 --param ParentGuid=farm-root --param ObjectName=daily-backup --param JobsTag=maintenance
RUN_ARGS = --script $(INPUT) $(PARAMS) --log $(LOG)

include ../../mk/go.mk

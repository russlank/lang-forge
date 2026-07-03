REPO_ROOT ?= ../../..
SPEC := recovery.lf
BIN = $(DIST_DIR)/parser-recovery-c-demo
INPUT ?= input.recovery
LOG ?= $(DIST_DIR)/parser-recovery-c-demo.log
RUN_ARGS = $(INPUT) --log $(LOG) --assert
TEST_ARGS = $(INPUT) --log $(LOG) --assert

include ../../mk/c.mk

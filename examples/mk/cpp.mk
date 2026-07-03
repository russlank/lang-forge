# Shared targets for C++ LangForge examples.

LF_TARGET := cpp
GENERATED_DIR ?= generated
DIST_DIR ?= dist
CXX ?= g++
CXXFLAGS ?= -std=c++17 -Wall -Wextra -O2
INCLUDES ?= -I$(GENERATED_DIR)
SOURCES ?= main.cpp $(GENERATED_DIR)/scanner.cpp $(GENERATED_DIR)/parser.cpp
LDLIBS ?=
RUN_ARGS ?= $(INPUT) --log $(LOG)
TEST_ARGS ?= --assert $(INPUT) --log $(LOG)

include $(REPO_ROOT)/examples/mk/langforge.mk

.PHONY: build run test clean

build: generate
	@if ! command -v "$(CXX)" >/dev/null 2>&1; then \
		echo "skip: C++ compiler '$(CXX)' not found"; \
	else \
		mkdir -p $(DIST_DIR); \
		$(CXX) $(CXXFLAGS) $(INCLUDES) $(SOURCES) -o $(BIN) $(LDLIBS); \
	fi

run: build
	@if ! command -v "$(CXX)" >/dev/null 2>&1; then \
		echo "skip: C++ compiler '$(CXX)' not found"; \
	else \
		./$(BIN) $(RUN_ARGS); \
	fi

test: build
	@if ! command -v "$(CXX)" >/dev/null 2>&1; then \
		echo "skip: C++ compiler '$(CXX)' not found"; \
	else \
		./$(BIN) $(TEST_ARGS); \
	fi

clean:
	rm -rf $(GENERATED_DIR) $(DIST_DIR)

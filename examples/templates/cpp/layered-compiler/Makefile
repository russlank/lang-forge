REPO_ROOT ?= ../../../..
SPEC := grammar.lf
GENERATED_DIR = generated
BIN = $(DIST_DIR)/layered-cpp-demo
INPUT ?= input.mini
LOG ?= $(DIST_DIR)/layered-cpp.log
INCLUDES = -Iinclude -I.
SOURCES = src/main.cpp src/parser.cpp src/compiler.cpp src/diagnostics.cpp $(GENERATED_DIR)/scanner.cpp $(GENERATED_DIR)/parser.cpp
CMAKE_BUILD_DIR ?= $(DIST_DIR)/cmake-build
CMAKE_LANG_FORGE ?= $(abspath $(REPO_ROOT))/dist/lang-forge

include ../../../mk/cpp.mk

.PHONY: cmake-test

cmake-test:
	@if ! command -v cmake >/dev/null 2>&1; then \
		echo "skip: cmake not found"; \
	else \
		$(MAKE) -C $(REPO_ROOT) GO=$(GO) build; \
		cmake -S . -B $(CMAKE_BUILD_DIR) -DLANG_FORGE=$(CMAKE_LANG_FORGE); \
		cmake --build $(CMAKE_BUILD_DIR); \
		ctest --test-dir $(CMAKE_BUILD_DIR) --output-on-failure; \
	fi

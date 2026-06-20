GO ?= /usr/local/go/bin/go
CXX ?= g++
CXXFLAGS ?= -std=c++17 -Wall -Wextra -O2
LANG_FORGE ?= $(GO) run ../../../cmd/lang-forge
GENERATED_DIR := generated
DIST_DIR := dist
BIN := $(DIST_DIR)/draw-cpp-demo
INPUT ?= sample.draw
OUTPUT ?= $(DIST_DIR)/sample-cpp.png
LOG ?= $(DIST_DIR)/draw-cpp-demo.log
SRCS := main.cpp io.cpp parser_adapter.cpp renderer.cpp png_writer.cpp report.cpp $(GENERATED_DIR)/scanner.cpp $(GENERATED_DIR)/parser.cpp

.PHONY: validate generate build run test clean

validate:
	$(LANG_FORGE) validate --spec draw.lf

generate: validate
	$(LANG_FORGE) generate --spec draw.lf --target cpp --out $(GENERATED_DIR)

build: generate
	@if ! command -v "$(CXX)" >/dev/null 2>&1; then \
		echo "skip: C++ compiler '$(CXX)' not found"; \
	else \
		mkdir -p $(DIST_DIR); \
		$(CXX) $(CXXFLAGS) -I. -I$(GENERATED_DIR) $(SRCS) -o $(BIN); \
	fi

run: build
	@if ! command -v "$(CXX)" >/dev/null 2>&1; then \
		echo "skip: C++ compiler '$(CXX)' not found"; \
	else \
		./$(BIN) $(INPUT) --output $(OUTPUT) --log $(LOG); \
	fi

test: build
	@if ! command -v "$(CXX)" >/dev/null 2>&1; then \
		echo "skip: C++ compiler '$(CXX)' not found"; \
	else \
		./$(BIN) --assert $(INPUT) --output $(OUTPUT) --log $(LOG); \
	fi

clean:
	rm -rf $(GENERATED_DIR) $(DIST_DIR)

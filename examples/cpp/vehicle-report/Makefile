GO ?= /usr/local/go/bin/go
CXX ?= g++
CXXFLAGS ?= -std=c++17 -Wall -Wextra -O2
LANG_FORGE ?= $(GO) run ../../../cmd/lang-forge
GENERATED_DIR := generated
DIST_DIR := dist
BIN := $(DIST_DIR)/vehicle-report-cpp-demo
INPUT ?= sample.vehicle
LOG ?= $(DIST_DIR)/vehicle-report-cpp-demo.log

.PHONY: validate generate build run test clean

validate:
	$(LANG_FORGE) validate --spec vehicle.lf

generate: validate
	$(LANG_FORGE) generate --spec vehicle.lf --target cpp --out $(GENERATED_DIR)

build: generate
	@if ! command -v "$(CXX)" >/dev/null 2>&1; then \
		echo "skip: C++ compiler '$(CXX)' not found"; \
	else \
		mkdir -p $(DIST_DIR); \
		$(CXX) $(CXXFLAGS) -I$(GENERATED_DIR) main.cpp $(GENERATED_DIR)/scanner.cpp $(GENERATED_DIR)/parser.cpp -o $(BIN); \
	fi

run: build
	@if ! command -v "$(CXX)" >/dev/null 2>&1; then \
		echo "skip: C++ compiler '$(CXX)' not found"; \
	else \
		./$(BIN) $(INPUT) --log $(LOG); \
	fi

test: build
	@if ! command -v "$(CXX)" >/dev/null 2>&1; then \
		echo "skip: C++ compiler '$(CXX)' not found"; \
	else \
		./$(BIN) --assert $(INPUT) --log $(LOG); \
	fi

clean:
	rm -rf $(GENERATED_DIR) $(DIST_DIR)

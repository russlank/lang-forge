GO ?= /usr/local/go/bin/go
DOTNET ?= dotnet
LANG_FORGE ?= $(GO) run ../../../cmd/lang-forge
GENERATED_DIR := Generated
DIST_DIR := dist
INPUT ?= sample.draw
OUTPUT ?= $(DIST_DIR)/sample-csharp.png
LOG ?= $(DIST_DIR)/draw-csharp-demo.log

.PHONY: validate generate build run test clean

validate:
	$(LANG_FORGE) validate --spec draw.lf

generate: validate
	$(LANG_FORGE) generate --spec draw.lf --target csharp --out $(GENERATED_DIR)

build: generate
	$(DOTNET) build DrawCSharp.csproj

run: generate
	$(DOTNET) run --project DrawCSharp.csproj -- $(INPUT) --output $(OUTPUT) --log $(LOG)

test: generate
	$(DOTNET) run --project DrawCSharp.csproj -- --assert $(INPUT) --output $(OUTPUT) --log $(LOG)

clean:
	rm -rf $(GENERATED_DIR) $(DIST_DIR) bin obj

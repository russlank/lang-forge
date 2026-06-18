GO ?= /usr/local/go/bin/go
DOTNET ?= dotnet
LANG_FORGE ?= $(GO) run ../../../cmd/lang-forge
GENERATED_DIR := Generated
DIST_DIR := dist
INPUT ?= sample.dks
LOG ?= $(DIST_DIR)/datakeeper-csharp-demo.log

.PHONY: validate generate build run test clean

validate:
	$(LANG_FORGE) validate --spec datakeeper.lf

generate: validate
	$(LANG_FORGE) generate --spec datakeeper.lf --target csharp --out $(GENERATED_DIR)

build: generate
	$(DOTNET) build DataKeeperCSharp.csproj

run: generate
	$(DOTNET) run --project DataKeeperCSharp.csproj -- $(INPUT) --log $(LOG)

test: generate
	$(DOTNET) run --project DataKeeperCSharp.csproj -- --assert $(INPUT) --log $(LOG)

clean:
	rm -rf $(GENERATED_DIR) $(DIST_DIR) bin obj

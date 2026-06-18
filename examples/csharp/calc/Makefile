GO ?= /usr/local/go/bin/go
DOTNET ?= dotnet
LANG_FORGE ?= $(GO) run ../../../cmd/lang-forge
GENERATED_DIR := Generated

.PHONY: validate generate build run test clean

validate:
	$(LANG_FORGE) validate --spec calc.lf

generate: validate
	$(LANG_FORGE) generate --spec calc.lf --target csharp --out $(GENERATED_DIR)

build: generate
	$(DOTNET) build CalcCSharp.csproj --nologo

run: generate
	$(DOTNET) run --project CalcCSharp.csproj -- input.calc

test: generate
	$(DOTNET) run --project CalcCSharp.csproj -- --assert input.calc

clean:
	rm -rf $(GENERATED_DIR) dist bin obj

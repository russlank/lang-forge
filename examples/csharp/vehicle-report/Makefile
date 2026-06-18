GO ?= /usr/local/go/bin/go
DOTNET ?= dotnet
LANG_FORGE ?= $(GO) run ../../../cmd/lang-forge
GENERATED_DIR := Generated
DIST_DIR := dist
INPUT ?= sample.vehicle
LOG ?= $(DIST_DIR)/vehicle-report-csharp-demo.log

.PHONY: validate generate build run test clean

validate:
	$(LANG_FORGE) validate --spec vehicle.lf

generate: validate
	$(LANG_FORGE) generate --spec vehicle.lf --target csharp --out $(GENERATED_DIR)

build: generate
	$(DOTNET) build VehicleReportCSharp.csproj

run: generate
	$(DOTNET) run --project VehicleReportCSharp.csproj -- $(INPUT) --log $(LOG)

test: generate
	$(DOTNET) run --project VehicleReportCSharp.csproj -- --assert $(INPUT) --log $(LOG)

clean:
	rm -rf $(GENERATED_DIR) $(DIST_DIR) bin obj

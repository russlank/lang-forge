REPO_ROOT ?= ../../..
SPEC := vehicle.lf
PROJECT := VehicleReportCSharp.csproj
INPUT ?= ../../testdata/vehicle-report/valid/basic.vehicle
LOG ?= $(DIST_DIR)/vehicle-report-csharp-demo.log
RUN_ARGS = $(INPUT) --log $(LOG)
TEST_ARGS = --assert $(INPUT) --log $(LOG)

include ../../mk/csharp.mk

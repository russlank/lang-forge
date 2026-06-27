//go:build langforge_generated

package vehiclereport

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestParseSampleVehicle(t *testing.T) {
	data, err := os.ReadFile("sample.vehicle")
	if err != nil {
		t.Fatal(err)
	}
	vehicle, err := Parse(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if vehicle.Model != "KIA" || vehicle.License != "198783-damascus" || vehicle.Distance != 10000 {
		t.Fatalf("unexpected vehicle: %#v", vehicle)
	}
	if len(vehicle.Features) != 4 {
		t.Fatalf("features len = %d", len(vehicle.Features))
	}
	if len(vehicle.Repairs) != 3 {
		t.Fatalf("repairs len = %d", len(vehicle.Repairs))
	}
}

func TestReducerCoverageAndTypedActionManifest(t *testing.T) {
	if err := vehicleReducers.ValidateCoverage(); err != nil {
		t.Fatalf("vehicle reducer coverage: %v", err)
	}
	manifest, err := os.ReadFile("generated/langforge.actions.json")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(manifest, []byte(`"typed": false`)) || bytes.Contains(manifest, []byte(`"typeIssue"`)) {
		t.Fatalf("vehicle action manifest contains an untyped action:\n%s", manifest)
	}
	for _, want := range [][]byte{
		[]byte(`"label": "info"`),
		[]byte(`"label": "model"`),
		[]byte(`"label": "features"`),
		[]byte(`"label": "description"`),
		[]byte(`"returnType": "*vehiclemodel.Vehicle"`),
	} {
		if !bytes.Contains(manifest, want) {
			t.Fatalf("vehicle action manifest missing %s:\n%s", want, manifest)
		}
	}
}

func TestParseAcceptsEmptyListsAndLegacyReparationsName(t *testing.T) {
	vehicle, err := Parse(`car = {
  model = "KIA",
  license = "198783-damascus",
  distance = 10000,
  features = {},
  reparations = {}
}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(vehicle.Features) != 0 || len(vehicle.Repairs) != 0 {
		t.Fatalf("lists = features %d repairs %d", len(vehicle.Features), len(vehicle.Repairs))
	}
}

func TestParseRejectsMalformedInput(t *testing.T) {
	_, err := Parse(`car = {
  model = "KIA",
  license = "198783-damascus",
  distance = 10000
  features = {},
  repairs = {}
}`)
	if err == nil {
		t.Fatal("expected parse error for missing comma")
	}
}

func TestWriteReport(t *testing.T) {
	vehicle := &Vehicle{
		Model:    "KIA",
		License:  "198783-damascus",
		Distance: 10000,
		Features: []Feature{{Name: "color", Value: "Red"}},
		Repairs:  []Repair{{Date: "21-05-2009", Description: "first service"}},
	}
	var report bytes.Buffer
	WriteReport(&report, "sample.vehicle", vehicle)
	text := report.String()
	for _, want := range []string{"Vehicle Report", "Model: KIA", "<color>Red</color>", "<date>21-05-2009</date>"} {
		if !strings.Contains(text, want) {
			t.Fatalf("report missing %q:\n%s", want, text)
		}
	}
}

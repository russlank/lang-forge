package vehiclereport

import vehiclemodel "github.com/russlank/lang-forge/examples/go/vehicle-report/model"

// The public package keeps the original type names as aliases while generated
// typed reducer contexts and handwritten report code share the cycle-free model
// package.
type (
	Vehicle = vehiclemodel.Vehicle
	Feature = vehiclemodel.Feature
	Repair  = vehiclemodel.Repair

	vehicleInfo = vehiclemodel.Info
)

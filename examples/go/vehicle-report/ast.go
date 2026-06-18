package vehiclereport

// Vehicle is the parsed vehicle description.
type Vehicle struct {
	Model    string
	License  string
	Distance int
	Features []Feature
	Repairs  []Repair
}

// Feature is one named vehicle feature.
type Feature struct {
	Name  string
	Value string
}

// Repair is one repair or service event.
type Repair struct {
	Date        string
	Description string
}

type vehicleInfo struct {
	model    string
	license  string
	distance int
	features []Feature
	repairs  []Repair
}

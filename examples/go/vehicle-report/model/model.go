// Package model defines the vehicle-report data model shared by generated
// typed reducer contexts and handwritten reporting code.
//
// Keeping these types in a dependency-only package prevents an import cycle:
// generated parser code can reference model types, while the public
// vehicle-report package imports the generated parser.
package model

// Vehicle is the parsed vehicle description.
type Vehicle struct {
	Model    string
	License  string
	Distance int
	Features []Feature
	Repairs  []Repair
}

// Info is the normalized set of fields gathered before creating a Vehicle.
type Info struct {
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

package vehiclereport

import (
	"fmt"
	"io"
)

// WriteReport writes a readable report plus XML-like output for vehicle.
func WriteReport(w io.Writer, source string, vehicle *Vehicle) {
	fmt.Fprintf(w, "Vehicle Report\n")
	fmt.Fprintf(w, "Source: %s\n", source)
	if vehicle == nil {
		fmt.Fprintln(w, "Status: no vehicle parsed")
		return
	}
	fmt.Fprintf(w, "Model: %s\n", vehicle.Model)
	fmt.Fprintf(w, "License: %s\n", vehicle.License)
	fmt.Fprintf(w, "Distance: %d km\n", vehicle.Distance)
	fmt.Fprintf(w, "Features: %d\n", len(vehicle.Features))
	for _, feature := range vehicle.Features {
		fmt.Fprintf(w, "  - %s = %s\n", feature.Name, feature.Value)
	}
	fmt.Fprintf(w, "Repairs: %d\n", len(vehicle.Repairs))
	for _, repair := range vehicle.Repairs {
		fmt.Fprintf(w, "  - %s: %s\n", repair.Date, repair.Description)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "<car>")
	fmt.Fprintf(w, "  <model>%s</model>\n", vehicle.Model)
	fmt.Fprintf(w, "  <license>%s</license>\n", vehicle.License)
	fmt.Fprintf(w, "  <distance>%d</distance>\n", vehicle.Distance)
	fmt.Fprintln(w, "  <features>")
	for _, feature := range vehicle.Features {
		fmt.Fprintf(w, "    <%s>%s</%s>\n", feature.Name, feature.Value, feature.Name)
	}
	fmt.Fprintln(w, "  </features>")
	fmt.Fprintln(w, "  <repairs>")
	for _, repair := range vehicle.Repairs {
		fmt.Fprintln(w, "    <repair>")
		fmt.Fprintf(w, "      <date>%s</date>\n", repair.Date)
		fmt.Fprintf(w, "      <description>%s</description>\n", repair.Description)
		fmt.Fprintln(w, "    </repair>")
	}
	fmt.Fprintln(w, "  </repairs>")
	fmt.Fprintln(w, "</car>")
}

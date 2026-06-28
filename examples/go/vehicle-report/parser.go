//go:build langforge_generated

package vehiclereport

import (
	"fmt"
	"strconv"
	"strings"

	vehiclegen "github.com/russlank/lang-forge/examples/go/vehicle-report/generated"
)

// Parse converts vehicle-report source text into a Vehicle AST.
//
// The grammar declares named RHS labels and semantic result types. LangForge
// uses those declarations to generate typed reducer contexts such as
// VehicleReduction and FeatureReduction, keeping this adapter free from
// positional semantic-value casts.
func Parse(source string) (*Vehicle, error) {
	lexemes, err := vehiclegen.Tokenize(source)
	if err != nil {
		return nil, err
	}
	value, err := vehiclegen.ParseWithReducer(lexemes, vehicleReducers)
	if err != nil {
		return nil, err
	}
	vehicle, ok := value.(*Vehicle)
	if !ok {
		return nil, fmt.Errorf("generated parser returned %T, want *Vehicle", value)
	}
	return vehicle, nil
}

// vehicleReducers maps `{go: ...}` action labels from vehicle.lf to ordinary
// Go functions. The generated typed adapters check the reduction shape once,
// then the reducer body can work with named fields such as ctx.Description.
var vehicleReducers = vehiclegen.ReducerMap{
	vehiclegen.SemanticActionVehicle:          vehiclegen.TypedVehicle(reduceVehicle),
	vehiclegen.SemanticActionInfo:             vehiclegen.TypedInfo(reduceInfo),
	vehiclegen.SemanticActionFieldModel:       vehiclegen.TypedFieldModel(reduceModelField),
	vehiclegen.SemanticActionFieldLicense:     vehiclegen.TypedFieldLicense(reduceLicenseField),
	vehiclegen.SemanticActionFieldDistance:    vehiclegen.TypedFieldDistance(reduceDistanceField),
	vehiclegen.SemanticActionFieldFeatures:    vehiclegen.TypedFieldFeatures(reduceFeaturesField),
	vehiclegen.SemanticActionFeatureItems:     vehiclegen.TypedFeatureItems(reduceFeatureItems),
	vehiclegen.SemanticActionFeatureEmpty:     vehiclegen.TypedFeatureEmpty(reduceFeatureEmpty),
	vehiclegen.SemanticActionFeatureTailMore:  vehiclegen.TypedFeatureTailMore(reduceFeatureTailMore),
	vehiclegen.SemanticActionFeatureTailEmpty: vehiclegen.TypedFeatureTailEmpty(reduceFeatureTailEmpty),
	vehiclegen.SemanticActionFeature:          vehiclegen.TypedFeature(reduceFeature),
	vehiclegen.SemanticActionFieldRepairs:     vehiclegen.TypedFieldRepairs(reduceRepairsField),
	vehiclegen.SemanticActionRepairItems:      vehiclegen.TypedRepairItems(reduceRepairItems),
	vehiclegen.SemanticActionRepairEmpty:      vehiclegen.TypedRepairEmpty(reduceRepairEmpty),
	vehiclegen.SemanticActionRepairTailMore:   vehiclegen.TypedRepairTailMore(reduceRepairTailMore),
	vehiclegen.SemanticActionRepairTailEmpty:  vehiclegen.TypedRepairTailEmpty(reduceRepairTailEmpty),
	vehiclegen.SemanticActionRepair:           vehiclegen.TypedRepair(reduceRepair),
}

func reduceVehicle(ctx vehiclegen.VehicleReduction) (*Vehicle, error) {
	info := ctx.Info
	return &Vehicle{
		Model:    info.Model,
		License:  info.License,
		Distance: info.Distance,
		Features: info.Features,
		Repairs:  info.Repairs,
	}, nil
}

func reduceInfo(ctx vehiclegen.InfoReduction) (vehicleInfo, error) {
	return vehicleInfo{
		Model:    ctx.Model,
		License:  ctx.License,
		Distance: ctx.Distance,
		Features: ctx.Features,
		Repairs:  ctx.Repairs,
	}, nil
}

func reduceModelField(ctx vehiclegen.FieldModelReduction) (string, error) {
	return decodeQuoted(ctx.Literal.Text), nil
}

func reduceLicenseField(ctx vehiclegen.FieldLicenseReduction) (string, error) {
	return decodeQuoted(ctx.Literal.Text), nil
}

func reduceDistanceField(ctx vehiclegen.FieldDistanceReduction) (int, error) {
	value, err := strconv.Atoi(ctx.Literal.Text)
	if err != nil {
		return 0, fmt.Errorf("rule %d invalid distance %q: %w", ctx.Reduction.Rule, ctx.Literal.Text, err)
	}
	return value, nil
}

func reduceFeaturesField(ctx vehiclegen.FieldFeaturesReduction) ([]Feature, error) {
	return ctx.Items, nil
}

func reduceFeatureItems(ctx vehiclegen.FeatureItemsReduction) ([]Feature, error) {
	return prependFeature(ctx.Head, ctx.Tail), nil
}

func reduceFeatureEmpty(vehiclegen.FeatureEmptyReduction) ([]Feature, error) {
	return []Feature{}, nil
}

func reduceFeatureTailMore(ctx vehiclegen.FeatureTailMoreReduction) ([]Feature, error) {
	return prependFeature(ctx.Head, ctx.Tail), nil
}

func reduceFeatureTailEmpty(vehiclegen.FeatureTailEmptyReduction) ([]Feature, error) {
	return []Feature{}, nil
}

func reduceFeature(ctx vehiclegen.FeatureReduction) (Feature, error) {
	return Feature{Name: ctx.Name.Text, Value: decodeQuoted(ctx.Value.Text)}, nil
}

func reduceRepairsField(ctx vehiclegen.FieldRepairsReduction) ([]Repair, error) {
	return ctx.Items, nil
}

func reduceRepairItems(ctx vehiclegen.RepairItemsReduction) ([]Repair, error) {
	return prependRepair(ctx.Head, ctx.Tail), nil
}

func reduceRepairEmpty(vehiclegen.RepairEmptyReduction) ([]Repair, error) {
	return []Repair{}, nil
}

func reduceRepairTailMore(ctx vehiclegen.RepairTailMoreReduction) ([]Repair, error) {
	return prependRepair(ctx.Head, ctx.Tail), nil
}

func reduceRepairTailEmpty(vehiclegen.RepairTailEmptyReduction) ([]Repair, error) {
	return []Repair{}, nil
}

func reduceRepair(ctx vehiclegen.RepairReduction) (Repair, error) {
	return Repair{
		Date:        decodeQuoted(ctx.Date.Text),
		Description: decodeQuoted(ctx.Description.Text),
	}, nil
}

func prependFeature(head Feature, tail []Feature) []Feature {
	return append([]Feature{head}, tail...)
}

func prependRepair(head Repair, tail []Repair) []Repair {
	return append([]Repair{head}, tail...)
}

// decodeQuoted is intentionally semantic code, not lexer code: the token tells
// us a quoted literal was recognized, and the reducer decides how that literal
// becomes a report field.
func decodeQuoted(text string) string {
	return strings.TrimSuffix(strings.TrimPrefix(text, "\""), "\"")
}

//go:build langforge_generated

package vehiclereport

import (
	"fmt"
	"strconv"
	"strings"

	vehiclegen "github.com/russlank/lang-forge/examples/go/vehicle-report/generated"
)

// Parse converts vehicle-report source text into a Vehicle AST.
func Parse(source string) (*Vehicle, error) {
	lexemes, err := vehiclegen.Tokenize(source)
	if err != nil {
		return nil, err
	}
	value, err := vehiclegen.ParseWithReducer(lexemes, vehiclegen.ReducerFunc(vehicleReduce))
	if err != nil {
		return nil, err
	}
	vehicle, ok := value.(*Vehicle)
	if !ok {
		return nil, fmt.Errorf("generated parser returned %T, want *Vehicle", value)
	}
	return vehicle, nil
}

var vehicleReducers = vehiclegen.ReducerMap{
	vehiclegen.SemanticActionVehicle:          reduceVehicle,
	vehiclegen.SemanticActionInfo:             reduceInfo,
	vehiclegen.SemanticActionFieldModel:       reduceModelField,
	vehiclegen.SemanticActionFieldLicense:     reduceLicenseField,
	vehiclegen.SemanticActionFieldDistance:    reduceDistanceField,
	vehiclegen.SemanticActionFieldFeatures:    reduceFeaturesField,
	vehiclegen.SemanticActionFeatureItems:     reduceFeatureItems,
	vehiclegen.SemanticActionFeatureEmpty:     reduceEmptyFeatures,
	vehiclegen.SemanticActionFeatureTailMore:  reduceFeatureTailMore,
	vehiclegen.SemanticActionFeatureTailEmpty: reduceEmptyFeatures,
	vehiclegen.SemanticActionFeature:          reduceFeature,
	vehiclegen.SemanticActionFieldRepairs:     reduceRepairsField,
	vehiclegen.SemanticActionRepairItems:      reduceRepairItems,
	vehiclegen.SemanticActionRepairEmpty:      reduceEmptyRepairs,
	vehiclegen.SemanticActionRepairTailMore:   reduceRepairTailMore,
	vehiclegen.SemanticActionRepairTailEmpty:  reduceEmptyRepairs,
	vehiclegen.SemanticActionRepair:           reduceRepair,
}

func vehicleReduce(ctx vehiclegen.Reduction) (vehiclegen.Value, error) {
	return vehicleReducers.Reduce(ctx)
}

func reduceVehicle(ctx vehiclegen.Reduction) (vehiclegen.Value, error) {
	info, err := infoArg(ctx, 3)
	if err != nil {
		return nil, err
	}
	return &Vehicle{
		Model:    info.model,
		License:  info.license,
		Distance: info.distance,
		Features: info.features,
		Repairs:  info.repairs,
	}, nil
}

func reduceInfo(ctx vehiclegen.Reduction) (vehiclegen.Value, error) {
	model, err := stringArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	license, err := stringArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	distance, err := intArg(ctx, 4)
	if err != nil {
		return nil, err
	}
	features, err := featureSliceArg(ctx, 6)
	if err != nil {
		return nil, err
	}
	repairs, err := repairSliceArg(ctx, 8)
	if err != nil {
		return nil, err
	}
	return vehicleInfo{model: model, license: license, distance: distance, features: features, repairs: repairs}, nil
}

func reduceModelField(ctx vehiclegen.Reduction) (vehiclegen.Value, error) {
	text, err := lexemeTextArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	return decodeQuoted(text), nil
}

func reduceLicenseField(ctx vehiclegen.Reduction) (vehiclegen.Value, error) {
	text, err := lexemeTextArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	return decodeQuoted(text), nil
}

func reduceDistanceField(ctx vehiclegen.Reduction) (vehiclegen.Value, error) {
	text, err := lexemeTextArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	value, err := strconv.Atoi(text)
	if err != nil {
		return nil, fmt.Errorf("invalid distance %q: %w", text, err)
	}
	return value, nil
}

func reduceFeaturesField(ctx vehiclegen.Reduction) (vehiclegen.Value, error) {
	return featureSliceArg(ctx, 3)
}

func reduceFeatureItems(ctx vehiclegen.Reduction) (vehiclegen.Value, error) {
	feature, err := featureArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	tail, err := featureSliceArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	return append([]Feature{feature}, tail...), nil
}

func reduceFeatureTailMore(ctx vehiclegen.Reduction) (vehiclegen.Value, error) {
	feature, err := featureArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	tail, err := featureSliceArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	return append([]Feature{feature}, tail...), nil
}

func reduceEmptyFeatures(vehiclegen.Reduction) (vehiclegen.Value, error) {
	return []Feature{}, nil
}

func reduceFeature(ctx vehiclegen.Reduction) (vehiclegen.Value, error) {
	name, err := lexemeTextArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	value, err := lexemeTextArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	return Feature{Name: name, Value: decodeQuoted(value)}, nil
}

func reduceRepairsField(ctx vehiclegen.Reduction) (vehiclegen.Value, error) {
	return repairSliceArg(ctx, 3)
}

func reduceRepairItems(ctx vehiclegen.Reduction) (vehiclegen.Value, error) {
	repair, err := repairArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	tail, err := repairSliceArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	return append([]Repair{repair}, tail...), nil
}

func reduceRepairTailMore(ctx vehiclegen.Reduction) (vehiclegen.Value, error) {
	repair, err := repairArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	tail, err := repairSliceArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	return append([]Repair{repair}, tail...), nil
}

func reduceEmptyRepairs(vehiclegen.Reduction) (vehiclegen.Value, error) {
	return []Repair{}, nil
}

func reduceRepair(ctx vehiclegen.Reduction) (vehiclegen.Value, error) {
	date, err := lexemeTextArg(ctx, 3)
	if err != nil {
		return nil, err
	}
	description, err := lexemeTextArg(ctx, 7)
	if err != nil {
		return nil, err
	}
	return Repair{Date: decodeQuoted(date), Description: decodeQuoted(description)}, nil
}

func valueArg(ctx vehiclegen.Reduction, index int) (vehiclegen.Value, error) {
	if index < 0 || index >= len(ctx.Values) {
		return nil, fmt.Errorf("rule %d action %q missing argument %d", ctx.Rule, ctx.Action, index+1)
	}
	return ctx.Values[index], nil
}

func lexemeTextArg(ctx vehiclegen.Reduction, index int) (string, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return "", err
	}
	lexeme, ok := value.(vehiclegen.Lexeme)
	if !ok {
		return "", fmt.Errorf("rule %d action %q argument %d has type %T, want Lexeme", ctx.Rule, ctx.Action, index+1, value)
	}
	return lexeme.Text, nil
}

func stringArg(ctx vehiclegen.Reduction, index int) (string, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return "", err
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("rule %d action %q argument %d has type %T, want string", ctx.Rule, ctx.Action, index+1, value)
	}
	return text, nil
}

func intArg(ctx vehiclegen.Reduction, index int) (int, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return 0, err
	}
	number, ok := value.(int)
	if !ok {
		return 0, fmt.Errorf("rule %d action %q argument %d has type %T, want int", ctx.Rule, ctx.Action, index+1, value)
	}
	return number, nil
}

func infoArg(ctx vehiclegen.Reduction, index int) (vehicleInfo, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return vehicleInfo{}, err
	}
	info, ok := value.(vehicleInfo)
	if !ok {
		return vehicleInfo{}, fmt.Errorf("rule %d action %q argument %d has type %T, want vehicleInfo", ctx.Rule, ctx.Action, index+1, value)
	}
	return info, nil
}

func featureArg(ctx vehiclegen.Reduction, index int) (Feature, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return Feature{}, err
	}
	feature, ok := value.(Feature)
	if !ok {
		return Feature{}, fmt.Errorf("rule %d action %q argument %d has type %T, want Feature", ctx.Rule, ctx.Action, index+1, value)
	}
	return feature, nil
}

func featureSliceArg(ctx vehiclegen.Reduction, index int) ([]Feature, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return nil, err
	}
	features, ok := value.([]Feature)
	if !ok {
		return nil, fmt.Errorf("rule %d action %q argument %d has type %T, want []Feature", ctx.Rule, ctx.Action, index+1, value)
	}
	return features, nil
}

func repairArg(ctx vehiclegen.Reduction, index int) (Repair, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return Repair{}, err
	}
	repair, ok := value.(Repair)
	if !ok {
		return Repair{}, fmt.Errorf("rule %d action %q argument %d has type %T, want Repair", ctx.Rule, ctx.Action, index+1, value)
	}
	return repair, nil
}

func repairSliceArg(ctx vehiclegen.Reduction, index int) ([]Repair, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return nil, err
	}
	repairs, ok := value.([]Repair)
	if !ok {
		return nil, fmt.Errorf("rule %d action %q argument %d has type %T, want []Repair", ctx.Rule, ctx.Action, index+1, value)
	}
	return repairs, nil
}

func decodeQuoted(text string) string {
	return strings.TrimSuffix(strings.TrimPrefix(text, "\""), "\"")
}

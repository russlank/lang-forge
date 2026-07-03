package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/russlank/lang-forge/internal/action"
	"github.com/russlank/lang-forge/internal/diagnostics"
	"github.com/russlank/lang-forge/internal/lex"
	"github.com/russlank/lang-forge/internal/parse"
	"github.com/russlank/lang-forge/internal/spec"
)

type exampleFamily struct {
	Name  string
	Specs []targetSpec
}

type targetSpec struct {
	Target string
	Path   string
}

type allowlist struct {
	Allow []allowedDifference `json:"allow"`
}

type allowedDifference struct {
	Family string `json:"family"`
	Target string `json:"target"`
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type manifestContract struct {
	Actions  []actionContract `json:"actions,omitempty"`
	Recovery recoveryContract `json:"recovery"`
	Expected expectedContract `json:"expected"`
}

type actionContract struct {
	ID                int            `json:"id"`
	Name              string         `json:"name"`
	Typed             bool           `json:"typed"`
	TypeIssue         string         `json:"typeIssue,omitempty"`
	ReturnType        string         `json:"returnType"`
	Rules             []ruleContract `json:"rules"`
	ConsistentContext bool           `json:"consistentContext"`
}

type ruleContract struct {
	ID         int               `json:"id"`
	LHS        string            `json:"lhs"`
	ReturnType string            `json:"returnType"`
	RHS        []operandContract `json:"rhs,omitempty"`
	Typed      bool              `json:"typed"`
	TypeIssue  string            `json:"typeIssue,omitempty"`
}

type operandContract struct {
	Position int    `json:"position"`
	Symbol   string `json:"symbol"`
	Label    string `json:"label,omitempty"`
	Type     string `json:"type"`
}

type recoveryContract struct {
	Enabled     bool                 `json:"enabled"`
	Productions []recoveryProduction `json:"productions,omitempty"`
}

type recoveryProduction struct {
	ID     int      `json:"id"`
	LHS    string   `json:"lhs"`
	RHS    []string `json:"rhs"`
	Labels []string `json:"labels,omitempty"`
}

type expectedContract struct {
	Aliases []expectedAlias `json:"aliases,omitempty"`
	Groups  []expectedGroup `json:"groups,omitempty"`
	Hidden  []string        `json:"hidden,omitempty"`
}

type expectedAlias struct {
	Token string `json:"token"`
	Label string `json:"label"`
}

type expectedGroup struct {
	Name   string   `json:"name"`
	Tokens []string `json:"tokens"`
}

type contractDiff struct {
	Path string
	Want string
	Got  string
}

func main() {
	familyFlag := flag.String("family", "all", "example family to check: all, calc, datakeeper, draw, vehicle-report, parser-recovery, or mini-compiler")
	allowlistPath := flag.String("allowlist", "examples/manifest-parity.allowlist.json", "JSON allowlist for intentional target-specific manifest differences")
	flag.Parse()

	allowed, err := readAllowlist(*allowlistPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	checked := 0
	for _, family := range exampleFamilies() {
		if *familyFlag != "all" && *familyFlag != family.Name {
			continue
		}
		if err := checkFamily(family, allowed); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		checked++
	}
	if checked == 0 {
		fmt.Fprintf(os.Stderr, "unknown example family %q\n", *familyFlag)
		os.Exit(2)
	}
	fmt.Printf("example action manifest parity check passed for %d family set(s)\n", checked)
}

func exampleFamilies() []exampleFamily {
	return []exampleFamily{
		{
			Name: "calc",
			Specs: []targetSpec{
				{Target: "go", Path: "examples/go/calc/calc.lf"},
				{Target: "csharp", Path: "examples/csharp/calc/calc.lf"},
				{Target: "c", Path: "examples/c/calc/calc.lf"},
				{Target: "cpp", Path: "examples/cpp/calc/calc.lf"},
			},
		},
		{
			Name: "datakeeper",
			Specs: []targetSpec{
				{Target: "go", Path: "examples/go/datakeeper/datakeeper.lf"},
				{Target: "csharp", Path: "examples/csharp/datakeeper/datakeeper.lf"},
				{Target: "c", Path: "examples/c/datakeeper/datakeeper.lf"},
				{Target: "cpp", Path: "examples/cpp/datakeeper/datakeeper.lf"},
			},
		},
		{
			Name: "draw",
			Specs: []targetSpec{
				{Target: "go", Path: "examples/go/draw/draw.lf"},
				{Target: "csharp", Path: "examples/csharp/draw/draw.lf"},
				{Target: "c", Path: "examples/c/draw/draw.lf"},
				{Target: "cpp", Path: "examples/cpp/draw/draw.lf"},
			},
		},
		{
			Name: "vehicle-report",
			Specs: []targetSpec{
				{Target: "go", Path: "examples/go/vehicle-report/vehicle.lf"},
				{Target: "csharp", Path: "examples/csharp/vehicle-report/vehicle.lf"},
				{Target: "c", Path: "examples/c/vehicle-report/vehicle.lf"},
				{Target: "cpp", Path: "examples/cpp/vehicle-report/vehicle.lf"},
			},
		},
		{
			Name: "parser-recovery",
			Specs: []targetSpec{
				{Target: "go", Path: "examples/go/parser-recovery/recovery.lf"},
				{Target: "csharp", Path: "examples/csharp/parser-recovery/recovery.lf"},
				{Target: "c", Path: "examples/c/parser-recovery/recovery.lf"},
				{Target: "cpp", Path: "examples/cpp/parser-recovery/recovery.lf"},
			},
		},
		{
			Name: "mini-compiler",
			Specs: []targetSpec{
				{Target: "go", Path: "examples/templates/go/mini-compiler/mini.lf"},
				{Target: "csharp", Path: "examples/templates/csharp/mini-compiler/mini.lf"},
				{Target: "c", Path: "examples/templates/c/mini-compiler/mini.lf"},
				{Target: "cpp", Path: "examples/templates/cpp/mini-compiler/mini.lf"},
			},
		},
	}
}

func readAllowlist(path string) (allowlist, error) {
	if strings.TrimSpace(path) == "" {
		return allowlist{}, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return allowlist{}, nil
	}
	if err != nil {
		return allowlist{}, fmt.Errorf("read action manifest parity allowlist %s: %w", path, err)
	}
	var out allowlist
	if err := json.Unmarshal(data, &out); err != nil {
		return allowlist{}, fmt.Errorf("parse action manifest parity allowlist %s: %w", path, err)
	}
	for i, allowed := range out.Allow {
		if allowed.Family == "" || allowed.Target == "" || allowed.Path == "" || allowed.Reason == "" {
			return allowlist{}, fmt.Errorf("allowlist entry %d must include family, target, path, and reason", i+1)
		}
	}
	return out, nil
}

func checkFamily(family exampleFamily, allowed allowlist) error {
	var baselineSpec targetSpec
	var baseline manifestContract
	for i, spec := range family.Specs {
		contract, err := buildContract(spec)
		if err != nil {
			return fmt.Errorf("%s: build manifest contract for %s: %w", family.Name, spec.Path, err)
		}
		if i == 0 {
			baselineSpec = spec
			baseline = contract
			continue
		}
		diffs := diffContracts(baseline, contract)
		unallowed := filterUnallowed(family.Name, spec.Target, diffs, allowed)
		if len(unallowed) > 0 {
			return formatParityError(family.Name, baselineSpec, spec, unallowed)
		}
		if len(diffs) > 0 {
			fmt.Printf("%s action manifest parity allowed differences between %s and %s (%d difference(s))\n", family.Name, baselineSpec.Path, spec.Path, len(diffs))
		}
	}
	fmt.Printf("%s action manifest parity ok (%d target specs)\n", family.Name, len(family.Specs))
	return nil
}

func buildContract(target targetSpec) (manifestContract, error) {
	data, err := os.ReadFile(target.Path)
	if err != nil {
		return manifestContract{}, err
	}
	parsed, diags := spec.ParseCombined(data, target.Path)
	if parsed == nil {
		return manifestContract{}, diagnosticsOrError(diags, "parse failed")
	}
	_, lexDiags := lex.BuildFromSpecWithScanner(parsed.Lexer, parsed.Scanner)
	diags = append(diags, lexDiags...)
	grammar, grammarDiags := parse.FromSpec(*parsed)
	diags = append(diags, grammarDiags...)
	if diags.HasErrors() {
		return manifestContract{}, diags
	}
	table := parse.Build(grammar, parsed.Grammar.Algorithm)
	if len(table.Conflicts) > 0 {
		return manifestContract{}, fmt.Errorf("parser table has %d conflict(s)", len(table.Conflicts))
	}
	manifest := roundTripManifestJSON(action.Build(grammar, parsed.Semantics, target.Target))
	return normalizeManifest(manifest, grammar, table), nil
}

func diagnosticsOrError(diags diagnostics.List, fallback string) error {
	if len(diags) == 0 {
		return errors.New(fallback)
	}
	return diags
}

func roundTripManifestJSON(manifest action.Manifest) action.Manifest {
	data, err := json.Marshal(manifest)
	if err != nil {
		panic(err)
	}
	var decoded action.Manifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		panic(err)
	}
	return decoded
}

func normalizeManifest(manifest action.Manifest, grammar *parse.Grammar, table *parse.Table) manifestContract {
	out := manifestContract{
		Recovery: recoveryContract{
			Enabled:     table.ErrorRecovery,
			Productions: recoveryProductions(grammar),
		},
		Expected: expectedTokenContract(grammar.ExpectedTokens),
	}
	for _, manifestAction := range manifest.Actions {
		normalizedAction := actionContract{
			ID:                manifestAction.ID,
			Name:              manifestAction.Name,
			Typed:             manifestAction.Typed,
			TypeIssue:         manifestAction.TypeIssue,
			ReturnType:        typeRole(grammar, firstActionLHS(manifestAction), manifestAction.ReturnType),
			ConsistentContext: manifestAction.ConsistentContext,
		}
		for _, manifestRule := range manifestAction.Rules {
			normalizedRule := ruleContract{
				ID:         manifestRule.ID,
				LHS:        manifestRule.LHS,
				ReturnType: typeRole(grammar, manifestRule.LHS, manifestRule.ReturnType),
				Typed:      manifestRule.Typed,
				TypeIssue:  manifestRule.TypeIssue,
			}
			for _, operand := range manifestRule.RHS {
				normalizedRule.RHS = append(normalizedRule.RHS, operandContract{
					Position: operand.Position,
					Symbol:   operand.Symbol,
					Label:    operand.Label,
					Type:     typeRole(grammar, operand.Symbol, operand.Type),
				})
			}
			normalizedAction.Rules = append(normalizedAction.Rules, normalizedRule)
		}
		out.Actions = append(out.Actions, normalizedAction)
	}
	return out
}

func firstActionLHS(action action.Action) string {
	if len(action.Rules) == 0 {
		return ""
	}
	return action.Rules[0].LHS
}

func typeRole(grammar *parse.Grammar, symbol string, raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "missing"
	}
	switch {
	case grammar.Terminals[symbol]:
		return "terminal:" + symbol
	case grammar.Nonterminals[symbol]:
		return "nonterminal:" + symbol
	default:
		return "present"
	}
}

func recoveryProductions(grammar *parse.Grammar) []recoveryProduction {
	var out []recoveryProduction
	for _, rule := range grammar.Rules {
		if !contains(rule.RHS, parse.Error) {
			continue
		}
		out = append(out, recoveryProduction{
			ID:     rule.ID,
			LHS:    rule.LHS,
			RHS:    append([]string(nil), rule.RHS...),
			Labels: trimTrailingEmptyLabels(rule.Labels),
		})
	}
	return out
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func trimTrailingEmptyLabels(labels []string) []string {
	out := append([]string(nil), labels...)
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func expectedTokenContract(expected spec.ExpectedTokenSpec) expectedContract {
	out := expectedContract{}
	for _, alias := range expected.Aliases {
		out.Aliases = append(out.Aliases, expectedAlias{Token: alias.Token, Label: alias.Label})
	}
	sort.Slice(out.Aliases, func(i, j int) bool {
		if out.Aliases[i].Token == out.Aliases[j].Token {
			return out.Aliases[i].Label < out.Aliases[j].Label
		}
		return out.Aliases[i].Token < out.Aliases[j].Token
	})
	for _, group := range expected.Groups {
		tokens := append([]string(nil), group.Tokens...)
		sort.Strings(tokens)
		out.Groups = append(out.Groups, expectedGroup{Name: group.Name, Tokens: tokens})
	}
	sort.Slice(out.Groups, func(i, j int) bool { return out.Groups[i].Name < out.Groups[j].Name })
	for _, hidden := range expected.Hidden {
		out.Hidden = append(out.Hidden, hidden.Token)
	}
	sort.Strings(out.Hidden)
	return out
}

func diffContracts(want manifestContract, got manifestContract) []contractDiff {
	wantData := mustJSON(want)
	gotData := mustJSON(got)
	if string(wantData) == string(gotData) {
		return nil
	}
	var diffs []contractDiff
	compareActions(&diffs, "actions", want.Actions, got.Actions)
	compareRecovery(&diffs, "recovery", want.Recovery, got.Recovery)
	compareExpected(&diffs, "expected", want.Expected, got.Expected)
	return diffs
}

func mustJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func compareActions(diffs *[]contractDiff, path string, want []actionContract, got []actionContract) {
	compareInt(diffs, path+".length", len(want), len(got))
	limit := min(len(want), len(got))
	for i := 0; i < limit; i++ {
		actionPath := fmt.Sprintf("%s[%d]", path, i)
		compareInt(diffs, actionPath+".id", want[i].ID, got[i].ID)
		compareString(diffs, actionPath+".name", want[i].Name, got[i].Name)
		compareBool(diffs, actionPath+".typed", want[i].Typed, got[i].Typed)
		compareString(diffs, actionPath+".typeIssue", want[i].TypeIssue, got[i].TypeIssue)
		compareString(diffs, actionPath+".returnType", want[i].ReturnType, got[i].ReturnType)
		compareBool(diffs, actionPath+".consistentContext", want[i].ConsistentContext, got[i].ConsistentContext)
		compareRules(diffs, actionPath+".rules", want[i].Rules, got[i].Rules)
	}
}

func compareRules(diffs *[]contractDiff, path string, want []ruleContract, got []ruleContract) {
	compareInt(diffs, path+".length", len(want), len(got))
	limit := min(len(want), len(got))
	for i := 0; i < limit; i++ {
		rulePath := fmt.Sprintf("%s[%d]", path, i)
		compareInt(diffs, rulePath+".id", want[i].ID, got[i].ID)
		compareString(diffs, rulePath+".lhs", want[i].LHS, got[i].LHS)
		compareString(diffs, rulePath+".returnType", want[i].ReturnType, got[i].ReturnType)
		compareBool(diffs, rulePath+".typed", want[i].Typed, got[i].Typed)
		compareString(diffs, rulePath+".typeIssue", want[i].TypeIssue, got[i].TypeIssue)
		compareOperands(diffs, rulePath+".rhs", want[i].RHS, got[i].RHS)
	}
}

func compareOperands(diffs *[]contractDiff, path string, want []operandContract, got []operandContract) {
	compareInt(diffs, path+".length", len(want), len(got))
	limit := min(len(want), len(got))
	for i := 0; i < limit; i++ {
		operandPath := fmt.Sprintf("%s[%d]", path, i)
		compareInt(diffs, operandPath+".position", want[i].Position, got[i].Position)
		compareString(diffs, operandPath+".symbol", want[i].Symbol, got[i].Symbol)
		compareString(diffs, operandPath+".label", want[i].Label, got[i].Label)
		compareString(diffs, operandPath+".type", want[i].Type, got[i].Type)
	}
}

func compareRecovery(diffs *[]contractDiff, path string, want recoveryContract, got recoveryContract) {
	compareBool(diffs, path+".enabled", want.Enabled, got.Enabled)
	compareInt(diffs, path+".productions.length", len(want.Productions), len(got.Productions))
	limit := min(len(want.Productions), len(got.Productions))
	for i := 0; i < limit; i++ {
		productionPath := fmt.Sprintf("%s.productions[%d]", path, i)
		compareInt(diffs, productionPath+".id", want.Productions[i].ID, got.Productions[i].ID)
		compareString(diffs, productionPath+".lhs", want.Productions[i].LHS, got.Productions[i].LHS)
		compareStringSlice(diffs, productionPath+".rhs", want.Productions[i].RHS, got.Productions[i].RHS)
		compareStringSlice(diffs, productionPath+".labels", want.Productions[i].Labels, got.Productions[i].Labels)
	}
}

func compareExpected(diffs *[]contractDiff, path string, want expectedContract, got expectedContract) {
	compareInt(diffs, path+".aliases.length", len(want.Aliases), len(got.Aliases))
	aliasLimit := min(len(want.Aliases), len(got.Aliases))
	for i := 0; i < aliasLimit; i++ {
		aliasPath := fmt.Sprintf("%s.aliases[%d]", path, i)
		compareString(diffs, aliasPath+".token", want.Aliases[i].Token, got.Aliases[i].Token)
		compareString(diffs, aliasPath+".label", want.Aliases[i].Label, got.Aliases[i].Label)
	}
	compareInt(diffs, path+".groups.length", len(want.Groups), len(got.Groups))
	groupLimit := min(len(want.Groups), len(got.Groups))
	for i := 0; i < groupLimit; i++ {
		groupPath := fmt.Sprintf("%s.groups[%d]", path, i)
		compareString(diffs, groupPath+".name", want.Groups[i].Name, got.Groups[i].Name)
		compareStringSlice(diffs, groupPath+".tokens", want.Groups[i].Tokens, got.Groups[i].Tokens)
	}
	compareStringSlice(diffs, path+".hidden", want.Hidden, got.Hidden)
}

func compareInt(diffs *[]contractDiff, path string, want int, got int) {
	if want != got {
		*diffs = append(*diffs, contractDiff{Path: path, Want: fmt.Sprint(want), Got: fmt.Sprint(got)})
	}
}

func compareBool(diffs *[]contractDiff, path string, want bool, got bool) {
	if want != got {
		*diffs = append(*diffs, contractDiff{Path: path, Want: fmt.Sprint(want), Got: fmt.Sprint(got)})
	}
}

func compareString(diffs *[]contractDiff, path string, want string, got string) {
	if want != got {
		*diffs = append(*diffs, contractDiff{Path: path, Want: quote(want), Got: quote(got)})
	}
}

func compareStringSlice(diffs *[]contractDiff, path string, want []string, got []string) {
	compareInt(diffs, path+".length", len(want), len(got))
	limit := min(len(want), len(got))
	for i := 0; i < limit; i++ {
		compareString(diffs, fmt.Sprintf("%s[%d]", path, i), want[i], got[i])
	}
}

func quote(value string) string {
	if value == "" {
		return `""`
	}
	return fmt.Sprintf("%q", value)
}

func filterUnallowed(family string, target string, diffs []contractDiff, allowed allowlist) []contractDiff {
	var out []contractDiff
	for _, diff := range diffs {
		if allowed.allows(family, target, diff.Path) {
			continue
		}
		out = append(out, diff)
	}
	return out
}

func (a allowlist) allows(family string, target string, path string) bool {
	for _, allowed := range a.Allow {
		if matchField(allowed.Family, family) && matchField(allowed.Target, target) && matchField(allowed.Path, path) {
			return true
		}
	}
	return false
}

func matchField(pattern string, value string) bool {
	return pattern == "*" || pattern == value
}

func formatParityError(family string, baseline targetSpec, candidate targetSpec, diffs []contractDiff) error {
	var b strings.Builder
	fmt.Fprintf(&b, "%s action manifest parity mismatch between %s and %s", family, baseline.Path, candidate.Path)
	for _, diff := range diffs {
		fmt.Fprintf(&b, "\n  %s: want %s, got %s", diff.Path, diff.Want, diff.Got)
	}
	fmt.Fprintf(&b, "\nAdd a documented entry to examples/manifest-parity.allowlist.json only when the target-specific difference is intentional.")
	return errors.New(b.String())
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

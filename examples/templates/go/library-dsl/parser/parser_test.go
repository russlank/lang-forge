//go:build langforge_generated

package parser

import (
	"strings"
	"testing"

	librarydslgenerated "github.com/russlank/lang-forge/examples/templates/go/library-dsl/generated"
	"github.com/russlank/lang-forge/examples/templates/go/library-dsl/model"
)

func TestParserFacade(t *testing.T) {
	document, err := New().Parse("set retries = 3;\nset title = \"nightly\";\nenable audit;")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	settings := document.Settings()
	if settings["retries"].Kind != model.NumberValue || settings["retries"].Number != 3 {
		t.Fatalf("unexpected retries setting: %#v", settings["retries"])
	}
	if settings["title"].Text != "nightly" {
		t.Fatalf("unexpected title setting: %#v", settings["title"])
	}
	if !settings["audit"].Bool {
		t.Fatalf("expected audit flag")
	}
}

func TestParserFacadeCollectionCompatibility(t *testing.T) {
	tokens, err := librarydslgenerated.Tokenize("enable audit;")
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	document, err := New().ParseTokens(tokens)
	if err != nil {
		t.Fatalf("parse tokens: %v", err)
	}
	if len(document.Entries) != 1 || document.Entries[0].Name != "audit" {
		t.Fatalf("unexpected document: %#v", document)
	}
}

func TestParserFacadeErrors(t *testing.T) {
	if _, err := New().Parse("set retries = ;"); err == nil {
		t.Fatalf("expected parser error")
	} else if !strings.Contains(FormatError(err), "syntax error") {
		t.Fatalf("expected formatted syntax error, got %q", FormatError(err))
	}
	if _, err := New().Parse("set retries = 999999999999999999999999;"); err == nil {
		t.Fatalf("expected reducer error")
	} else if !strings.Contains(err.Error(), "value.number") && !strings.Contains(err.Error(), "ValueNumber") {
		t.Fatalf("expected number reducer error, got %v", err)
	}
}

package parseralgo

import "testing"

func TestNormalizeAndParse(t *testing.T) {
	if got, ok := Normalize(""); !ok || got != Default {
		t.Fatalf("Normalize empty = %q, %t; want %q, true", got, ok, Default)
	}
	if got, ok := Parse("CANONICAL"); !ok || got != Canonical {
		t.Fatalf("Parse canonical = %q, %t; want %q, true", got, ok, Canonical)
	}
	if got, ok := Parse("IELR"); !ok || got != IELR {
		t.Fatalf("Parse ielr = %q, %t; want %q, true", got, ok, IELR)
	}
	if _, ok := Parse("magic"); ok {
		t.Fatal("Parse accepted unknown algorithm")
	}
}

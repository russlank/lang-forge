package lex

import "testing"

func TestPartition_SplitsOverlappingRanges(t *testing.T) {
	parts := Partition([]RangeSet{
		{{Lo: 'a', Hi: 'f'}},
		{{Lo: 'd', Hi: 'z'}},
	}, RangeSet{{Lo: 'a', Hi: 'z'}})
	want := []RangeSet{
		{{Lo: 'a', Hi: 'c'}},
		{{Lo: 'd', Hi: 'f'}},
		{{Lo: 'g', Hi: 'z'}},
	}
	if len(parts) != len(want) {
		t.Fatalf("parts len = %d, want %d: %#v", len(parts), len(want), parts)
	}
	for i := range want {
		if parts[i].String() != want[i].String() {
			t.Fatalf("part %d = %s, want %s", i, parts[i], want[i])
		}
	}
}

func TestRangeSet_DifferenceCutsMiddle(t *testing.T) {
	got := (RangeSet{{Lo: 'a', Hi: 'z'}}).Difference(RangeSet{{Lo: 'm', Hi: 'p'}})
	want := RangeSet{{Lo: 'a', Hi: 'l'}, {Lo: 'q', Hi: 'z'}}
	if got.String() != want.String() {
		t.Fatalf("difference = %s, want %s", got, want)
	}
}

package lex

import (
	"fmt"
	"sort"
	"unicode/utf8"
)

// Range is an inclusive rune interval.
type Range struct {
	Lo rune `json:"lo"`
	Hi rune `json:"hi"`
}

// RangeSet is a normalized-or-normalizable collection of rune intervals.
type RangeSet []Range

// Single returns a range set containing one rune.
func Single(r rune) RangeSet {
	return RangeSet{{Lo: r, Hi: r}}
}

// AnyByte returns the byte-oriented scanner domain.
func AnyByte() RangeSet {
	return RangeSet{{Lo: 0, Hi: 255}}
}

// UnicodeScalarDomain returns every valid Unicode scalar value.
func UnicodeScalarDomain() RangeSet {
	return RangeSet{
		{Lo: 0, Hi: surrogateMin - 1},
		{Lo: surrogateMax + 1, Hi: utf8.MaxRune},
	}
}

const (
	surrogateMin rune = 0xD800
	surrogateMax rune = 0xDFFF
)

// IsUnicodeScalar reports whether r is a valid Unicode scalar value.
func IsUnicodeScalar(r rune) bool {
	return r >= 0 && r <= utf8.MaxRune && (r < surrogateMin || r > surrogateMax)
}

// Normalize sorts and merges overlapping or adjacent ranges.
func (s RangeSet) Normalize() RangeSet {
	if len(s) == 0 {
		return nil
	}
	cp := append(RangeSet(nil), s...)
	sort.Slice(cp, func(i, j int) bool {
		if cp[i].Lo == cp[j].Lo {
			return cp[i].Hi < cp[j].Hi
		}
		return cp[i].Lo < cp[j].Lo
	})
	out := make(RangeSet, 0, len(cp))
	for _, r := range cp {
		if r.Hi < r.Lo {
			r.Lo, r.Hi = r.Hi, r.Lo
		}
		if len(out) == 0 || r.Lo > out[len(out)-1].Hi+1 {
			out = append(out, r)
			continue
		}
		if r.Hi > out[len(out)-1].Hi {
			out[len(out)-1].Hi = r.Hi
		}
	}
	return out
}

// Contains reports whether r belongs to the range set.
func (s RangeSet) Contains(r rune) bool {
	for _, rr := range s {
		if r < rr.Lo {
			return false
		}
		if r <= rr.Hi {
			return true
		}
	}
	return false
}

// IsSubsetOf reports whether every rune in s is part of domain.
func (s RangeSet) IsSubsetOf(domain RangeSet) bool {
	return len(s.Difference(domain)) == 0
}

// Union returns the normalized union of two range sets.
func (s RangeSet) Union(other RangeSet) RangeSet {
	return append(append(RangeSet(nil), s...), other...).Normalize()
}

// Intersects reports whether two range sets overlap.
func (s RangeSet) Intersects(other RangeSet) bool {
	i, j := 0, 0
	a := s.Normalize()
	b := other.Normalize()
	for i < len(a) && j < len(b) {
		if a[i].Hi < b[j].Lo {
			i++
		} else if b[j].Hi < a[i].Lo {
			j++
		} else {
			return true
		}
	}
	return false
}

// Intersection returns the normalized intersection of two range sets.
func (s RangeSet) Intersection(other RangeSet) RangeSet {
	a := s.Normalize()
	b := other.Normalize()
	var out RangeSet
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		lo := maxRune(a[i].Lo, b[j].Lo)
		hi := minRune(a[i].Hi, b[j].Hi)
		if lo <= hi {
			out = append(out, Range{Lo: lo, Hi: hi})
		}
		if a[i].Hi < b[j].Hi {
			i++
		} else {
			j++
		}
	}
	return out.Normalize()
}

// Difference returns s without any runes contained in other.
func (s RangeSet) Difference(other RangeSet) RangeSet {
	current := s.Normalize()
	for _, cut := range other.Normalize() {
		var next RangeSet
		for _, r := range current {
			if cut.Hi < r.Lo || cut.Lo > r.Hi {
				next = append(next, r)
				continue
			}
			if cut.Lo > r.Lo {
				next = append(next, Range{Lo: r.Lo, Hi: cut.Lo - 1})
			}
			if cut.Hi < r.Hi {
				next = append(next, Range{Lo: cut.Hi + 1, Hi: r.Hi})
			}
		}
		current = next
	}
	return current.Normalize()
}

// String returns a stable diagnostic representation of the range set.
func (s RangeSet) String() string {
	s = s.Normalize()
	if len(s) == 0 {
		return "[]"
	}
	out := "["
	for i, r := range s {
		if i > 0 {
			out += ","
		}
		if r.Lo == r.Hi {
			out += fmt.Sprintf("%q", r.Lo)
		} else {
			out += fmt.Sprintf("%q-%q", r.Lo, r.Hi)
		}
	}
	return out + "]"
}

// Partition splits the domain into disjoint classes for the supplied sets.
func Partition(sets []RangeSet, domain RangeSet) []RangeSet {
	parts := []RangeSet{domain.Normalize()}
	for _, set := range sets {
		set = set.Normalize()
		var next []RangeSet
		for _, part := range parts {
			inside := part.Intersection(set)
			outside := part.Difference(set)
			if len(inside) > 0 {
				next = append(next, inside)
			}
			if len(outside) > 0 {
				next = append(next, outside)
			}
		}
		parts = next
	}
	sort.Slice(parts, func(i, j int) bool {
		return parts[i][0].Lo < parts[j][0].Lo
	})
	return parts
}

func maxRune(a, b rune) rune {
	if a > b {
		return a
	}
	return b
}

func minRune(a, b rune) rune {
	if a < b {
		return a
	}
	return b
}

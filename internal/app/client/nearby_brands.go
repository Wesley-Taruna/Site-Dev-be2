package client

// Competitor brands of the nearby search (Wes, 23 Oct): which chains get
// their own Google Text Search and which names count as a chain. The list
// comes from the ACTIVE CompetitorBrands rows (the nearby service reads and
// caches them), so switching a brand on or off is a data change; the
// clients only get the result as a BrandSet. DefaultBrands is the same
// list in code, used when the table cannot be read or has no active row.

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"
)

// Brand is one competitor chain as the nearby search uses it.
type Brand struct {
	// Name is the brand answered in places[].brand ("Indomaret").
	Name string
	// Query is the Google Text Search query; "" = no own search (found by
	// the convenience_store Nearby Search or the OSM shop tags, e.g. K3
	// Mart).
	Query string
	// Patterns are case-insensitive regexes on the place name (Go syntax,
	// the CompetitorBrands.Patterns JSON list); empty = the name itself.
	Patterns []string
}

// DefaultBrands = the active rows of scripts/localdev/sql/02_seed_master.sql
// (23 Oct), in Sort order. Patterns = API Lab src/lib/brands.ts + runner.ts
// (spelling variants "Indomart", "Indo Maret", "Alfa Expres"; "Tomoro"
// matches "TOMORO COFFEE").
var DefaultBrands = []Brand{
	{Name: "Indomaret", Query: "Indomaret", Patterns: []string{`indo\s*-?\s*mare?t`}},
	{Name: "Alfamart", Query: "Alfamart", Patterns: []string{`alfa\s*mart`, `alfa\s*expres`}},
	{Name: "Alfamidi", Query: "Alfamidi", Patterns: []string{`alfa\s*midi`}},
	{Name: "Lawson", Query: "Lawson", Patterns: []string{`lawson`}},
	{Name: "Circle K", Query: "Circle K", Patterns: []string{`circle\s*-?\s*k\b`}},
	{Name: "K3 Mart", Patterns: []string{`k3\s*mart`}},
	{Name: "Kopi Kenangan", Query: "Kopi Kenangan", Patterns: []string{`kopi\s*kenangan`}},
	{Name: "Tomoro Coffee", Query: "Tomoro Coffee", Patterns: []string{`tomoro`}},
}

type compiledBrand struct {
	Brand
	re *regexp.Regexp
	// osm is the Overpass (POSIX) form of the patterns, "" when a pattern
	// cannot be written that way
	osm []string
}

// BrandSet is the compiled brand list; the zero value matches nothing.
type BrandSet struct {
	list []compiledBrand
	key  string
}

// NewBrandSet compiles the brands in the given order (= match order). A
// brand without a name or a pattern that does not compile is skipped or
// trimmed, and named in the errors (one bad row never breaks the search).
func NewBrandSet(brands []Brand) (BrandSet, []error) {
	var errs []error
	set := BrandSet{}
	h := fnv.New32a()
	for _, b := range brands {
		b.Name = strings.TrimSpace(b.Name)
		b.Query = strings.TrimSpace(b.Query)
		if b.Name == "" {
			errs = append(errs, fmt.Errorf("competitor brand without a name skipped"))
			continue
		}
		patterns := b.Patterns
		if len(patterns) == 0 {
			patterns = []string{strings.Join(strings.Fields(regexp.QuoteMeta(strings.ToLower(b.Name))), `\s*`)}
		}
		var ok []string
		for _, p := range patterns {
			if _, err := regexp.Compile(p); err != nil || strings.TrimSpace(p) == "" {
				errs = append(errs, fmt.Errorf("competitor brand %s: pattern %q skipped: %v", b.Name, p, err))
				continue
			}
			ok = append(ok, p)
		}
		if len(ok) == 0 {
			errs = append(errs, fmt.Errorf("competitor brand %s skipped: no usable pattern", b.Name))
			continue
		}
		b.Patterns = ok
		c := compiledBrand{Brand: b, re: regexp.MustCompile(`(?i)` + strings.Join(ok, "|"))}
		for _, p := range ok {
			if o := overpassPattern(p); o != "" {
				c.osm = append(c.osm, o)
			}
		}
		set.list = append(set.list, c)
		fmt.Fprintf(h, "%s\x00%s\x00%s\x01", b.Name, b.Query, strings.Join(ok, "\x00"))
	}
	if len(set.list) > 0 {
		set.key = fmt.Sprintf("%08x", h.Sum32())
	}
	return set, errs
}

// DefaultBrandSet is DefaultBrands compiled.
func DefaultBrandSet() BrandSet {
	s, _ := NewBrandSet(DefaultBrands)
	return s
}

// Len is the number of brands.
func (s BrandSet) Len() int { return len(s.list) }

// Key is a short fingerprint of the list (part of the nearby cache key:
// another brand list = other cached areas).
func (s BrandSet) Key() string { return s.key }

// Names are the brand names in order.
func (s BrandSet) Names() []string {
	out := make([]string, 0, len(s.list))
	for _, b := range s.list {
		out = append(out, b.Name)
	}
	return out
}

// Searched are the brands with their own Text Search, in order.
func (s BrandSet) Searched() []Brand {
	var out []Brand
	for _, b := range s.list {
		if b.Query != "" {
			out = append(out, b.Brand)
		}
	}
	return out
}

// Of is the brand found in text, or "".
func (s BrandSet) Of(text string) string {
	for _, b := range s.list {
		if b.re.MatchString(text) {
			return b.Name
		}
	}
	return ""
}

// overpassNames is the name regex of the Overpass query ("" = no brand).
func (s BrandSet) overpassNames() string {
	var parts []string
	for _, b := range s.list {
		parts = append(parts, b.osm...)
	}
	return strings.Join(parts, "|")
}

// overpassPattern writes a Go pattern in Overpass's POSIX syntax: \s → a
// space, \b dropped (the Go pattern checks again afterwards); "" when
// other escapes or quotes are left.
func overpassPattern(p string) string {
	o := strings.NewReplacer(`\s`, " ", `\b`, "").Replace(p)
	if strings.ContainsAny(o, "\\\"") {
		return ""
	}
	return o
}

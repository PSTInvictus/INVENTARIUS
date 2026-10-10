// Package codes formats and parses the short label codes printed on labels:
// a one-letter prefix and the database id, e.g. L42 (location), A107 (asset).
package codes

import (
	"fmt"
	"strconv"
	"strings"
)

type Kind byte

const (
	Location Kind = 'L'
	Item     Kind = 'I'
	Stock    Kind = 'S'
	Asset    Kind = 'A'
)

func (k Kind) Name() string {
	switch k {
	case Location:
		return "location"
	case Item:
		return "item"
	case Stock:
		return "stock"
	case Asset:
		return "asset"
	}
	return "unknown"
}

func Format(k Kind, id int64) string { return fmt.Sprintf("%c%d", k, id) }

// Parse accepts a code in any case, with surrounding spaces, or the full
// URL a QR code encodes (".../A107").
func Parse(s string) (Kind, int64, error) {
	s = strings.TrimSpace(s)
	if i := strings.LastIndexByte(s, '/'); i >= 0 {
		s = s[i+1:]
	}
	s = strings.ToUpper(s)
	if len(s) < 2 {
		return 0, 0, fmt.Errorf("invalid code %q", s)
	}
	k := Kind(s[0])
	switch k {
	case Location, Item, Stock, Asset:
	default:
		return 0, 0, fmt.Errorf("invalid code %q: unknown prefix", s)
	}
	id, err := strconv.ParseInt(s[1:], 10, 64)
	if err != nil || id <= 0 || s[1] == '0' {
		return 0, 0, fmt.Errorf("invalid code %q", s)
	}
	return k, id, nil
}

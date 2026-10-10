package api

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"inventarius/internal/inventory"
)

// parsedQuantity is a quantity as people write it in spreadsheets.
type parsedQuantity struct {
	Value float64
	State string // inventory.Exact, Approx or Uncounted
	Unit  string // from a suffix like "5 m"; "" if none
	Note  string // a remark in parentheses, e.g. "(3x4)"
}

// Words meaning "plenty, nobody counted".
var uncountedWords = map[string]bool{
	"+++": true, "++": true, "muitos": true, "muitas": true, "muito": true, "muita": true,
	"demasiados": true, "demasiadas": true, "vários": true, "varios": true, "várias": true,
	"lots": true, "many": true, "plenty": true,
}

var unitAliases = map[string]string{
	"pcs": "pcs", "pc": "pcs", "un": "pcs", "unid": "pcs", "x": "pcs",
	"m": "m", "cm": "cm", "mm": "mm", "g": "g", "kg": "kg", "ml": "ml", "l": "L",
}

var (
	quantityRe = regexp.MustCompile(`^([0-9.,]+(?:\s*\+\s*[0-9.,]+)*)\s*([a-zA-Z]*)$`)
	approxRe   = regexp.MustCompile(`^(~|≈|aprox\.?|approx\.?|cerca de|ca\.)\s*`)
)

// parseQuantity understands "12", "12,5", "9+1", "25+2+3", "~10", "12 (3x4)",
// "5 m", "530 PCS", and "+++"/"muitos" for uncounted.
func parseQuantity(s string) (parsedQuantity, error) {
	s = strings.TrimSpace(s)
	p := parsedQuantity{State: inventory.Exact}
	if s == "" {
		return p, nil
	}
	if uncountedWords[strings.ToLower(s)] {
		p.State = inventory.Uncounted
		return p, nil
	}
	if i := strings.IndexByte(s, '('); i >= 0 {
		p.Note = strings.TrimSpace(s[i:])
		s = strings.TrimSpace(s[:i])
	}
	if m := approxRe.FindString(strings.ToLower(s)); m != "" {
		p.State = inventory.Approx
		s = strings.TrimSpace(s[len(m):])
	}
	m := quantityRe.FindStringSubmatch(s)
	if m == nil {
		return p, fmt.Errorf("quantity %q is not a number (use 12, 9+1, ~10, 5 m, or +++ for not counted)", s)
	}
	for _, part := range strings.Split(m[1], "+") {
		v, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(part), ",", "."), 64)
		if err != nil {
			return p, fmt.Errorf("quantity %q is not a number", s)
		}
		p.Value += v
	}
	if m[2] != "" {
		u, ok := unitAliases[strings.ToLower(m[2])]
		if !ok {
			return p, fmt.Errorf("quantity %q: unknown unit %q", s, m[2])
		}
		p.Unit = u
	}
	return p, nil
}

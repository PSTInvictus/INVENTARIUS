package api

import "testing"

func TestParseQuantity(t *testing.T) {
	tests := []struct {
		in    string
		value float64
		state string
		unit  string
		note  string
	}{
		{"", 0, "exact", "", ""},
		{"12", 12, "exact", "", ""},
		{"12.0", 12, "exact", "", ""},
		{"12,5", 12.5, "exact", "", ""},
		{"9+1", 10, "exact", "", ""},
		{"25+2+3", 30, "exact", "", ""},
		{"~10", 10, "approx", "", ""},
		{"≈ 4", 4, "approx", "", ""},
		{"12 (3x4)", 12, "exact", "", "(3x4)"},
		{"4 (+1 sujo)", 4, "exact", "", "(+1 sujo)"},
		{"5 m", 5, "exact", "m", ""},
		{"530 PCS", 530, "exact", "pcs", ""},
		{"+++", 0, "uncounted", "", ""},
		{"muitos", 0, "uncounted", "", ""},
		{"Demasiadas", 0, "uncounted", "", ""},
	}
	for _, tc := range tests {
		p, err := parseQuantity(tc.in)
		if err != nil {
			t.Errorf("%q: %v", tc.in, err)
			continue
		}
		if p.Value != tc.value || p.State != tc.state || p.Unit != tc.unit || p.Note != tc.note {
			t.Errorf("%q = %+v, want %v %s %q %q", tc.in, p, tc.value, tc.state, tc.unit, tc.note)
		}
	}
	for _, bad := range []string{"Nylon", "---", "5 parsecs", "x2"} {
		if _, err := parseQuantity(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

package search

import (
	"reflect"
	"testing"
)

func doc(name, extra string) Doc {
	return NewDoc(Field{name, 3}, Field{extra, 1})
}

var catalogue = []struct{ name, extra string }{
	{"Resistência 10 Ω", "0402 Eletrónica Resistências"},
	{"Resistência 10k Ω", "0603 Eletrónica Resistências"},
	{"Resistência 100k Ω", "0603 Eletrónica Resistências"},
	{"Condensador 100 nF", "0402 Eletrónica Condensadores"},
	{"Condensador 4.7 uF", "1206 Eletrónica Condensadores"},
	{"Indutor 2.2 uH SPM10065VT-2R2M-D", "Eletrónica Indutores"},
	{"Porca M6", "Fixação"},
	{"Parafuso M4x12", "Estruturas Fixação"},
	{"Sensor de pressão BMP581", "Eletrónica Sensores"},
	{"Paraquedas drogue 30\"", "Recuperação Paraquedas"},
	{"Botão de pressão", "Interruptores"},
	{"Kit M3x20 com anilhas e porcas", "Estruturas"},
	{"Disco de paraquedas", "Recuperação"},
	{"Fusível rearmável 1 µF", "Proteção"}, // a µ to test folding
}

func search(t *testing.T, q string) []string {
	t.Helper()
	docs := make([]Doc, len(catalogue))
	for i, c := range catalogue {
		docs[i] = doc(c.name, c.extra)
	}
	var names []string
	for _, i := range Rank(Parse(q), docs) {
		names = append(names, catalogue[i].name)
	}
	return names
}

func TestSearch(t *testing.T) {
	tests := []struct {
		q     string
		first string // the best match
		count int    // number of matches; -1 = don't check
	}{
		{"resistencia", "Resistência 10 Ω", 3},
		{"resistencias", "Resistência 10 Ω", 3},
		{"RESISTÊNCIA", "Resistência 10 Ω", 3},
		{"resist", "Resistência 10 Ω", 3},
		{"10 ohm", "Resistência 10 Ω", 3}, // 10k and 100k also start with "10"
		{"10ohm", "Resistência 10 Ω", 3},
		{"10 ohms", "Resistência 10 Ω", 3},
		{"10k ohm", "Resistência 10k Ω", 1},
		{"10kohm", "Resistência 10k Ω", 1},
		{"10kΩ", "Resistência 10k Ω", 1},
		{"100k", "Resistência 100k Ω", 1},
		{"resistencia 0603", "Resistência 10k Ω", 2},
		{"100nf", "Condensador 100 nF", 1},
		{"condensadores 100 n", "Condensador 100 nF", 1},
		{"4,7uf", "Condensador 4.7 uF", 1},
		{"1 uf", "Fusível rearmável 1 µF", -1}, // "1" also starts the 1206 package
		{"2.2uh", "Indutor 2.2 uH SPM10065VT-2R2M-D", 1},
		{"spm10065", "Indutor 2.2 uH SPM10065VT-2R2M-D", 1},
		{"porcas m6", "Porca M6", 1},
		{"parafusos", "Parafuso M4x12", 1},
		{"m4x12", "Parafuso M4x12", 1},
		{"bmp581", "Sensor de pressão BMP581", 1},
		{"pressao", "Sensor de pressão BMP581", 2},
		{"recuperacao", "Paraquedas drogue 30\"", 2},
		{"botoes", "Botão de pressão", 1},
		{"fusivel rearmavel", "Fusível rearmável 1 µF", 1},
		{"porcas", "Porca M6", -1}, // the item that *is* a nut, before one that mentions nuts
		{"paraquedas", "Paraquedas drogue 30\"", -1},
		{"capacitor", "Condensador 100 nF", 2},
		{"capacitors 100nf", "Condensador 100 nF", 1},
		{"capac", "Condensador 100 nF", 2},
		{"resistor 10k", "Resistência 10k Ω", 1},
		{"resistors", "Resistência 10 Ω", 3},
		{"screws", "Parafuso M4x12", 1},
		{"nut m6", "Porca M6", 1},
		{"parachute", "Paraquedas drogue 30\"", 2},
		{"pressure sensor", "Sensor de pressão BMP581", 1},
		{"condensador", "Condensador 100 nF", 2},
		{"nothing like this", "", 0},
	}
	for _, tc := range tests {
		got := search(t, tc.q)
		if tc.count == 0 {
			if len(got) != 0 {
				t.Errorf("%q matched %v", tc.q, got)
			}
			continue
		}
		if len(got) == 0 || got[0] != tc.first {
			t.Errorf("%q: first result %v, want %q", tc.q, got, tc.first)
		}
		if tc.count >= 0 && len(got) != tc.count {
			t.Errorf("%q: %d results %v, want %d", tc.q, len(got), got, tc.count)
		}
	}
}

func TestTokens(t *testing.T) {
	got := Tokens("Resistência 10k Ω · 0603, Condensador 0,47 µF (2.2 uH)")
	want := []string{"resistencia", "10k", "ohm", "0603", "condensador", "0.47", "uf", "2.2", "uh"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Tokens = %q, want %q", got, want)
	}
}

func TestStem(t *testing.T) {
	for in, want := range map[string]string{
		"resistencias":  "resistencia",
		"condensadores": "condensador",
		"botoes":        "botao",
		"metais":        "metal",
		"porcas":        "porca",
		"cruzes":        "cruz",
		"glass":         "glass",
		"m6":            "m6",
		"10s":           "10s",
	} {
		if got := Stem(in); got != want {
			t.Errorf("Stem(%q) = %q, want %q", in, got, want)
		}
	}
}

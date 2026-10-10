// Package search matches free text the way people type it: without
// accents ("resistencia" finds "Resistência"), in plural or singular
// ("porcas" finds "Porca"), in Portuguese or English ("capacitor" finds
// "Condensador"), with "ohm" for Ω and "u" for µ, units run together
// ("10kohm", "100nf"), and any word cut short ("cond").
package search

import (
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Field is one piece of text of a document, with how much a match in it counts.
type Field struct {
	Text   string
	Weight float64
}

var (
	symbols = strings.NewReplacer(
		"ω", " ohm ", "μ", "u", // after NFKD folding: Ω/Ω -> ω (lowercased), µ -> μ
		"ohms", "ohm",
	)
	foldChain = transform.Chain(norm.NFKD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
)

// Normalize lower-cases, removes accents and spells out unit symbols.
func Normalize(s string) string {
	folded, _, err := transform.String(foldChain, s)
	if err != nil {
		folded = s
	}
	return symbols.Replace(strings.ToLower(folded))
}

// Tokens splits normalized text into words. Decimal points and commas
// inside numbers are kept ("2.2", "0,47" -> "0.47").
func Tokens(s string) []string {
	s = Normalize(s)
	var out []string
	var b strings.Builder
	flush := func() {
		if b.Len() > 0 {
			out = append(out, b.String())
			b.Reset()
		}
	}
	rs := []rune(s)
	for i, r := range rs {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case (r == '.' || r == ',') && b.Len() > 0 && i+1 < len(rs) &&
			unicode.IsDigit(rs[i-1]) && unicode.IsDigit(rs[i+1]):
			b.WriteRune('.')
		default:
			flush()
		}
	}
	flush()
	return out
}

// Stem reduces Portuguese and English plurals to the singular, so that
// "resistencias", "condensadores", "botoes" and "screws" match their singular.
func Stem(w string) string {
	switch n := len(w); {
	case n <= 3 || !unicode.IsLetter(rune(w[n-1])):
		return w
	case strings.HasSuffix(w, "oes"):
		return w[:n-3] + "ao"
	case strings.HasSuffix(w, "ais"):
		return w[:n-3] + "al"
	case strings.HasSuffix(w, "res"), strings.HasSuffix(w, "zes"), strings.HasSuffix(w, "les"),
		strings.HasSuffix(w, "ses"), strings.HasSuffix(w, "xes"), strings.HasSuffix(w, "ches"):
		return w[:n-2]
	case strings.HasSuffix(w, "ss"):
		return w
	case strings.HasSuffix(w, "s"):
		return w[:n-1]
	}
	return w
}

// Query is a parsed search: every word must match some word of a document.
type Query struct {
	words [][]alt // per query word: alternatives; one of them must match
}

// alt is one way a query word can match: all parts must be found.
type alt struct {
	parts  []string
	factor float64 // < 1 for looser forms (split units, translations)
}

// Parse prepares a query. A word that runs a number into a unit ("10kohm",
// "100nf") may also match as its parts ("10k" + "ohm", "100" + "nf").
func Parse(q string) Query {
	var query Query
	for _, w := range Tokens(q) {
		alts := []alt{{[]string{w}, 1}}
		for i := 1; i < len(w); i++ {
			left, right := w[:i], w[i:]
			if hasDigit(left) && !hasDigit(right) {
				alts = append(alts, alt{[]string{left, right}, 0.9})
			}
		}
		for _, t := range translations(w) {
			alts = append(alts, alt{[]string{t}, 0.95})
		}
		query.words = append(query.words, alts)
	}
	return query
}

func (q Query) Empty() bool { return len(q.words) == 0 }

func hasDigit(s string) bool {
	return strings.IndexFunc(s, unicode.IsDigit) >= 0
}

type token struct{ word, stem string }

// Doc is a document prepared for matching.
type Doc struct {
	fields [][]token
	weight []float64
}

func NewDoc(fields ...Field) Doc {
	var d Doc
	for _, f := range fields {
		var ts []token
		for _, t := range Tokens(f.Text) {
			ts = append(ts, token{t, Stem(t)})
		}
		d.fields = append(d.fields, ts)
		d.weight = append(d.weight, f.Weight)
	}
	return d
}

// matchPart scores how well one query part matches the document: the same
// word (singular or plural) 3, the start of a word 1, plus 0.5 when it is
// the first word of the field (what the thing is: "Porca M6" over "Kit ...
// e porcas"); weighted by field, 0 if nowhere.
func (d Doc) matchPart(p string) float64 {
	ps := Stem(p)
	best := 0.0
	for i, ts := range d.fields {
		for j, t := range ts {
			s := 0.0
			switch {
			case t.word == p, t.stem == ps:
				s = 3
			case strings.HasPrefix(t.word, p), strings.HasPrefix(t.stem, ps):
				s = 1
			}
			if s > 0 && j == 0 {
				s += 0.5
			}
			if s *= d.weight[i]; s > best {
				best = s
			}
		}
	}
	return best
}

// Score returns how well the document matches, and whether it matches at all.
func (q Query) Score(d Doc) (float64, bool) {
	total := 0.0
	for _, alts := range q.words {
		best := 0.0
		for _, a := range alts {
			s := 0.0
			for _, p := range a.parts {
				ps := d.matchPart(p)
				if ps == 0 {
					s = 0
					break
				}
				s += ps
			}
			best = max(best, s*a.factor)
		}
		if best == 0 {
			return 0, false
		}
		total += best
	}
	return total, true
}

// Rank returns the indexes of the matching documents, best first. Ties keep
// their original order.
func Rank(q Query, docs []Doc) []int {
	type hit struct {
		i     int
		score float64
	}
	var hits []hit
	for i, d := range docs {
		if s, ok := q.Score(d); ok {
			hits = append(hits, hit{i, s})
		}
	}
	sort.SliceStable(hits, func(a, b int) bool { return hits[a].score > hits[b].score })
	out := make([]int, len(hits))
	for i, h := range hits {
		out[i] = h.i
	}
	return out
}

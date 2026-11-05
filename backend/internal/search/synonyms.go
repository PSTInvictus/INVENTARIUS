package search

import (
	_ "embed"
	"sort"
	"strings"
)

// synonymGroups lists words that mean the same thing, one group per line
// (see the file for the rules).
//
//go:embed synonyms.txt
var synonymGroups string

// synonyms maps a word's stem to the other words of its groups.
var synonyms = func() map[string][]string {
	m := map[string][]string{}
	for _, line := range strings.Split(synonymGroups, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		words := strings.Fields(line)
		for _, w := range words {
			k := Stem(w)
			for _, o := range words {
				if o != w {
					m[k] = append(m[k], o)
				}
			}
		}
	}
	return m
}()

// translations returns the other-language words for w, each once. Words of 4
// letters or more also expand while still being typed ("capac" ->
// "condensador").
func translations(w string) []string {
	ws := Stem(w)
	var found [][]string
	if len(ws) < 4 {
		found = append(found, synonyms[ws])
	} else {
		for k, v := range synonyms {
			if strings.HasPrefix(k, ws) {
				found = append(found, v)
			}
		}
	}
	seen := map[string]bool{w: true}
	var out []string
	for _, v := range found {
		for _, t := range v {
			if !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	sort.Strings(out)
	return out
}

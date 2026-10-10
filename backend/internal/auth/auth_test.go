package auth

import (
	"testing"

	"inventarius/internal/config"
)

func TestCheckAccount(t *testing.T) {
	a := &Auth{cfg: config.Config{AllowedDomain: "portospaceteam.pt"}}
	tests := []struct {
		email    string
		verified bool
		hd       string
		ok       bool
	}{
		{"ana@portospaceteam.pt", true, "portospaceteam.pt", true},
		{"Ana@PortoSpaceTeam.pt", true, "portospaceteam.pt", true},
		{"ana@portospaceteam.pt", false, "portospaceteam.pt", false}, // unverified
		{"ana@gmail.com", true, "", false},                           // personal account
		{"ana@portospaceteam.pt", true, "", false},                   // no hd claim: not a Workspace account
		{"ana@evil.com", true, "portospaceteam.pt", false},           // mismatched email
		{"ana@xportospaceteam.pt", true, "portospaceteam.pt", false},
	}
	for _, tc := range tests {
		if err := a.checkAccount(tc.email, tc.verified, tc.hd); (err == nil) != tc.ok {
			t.Errorf("checkAccount(%q, %v, %q) = %v, want ok=%v", tc.email, tc.verified, tc.hd, err, tc.ok)
		}
	}
}

func TestSafeNext(t *testing.T) {
	for in, want := range map[string]string{
		"/items/3":         "/items/3",
		"":                 "/",
		"https://evil.com": "/",
		"//evil.com":       "/",
		"/\\evil.com":      "/",
		"/c/A107?x=1":      "/c/A107?x=1",
	} {
		if got := safeNext(in); got != want {
			t.Errorf("safeNext(%q) = %q, want %q", in, got, want)
		}
	}
}

package api

import (
	"net/http"
	"net/url"
	"testing"
)

func TestPartLookupWithoutKey(t *testing.T) {
	e := newEnv(t)
	c := e.login("ana@portospaceteam.pt")
	var meta struct {
		PartLookup bool `json:"part_lookup"`
	}
	c.do("GET", "/api/meta", nil, http.StatusOK, &meta)
	if meta.PartLookup {
		t.Fatal("part_lookup on without an API key")
	}
	c.do("GET", "/api/parts?mpn=LM358P", nil, http.StatusServiceUnavailable, nil)
}

func TestPartImageOnlyFromMouser(t *testing.T) {
	e := newEnv(t)
	c := e.login("ana@portospaceteam.pt")
	for _, u := range []string{
		"http://www.mouser.com/images/x.jpg",           // not https
		"https://evil.example/images/x.jpg",            // other host
		"https://mouser.com.evil.example/images/x.jpg", // look-alike host
		"https://www.mouser.com/api/v1/x",              // not an image path
		"https://127.0.0.1/images/x.jpg",
	} {
		c.do("GET", "/api/parts/image?url="+url.QueryEscape(u), nil, http.StatusBadRequest, nil)
	}
}

func TestLargeImage(t *testing.T) {
	got := largeImage("https://www.mouser.com/images/boschsensortec/images/262-BMP390_DSL.jpg")
	if want := "https://www.mouser.com/images/boschsensortec/lrg/262-BMP390_DSL.jpg"; got != want {
		t.Fatalf("largeImage = %q, want %q", got, want)
	}
}

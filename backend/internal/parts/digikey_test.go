package parts

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

const digikeyAnswer = `{"Product":{
 "Description":{"ProductDescription":"CAP CER 0.1UF 50V X7R 0603","DetailedDescription":"0.1 µF ±10% 50V Ceramic Capacitor X7R 0603 (1608 Metric)"},
 "Manufacturer":{"Id":490,"Name":"Murata Electronics"},
 "ManufacturerProductNumber":"GRM188R71H104KA93D",
 "ProductUrl":"https://www.digikey.pt/en/products/detail/murata-electronics/GRM188R71H104KA93D/587207",
 "DatasheetUrl":"//www.murata.com/en-us/products/productdata/8796738650142/GRM188R71H104KA93-01.pdf",
 "PhotoUrl":"https://mm.digikey.com/Volume0/opasdata/d220001/medias/images/1/MFG_GRM18.jpg",
 "ProductVariations":[{"DigiKeyProductNumber":"490-1519-1-ND","PackageType":{"Name":"Cut Tape (CT)"}}],
 "Parameters":[{"ParameterId":16,"ParameterText":"Package / Case","ValueText":"0603 (1608 Metric)"}],
 "Category":{"CategoryId":3,"Name":"Capacitors","ChildCategories":[{"CategoryId":60,"Name":"Ceramic Capacitors","ChildCategories":[]}]}}}`

// fakeDigiKey answers like the token and productdetails endpoints. product
// gives the status and body for a part number.
func fakeDigiKey(t *testing.T, tokens *atomic.Int32, product func(mpn, token string) (int, string)) *DigiKey {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/oauth2/token":
			r.ParseForm()
			if r.Form.Get("client_id") != "ID" || r.Form.Get("client_secret") != "SECRET" || r.Form.Get("grant_type") != "client_credentials" {
				w.WriteHeader(http.StatusUnauthorized)
				io.WriteString(w, `{"error":"invalid_client","error_description":"The client credentials are invalid"}`)
				return
			}
			n := tokens.Add(1)
			io.WriteString(w, `{"access_token":"T`+string(rune('0'+n))+`","expires_in":599,"token_type":"Bearer"}`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/products/v4/search/") && strings.HasSuffix(r.URL.Path, "/productdetails"):
			if r.Header.Get("X-DIGIKEY-Client-Id") != "ID" {
				t.Errorf("no client id header")
			}
			mpn := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/products/v4/search/"), "/productdetails")
			code, body := product(mpn, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
			w.WriteHeader(code)
			io.WriteString(w, body)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ts.Close)
	return &DigiKey{ClientID: "ID", ClientSecret: "SECRET", BaseURL: ts.URL}
}

func TestDigiKeyLookup(t *testing.T) {
	var tokens atomic.Int32
	d := fakeDigiKey(t, &tokens, func(mpn, token string) (int, string) {
		if mpn != "GRM188R71H104KA93D" {
			return 404, `{"title":"Not Found","status":404,"detail":"Product not found"}`
		}
		return 200, digikeyAnswer
	})
	p, err := d.Lookup(context.Background(), "GRM188R71H104KA93D")
	if err != nil {
		t.Fatal(err)
	}
	want := Part{
		Supplier:     "DigiKey",
		MPN:          "GRM188R71H104KA93D",
		Manufacturer: "Murata Electronics",
		Description:  "0.1 µF ±10% 50V Ceramic Capacitor X7R 0603 (1608 Metric)",
		Category:     "Ceramic Capacitors",
		Package:      "0603",
		DatasheetURL: "https://www.murata.com/en-us/products/productdata/8796738650142/GRM188R71H104KA93-01.pdf",
		ProductURL:   "https://www.digikey.pt/en/products/detail/murata-electronics/GRM188R71H104KA93D/587207",
		ImageURL:     "https://mm.digikey.com/Volume0/opasdata/d220001/medias/images/1/MFG_GRM18.jpg",
		SupplierPart: "490-1519-1-ND",
	}
	if p != want {
		t.Fatalf("got  %+v\nwant %+v", p, want)
	}
	if g := GroupOf(p.Category); g == nil || g.Name != "Condensadores" {
		t.Fatalf("group of %q = %v", p.Category, g)
	}
	if got := ShortLabel(p, GroupOf(p.Category)); got != "Capacitors 0.1uF 0603" {
		t.Fatalf("label = %q", got)
	}
	// The token is reused.
	if _, err := d.Lookup(context.Background(), "NOPE"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown part: %v", err)
	}
	if n := tokens.Load(); n != 1 {
		t.Fatalf("%d tokens fetched, want 1", n)
	}
}

func TestDigiKeyRenewsRejectedToken(t *testing.T) {
	var tokens atomic.Int32
	d := fakeDigiKey(t, &tokens, func(mpn, token string) (int, string) {
		if token == "T1" {
			return 401, `{"title":"Unauthorized","detail":"Bearer token expired"}`
		}
		return 200, digikeyAnswer
	})
	if _, err := d.Lookup(context.Background(), "GRM188R71H104KA93D"); err != nil {
		t.Fatal(err)
	}
	if n := tokens.Load(); n != 2 {
		t.Fatalf("%d tokens fetched, want 2", n)
	}
}

func TestDigiKeyBadCredentials(t *testing.T) {
	var tokens atomic.Int32
	d := fakeDigiKey(t, &tokens, func(string, string) (int, string) { return 200, digikeyAnswer })
	d.ClientSecret = "WRONG"
	_, err := d.Lookup(context.Background(), "GRM188R71H104KA93D")
	if err == nil || !strings.Contains(err.Error(), "client credentials are invalid") {
		t.Fatalf("err = %v", err)
	}
}

type fakeSource struct {
	name string
	err  error
}

func (f fakeSource) Name() string { return f.name }
func (f fakeSource) Lookup(context.Context, string) (Part, error) {
	if f.err != nil {
		return Part{}, f.err
	}
	return Part{Supplier: f.name}, nil
}

func TestChain(t *testing.T) {
	ctx := context.Background()
	down := errors.New("down")
	missing := fakeSource{"Mouser", ErrNotFound}
	for _, tc := range []struct {
		chain Chain
		want  string
		err   error
	}{
		{Chain{missing, fakeSource{"DigiKey", nil}}, "DigiKey", nil},
		{Chain{fakeSource{"Mouser", nil}, fakeSource{"DigiKey", nil}}, "Mouser", nil},
		{Chain{missing, fakeSource{"DigiKey", ErrNotFound}}, "", ErrNotFound},
		{Chain{fakeSource{"Mouser", down}, fakeSource{"DigiKey", ErrNotFound}}, "", down},
		{Chain{fakeSource{"Mouser", down}, fakeSource{"DigiKey", nil}}, "DigiKey", nil},
	} {
		p, err := tc.chain.Lookup(ctx, "X")
		if p.Supplier != tc.want || !errors.Is(err, tc.err) {
			t.Errorf("%v: got %q, %v; want %q, %v", tc.chain.Names(), p.Supplier, err, tc.want, tc.err)
		}
	}
	c := Chain{fakeSource{"Mouser", nil}, fakeSource{"DigiKey", nil}}
	if got := strings.Join(c.Prefer("digikey").Names(), ","); got != "DigiKey,Mouser" {
		t.Errorf("Prefer(digikey) = %s", got)
	}
	if got := strings.Join(c.Prefer("lcsc").Names(), ","); got != "Mouser,DigiKey" {
		t.Errorf("Prefer(lcsc) = %s", got)
	}
}

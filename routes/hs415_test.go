package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-market/routes"
)

func TestRegroupingAnOfferIsOneAddressAndTheToolTakesOfferIdAndProduct(t *testing.T) {
	who := "ann"
	built, err := routes.BuildFor(manager(t), routes.Mount{Caller: func(context.Context) string { return who }})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	for _, rt := range built {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	do := func(method, target, body string, out any) int {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(method, target, strings.NewReader(body)))
		_ = json.Unmarshal(w.Body.Bytes(), out)
		return w.Code
	}
	var p, o, out map[string]any
	do("POST", "/posters", `{"name":"Ann"}`, &p)
	pid := p["id"].(string)
	who = "coordinator"
	do("POST", "/posters/"+pid+"/approve", "", &p)
	who = "ann"
	do("POST", "/offers", `{"poster_id":"`+pid+`","product_words":"maharagwe","price":120,"product":"beans"}`, &o)
	oid := o["id"].(string)
	who = "coordinator"
	if code := do("POST", "/offers/"+oid+"/regroup", `{"product":"cowpeas"}`, &out); code != 200 || out["product"] != "cowpeas" || out["price"] != 120.0 {
		t.Fatalf("regroup %d %v", code, out)
	}
	if code := do("POST", "/offers/"+oid+"/regroup", `{"product":"cowpeas"}`, &out); code != 409 {
		t.Fatalf("same product: %d", code)
	}
	if code := do("POST", "/offers/"+oid+"/regroup", `{"product":""}`, &out); code != 400 {
		t.Fatalf("empty product: %d", code)
	}
	if code := do("POST", "/offers/nope/regroup", `{"product":"x"}`, &out); code != 404 {
		t.Fatalf("unknown offer: %d", code)
	}
}

func TestTheRegroupToolIsNamedAfterItsActionAndTakesOfferIdAndProduct(t *testing.T) {
	tools, err := routes.BuildTools(manager(t), routes.Mount{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tl := range tools {
		if tl.Name != "market_regroup_offer" {
			continue
		}
		if tl.Action != "market.offer.regroup" {
			t.Fatalf("action %s", tl.Action)
		}
		var schema struct {
			Properties map[string]any `json:"properties"`
		}
		if err := json.Unmarshal(tl.InputSchema, &schema); err != nil {
			t.Fatal(err)
		}
		for _, k := range []string{"offer_id", "product"} {
			if _, ok := schema.Properties[k]; !ok {
				t.Fatalf("no %s in %v", k, schema.Properties)
			}
		}
		return
	}
	t.Fatal("no market_regroup_offer tool")
}

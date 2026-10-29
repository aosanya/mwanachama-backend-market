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

func TestTheGroupOrderListTakesLimitAndAfterFromTheQueryAndTheTrailHasItsOwnAddress(t *testing.T) {
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
	var p map[string]any
	do("POST", "/posters", `{"name":"Ann"}`, &p)
	pid := p["id"].(string)
	who = "coordinator"
	do("POST", "/posters/"+pid+"/approve", "", &p)
	var first, second map[string]any
	for _, g := range []*map[string]any{&first, &second} {
		if code := do("POST", "/group-orders", `{"poster_id":"`+pid+`","price":3,"minimum":10}`, g); code != 201 {
			t.Fatalf("start %d", code)
		}
	}
	var page []map[string]any
	if code := do("GET", "/group-orders?limit=1", "", &page); code != 200 || len(page) != 1 {
		t.Fatalf("first page %d %v", code, page)
	}
	var rest []map[string]any
	if code := do("GET", "/group-orders?limit=1&after="+page[0]["id"].(string), "", &rest); code != 200 || len(rest) != 1 || rest[0]["id"] == page[0]["id"] {
		t.Fatalf("next page %d %v", code, rest)
	}
	var trail []map[string]any
	if code := do("GET", "/group-orders/"+first["id"].(string)+"/trail", "", &trail); code != 200 || len(trail) != 1 || trail[0]["what"] != "started" {
		t.Fatalf("trail %d %v", code, trail)
	}
	who = "stranger"
	if code := do("GET", "/group-orders/"+first["id"].(string)+"/trail", "", &trail); code != 404 {
		t.Fatalf("a stranger read the trail: %d", code)
	}
}

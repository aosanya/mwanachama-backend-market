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

func TestKindsAndCategoryCorrectionAreReachableByAddress(t *testing.T) {
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
	var p, o map[string]any
	do("POST", "/posters", `{"name":"Ann"}`, &p)
	pid := p["id"].(string)
	who = "coordinator"
	do("POST", "/posters/"+pid+"/approve", "", &p)
	who = "ann"
	do("POST", "/offers", `{"poster_id":"`+pid+`","product_words":"dap","price":10,"product":"dap","category":"seed"}`, &o)
	do("POST", "/offers", `{"poster_id":"`+pid+`","product_words":"beans","price":10,"product":"beans","category":"grain"}`, &o)
	who = "bea"
	do("POST", "/asks", `{"poster_id":"`+pid+`","product":"dap","question":"have any?"}`, &o)
	who = "coordinator"
	var changed []map[string]any
	if code := do("POST", "/products/category", `{"product":"dap","category":"fertiliser"}`, &changed); code != 200 || len(changed) != 1 || changed[0]["category"] != "fertiliser" {
		t.Fatalf("categorise %d %v", code, changed)
	}
	var out map[string]any
	if code := do("POST", "/products/category", `{"product":"dap","category":"fertiliser"}`, &out); code != 409 {
		t.Fatalf("same category %d", code)
	}
	if code := do("POST", "/products/category", `{"product":"dap","category":""}`, &out); code != 400 {
		t.Fatalf("empty category %d", code)
	}
	if code := do("POST", "/products/category", `{"product":"nope","category":"x"}`, &out); code != 404 {
		t.Fatalf("unknown product %d", code)
	}
	var kinds []map[string]any
	if code := do("GET", "/kinds?limit=1", "", &kinds); code != 200 || len(kinds) != 1 || kinds[0]["category"] != "fertiliser" || kinds[0]["asked"] != 1.0 || kinds[0]["product"] != "dap" {
		t.Fatalf("kinds %d %v", code, kinds)
	}
	if code := do("GET", "/kinds", "", &kinds); code != 200 || len(kinds) != 2 {
		t.Fatalf("kinds %d %v", code, kinds)
	}
}

func TestTheKindToolsAreNamedAfterTheirActionsAndTakeTheirFields(t *testing.T) {
	tools, err := routes.BuildTools(manager(t), routes.Mount{})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"market_categorise_product": {"product", "category"},
		"market_list_kinds":         {"limit"},
	}
	seen := 0
	for _, tl := range tools {
		fields, ok := want[tl.Name]
		if !ok {
			continue
		}
		seen++
		var schema struct {
			Properties map[string]any `json:"properties"`
		}
		if err := json.Unmarshal(tl.InputSchema, &schema); err != nil {
			t.Fatal(err)
		}
		for _, k := range fields {
			if _, ok := schema.Properties[k]; !ok {
				t.Fatalf("%s has no %s in %v", tl.Name, k, schema.Properties)
			}
		}
	}
	if seen != 2 {
		t.Fatalf("found %d of 2 tools", seen)
	}
}

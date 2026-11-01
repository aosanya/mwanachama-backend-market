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

func TestTheOrderPosterToolNamesThePosterAGroupOrderBuysFromAndStartAgreesWithIt(t *testing.T) {
	who := "ann"
	built, err := routes.BuildFor(manager(t), routes.Mount{Caller: func(context.Context) string { return who }})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	for _, rt := range built {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	do := func(method, target, body string) (int, map[string]any) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(method, target, strings.NewReader(body)))
		out := map[string]any{}
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}
	if code, _ := do("GET", "/group-order-poster", ""); code != 409 {
		t.Fatalf("no approved poster: %d", code)
	}
	_, p := do("POST", "/posters", `{"name":"Ann Seeds"}`)
	pid := p["id"].(string)
	who = "coordinator"
	do("POST", "/posters/"+pid+"/approve", "")
	code, got := do("GET", "/group-order-poster", "")
	if code != 200 || got["poster_id"] != pid || got["poster_name"] != "Ann Seeds" || got["source"] != "only_approved_poster" {
		t.Fatalf("order poster %d %v", code, got)
	}
	code, started := do("POST", "/group-orders", `{"product":"beans","price":100,"minimum":50}`)
	if code != 201 || started["poster_id"] != pid {
		t.Fatalf("start without a poster %d %v", code, started)
	}
	code, got = do("GET", "/group-order-poster?groupOrderID="+started["id"].(string), "")
	if code != 200 || got["poster_id"] != pid || got["source"] != "group_order" {
		t.Fatalf("poster of the order %d %v", code, got)
	}
	who = "bob"
	_, q := do("POST", "/posters", `{"name":"Bob Seeds"}`)
	who = "coordinator"
	do("POST", "/posters/"+q["id"].(string)+"/approve", "")
	if code, _ := do("GET", "/group-order-poster", ""); code != 400 {
		t.Fatalf("two approved posters: %d", code)
	}
	if code, _ := do("POST", "/group-orders", `{"product":"beans","price":100,"minimum":50}`); code != 400 {
		t.Fatalf("start with two approved posters and none named: %d", code)
	}
	code, got = do("GET", "/group-order-poster?groupOrderID="+started["id"].(string), "")
	if code != 200 || got["poster_id"] != pid {
		t.Fatalf("the order keeps its poster %d %v", code, got)
	}
	if code, _ := do("GET", "/group-order-poster?groupOrderID=nope", ""); code != 404 {
		t.Fatalf("unknown order: %d", code)
	}
}

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

func TestTheGroupOrderListTakesMineAndStateFromTheQuery(t *testing.T) {
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
	var orders [3]map[string]any
	for i := range orders {
		if code := do("POST", "/group-orders", `{"poster_id":"`+pid+`","price":3,"minimum":10}`, &orders[i]); code != 201 {
			t.Fatalf("start %d", code)
		}
	}
	who = "bea"
	var line map[string]any
	if code := do("POST", "/group-orders/"+orders[1]["id"].(string)+"/join", `{"quantity":1}`, &line); code != 201 {
		t.Fatalf("join %d %v", code, line)
	}

	var all, mine, openMine, cancelled []map[string]any
	if code := do("GET", "/group-orders", "", &all); code != 200 || len(all) != 3 {
		t.Fatalf("all %d %v", code, all)
	}
	if code := do("GET", "/group-orders?mine=true", "", &mine); code != 200 || len(mine) != 1 || mine[0]["id"] != orders[1]["id"] {
		t.Fatalf("mine %d %v", code, mine)
	}
	if code := do("GET", "/group-orders?mine=true&state=open", "", &openMine); code != 200 || len(openMine) != 1 {
		t.Fatalf("open and mine %d %v", code, openMine)
	}
	if code := do("GET", "/group-orders?state=cancelled", "", &cancelled); code != 200 || len(cancelled) != 0 {
		t.Fatalf("cancelled %d %v", code, cancelled)
	}
	var refused map[string]any
	if code := do("GET", "/group-orders?state=nonsense", "", &refused); code != 400 {
		t.Fatalf("an unknown state answered %d", code)
	}
}

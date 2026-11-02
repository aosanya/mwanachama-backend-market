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

func TestTheAskAndYesListsTakeRequestAnswerAndStateFromTheQuery(t *testing.T) {
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
	who = "bea"
	var first, second map[string]any
	do("POST", "/requests", `{"wanted":"hybrid maize seed"}`, &first)
	do("POST", "/requests", `{"wanted":"a ride"}`, &second)
	var askOne, askTwo map[string]any
	for request, out := range map[string]*map[string]any{first["id"].(string): &askOne, second["id"].(string): &askTwo} {
		if code := do("POST", "/asks", `{"poster_id":"`+pid+`","request_id":"`+request+`","question":"can you?"}`, out); code != 201 {
			t.Fatalf("ask %d %v", code, *out)
		}
	}
	who = "ann"
	var answered map[string]any
	if code := do("POST", "/asks/"+askOne["id"].(string)+"/answer", `{"answer":"yes","date":"2026-11-03"}`, &answered); code != 200 {
		t.Fatalf("answer %d %v", code, answered)
	}
	who = "bea"

	var all, ofFirst, waiting []map[string]any
	if code := do("GET", "/asks", "", &all); code != 200 || len(all) != 2 {
		t.Fatalf("all %d %v", code, all)
	}
	if code := do("GET", "/asks?request_id="+first["id"].(string), "", &ofFirst); code != 200 || len(ofFirst) != 1 || ofFirst[0]["id"] != askOne["id"] {
		t.Fatalf("one request %d %v", code, ofFirst)
	}
	if code := do("GET", "/asks?answer=waiting", "", &waiting); code != 200 || len(waiting) != 1 || waiting[0]["id"] != askTwo["id"] {
		t.Fatalf("waiting %d %v", code, waiting)
	}
	var refused map[string]any
	if code := do("GET", "/asks?answer=maybe", "", &refused); code != 400 {
		t.Fatalf("an unknown answer answered %d", code)
	}

	var yeses, noYeses []map[string]any
	if code := do("GET", "/yeses?request_id="+first["id"].(string)+"&state=open", "", &yeses); code != 200 || len(yeses) != 1 || yeses[0]["ask_id"] != askOne["id"] {
		t.Fatalf("yeses of the first request %d %v", code, yeses)
	}
	if code := do("GET", "/yeses?request_id="+second["id"].(string), "", &noYeses); code != 200 || len(noYeses) != 0 {
		t.Fatalf("yeses of the second request %d %v", code, noYeses)
	}
	if code := do("GET", "/yeses?state=maybe", "", &refused); code != 400 {
		t.Fatalf("an unknown state answered %d", code)
	}
}

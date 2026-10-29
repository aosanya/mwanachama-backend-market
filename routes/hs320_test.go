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

func TestWithdrawingAnAskIsOneAddressTheAskerOwnsAndAnAnsweredAskRefuses(t *testing.T) {
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
	var p, first, second map[string]any
	do("POST", "/posters", `{"name":"Ann"}`, &p)
	pid := p["id"].(string)
	who = "coordinator"
	do("POST", "/posters/"+pid+"/approve", "", &p)
	who = "bea"
	for i, a := range []*map[string]any{&first, &second} {
		if code := do("POST", "/asks", `{"poster_id":"`+pid+`","request_id":"r`+string(rune('1'+i))+`","question":"can you come?","detail":"Siaya market to Ndori"}`, a); code != 201 {
			t.Fatalf("ask %d", code)
		}
		who = "bea"
	}
	firstID, secondID := first["id"].(string), second["id"].(string)
	var out map[string]any
	who = "cy"
	if code := do("POST", "/asks/"+firstID+"/withdraw", "", &out); code != 404 {
		t.Fatalf("a stranger withdrew: %d", code)
	}
	who = "bea"
	if code := do("POST", "/asks/"+firstID+"/withdraw", "", &out); code != 200 || out["answer"] != "withdrawn" {
		t.Fatalf("withdraw %d %v", code, out)
	}
	if code := do("POST", "/asks/"+firstID+"/withdraw", "", &out); code != 409 {
		t.Fatalf("withdrew twice: %d", code)
	}
	who = "ann"
	if code := do("POST", "/asks/"+firstID+"/answer", `{"answer":"yes","date":"2026-11-01"}`, &out); code != 409 {
		t.Fatalf("answered a withdrawn ask: %d", code)
	}
	if code := do("POST", "/asks/"+secondID+"/answer", `{"answer":"no"}`, &out); code != 200 {
		t.Fatalf("answer %d", code)
	}
	who = "bea"
	if code := do("POST", "/asks/"+secondID+"/withdraw", "", &out); code != 409 {
		t.Fatalf("withdrew an answered ask: %d", code)
	}
	var asks []map[string]any
	if code := do("GET", "/asks", "", &asks); code != 200 || len(asks) != 2 {
		t.Fatalf("list %d %v", code, asks)
	}
}

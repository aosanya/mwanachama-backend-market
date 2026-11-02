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

func TestHS706_AGroupOrderEveryFarmerAnsweredYesToIsSentOnceTheCoordinatorTakesItBackToOpen(t *testing.T) {
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
	_, poster := do("POST", "/posters", `{"name":"Ann"}`)
	posterID := poster["id"].(string)
	who = "coordinator"
	do("POST", "/posters/"+posterID+"/approve", "")
	_, started := do("POST", "/group-orders", `{"poster_id":"`+posterID+`","price":3,"minimum":10}`)
	base := "/group-orders/" + started["id"].(string)
	farmers := []string{"bea", "cal", "dee"}
	for _, farmer := range farmers {
		who = farmer
		if code, out := do("POST", base+"/join", `{"quantity":4}`); code != 201 && code != 200 {
			t.Fatalf("HS706: %s joins the order, got %d %v", farmer, code, out)
		}
	}
	who = "ann"
	if code, out := do("POST", base+"/revision", `{"price":4,"minimum":10}`); code != 200 || out["state"] != "revising" {
		t.Fatalf("HS706: the poster asks for new terms, got %d %v, want 200 and revising", code, out)
	}
	for _, farmer := range farmers {
		who = farmer
		if code, out := do("POST", base+"/revision/answer", `{"answer":"yes"}`); code != 200 || out["answer"] != "yes" {
			t.Fatalf("HS706: %s answers yes to the new terms, got %d %v", farmer, code, out)
		}
	}
	who = "coordinator"
	if code, got := do("GET", base, ""); code != 200 || got["state"] != "revising" {
		t.Fatalf("HS706: the yeses alone do not reopen the order, got %v (status %d), want revising", got["state"], code)
	}
	if code, out := do("POST", base+"/send", ""); code != 409 {
		t.Fatalf("HS706: a revising order is not sent, got %d %v, want 409", code, out)
	}
	who = "bea"
	if code, out := do("POST", base+"/reopen", ""); code != 403 {
		t.Fatalf("HS706: a farmer does not reopen the order, got %d %v, want 403", code, out)
	}
	who = "coordinator"
	code, got := do("POST", base+"/reopen", "")
	if code != 200 || got["state"] != "open" || got["terms_version"] != float64(2) {
		t.Fatalf("HS706: the coordinator reopens the order at terms version 2, got %v at version %v (status %d), want open at version 2", got["state"], got["terms_version"], code)
	}
	if code, out := do("POST", base+"/reopen", ""); code != 409 {
		t.Fatalf("HS706: an open order is not reopened again, got %d %v, want 409", code, out)
	}
	if code, out := do("POST", base+"/send", ""); code != 200 || out["state"] != "sent" {
		t.Fatalf("HS706: the coordinator sends the order every farmer said yes to, got %d %v, want 200 and sent", code, out)
	}
}

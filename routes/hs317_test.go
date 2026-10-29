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

func TestTheAnswerToAnAskTakesMinutesFromTheBodyAndTheYesReturnsThem(t *testing.T) {
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
	var p, req, ask, answered map[string]any
	do("POST", "/posters", `{"name":"Ann"}`, &p)
	pid := p["id"].(string)
	who = "coordinator"
	do("POST", "/posters/"+pid+"/approve", "", &p)
	who = "bea"
	do("POST", "/requests", `{"wanted":"a ride"}`, &req)
	if code := do("POST", "/asks", `{"poster_id":"`+pid+`","request_id":"`+req["id"].(string)+`","question":"can you come?"}`, &ask); code != 201 {
		t.Fatalf("ask %d %v", code, ask)
	}
	who = "ann"
	if code := do("POST", "/asks/"+ask["id"].(string)+"/answer", `{"answer":"yes","date":"2026-11-01","minutes":12}`, &answered); code != 200 {
		t.Fatalf("answer %d %v", code, answered)
	}
	var yeses []map[string]any
	who = "bea"
	if code := do("GET", "/yeses", "", &yeses); code != 200 || len(yeses) != 1 || yeses[0]["minutes"] != float64(12) {
		t.Fatalf("yeses %d %v", code, yeses)
	}
}

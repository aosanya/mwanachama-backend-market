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

func TestTheGroupGateRoutesAnswerTheStatusesTheStatesCall(t *testing.T) {
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
	var p, g, l, out map[string]any
	do("POST", "/posters", `{"name":"Ann"}`, &p)
	pid := p["id"].(string)
	who = "coordinator"
	do("POST", "/posters/"+pid+"/approve", "", &p)
	do("POST", "/group-orders", `{"poster_id":"`+pid+`","price":3,"minimum":10}`, &g)
	gid := g["id"].(string)
	who = "bea"
	do("POST", "/group-orders/"+gid+"/join", `{"quantity":6}`, &l)
	lid := l["id"].(string)
	who = "ann"
	do("POST", "/group-orders/"+gid+"/revision", `{"price":4,"minimum":10}`, &out)
	who = "coordinator"
	base := "/group-orders/" + gid
	if code := do("POST", base+"/lines/"+lid+"/decision", `{"decision":"later"}`, &out); code != 400 {
		t.Fatalf("unknown decision: %d", code)
	}
	if code := do("POST", base+"/lines/"+lid+"/decision", `{"decision":"keep"}`, &out); code != 200 || out["answer"] != "yes" || out["decided_by_coordinator"] != true {
		t.Fatalf("keep %d %v", code, out)
	}
	if code := do("POST", base+"/lines/"+lid+"/decision", `{"decision":"take_off"}`, &out); code != 409 {
		t.Fatalf("second decision: %d", code)
	}
	who = "bea"
	if code := do("POST", base+"/call-off", "", &out); code != 403 {
		t.Fatalf("a picker calls off: %d", code)
	}
	who = "coordinator"
	if code := do("POST", base+"/call-off", "", &out); code != 200 || out["state"] != "cancelled" {
		t.Fatalf("call off %d %v", code, out)
	}
	if code := do("POST", base+"/call-off", "", &out); code != 409 {
		t.Fatalf("second call off: %d", code)
	}
	if code := do("POST", "/group-orders/nope/call-off", "", &out); code != 404 {
		t.Fatalf("unknown order: %d", code)
	}
}

func TestTheGroupGateToolsAreNamedAfterTheirActionsAndTakeTheConsolesArguments(t *testing.T) {
	tools, err := routes.BuildTools(manager(t), routes.Mount{})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		action string
		args   []string
	}{
		"market_decide_silent_line": {"market.group_order.decide_silent_line", []string{"groupOrderID", "lineID", "decision"}},
		"market_call_off":           {"market.group_order.call_off", []string{"groupOrderID"}},
	}
	seen := 0
	for _, tl := range tools {
		w, ok := want[tl.Name]
		if !ok {
			continue
		}
		seen++
		if tl.Action != w.action {
			t.Fatalf("%s action %s", tl.Name, tl.Action)
		}
		var schema struct {
			Properties map[string]any `json:"properties"`
		}
		if err := json.Unmarshal(tl.InputSchema, &schema); err != nil {
			t.Fatal(err)
		}
		for _, k := range w.args {
			if _, ok := schema.Properties[k]; !ok {
				t.Fatalf("%s has no %s in %v", tl.Name, k, schema.Properties)
			}
		}
	}
	if seen != len(want) {
		t.Fatalf("found %d of %d tools", seen, len(want))
	}
}

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

func TestThePosterAndNoticeListsAnswerEachCallerWhatTheyMaySee(t *testing.T) {
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
	var ann, cat map[string]any
	do("POST", "/posters", `{"name":"Ann"}`, &ann)
	who = "cat"
	do("POST", "/posters", `{"name":"Cat"}`, &cat)
	who = "coordinator"
	do("POST", "/posters/"+ann["id"].(string)+"/approve", "", &ann)
	who = "bea"
	var notice map[string]any
	do("POST", "/notices", `{"wanted":"a ride to Bondo"}`, &notice)

	namesIn := func(list []map[string]any) []string {
		out := []string{}
		for _, p := range list {
			out = append(out, p["name"].(string))
		}
		return out
	}
	seenBy := func(caller string) []string {
		who = caller
		var list []map[string]any
		do("GET", "/posters", "", &list)
		return namesIn(list)
	}
	if got := seenBy("bea"); len(got) != 1 || got[0] != "Ann" {
		t.Fatalf("a farmer sees %v", got)
	}
	if got := seenBy("cat"); len(got) != 2 {
		t.Fatalf("the shop's owner sees %v", got)
	}
	if got := seenBy("coordinator"); len(got) != 2 {
		t.Fatalf("the person who decides sees %v", got)
	}

	who = "dan"
	var notices []map[string]any
	if code := do("GET", "/notices", "", &notices); code != http.StatusOK || len(notices) != 1 {
		t.Fatalf("notices %d %v", code, notices)
	}
	if _, has := notices[0]["picker"]; has {
		t.Fatalf("a neighbour's notice carries the picker: %v", notices[0])
	}
	who = "bea"
	do("GET", "/notices", "", &notices)
	if notices[0]["picker"] != "bea" {
		t.Fatalf("the picker's own notice lost the picker: %v", notices[0])
	}
}

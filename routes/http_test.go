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

func TestTheCallerComesFromTheMountAndAPickerCannotMarkPaid(t *testing.T) {
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
		r := httptest.NewRequest(method, target, strings.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}
	code, p := do("POST", "/posters", `{"name":"Ann","owner":"mallory"}`)
	if code != 201 || p["owner"] != "ann" {
		t.Fatalf("%d %v", code, p)
	}
	pid := p["id"].(string)
	who = "coordinator"
	if code, _ := do("POST", "/posters/"+pid+"/approve", ""); code != 200 {
		t.Fatalf("approve %d", code)
	}
	who = "ann"
	code, o := do("POST", "/offers", `{"poster_id":"`+pid+`","product_words":"beans","price":4}`)
	if code != 201 {
		t.Fatalf("offer %d %v", code, o)
	}
	who = "bea"
	code, ord := do("POST", "/offers/"+o["id"].(string)+"/buy", `{"quantity":2}`)
	if code != 201 || ord["unit_price"] != float64(4) || ord["picker"] != "bea" {
		t.Fatalf("buy %d %v", code, ord)
	}
	if code, _ := do("POST", "/orders/"+ord["id"].(string)+"/paid", ""); code != 403 {
		t.Fatalf("a picker marked paid: %d", code)
	}
	who = "ann"
	if code, _ := do("POST", "/orders/"+ord["id"].(string)+"/paid", ""); code != 200 {
		t.Fatalf("poster mark paid: %d", code)
	}
}

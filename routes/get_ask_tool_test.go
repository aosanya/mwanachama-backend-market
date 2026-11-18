package routes_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-market/routes"
)

func TestTheGetAskToolIsDeclared(t *testing.T) {
	tools, err := routes.BuildTools(manager(t), routes.Mount{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tl := range tools {
		if tl.Name == "market_get_ask" {
			return
		}
	}
	t.Fatal("missing tool market_get_ask")
}

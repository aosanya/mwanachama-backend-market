package market_test

import (
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

func TestAnAskKeepsTheProductItIsAboutForBothSides(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	r, _ := m.CreateRequest(ctx, "bea", market.Request{Wanted: "beans"})
	a, err := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: r.ID, Product: "maize flour", Question: "have any?"})
	if err != nil || a.Product != "maize flour" {
		t.Fatalf("ask %+v err %v", a, err)
	}
	for _, caller := range []string{"bea", "ann"} {
		asks, _ := m.ListAsks(ctx, caller)
		if len(asks) != 1 || asks[0].Product != "maize flour" {
			t.Fatalf("%s sees %+v", caller, asks)
		}
	}
	answered, err := m.AnswerAsk(ctx, "ann", a.ID, models.AnswerYes, "2026-11-01", "")
	if err != nil || answered.Product != "maize flour" {
		t.Fatalf("answered %+v err %v", answered, err)
	}
	plain, _ := m.CreateAsk(ctx, "cat", market.Ask{PosterID: p.ID, Question: "anything?"})
	if plain.Product != "" {
		t.Fatalf("an ask without a product: %+v", plain)
	}
}

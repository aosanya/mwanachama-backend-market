package market_test

import (
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

func TestAYesCanPointAtTheShopsOfferAndQuoteAPrice(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	o, _ := m.CreateOffer(ctx, "ann", market.Offer{PosterID: p.ID, ProductWords: "beans", Price: 10})
	r, _ := m.CreateRequest(ctx, "bea", market.Request{Wanted: "beans"})
	a, _ := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: r.ID, Question: "have any?"})
	if _, err := m.AnswerAskWithOffer(ctx, "ann", a.ID, models.AnswerYes, "2026-11-01", "", o.ID, 9.5); err != nil {
		t.Fatal(err)
	}
	for _, caller := range []string{"bea", "ann"} {
		ys, _ := m.ListYeses(ctx, caller)
		if len(ys) != 1 || ys[0].OfferID != o.ID || ys[0].Price != 9.5 || ys[0].Date != "2026-11-01" {
			t.Fatalf("%s sees %+v", caller, ys)
		}
	}
	ys, _ := m.ListYeses(ctx, "bea")
	ord, err := m.BuyNow(ctx, "bea", ys[0].OfferID, 2, ys[0].ID)
	if err != nil || ord.YesID != ys[0].ID {
		t.Fatalf("buy now from the yes: %+v %v", ord, err)
	}
}

func TestAYesWithoutAnOfferOrPriceIsAsBefore(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	r, _ := m.CreateRequest(ctx, "bea", market.Request{Wanted: "beans"})
	a, _ := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: r.ID, Question: "have any?"})
	if _, err := m.AnswerAsk(ctx, "ann", a.ID, models.AnswerYes, "2026-11-01", "tomorrow"); err != nil {
		t.Fatal(err)
	}
	ys, _ := m.ListYeses(ctx, "bea")
	if len(ys) != 1 || ys[0].OfferID != "" || ys[0].Price != 0 || ys[0].Note != "tomorrow" {
		t.Fatalf("yeses %+v", ys)
	}
}

func TestAYesRefusesAnotherPostersOfferAnUnknownOfferAndANegativePrice(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	q := approvedPoster(t, m, ctx, "dan")
	mine, _ := m.CreateOffer(ctx, "ann", market.Offer{PosterID: p.ID, ProductWords: "beans", Price: 10})
	theirs, _ := m.CreateOffer(ctx, "dan", market.Offer{PosterID: q.ID, ProductWords: "rice", Price: 8})
	r, _ := m.CreateRequest(ctx, "bea", market.Request{Wanted: "beans"})
	a, _ := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: r.ID, Question: "have any?"})
	cases := []struct {
		name  string
		offer string
		price float64
	}{{"another poster's offer", theirs.ID, 0}, {"an unknown offer", "nope", 0}, {"a negative price", mine.ID, -1}}
	for _, c := range cases {
		if _, err := m.AnswerAskWithOffer(ctx, "ann", a.ID, models.AnswerYes, "2026-11-01", "", c.offer, c.price); !errors.Is(err, market.ErrInvalid) {
			t.Fatalf("%s: %v", c.name, err)
		}
	}
	if ys, _ := m.ListYeses(ctx, "bea"); len(ys) != 0 {
		t.Fatal("a refused answer left a yes behind")
	}
	if got, _ := m.ListAsks(ctx, "bea"); got[0].Answer != models.AnswerWaiting {
		t.Fatal("a refused answer closed the ask")
	}
}

func TestANoCarriesNeitherAnOfferNorAPrice(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	o, _ := m.CreateOffer(ctx, "ann", market.Offer{PosterID: p.ID, ProductWords: "beans", Price: 10})
	r, _ := m.CreateRequest(ctx, "bea", market.Request{Wanted: "beans"})
	a, _ := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: r.ID, Question: "have any?"})
	if _, err := m.AnswerAskWithOffer(ctx, "ann", a.ID, models.AnswerNo, "", "", o.ID, 0); !errors.Is(err, market.ErrInvalid) {
		t.Fatalf("a no with an offer: %v", err)
	}
	if _, err := m.AnswerAskWithOffer(ctx, "ann", a.ID, models.AnswerNo, "", "", "", 5); !errors.Is(err, market.ErrInvalid) {
		t.Fatalf("a no with a price: %v", err)
	}
	if _, err := m.AnswerAskWithOffer(ctx, "ann", a.ID, models.AnswerNo, "", "", "", 0); err != nil {
		t.Fatalf("a plain no: %v", err)
	}
}

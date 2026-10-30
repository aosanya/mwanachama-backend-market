package market_test

import (
	"context"
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
)

func groupedOffer(t *testing.T) (market.MarketManager, market.Offer, context.Context) {
	t.Helper()
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	o, err := m.CreateOffer(ctx, "ann", market.Offer{
		PosterID: p.ID, ProductWords: "maharagwe", UnitWords: "kg", Price: 120, BulkMin: 50, BulkPrice: 100, Product: "beans",
	})
	if err != nil {
		t.Fatal(err)
	}
	return m, o, ctx
}

func TestRegroupingAnOfferChangesItsProductAndNothingElse(t *testing.T) {
	m, o, ctx := groupedOffer(t)
	got, err := m.RegroupOffer(ctx, "coordinator", o.ID, "cowpeas")
	if err != nil {
		t.Fatal(err)
	}
	want := o
	want.Product = "cowpeas"
	if got != want {
		t.Fatalf("returned %+v, want %+v", got, want)
	}
	offers, err := m.ListOffers(ctx, "")
	if err != nil || len(offers) != 1 || offers[0] != want {
		t.Fatalf("stored %+v %v, want %+v", offers, err, want)
	}
}

func TestRegroupingTrimsTheProductName(t *testing.T) {
	m, o, ctx := groupedOffer(t)
	got, err := m.RegroupOffer(ctx, "coordinator", o.ID, "  cowpeas ")
	if err != nil || got.Product != "cowpeas" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestRegroupingAnUnknownOfferIsNotFound(t *testing.T) {
	m, _, ctx := groupedOffer(t)
	if _, err := m.RegroupOffer(ctx, "coordinator", "no-such-offer", "cowpeas"); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestRegroupingToAnEmptyProductIsRefusedAndTheOfferKeepsItsProduct(t *testing.T) {
	m, o, ctx := groupedOffer(t)
	for _, product := range []string{"", "   "} {
		if _, err := m.RegroupOffer(ctx, "coordinator", o.ID, product); !errors.Is(err, market.ErrInvalid) {
			t.Fatalf("%q: want invalid, got %v", product, err)
		}
	}
	offers, _ := m.ListOffers(ctx, "")
	if offers[0] != o {
		t.Fatalf("the offer changed: %+v", offers[0])
	}
}

func TestRegroupingOntoTheProductItAlreadyHasIsRefusedAsAConflictThatSaysSo(t *testing.T) {
	m, o, ctx := groupedOffer(t)
	_, err := m.RegroupOffer(ctx, "coordinator", o.ID, "beans")
	if !errors.Is(err, market.ErrConflict) || err.Error() != "market: conflict: the offer is already under beans" {
		t.Fatalf("got %v", err)
	}
}

func TestRegroupingNeedsACaller(t *testing.T) {
	m, o, ctx := groupedOffer(t)
	if _, err := m.RegroupOffer(ctx, "", o.ID, "cowpeas"); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("want forbidden, got %v", err)
	}
}

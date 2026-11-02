package market_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-market"
)

func offerWithCategory(t *testing.T) (market.MarketManager, market.Poster, market.Offer) {
	t.Helper()
	m, ctx := newManager(t)
	ann := approvedPoster(t, m, ctx, "ann")
	off, err := m.CreateOffer(ctx, "ann", market.Offer{PosterID: ann.ID, ProductWords: "beans", Price: 5, Product: "beans", Category: "pulses"})
	if err != nil {
		t.Fatal(err)
	}
	return m, ann, off
}

func TestUpdateOfferThatLeavesCategoryOutKeepsIt(t *testing.T) {
	m, ann, off := offerWithCategory(t)
	ctx := t.Context()
	updated, err := m.UpdateOffer(ctx, "ann", market.Offer{ID: off.ID, PosterID: ann.ID, ProductWords: "beans", UnitWords: "kg", Price: 6, Product: "beans"})
	if err != nil {
		t.Fatal(err)
	}
	read, err := m.ListOffers(ctx, ann.ID)
	if err != nil || len(read) != 1 {
		t.Fatal(err, read)
	}
	if read[0].Category != "pulses" || updated.Category != "pulses" {
		t.Errorf("category after an update that omits it: stored %q, returned %q; want pulses", read[0].Category, updated.Category)
	}
}

func TestUpdateOfferThatLeavesProductOutKeepsItAndItsCategory(t *testing.T) {
	m, ann, off := offerWithCategory(t)
	ctx := t.Context()
	updated, err := m.UpdateOffer(ctx, "ann", market.Offer{ID: off.ID, PosterID: ann.ID, ProductWords: "beans", Price: 7})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Product != "beans" || updated.Category != "pulses" {
		t.Errorf("product %q category %q after an update that omits both; want beans and pulses", updated.Product, updated.Category)
	}
}

func TestUpdateOfferToANewProductDoesNotKeepTheOldCategory(t *testing.T) {
	m, ann, off := offerWithCategory(t)
	updated, err := m.UpdateOffer(t.Context(), "ann", market.Offer{ID: off.ID, PosterID: ann.ID, ProductWords: "maize", Price: 4, Product: "maize"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Product != "maize" || updated.Category == "pulses" {
		t.Errorf("product %q category %q after moving to maize without a category; the beans category must not follow", updated.Product, updated.Category)
	}
}

func TestUpdateOfferThatSendsACategoryChangesIt(t *testing.T) {
	m, ann, off := offerWithCategory(t)
	updated, err := m.UpdateOffer(t.Context(), "ann", market.Offer{ID: off.ID, PosterID: ann.ID, ProductWords: "beans", Price: 5, Product: "beans", Category: "legumes"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Category != "legumes" {
		t.Errorf("category %q; want the one sent", updated.Category)
	}
}

func TestStartGroupOrderToolSaysPosterIDCanBeLeftOut(t *testing.T) {
	var file struct {
		Operations map[string]struct {
			Tool        string `json:"tool"`
			Description string `json:"description"`
		} `json:"operations"`
	}
	if err := json.Unmarshal([]byte(readFile(t, "market.operations.json")), &file); err != nil {
		t.Fatal(err)
	}
	for _, op := range file.Operations {
		if op.Tool == "market_start_group_order" && !strings.Contains(op.Description, "poster_id") {
			t.Errorf("market_start_group_order describes itself as %q; want it to say poster_id can be left out", op.Description)
		}
	}
}

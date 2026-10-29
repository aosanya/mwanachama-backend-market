package market_test

import (
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
)

func TestAGroupOrderCarriesTheShopsPromisedDateAndNoteToEveryReader(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	g, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	m.JoinGroupOrder(ctx, "bea", g.ID, 4)
	before, _ := m.GetGroupOrder(ctx, "bea", g.ID)
	if before.PromisedDate != "" || before.Note != "" {
		t.Fatalf("nothing was promised yet: %+v", before.GroupOrder)
	}
	if _, err := m.PromiseGroupOrder(ctx, "ann", g.ID, "2026-11-05", "bags ready at the gate"); err != nil {
		t.Fatal(err)
	}
	for _, caller := range []string{"bea", "coord", "ann"} {
		v, _ := m.GetGroupOrder(ctx, caller, g.ID)
		if v.PromisedDate != "2026-11-05" || v.Note != "bags ready at the gate" || v.Price != 3 || v.Minimum != 10 {
			t.Fatalf("%s sees %+v", caller, v.GroupOrder)
		}
	}
	list, _ := m.ListGroupOrders(ctx, "bea")
	if len(list) != 1 || list[0].PromisedDate != "2026-11-05" {
		t.Fatalf("list %+v", list)
	}
	again, _ := m.PromiseGroupOrder(ctx, "ann", g.ID, "2026-11-08", "")
	if again.PromisedDate != "2026-11-08" || again.Note != "" {
		t.Fatalf("a second promise replaces the first: %+v", again)
	}
}

func TestOnlyTheShopPromisesAGroupOrderAndItNeedsADate(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	g, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	for _, caller := range []string{"coord", "bea", ""} {
		if _, err := m.PromiseGroupOrder(ctx, caller, g.ID, "2026-11-05", ""); !errors.Is(err, market.ErrForbidden) {
			t.Fatalf("%q promised: %v", caller, err)
		}
	}
	if _, err := m.PromiseGroupOrder(ctx, "ann", g.ID, "", "soon"); !errors.Is(err, market.ErrInvalid) {
		t.Fatalf("a promise without a date: %v", err)
	}
	if _, err := m.PromiseGroupOrder(ctx, "ann", "nope", "2026-11-05", ""); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("an unknown group order: %v", err)
	}
}

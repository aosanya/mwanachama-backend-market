package market_test

import (
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
)

func TestAGroupOrderTrailNamesEveryStepAndShowsAPickerOnlyHerOwnAndTheLeadersSteps(t *testing.T) {
	m, ctx := newManager(t)
	m.UseNames(upperNames)
	p := approvedPoster(t, m, ctx, "ann")
	g, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	m.JoinGroupOrder(ctx, "bea", g.ID, 4)
	m.JoinGroupOrder(ctx, "cat", g.ID, 2)
	m.PromiseGroupOrder(ctx, "ann", g.ID, "2026-11-05", "")
	m.AskRevision(ctx, "ann", g.ID, 4, 10)
	m.AnswerRevision(ctx, "bea", g.ID, "yes")

	all, err := m.GroupOrderTrail(ctx, "coord", g.ID)
	if err != nil || len(all) != 6 {
		t.Fatalf("coordinator sees %d: %+v %v", len(all), all, err)
	}
	want := []string{"started", "joined", "joined", "promised 2026-11-05", "asked for new terms", "answered yes"}
	for i, w := range want {
		if all[i].What != w || all[i].ActorName != "Name of "+all[i].Actor {
			t.Fatalf("step %d %+v", i, all[i])
		}
	}
	byPoster, _ := m.GroupOrderTrail(ctx, "ann", g.ID)
	if len(byPoster) != 6 {
		t.Fatalf("poster sees %d", len(byPoster))
	}
	mine, err := m.GroupOrderTrail(ctx, "bea", g.ID)
	if err != nil || len(mine) != 5 {
		t.Fatalf("bea sees %d: %+v %v", len(mine), mine, err)
	}
	for _, s := range mine {
		if s.Actor == "cat" {
			t.Fatalf("bea sees cat's step %+v", s)
		}
	}
	if _, err := m.GroupOrderTrail(ctx, "dee", g.ID); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("a stranger read the trail: %v", err)
	}
	if _, err := m.GroupOrderTrail(ctx, "coord", "nothing"); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("unknown group order: %v", err)
	}
}

func TestTheGroupOrderListComesInPagesByKeyAndEveryOrderIsReadOnce(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	for i := 0; i < 5; i++ {
		if _, err := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10}); err != nil {
			t.Fatal(err)
		}
	}
	seen := map[string]bool{}
	after, pages := "", 0
	for {
		page, err := m.ListGroupOrders(ctx, "coord", 2, after)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		if len(page) > 2 {
			t.Fatalf("a page of %d", len(page))
		}
		for _, v := range page {
			if seen[v.ID] || v.ID <= after {
				t.Fatalf("%s repeated or out of order after %s", v.ID, after)
			}
			seen[v.ID] = true
		}
		after = page[len(page)-1].ID
		pages++
	}
	if len(seen) != 5 || pages != 3 {
		t.Fatalf("saw %d orders in %d pages", len(seen), pages)
	}
	whole, _ := m.ListGroupOrders(ctx, "coord", 0, "")
	if len(whole) != 5 {
		t.Fatalf("the default page held %d", len(whole))
	}
	if _, err := m.ListGroupOrders(ctx, "coord", -1, ""); !errors.Is(err, market.ErrInvalid) {
		t.Fatalf("a negative page: %v", err)
	}
}

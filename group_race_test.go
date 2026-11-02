package market_test

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

func managerOn(t *testing.T) (*gorm.DB, market.MarketManager, context.Context) {
	t.Helper()
	db := newDB(t)
	s, err := market.SpecFor("app")
	if err != nil {
		t.Fatal(err)
	}
	if err := market.Provision(db, s); err != nil {
		t.Fatal(err)
	}
	m, err := market.NewMarketManager(db, s)
	if err != nil {
		t.Fatal(err)
	}
	m.UseCoordinators(func(_ context.Context, c string) bool { return c == "coordinator" || c == "coord" })
	return db, m, context.Background()
}

func interleave(t *testing.T, db *gorm.DB, groupID, set string) {
	t.Helper()
	done := false
	err := db.Callback().Update().Before("gorm:update").Register("interleave", func(tx *gorm.DB) {
		if done {
			return
		}
		done = true
		if _, err := tx.Statement.ConnPool.ExecContext(context.Background(), "UPDATE "+tx.Statement.Table+" SET "+set+" WHERE id = ?", groupID); err != nil {
			t.Error(err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}

func stateOf(t *testing.T, m market.MarketManager, ctx context.Context, id string) string {
	t.Helper()
	g, err := m.GetGroupOrder(ctx, "coord", id)
	if err != nil {
		t.Fatal(err)
	}
	return g.State
}

func TestAReopenDoesNotEraseACallOffThatLandedAfterItsRead(t *testing.T) {
	db, m, ctx := managerOn(t)
	p := approvedPoster(t, m, ctx, "ann")
	g, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	m.JoinGroupOrder(ctx, "bea", g.ID, 6)
	if _, err := m.AskRevision(ctx, "ann", g.ID, 4, 10); err != nil {
		t.Fatal(err)
	}
	interleave(t, db, g.ID, "state = '"+models.GroupCancelled+"'")
	if _, err := m.ReopenGroupOrder(ctx, "coord", g.ID); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("a reopen over a call-off: want conflict, got %v", err)
	}
	if got := stateOf(t, m, ctx, g.ID); got != models.GroupCancelled {
		t.Fatalf("the order is %q; want the call-off to stand", got)
	}
}

func TestASendDoesNotEraseARevisionThatLandedAfterItsRead(t *testing.T) {
	db, m, ctx := managerOn(t)
	p := approvedPoster(t, m, ctx, "ann")
	g, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	m.JoinGroupOrder(ctx, "bea", g.ID, 6)
	m.JoinGroupOrder(ctx, "cy", g.ID, 5)
	interleave(t, db, g.ID, "state = '"+models.GroupRevising+"', terms_version = terms_version + 1, price = 9")
	if _, err := m.SendGroupOrder(ctx, "coord", g.ID); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("a send over a revision: want conflict, got %v", err)
	}
	got, err := m.GetGroupOrder(ctx, "coord", g.ID)
	if err != nil || got.State != models.GroupRevising || got.Price != 9 {
		t.Fatalf("the revision should stand: %+v %v", got, err)
	}
}

func TestAPromiseDoesNotEraseARevisionThatLandedAfterItsRead(t *testing.T) {
	db, m, ctx := managerOn(t)
	p := approvedPoster(t, m, ctx, "ann")
	g, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	interleave(t, db, g.ID, "state = '"+models.GroupRevising+"', terms_version = terms_version + 1, price = 9")
	if _, err := m.PromiseGroupOrder(ctx, "ann", g.ID, "2026-11-10", ""); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("a promise over a revision: want conflict, got %v", err)
	}
	got, _ := m.GetGroupOrder(ctx, "coord", g.ID)
	if got.State != models.GroupRevising || got.Price != 9 {
		t.Fatalf("the revision should stand: %+v", got)
	}
}

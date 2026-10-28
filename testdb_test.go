package market_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

func newDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func newManager(t *testing.T) (market.MarketManager, context.Context) {
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
	return m, context.Background()
}

func approvedPoster(t *testing.T, m market.MarketManager, ctx context.Context, owner string) market.Poster {
	t.Helper()
	p, err := m.ApplyPoster(ctx, owner, market.Poster{Name: "Poster " + owner})
	if err != nil {
		t.Fatal(err)
	}
	p, err = m.ApprovePoster(ctx, "coordinator", p.ID)
	if err != nil || p.Status != models.PosterApproved {
		t.Fatal(err, p.Status)
	}
	return p
}

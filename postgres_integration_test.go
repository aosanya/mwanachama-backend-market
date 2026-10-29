//go:build integration

package market_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"

	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

func randomInstance(t *testing.T) string {
	t.Helper()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random instance: %v", err)
	}
	return "pg" + hex.EncodeToString(b)
}

func newPostgres(t *testing.T) (market.MarketManager, context.Context) {
	t.Helper()
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		t.Skip("POSTGRES_URL is not set")
	}
	db, err := gorm.Open(gormpostgres.Open(url), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	s, err := market.SpecFor(randomInstance(t))
	if err != nil {
		t.Fatalf("spec: %v", err)
	}
	if err := market.Provision(db, s); err != nil {
		t.Fatalf("provision: %v", err)
	}
	t.Cleanup(func() {
		for _, o := range s.Objects {
			db.Exec("drop table if exists " + s.TableFor(o) + " cascade")
		}
		db.Exec("drop table if exists " + s.NameRegistryTable() + " cascade")
	})
	m, err := market.NewMarketManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	m.UseCoordinators(func(_ context.Context, c string) bool { return c == "coordinator" })
	return m, context.Background()
}

func TestPG_ABuyNowRoundTrip(t *testing.T) {
	m, ctx := newPostgres(t)
	p, err := m.ApplyPoster(ctx, "ann", market.Poster{Name: "Ann"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := m.ApprovePoster(ctx, "coordinator", p.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	o, err := m.CreateOffer(ctx, "ann", market.Offer{PosterID: p.ID, ProductWords: "beans", Price: 10})
	if err != nil {
		t.Fatalf("offer: %v", err)
	}
	ord, err := m.BuyNow(ctx, "bea", o.ID, 2, "")
	if err != nil {
		t.Fatalf("buy now: %v", err)
	}
	got, err := m.GetOrder(ctx, "bea", ord.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if got.UnitPrice != 10 || got.Quantity != 2 || got.State != models.OrderPlaced {
		t.Fatalf("order did not round trip: %+v", got)
	}
}

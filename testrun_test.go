package market_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"
	"gorm.io/gorm"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

type testCopy struct {
	live market.MarketManager
	db   *gorm.DB
	spec *spec.Spec
	ctx  context.Context
}

func newTestCopy(t *testing.T) testCopy {
	t.Helper()
	db := newDB(t)
	s, err := market.SpecFor("app")
	if err != nil {
		t.Fatal(err)
	}
	if err := market.Provision(db, s); err != nil {
		t.Fatal(err)
	}
	live, err := market.NewMarketManager(db, s)
	if err != nil {
		t.Fatal(err)
	}
	live.UseCoordinators(func(_ context.Context, c string) bool { return c == "coordinator" })
	return testCopy{live: live, db: db, spec: s, ctx: context.Background()}
}

func (c testCopy) run(t *testing.T, id string) market.MarketManager {
	t.Helper()
	m, err := c.live.ForRun(id)
	if err != nil {
		t.Fatal(err)
	}
	m.UseCoordinators(func(_ context.Context, caller string) bool { return caller == "coordinator" })
	return m
}

func (c testCopy) count(t *testing.T, table string) int64 {
	t.Helper()
	var n int64
	if err := c.db.Table(table).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func (c testCopy) liveRows(t *testing.T) int64 {
	t.Helper()
	var total int64
	for _, o := range c.spec.Objects {
		total += c.count(t, c.spec.TableFor(o))
	}
	return total
}

func buyBeans(t *testing.T, c testCopy, m market.MarketManager) market.Order {
	t.Helper()
	p := approvedPoster(t, m, c.ctx, "ann")
	o, err := m.CreateOffer(c.ctx, "ann", market.Offer{PosterID: p.ID, ProductWords: "beans", Price: 10})
	if err != nil {
		t.Fatal(err)
	}
	ord, err := m.BuyNow(c.ctx, "bea", o.ID, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	return ord
}

func TestARunThroughAnOrderLeavesNoRowInAnyLiveTable(t *testing.T) {
	c := newTestCopy(t)
	buyBeans(t, c, c.run(t, "run-1"))

	if got := c.liveRows(t); got != 0 {
		t.Fatalf("live rows = %d after a run, want 0", got)
	}
	rows, err := specstore.RowsOfRun(c.ctx, c.db, c.spec, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	stored := map[string]int64{}
	for _, r := range rows {
		stored[r.Object] = r.Rows
	}
	for _, object := range []string{"poster", "offer", "order", "trail"} {
		if stored[object] == 0 {
			t.Fatalf("run-1 stored nothing in %s: %+v", object, rows)
		}
	}
}

func TestALiveWriteLeavesTheTestCopyEmpty(t *testing.T) {
	c := newTestCopy(t)
	buyBeans(t, c, c.live)

	for _, o := range c.spec.Objects {
		if got := c.count(t, c.spec.TestTableFor(o)); got != 0 {
			t.Fatalf("%s has %d rows after a live write, want 0", c.spec.TestTableFor(o), got)
		}
	}
}

func TestTwoRunsKeepTheirOrdersApart(t *testing.T) {
	c := newTestCopy(t)
	first, second := c.run(t, "run-1"), c.run(t, "run-2")
	ord := buyBeans(t, c, first)
	buyBeans(t, c, second)
	buyBeans(t, c, second)

	firstOrders, err := first.ListOrders(c.ctx, "bea")
	if err != nil {
		t.Fatal(err)
	}
	secondOrders, err := second.ListOrders(c.ctx, "bea")
	if err != nil {
		t.Fatal(err)
	}
	if len(firstOrders) != 1 || firstOrders[0].ID != ord.ID || len(secondOrders) != 2 {
		t.Fatalf("run-1 sees %d orders and run-2 sees %d, want 1 and 2", len(firstOrders), len(secondOrders))
	}
	if _, err := second.GetOrder(c.ctx, "bea", ord.ID); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("run-2 reading run-1's order = %v, want ErrNotFound", err)
	}
}

func TestACancelInARunChangesOnlyThatRunsOrder(t *testing.T) {
	c := newTestCopy(t)
	first, second := c.run(t, "run-1"), c.run(t, "run-2")
	a, b := buyBeans(t, c, first), buyBeans(t, c, second)

	if _, err := first.CancelOrder(c.ctx, "bea", a.ID); err != nil {
		t.Fatal(err)
	}
	got, err := second.GetOrder(c.ctx, "bea", b.ID)
	if err != nil || got.State == models.OrderCancelled {
		t.Fatalf("run-2's order = %+v, %v; want it untouched", got, err)
	}
}

func TestRemovingARunRemovesItsRowsAndKeepsTheOthers(t *testing.T) {
	c := newTestCopy(t)
	first, second := c.run(t, "run-1"), c.run(t, "run-2")
	buyBeans(t, c, first)
	buyBeans(t, c, second)

	removed, err := specstore.RemoveRun(c.ctx, c.db, c.spec, "run-1")
	if err != nil || removed == 0 {
		t.Fatalf("removed %d rows, %v", removed, err)
	}
	if rows, _ := specstore.RowsOfRun(c.ctx, c.db, c.spec, "run-1"); len(rows) != 0 {
		t.Fatalf("run-1 still holds %+v", rows)
	}
	if orders, _ := second.ListOrders(c.ctx, "bea"); len(orders) != 1 {
		t.Fatalf("run-2 lost its order: %d", len(orders))
	}
}

func TestARunWithoutAnIDIsRefused(t *testing.T) {
	c := newTestCopy(t)
	if _, err := c.live.ForRun(""); !errors.Is(err, specstore.ErrRunIDEmpty) {
		t.Fatalf("ForRun(\"\") = %v, want ErrRunIDEmpty", err)
	}
}

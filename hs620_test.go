package market_test

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	market "github.com/aosanya/mwanachama-backend-market"
)

func newCountedManager(t *testing.T) (market.MarketManager, context.Context, *int) {
	t.Helper()
	db := newDB(t)
	reads := 0
	if err := db.Callback().Query().Before("gorm:query").Register("count_reads", func(*gorm.DB) { reads++ }); err != nil {
		t.Fatal(err)
	}
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
	return m, context.Background(), &reads
}

func startOrdersJoinedBy(t *testing.T, m market.MarketManager, ctx context.Context, posterID, picker string, n int) []string {
	t.Helper()
	var ids []string
	for i := 0; i < n; i++ {
		g, err := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: posterID, Price: 3, Minimum: 10})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := m.JoinGroupOrder(ctx, picker, g.ID, 1); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, g.ID)
	}
	return ids
}

func readsOfOneList(t *testing.T, m market.MarketManager, ctx context.Context, reads *int, caller string) int {
	t.Helper()
	before := *reads
	if _, err := m.ListGroupOrders(ctx, caller, 0, ""); err != nil {
		t.Fatal(err)
	}
	return *reads - before
}

func TestAGroupOrderPageReadsTheSameNumberOfTimesWhateverItsSize(t *testing.T) {
	m, ctx, reads := newCountedManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	startOrdersJoinedBy(t, m, ctx, p.ID, "bea", 3)
	small := readsOfOneList(t, m, ctx, reads, "bea")
	startOrdersJoinedBy(t, m, ctx, p.ID, "bea", 40)
	large := readsOfOneList(t, m, ctx, reads, "bea")
	if small != large {
		t.Fatalf("a page of 3 orders read %d times and a page of 43 read %d", small, large)
	}
}

func idsOf(views []market.GroupOrderView) map[string]bool {
	out := map[string]bool{}
	for _, v := range views {
		out[v.ID] = true
	}
	return out
}

func TestAMemberOfSixtyGroupOrdersGetsExactlyHerThreeInOneCall(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	startOrdersJoinedBy(t, m, ctx, p.ID, "cat", 57)
	hers := startOrdersJoinedBy(t, m, ctx, p.ID, "bea", 3)

	got, err := m.ListGroupOrdersBy(ctx, "bea", 0, "", true, "")
	if err != nil || len(got) != 3 {
		t.Fatalf("bea got %d: %v", len(got), err)
	}
	for _, id := range hers {
		if !idsOf(got)[id] {
			t.Fatalf("bea's order %s is missing", id)
		}
	}
	for _, v := range got {
		if len(v.Lines) != 1 || v.Lines[0].Picker != "bea" {
			t.Fatalf("bea sees %+v", v.Lines)
		}
	}
	all, _ := m.ListGroupOrders(ctx, "bea", 0, "")
	if len(all) != 50 {
		t.Fatalf("the page without the filter held %d", len(all))
	}
	if none, _ := m.ListGroupOrdersBy(ctx, "dee", 0, "", true, ""); len(none) != 0 {
		t.Fatalf("a stranger got %d", len(none))
	}
	if none, _ := m.ListGroupOrdersBy(ctx, "", 0, "", true, ""); len(none) != 0 {
		t.Fatalf("nobody got %d", len(none))
	}
}

func TestTheMineFilterIsAppliedBeforeAPageIsCut(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	startOrdersJoinedBy(t, m, ctx, p.ID, "cat", 20)
	hers := startOrdersJoinedBy(t, m, ctx, p.ID, "bea", 3)
	startOrdersJoinedBy(t, m, ctx, p.ID, "cat", 20)

	first, err := m.ListGroupOrdersBy(ctx, "bea", 2, "", true, "")
	if err != nil || len(first) != 2 {
		t.Fatalf("the first page held %d: %v", len(first), err)
	}
	second, err := m.ListGroupOrdersBy(ctx, "bea", 2, first[1].ID, true, "")
	if err != nil || len(second) != 1 {
		t.Fatalf("the second page held %d: %v", len(second), err)
	}
	seen := idsOf(append(first, second...))
	for _, id := range hers {
		if !seen[id] {
			t.Fatalf("bea's order %s is missing", id)
		}
	}
}

func TestMineMeansAnOrderTheCallerHasALineInStartedOrOwnsThePoster(t *testing.T) {
	m, ctx := newManager(t)
	ann := approvedPoster(t, m, ctx, "ann")
	eve := approvedPoster(t, m, ctx, "eve")
	onAnns := startOrdersJoinedBy(t, m, ctx, ann.ID, "bea", 1)[0]
	onEves, err := m.StartGroupOrder(ctx, "coordinator", market.GroupOrder{PosterID: eve.ID, Price: 3, Minimum: 10})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{"bea": onAnns, "coord": onAnns, "ann": onAnns, "coordinator": onEves.ID, "eve": onEves.ID}
	for caller, id := range want {
		got, err := m.ListGroupOrdersBy(ctx, caller, 0, "", true, "")
		if err != nil || len(got) != 1 || got[0].ID != id {
			t.Fatalf("%s got %+v: %v", caller, idsOf(got), err)
		}
	}
	if got, _ := m.ListGroupOrdersBy(ctx, "dee", 0, "", true, ""); len(got) != 0 {
		t.Fatalf("a stranger got %+v", idsOf(got))
	}
}

func TestTheStateFilterListsOnlyOrdersInThatState(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	ids := startOrdersJoinedBy(t, m, ctx, p.ID, "bea", 4)
	for _, id := range ids[:1] {
		if _, err := m.CallOffGroupOrder(ctx, "coord", id); err != nil {
			t.Fatal(err)
		}
	}
	open, err := m.ListGroupOrdersBy(ctx, "bea", 0, "", true, "open")
	if err != nil || len(open) != 3 || idsOf(open)[ids[0]] {
		t.Fatalf("open: %+v %v", idsOf(open), err)
	}
	called, _ := m.ListGroupOrdersBy(ctx, "bea", 0, "", false, "cancelled")
	if len(called) != 1 || called[0].ID != ids[0] {
		t.Fatalf("cancelled: %+v", idsOf(called))
	}
	if _, err := m.ListGroupOrdersBy(ctx, "bea", 0, "", false, "nonsense"); !errors.Is(err, market.ErrInvalid) {
		t.Fatalf("an unknown state: %v", err)
	}
}

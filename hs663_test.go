package market_test

import (
	"context"
	"sort"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
)

func ownersOf(posters []market.Poster) []string {
	out := []string{}
	for _, p := range posters {
		out = append(out, p.Owner)
	}
	sort.Strings(out)
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type shopWorld struct {
	m   market.MarketManager
	ctx context.Context
}

func newShopWorld(t *testing.T) shopWorld {
	t.Helper()
	m, ctx := newManager(t)
	approvedPoster(t, m, ctx, "ann")
	if _, err := m.ApplyPoster(ctx, "cat", market.Poster{Name: "Cat's farm"}); err != nil {
		t.Fatal(err)
	}
	declined, err := m.ApplyPoster(ctx, "dan", market.Poster{Name: "Dan's farm"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.DeclinePoster(ctx, "coordinator", declined.ID); err != nil {
		t.Fatal(err)
	}
	return shopWorld{m: m, ctx: ctx}
}

func TestAFarmersPosterListHoldsApprovedShopsOnly(t *testing.T) {
	w := newShopWorld(t)
	for _, caller := range []string{"bea", ""} {
		got, err := w.m.ListPostersFor(w.ctx, caller, "")
		if err != nil || !sameStrings(ownersOf(got), []string{"ann"}) {
			t.Fatalf("caller %q sees %v, err %v", caller, ownersOf(got), err)
		}
		waiting, err := w.m.ListPostersFor(w.ctx, caller, "pending")
		if err != nil || len(waiting) != 0 {
			t.Fatalf("caller %q asked for pending and sees %v, err %v", caller, ownersOf(waiting), err)
		}
	}
}

func TestAShopsOwnerSeesTheStateOfItsOwnPosterBesideTheApprovedOnes(t *testing.T) {
	w := newShopWorld(t)
	got, err := w.m.ListPostersFor(w.ctx, "cat", "")
	if err != nil || !sameStrings(ownersOf(got), []string{"ann", "cat"}) {
		t.Fatalf("cat sees %v, err %v", ownersOf(got), err)
	}
	own, err := w.m.ListPostersFor(w.ctx, "cat", "pending")
	if err != nil || !sameStrings(ownersOf(own), []string{"cat"}) {
		t.Fatalf("cat asked for pending and sees %v, err %v", ownersOf(own), err)
	}
	declined, err := w.m.ListPostersFor(w.ctx, "dan", "declined")
	if err != nil || !sameStrings(ownersOf(declined), []string{"dan"}) {
		t.Fatalf("dan asked for declined and sees %v, err %v", ownersOf(declined), err)
	}
	other, err := w.m.ListPostersFor(w.ctx, "cat", "declined")
	if err != nil || len(other) != 0 {
		t.Fatalf("cat asked for declined and sees %v, err %v", ownersOf(other), err)
	}
}

func TestThePersonWhoDecidesSeesEveryPosterInEveryState(t *testing.T) {
	w := newShopWorld(t)
	got, err := w.m.ListPostersFor(w.ctx, "coordinator", "")
	if err != nil || !sameStrings(ownersOf(got), []string{"ann", "cat", "dan"}) {
		t.Fatalf("coordinator sees %v, err %v", ownersOf(got), err)
	}
	waiting, err := w.m.ListPostersFor(w.ctx, "coordinator", "pending")
	if err != nil || !sameStrings(ownersOf(waiting), []string{"cat"}) {
		t.Fatalf("coordinator asked for pending and sees %v, err %v", ownersOf(waiting), err)
	}
}

func TestTheUnfilteredPosterListStillHoldsEveryState(t *testing.T) {
	w := newShopWorld(t)
	got, err := w.m.ListPosters(w.ctx, "")
	if err != nil || !sameStrings(ownersOf(got), []string{"ann", "cat", "dan"}) {
		t.Fatalf("sees %v, err %v", ownersOf(got), err)
	}
}

func fullNames(_ context.Context, id string) string {
	return map[string]string{"bea": "Bea Otieno", "dan": "Dan Ouma"}[id]
}

func TestANoticeCarriesNoNeighboursIdAndOnlyTheirFirstName(t *testing.T) {
	m, ctx := newManager(t)
	m.UseNames(fullNames)
	if _, err := m.CreateNotice(ctx, "bea", market.Notice{Wanted: "a ride to Bondo", Quantity: 2, NeededBy: "2026-11-05"}); err != nil {
		t.Fatal(err)
	}
	seen, err := m.ListNoticesFor(ctx, "dan")
	if err != nil || len(seen) != 1 {
		t.Fatalf("notices %+v err %v", seen, err)
	}
	if seen[0].Picker != "" || seen[0].PickerName != "Bea" || seen[0].Wanted != "a ride to Bondo" || seen[0].Quantity != 2 || seen[0].NeededBy != "2026-11-05" {
		t.Fatalf("a neighbour sees %+v", seen[0])
	}
	none, err := m.ListNoticesFor(ctx, "")
	if err != nil || len(none) != 1 || none[0].Picker != "" || none[0].PickerName != "Bea" {
		t.Fatalf("nobody sees %+v err %v", none, err)
	}
}

func TestAPickerSeesTheirOwnNoticeWithTheirIdAndFullName(t *testing.T) {
	m, ctx := newManager(t)
	m.UseNames(fullNames)
	if _, err := m.CreateNotice(ctx, "bea", market.Notice{Wanted: "a ride to Bondo"}); err != nil {
		t.Fatal(err)
	}
	own, err := m.ListNoticesFor(ctx, "bea")
	if err != nil || len(own) != 1 || own[0].Picker != "bea" || own[0].PickerName != "Bea Otieno" {
		t.Fatalf("own %+v err %v", own, err)
	}
}

func TestANoticeWithNoNameSuppliedCarriesNoNameAndNoId(t *testing.T) {
	m, ctx := newManager(t)
	if _, err := m.CreateNotice(ctx, "bea", market.Notice{Wanted: "a ride to Bondo"}); err != nil {
		t.Fatal(err)
	}
	seen, err := m.ListNoticesFor(ctx, "dan")
	if err != nil || len(seen) != 1 || seen[0].Picker != "" || seen[0].PickerName != "" {
		t.Fatalf("a neighbour sees %+v err %v", seen, err)
	}
}

func TestTheUnfilteredNoticeListKeepsThePickerForTheHost(t *testing.T) {
	m, ctx := newManager(t)
	if _, err := m.CreateNotice(ctx, "bea", market.Notice{Wanted: "a ride to Bondo"}); err != nil {
		t.Fatal(err)
	}
	all, err := m.ListNotices(ctx)
	if err != nil || len(all) != 1 || all[0].Picker != "bea" {
		t.Fatalf("all %+v err %v", all, err)
	}
}

package market_test

import (
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
)

func TestARequestIsReadByItsPickerAndByAPosterAskedAboutIt(t *testing.T) {
	m, ctx := newManager(t)
	ann := approvedPoster(t, m, ctx, "ann")
	approvedPoster(t, m, ctx, "dan")
	r, _ := m.CreateRequest(ctx, "bea", market.Request{Wanted: "beans", Quantity: 5, NeededBy: "2026-11-01"})
	got, err := m.GetRequest(ctx, "bea", r.ID)
	if err != nil || got.Wanted != "beans" || got.Quantity != 5 || got.NeededBy != "2026-11-01" || got.Picker != "bea" {
		t.Fatalf("picker read %+v err %v", got, err)
	}
	if _, err := m.GetRequest(ctx, "ann", r.ID); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("a poster not yet asked read it: %v", err)
	}
	if _, err := m.CreateAsk(ctx, "bea", market.Ask{PosterID: ann.ID, RequestID: r.ID, Question: "have any?"}); err != nil {
		t.Fatal(err)
	}
	if got, err := m.GetRequest(ctx, "ann", r.ID); err != nil || got.ID != r.ID {
		t.Fatalf("the asked poster read %+v err %v", got, err)
	}
	for _, caller := range []string{"dan", "cat", ""} {
		if _, err := m.GetRequest(ctx, caller, r.ID); !errors.Is(err, market.ErrNotFound) {
			t.Fatalf("%q read it: %v", caller, err)
		}
	}
	if _, err := m.GetRequest(ctx, "bea", "nope"); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("unknown id: %v", err)
	}
}

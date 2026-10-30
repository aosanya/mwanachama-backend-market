package market_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/identity"

	market "github.com/aosanya/mwanachama-backend-market"
)

var _ identity.Owner = market.MarketManager(nil)

func holdOneOfEach(t *testing.T, m market.MarketManager, ctx context.Context) map[string]string {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	poster := approvedPoster(t, m, ctx, "ann")
	offer, err := m.CreateOffer(ctx, "ann", market.Offer{PosterID: poster.ID, ProductWords: "maharagwe", UnitWords: "kg", Price: 120, Product: "beans"})
	must(err)
	request, err := m.CreateRequest(ctx, "bea", market.Request{Wanted: "beans", Quantity: 5})
	must(err)
	ask, err := m.CreateAsk(ctx, "bea", market.Ask{PosterID: poster.ID, RequestID: request.ID, Question: "have you any?"})
	must(err)
	notice, err := m.CreateNotice(ctx, "bea", market.Notice{Wanted: "40 bags of maize", Quantity: 40})
	must(err)
	quote, err := m.SendQuote(ctx, "ann", notice.ID, market.Quote{PosterID: poster.ID, Price: 90})
	must(err)
	order, err := m.BuyNow(ctx, "bea", offer.ID, 2, "")
	must(err)
	trail, err := m.OrderTrail(ctx, "bea", order.ID)
	must(err)
	if len(trail) == 0 {
		t.Fatal("an order should leave a trail")
	}
	group, err := m.StartGroupOrder(ctx, "coordinator", market.GroupOrder{PosterID: poster.ID, Price: 3, Minimum: 10})
	must(err)
	line, err := m.JoinGroupOrder(ctx, "bea", group.ID, 4)
	must(err)
	stock, err := m.SetStock(ctx, "ann", market.Stock{PosterID: poster.ID, Product: "beans", Count: 9})
	must(err)
	return map[string]string{
		"poster": poster.ID, "offer": offer.ID, "request": request.ID, "ask": ask.ID, "notice": notice.ID,
		"quote": quote.ID, "order": order.ID, "trail": trail[0].ID, "group_order": group.ID, "line": line.ID, "stock": stock.ID,
	}
}

func TestStableIdentityIsTheRawMarketInstanceMountObjectAndRowId(t *testing.T) {
	m, ctx := newManager(t)
	for object, id := range holdOneOfEach(t, m, ctx) {
		got, err := m.StableIdentity(ctx, object, id)
		if err != nil {
			t.Fatalf("StableIdentity(%s): %v", object, err)
		}
		if want := "market/app/main/" + object + "/" + id; got != want {
			t.Errorf("StableIdentity(%s) = %q, want %q", object, got, want)
		}
	}
}

func TestResolveGivesBackTheObjectAndIdStableIdentityWasAskedFor(t *testing.T) {
	m, ctx := newManager(t)
	for object, id := range holdOneOfEach(t, m, ctx) {
		raw, err := m.StableIdentity(ctx, object, id)
		if err != nil {
			t.Fatalf("StableIdentity(%s): %v", object, err)
		}
		gotObject, gotID, err := m.Resolve(ctx, raw)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", raw, err)
		}
		if gotObject != object || gotID != id {
			t.Errorf("Resolve(%q) = (%q, %q), want (%q, %q)", raw, gotObject, gotID, object, id)
		}
	}
}

func TestAnOffersStableIdentityIsTheSameAfterItIsEditedAndRegrouped(t *testing.T) {
	m, o, ctx := groupedOffer(t)
	before, err := m.StableIdentity(ctx, "offer", o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.RegroupOffer(ctx, "coordinator", o.ID, "cowpeas"); err != nil {
		t.Fatal(err)
	}
	o.Price = 130
	if _, err := m.UpdateOffer(ctx, "ann", o); err != nil {
		t.Fatal(err)
	}
	after, err := m.StableIdentity(ctx, "offer", o.ID)
	if err != nil || before != after {
		t.Fatalf("identity moved from %q to %q (%v)", before, after, err)
	}
}

func TestStableIdentityNamesAMissAndAnObjectItDoesNotKnow(t *testing.T) {
	m, ctx := newManager(t)
	if _, err := m.StableIdentity(ctx, "poster", "missing"); !errors.Is(err, identity.ErrNotFound) {
		t.Errorf("a missing row: err = %v, want identity.ErrNotFound", err)
	}
	if _, err := m.StableIdentity(ctx, "invoice", "any"); !errors.Is(err, market.ErrNoStableIdentity) {
		t.Errorf("an object the market does not hold: err = %v, want ErrNoStableIdentity", err)
	}
}

func TestResolveNamesWhyARawIsNotAnswered(t *testing.T) {
	m, ctx := newManager(t)
	poster := approvedPoster(t, m, ctx, "ann")
	raw, err := m.StableIdentity(ctx, "poster", poster.ID)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct {
		raw  string
		want error
	}{
		"a string that does not parse":  {"not a raw", identity.ErrInvalidRaw},
		"another module's raw":          {strings.Replace(raw, "market/", "agency/", 1), identity.ErrNotOwned},
		"another instance's raw":        {strings.Replace(raw, "/app/", "/other/", 1), identity.ErrNotOwned},
		"another mount's raw":           {strings.Replace(raw, "/main/", "/harvest/", 1), identity.ErrNotOwned},
		"an identity that is not three": {"market/app/main/" + poster.ID, identity.ErrNotFound},
		"an object the market lacks":    {"market/app/main/invoice/" + poster.ID, identity.ErrNotFound},
		"a row that was never written":  {"market/app/main/poster/missing", identity.ErrNotFound},
	}
	for name, c := range cases {
		if _, _, err := m.Resolve(ctx, c.raw); !errors.Is(err, c.want) {
			t.Errorf("%s: Resolve(%q) err = %v, want %v", name, c.raw, err, c.want)
		}
	}
}

func TestTwoMountsOfOneInstanceDoNotAnswerForEachOthersThings(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()
	inputs := mountedMarket(t, db, "inputs")
	harvest := mountedMarket(t, db, "harvest")
	poster := approvedPoster(t, inputs, ctx, "ann")
	raw, err := inputs.StableIdentity(ctx, "poster", poster.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := "market/app/inputs/poster/" + poster.ID; raw != want {
		t.Fatalf("raw = %q, want %q", raw, want)
	}
	if _, _, err := harvest.Resolve(ctx, raw); !errors.Is(err, identity.ErrNotOwned) {
		t.Fatalf("the other mount: err = %v, want identity.ErrNotOwned", err)
	}
	if _, id, err := inputs.Resolve(ctx, raw); err != nil || id != poster.ID {
		t.Fatalf("its own mount: id = %q err = %v", id, err)
	}
}

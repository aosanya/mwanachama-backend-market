package market_test

import (
	"context"
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
)

type shelf struct {
	m   market.MarketManager
	ctx context.Context
	pid string
}

func newShelf(t *testing.T) shelf {
	t.Helper()
	m, ctx := newManager(t)
	return shelf{m: m, ctx: ctx, pid: approvedPoster(t, m, ctx, "ann").ID}
}

func (s shelf) offer(t *testing.T, product, category string) market.Offer {
	t.Helper()
	o, err := s.m.CreateOffer(s.ctx, "ann", market.Offer{PosterID: s.pid, ProductWords: product + " words", Price: 10, Product: product, Category: category})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func (s shelf) ask(t *testing.T, picker, product string) {
	t.Helper()
	if _, err := s.m.CreateAsk(s.ctx, picker, market.Ask{PosterID: s.pid, Product: product, Question: "have any?"}); err != nil {
		t.Fatal(err)
	}
}

func categories(kinds []market.Kind) []string {
	out := []string{}
	for _, k := range kinds {
		out = append(out, k.Category)
	}
	return out
}

func TestAnOfferCarriesTheCategoryTheAIGaveItsProduct(t *testing.T) {
	s := newShelf(t)
	o := s.offer(t, "dap", " fertiliser ")
	if o.Category != "fertiliser" {
		t.Fatalf("category %q", o.Category)
	}
	offers, _ := s.m.ListOffers(s.ctx, "")
	if len(offers) != 1 || offers[0].Category != "fertiliser" {
		t.Fatalf("stored %+v", offers)
	}
}

func TestAnOfferGroupedUnderAProductTakesTheCategoryTheProductAlreadyHas(t *testing.T) {
	s := newShelf(t)
	s.offer(t, "dap", "fertiliser")
	if got := s.offer(t, "dap", ""); got.Category != "fertiliser" {
		t.Fatalf("category %q", got.Category)
	}
	if got := s.offer(t, "beans", ""); got.Category != "" {
		t.Fatalf("a new product has no category yet, got %q", got.Category)
	}
}

func TestCorrectingAProductsCategoryMovesEveryOfferUnderIt(t *testing.T) {
	s := newShelf(t)
	a, b := s.offer(t, "dap", "seed"), s.offer(t, "dap", "seed")
	other := s.offer(t, "beans", "seed")
	got, err := s.m.CategoriseProduct(s.ctx, "coordinator", " dap ", " fertiliser ")
	if err != nil || len(got) != 2 {
		t.Fatalf("%+v %v", got, err)
	}
	offers, _ := s.m.ListOffers(s.ctx, "")
	for _, o := range offers {
		want := "fertiliser"
		if o.ID == other.ID {
			want = "seed"
		}
		if o.Category != want || (o.ID == a.ID && o.Price != a.Price) || (o.ID == b.ID && o.Product != "dap") {
			t.Fatalf("offer %+v want category %s", o, want)
		}
	}
}

func TestCorrectingACategoryRefusesTheEmptyTheUnknownAndTheSame(t *testing.T) {
	s := newShelf(t)
	s.offer(t, "dap", "fertiliser")
	cases := []struct {
		product, category string
		want              error
	}{
		{"dap", "", market.ErrInvalid},
		{"", "seed", market.ErrInvalid},
		{"nothing", "seed", market.ErrNotFound},
		{"dap", "fertiliser", market.ErrConflict},
	}
	for _, c := range cases {
		if _, err := s.m.CategoriseProduct(s.ctx, "coordinator", c.product, c.category); !errors.Is(err, c.want) {
			t.Fatalf("%+v: %v", c, err)
		}
	}
	if _, err := s.m.CategoriseProduct(s.ctx, "", "dap", "seed"); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("no caller: %v", err)
	}
}

func TestMovingAnOfferUnderAnotherProductGivesItThatProductsCategory(t *testing.T) {
	s := newShelf(t)
	s.offer(t, "dap", "fertiliser")
	o := s.offer(t, "urea", "seed")
	got, err := s.m.RegroupOffer(s.ctx, "coordinator", o.ID, "dap")
	if err != nil || got.Category != "fertiliser" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestMovingAnOfferUnderAProductWithNoCategoryKeepsItsOwn(t *testing.T) {
	s := newShelf(t)
	s.offer(t, "beans", "")
	o := s.offer(t, "urea", "fertiliser")
	got, err := s.m.RegroupOffer(s.ctx, "coordinator", o.ID, "beans")
	if err != nil || got.Category != "fertiliser" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestKindsAreTheCategoriesAskedForMostWithTheProductAskedForMostInEach(t *testing.T) {
	s := newShelf(t)
	s.offer(t, "dap", "fertiliser")
	s.offer(t, "urea", "fertiliser")
	s.offer(t, "h614", "seed")
	s.offer(t, "beans", "grain")
	s.ask(t, "bea", "dap")
	s.ask(t, "cy", "urea")
	s.ask(t, "di", "urea")
	s.ask(t, "ed", "h614")
	kinds, err := s.m.ListKinds(s.ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := categories(kinds); len(got) != 3 || got[0] != "fertiliser" || got[1] != "seed" || got[2] != "grain" {
		t.Fatalf("%+v", kinds)
	}
	if kinds[0].Asked != 3 || kinds[0].Product != "urea" || kinds[0].Offers != 2 || kinds[2].Asked != 0 {
		t.Fatalf("%+v", kinds)
	}
}

func TestAnAskThatNamesAnOfferCountsForTheOffersProduct(t *testing.T) {
	s := newShelf(t)
	s.offer(t, "dap", "fertiliser")
	o := s.offer(t, "h614", "seed")
	s.offer(t, "beans", "grain")
	if _, err := s.m.CreateAsk(s.ctx, "bea", market.Ask{PosterID: s.pid, OfferID: o.ID, Question: "how much?"}); err != nil {
		t.Fatal(err)
	}
	kinds, _ := s.m.ListKinds(s.ctx, 1)
	if len(kinds) != 1 || kinds[0].Category != "seed" || kinds[0].Asked != 1 || kinds[0].Product != "h614" {
		t.Fatalf("%+v", kinds)
	}
}

func TestOrdersCountAsAskingToo(t *testing.T) {
	s := newShelf(t)
	s.offer(t, "dap", "fertiliser")
	o := s.offer(t, "h614", "seed")
	if _, err := s.m.BuyNow(s.ctx, "bea", o.ID, 2, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.m.BuyNow(s.ctx, "cy", o.ID, 2, ""); err != nil {
		t.Fatal(err)
	}
	kinds, _ := s.m.ListKinds(s.ctx, 3)
	if kinds[0].Category != "seed" || kinds[0].Asked != 2 {
		t.Fatalf("%+v", kinds)
	}
}

func TestKindsAreBoundedAndAnOfferWithNoCategoryIsNotAKind(t *testing.T) {
	s := newShelf(t)
	for _, c := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"} {
		s.offer(t, "p-"+c, c)
	}
	s.offer(t, "loose", "")
	for limit, want := range map[int]int{0: 3, 1: 1, 5: 5, 10: 10, 500: 10} {
		kinds, err := s.m.ListKinds(s.ctx, limit)
		if err != nil || len(kinds) != want {
			t.Fatalf("limit %d: %d kinds %v", limit, len(kinds), err)
		}
	}
	if _, err := s.m.ListKinds(s.ctx, -1); !errors.Is(err, market.ErrInvalid) {
		t.Fatalf("negative: %v", err)
	}
}

func TestAMarketWithNoCategoriesOffersNoKinds(t *testing.T) {
	s := newShelf(t)
	s.offer(t, "loose", "")
	kinds, err := s.m.ListKinds(s.ctx, 3)
	if err != nil || len(kinds) != 0 {
		t.Fatalf("%+v %v", kinds, err)
	}
}

func TestAskedForMostFollowsACorrectionToTheCategory(t *testing.T) {
	s := newShelf(t)
	s.offer(t, "dap", "seed")
	s.offer(t, "h614", "seed")
	s.offer(t, "urea", "fertiliser")
	s.ask(t, "bea", "dap")
	s.ask(t, "cy", "dap")
	s.ask(t, "di", "urea")
	if _, err := s.m.CategoriseProduct(s.ctx, "coordinator", "dap", "fertiliser"); err != nil {
		t.Fatal(err)
	}
	kinds, _ := s.m.ListKinds(s.ctx, 3)
	if kinds[0].Category != "fertiliser" || kinds[0].Asked != 3 || kinds[0].Product != "dap" {
		t.Fatalf("%+v", kinds)
	}
}

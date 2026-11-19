package market

import (
	"context"
	"errors"
	"testing"
)

func offerUnder(t *testing.T, m *marketManager, owner, posterID, product, category string) Offer {
	t.Helper()
	o, err := m.CreateOffer(context.Background(), owner, Offer{PosterID: posterID, ProductWords: product, Price: 1, Product: product, Category: category})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestCategoriseProductRefusesAnonymousCallerAndBlankNames(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	if _, err := m.CategoriseProduct(ctx, "", "maize", "grain"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("anonymous caller: %v", err)
	}
	for _, args := range [][2]string{{"", "grain"}, {"maize", ""}, {"  ", "grain"}, {"maize", "  "}} {
		if _, err := m.CategoriseProduct(ctx, "cy", args[0], args[1]); !errors.Is(err, ErrInvalid) {
			t.Fatalf("%q: %v", args, err)
		}
	}
}

func TestCategoriseProductOnAnUnknownProductIsNotFound(t *testing.T) {
	m := newPosterFixture(t)
	if _, err := m.CategoriseProduct(context.Background(), "cy", "maize", "grain"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestCategoriseProductMovesEveryOfferOfTheProductAndRepeatingItConflicts(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	ada := approvedPoster(t, m, "ada", "Ada Farm")
	bob := approvedPoster(t, m, "bob", "Bob Shop")
	offerUnder(t, m, "ada", ada.ID, "maize", "")
	offerUnder(t, m, "bob", bob.ID, "maize", "veg")
	moved, err := m.CategoriseProduct(ctx, "cy", " maize ", " grain ")
	if err != nil {
		t.Fatal(err)
	}
	if len(moved) != 2 || moved[0].Category != "grain" || moved[1].Category != "grain" {
		t.Fatalf("offers not all moved: %+v", moved)
	}
	again, err := m.CategoriseProduct(ctx, "cy", "maize", "grain")
	if !errors.Is(err, ErrConflict) || len(again) != 2 {
		t.Fatalf("repeat: %v %+v", err, again)
	}
	listed, err := m.ListOffers(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range listed {
		if o.Category != "grain" {
			t.Fatalf("stored category %q", o.Category)
		}
	}
}

func TestCategoriseProductStillMovesWhenOnlySomeOffersAreAlreadyUnderIt(t *testing.T) {
	m := newPosterFixture(t)
	ada := approvedPoster(t, m, "ada", "Ada Farm")
	offerUnder(t, m, "ada", ada.ID, "maize", "grain")
	offerUnder(t, m, "ada", ada.ID, "maize", "veg")
	moved, err := m.CategoriseProduct(context.Background(), "cy", "maize", "grain")
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range moved {
		if o.Category != "grain" {
			t.Fatalf("left behind: %+v", o)
		}
	}
}

func TestListKindsRefusesNegativeLimit(t *testing.T) {
	m := newPosterFixture(t)
	if _, err := m.ListKinds(context.Background(), -1); !errors.Is(err, ErrInvalid) {
		t.Fatalf("got %v", err)
	}
}

func TestListKindsIsEmptyWithoutCategorisedOffers(t *testing.T) {
	m := newPosterFixture(t)
	ada := approvedPoster(t, m, "ada", "Ada Farm")
	offerUnder(t, m, "ada", ada.ID, "maize", "")
	kinds, err := m.ListKinds(context.Background(), 0)
	if err != nil || len(kinds) != 0 {
		t.Fatalf("%v %+v", err, kinds)
	}
}

func TestListKindsRanksByAskedThenOffersThenNameAndNamesTheTopProduct(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	ada := approvedPoster(t, m, "ada", "Ada Farm")
	offerUnder(t, m, "ada", ada.ID, "maize", "grain")
	offerUnder(t, m, "ada", ada.ID, "rice", "grain")
	offerUnder(t, m, "ada", ada.ID, "kale", "veg")
	offerUnder(t, m, "ada", ada.ID, "beans", "pulse")
	offerUnder(t, m, "ada", ada.ID, "peas", "pulse")
	for _, picker := range []string{"c1", "c2"} {
		if _, err := m.CreateAsk(ctx, picker, Ask{PosterID: ada.ID, Question: "any?", Product: "rice"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := m.CreateAsk(ctx, "c3", Ask{PosterID: ada.ID, Question: "any?", Product: "maize"}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.CreateAsk(ctx, "c4", Ask{PosterID: ada.ID, Question: "any?", Product: "unlisted"}); err != nil {
		t.Fatal(err)
	}
	kinds, err := m.ListKinds(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(kinds) != 3 {
		t.Fatalf("default limit gives three: %+v", kinds)
	}
	if kinds[0].Category != "grain" || kinds[0].Asked != 3 || kinds[0].Offers != 2 || kinds[0].Product != "rice" {
		t.Fatalf("first: %+v", kinds[0])
	}
	if kinds[1].Category != "pulse" || kinds[1].Asked != 0 || kinds[1].Offers != 2 || kinds[1].Product != "" {
		t.Fatalf("second: %+v", kinds[1])
	}
	if kinds[2].Category != "veg" || kinds[2].Offers != 1 {
		t.Fatalf("third: %+v", kinds[2])
	}
	one, err := m.ListKinds(ctx, 1)
	if err != nil || len(one) != 1 || one[0].Category != "grain" {
		t.Fatalf("limit 1: %v %+v", err, one)
	}
}

func TestListKindsCapsTheLimitAtTen(t *testing.T) {
	m := newPosterFixture(t)
	ada := approvedPoster(t, m, "ada", "Ada Farm")
	for _, c := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"} {
		offerUnder(t, m, "ada", ada.ID, "p"+c, c)
	}
	kinds, err := m.ListKinds(context.Background(), 99)
	if err != nil || len(kinds) != 10 {
		t.Fatalf("%v %d", err, len(kinds))
	}
	if kinds[0].Category != "a" || kinds[9].Category != "j" {
		t.Fatalf("ties are by category name: %+v", kinds)
	}
}

func TestAnAskNamingNoProductCountsThroughItsOffer(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	ada := approvedPoster(t, m, "ada", "Ada Farm")
	o := offerUnder(t, m, "ada", ada.ID, "maize", "grain")
	if _, err := m.CreateAsk(ctx, "c1", Ask{PosterID: ada.ID, Question: "any?", OfferID: o.ID}); err != nil {
		t.Fatal(err)
	}
	kinds, err := m.ListKinds(ctx, 0)
	if err != nil || len(kinds) != 1 || kinds[0].Asked != 1 || kinds[0].Product != "maize" {
		t.Fatalf("%v %+v", err, kinds)
	}
}

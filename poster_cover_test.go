package market

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aosanya/mwanachama-backend-market/models"
)

func newPosterFixture(t *testing.T) *marketManager {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "market.db") + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	s, err := SpecFor("postertest")
	if err != nil {
		t.Fatal(err)
	}
	if err := Provision(db, s); err != nil {
		t.Fatal(err)
	}
	mgr, err := NewMarketManager(db, s)
	if err != nil {
		t.Fatal(err)
	}
	m := mgr.(*marketManager)
	m.UseCoordinators(func(_ context.Context, caller string) bool { return caller == "admin" })
	m.UseNames(func(_ context.Context, id string) string { return "name-of-" + id })
	return m
}

func applyPoster(t *testing.T, m *marketManager, owner, name string) Poster {
	t.Helper()
	p, err := m.ApplyPoster(context.Background(), owner, Poster{Name: name})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func approvedPoster(t *testing.T, m *marketManager, owner, name string) Poster {
	t.Helper()
	p := applyPoster(t, m, owner, name)
	p, err := m.ApprovePoster(context.Background(), "admin", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestApplyPosterStartsPendingAndAvailableWhateverTheBodyClaims(t *testing.T) {
	m := newPosterFixture(t)
	p, err := m.ApplyPoster(context.Background(), "ada", Poster{ID: "forged", Name: "Ada Farm", Owner: "mallory",
		ApprovedBy: "mallory", Status: models.PosterApproved, Availability: models.Away})
	if err != nil {
		t.Fatal(err)
	}
	if p.ID == "forged" || p.Owner != "ada" || p.ApprovedBy != "" {
		t.Fatalf("server-owned fields were taken from the body: %+v", p)
	}
	if p.Status != models.PosterPending || p.Availability != models.Available {
		t.Fatalf("status %q availability %q", p.Status, p.Availability)
	}
	if p.OwnerName != "name-of-ada" {
		t.Fatalf("owner name %q", p.OwnerName)
	}
}

func TestApplyPosterRefusesAnonymousAndNamelessApplicants(t *testing.T) {
	m := newPosterFixture(t)
	if _, err := m.ApplyPoster(context.Background(), "", Poster{Name: "x"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("anonymous apply: %v", err)
	}
	if _, err := m.ApplyPoster(context.Background(), "ada", Poster{}); err == nil {
		t.Fatal("a poster with no name was accepted")
	}
}

func TestOnlyADeciderApprovesDeclinesOrRemovesAPoster(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	p := applyPoster(t, m, "ada", "Ada Farm")
	for name, call := range map[string]func(context.Context, string, string) (Poster, error){
		"approve": m.ApprovePoster, "decline": m.DeclinePoster, "remove": m.RemovePoster,
	} {
		for _, caller := range []string{"ada", "bob", ""} {
			if _, err := call(ctx, caller, p.ID); !errors.Is(err, ErrForbidden) {
				t.Fatalf("%s by %q: %v", name, caller, err)
			}
		}
	}
	got, err := m.ListPosters(ctx, "")
	if err != nil || len(got) != 1 || got[0].Status != models.PosterPending {
		t.Fatalf("refused decisions changed the poster: %+v %v", got, err)
	}
}

func TestDecisionOnAnUnknownPosterIsNotFound(t *testing.T) {
	m := newPosterFixture(t)
	if _, err := m.ApprovePoster(context.Background(), "admin", "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestApprovingRecordsTheApproverAndDecliningLeavesNone(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	a := applyPoster(t, m, "ada", "Ada Farm")
	b := applyPoster(t, m, "bob", "Bob Shop")
	a, err := m.ApprovePoster(ctx, "admin", a.ID)
	if err != nil || a.Status != models.PosterApproved || a.ApprovedBy != "admin" || a.ApprovedByName != "name-of-admin" {
		t.Fatalf("approve: %+v %v", a, err)
	}
	b, err = m.DeclinePoster(ctx, "admin", b.ID)
	if err != nil || b.Status != models.PosterDeclined || b.ApprovedBy != "" {
		t.Fatalf("decline: %+v %v", b, err)
	}
	a, err = m.RemovePoster(ctx, "admin", a.ID)
	if err != nil || a.Status != models.PosterRemoved {
		t.Fatalf("remove: %+v %v", a, err)
	}
}

func TestListPostersFiltersByStatusAndNamesPeople(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	approvedPoster(t, m, "ada", "Ada Farm")
	applyPoster(t, m, "bob", "Bob Shop")
	pending, err := m.ListPosters(ctx, models.PosterPending)
	if err != nil || len(pending) != 1 || pending[0].Name != "Bob Shop" || pending[0].OwnerName != "name-of-bob" {
		t.Fatalf("pending: %+v %v", pending, err)
	}
	all, err := m.ListPosters(ctx, "")
	if err != nil || len(all) != 2 {
		t.Fatalf("all: %+v %v", all, err)
	}
}

func TestListPostersForShowsEachCallerOnlyWhatTheyMaySee(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	approvedPoster(t, m, "ada", "Ada Farm")
	applyPoster(t, m, "bob", "Bob Shop")
	cases := []struct {
		caller string
		want   []string
	}{{"", []string{"Ada Farm"}}, {"cy", []string{"Ada Farm"}}, {"bob", []string{"Ada Farm", "Bob Shop"}}, {"admin", []string{"Ada Farm", "Bob Shop"}}}
	for _, c := range cases {
		got, err := m.ListPostersFor(ctx, c.caller, "")
		if err != nil {
			t.Fatal(err)
		}
		names := map[string]bool{}
		for _, p := range got {
			names[p.Name] = true
		}
		if len(got) != len(c.want) {
			t.Fatalf("caller %q saw %d posters", c.caller, len(got))
		}
		for _, w := range c.want {
			if !names[w] {
				t.Fatalf("caller %q did not see %s", c.caller, w)
			}
		}
	}
}

func TestListPostersForHidesTheApproverFromEverybodyButDeciders(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	approvedPoster(t, m, "ada", "Ada Farm")
	seen, _ := m.ListPostersFor(ctx, "cy", "")
	if len(seen) != 1 || seen[0].ApprovedBy != "" {
		t.Fatalf("a member saw the approver: %+v", seen)
	}
	decider, _ := m.ListPostersFor(ctx, "admin", "")
	if len(decider) != 1 || decider[0].ApprovedBy != "admin" {
		t.Fatalf("a decider did not see the approver: %+v", decider)
	}
}

func TestSetAvailabilityIsTheOwnersAndHeldToTheDeclaredValues(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	p := approvedPoster(t, m, "ada", "Ada Farm")
	if _, err := m.SetAvailability(ctx, "bob", p.ID, models.Away); !errors.Is(err, ErrForbidden) {
		t.Fatalf("another member: %v", err)
	}
	if _, err := m.SetAvailability(ctx, "ada", p.ID, "sleeping"); err == nil {
		t.Fatal("an undeclared availability was accepted")
	}
	got, err := m.SetAvailability(ctx, "ada", p.ID, models.Away)
	if err != nil || got.Availability != models.Away || got.OwnerName != "name-of-ada" {
		t.Fatalf("away: %+v %v", got, err)
	}
	listed, _ := m.ListPosters(ctx, "")
	if listed[0].Availability != models.Away {
		t.Fatalf("not stored: %+v", listed[0])
	}
}

func TestOfferNeedsAnApprovedPosterOwnedByTheCaller(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	pending := applyPoster(t, m, "ada", "Ada Farm")
	offer := Offer{PosterID: pending.ID, ProductWords: "maize", Price: 10, Product: "maize"}
	if _, err := m.CreateOffer(ctx, "ada", offer); !errors.Is(err, ErrPosterNotReady) {
		t.Fatalf("pending poster: %v", err)
	}
	ready := approvedPoster(t, m, "bob", "Bob Shop")
	offer.PosterID = ready.ID
	if _, err := m.CreateOffer(ctx, "ada", offer); !errors.Is(err, ErrForbidden) {
		t.Fatalf("not the owner: %v", err)
	}
	if _, err := m.CreateOffer(ctx, "", offer); !errors.Is(err, ErrForbidden) {
		t.Fatalf("anonymous: %v", err)
	}
	if _, err := m.CreateOffer(ctx, "bob", Offer{PosterID: "missing"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown poster: %v", err)
	}
}

func TestCreateOfferTrimsAndTakesTheCategoryOfAnEarlierOfferOnTheProduct(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	p := approvedPoster(t, m, "ada", "Ada Farm")
	first, err := m.CreateOffer(ctx, "ada", Offer{ID: "forged", PosterID: p.ID, ProductWords: "maize", Price: 10, Product: "  maize ", Category: " grain "})
	if err != nil || first.ID == "forged" || first.Product != "maize" || first.Category != "grain" {
		t.Fatalf("first: %+v %v", first, err)
	}
	second, err := m.CreateOffer(ctx, "ada", Offer{PosterID: p.ID, ProductWords: "maize again", Price: 12, Product: "maize"})
	if err != nil || second.Category != "grain" {
		t.Fatalf("category not inherited: %+v %v", second, err)
	}
	listed, err := m.ListOffers(ctx, p.ID)
	if err != nil || len(listed) != 2 {
		t.Fatalf("list: %+v %v", listed, err)
	}
	other, _ := m.ListOffers(ctx, "elsewhere")
	if len(other) != 0 {
		t.Fatalf("offers leaked across posters: %+v", other)
	}
}

func TestUpdateOfferKeepsItsPosterAndRecomputesTheCategoryOnlyWhenTheProductMoves(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	p := approvedPoster(t, m, "ada", "Ada Farm")
	maize, _ := m.CreateOffer(ctx, "ada", Offer{PosterID: p.ID, ProductWords: "maize", Price: 10, Product: "maize", Category: "grain"})
	m.CreateOffer(ctx, "ada", Offer{PosterID: p.ID, ProductWords: "beans", Price: 20, Product: "beans", Category: "pulse"})
	if _, err := m.UpdateOffer(ctx, "bob", Offer{ID: maize.ID, ProductWords: "x", Price: 1}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("not the owner: %v", err)
	}
	if _, err := m.UpdateOffer(ctx, "ada", Offer{ID: "missing"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown offer: %v", err)
	}
	cheaper, err := m.UpdateOffer(ctx, "ada", Offer{ID: maize.ID, PosterID: "forged", ProductWords: "maize", Price: 8})
	if err != nil || cheaper.PosterID != p.ID || cheaper.Product != "maize" || cheaper.Category != "grain" || cheaper.Price != 8 {
		t.Fatalf("price change: %+v %v", cheaper, err)
	}
	moved, err := m.UpdateOffer(ctx, "ada", Offer{ID: maize.ID, ProductWords: "beans", Price: 8, Product: "beans"})
	if err != nil || moved.Category != "pulse" {
		t.Fatalf("category after moving product: %+v %v", moved, err)
	}
}

func TestRegroupOfferRefusalsAndCategoryFollowsTheNewProduct(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	p := approvedPoster(t, m, "ada", "Ada Farm")
	maize, _ := m.CreateOffer(ctx, "ada", Offer{PosterID: p.ID, ProductWords: "maize", Price: 10, Product: "maize", Category: "grain"})
	m.CreateOffer(ctx, "ada", Offer{PosterID: p.ID, ProductWords: "beans", Price: 20, Product: "beans", Category: "pulse"})
	if _, err := m.RegroupOffer(ctx, "", maize.ID, "beans"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("anonymous: %v", err)
	}
	if _, err := m.RegroupOffer(ctx, "ada", maize.ID, "   "); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank product: %v", err)
	}
	if _, err := m.RegroupOffer(ctx, "ada", "missing", "beans"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown offer: %v", err)
	}
	if _, err := m.RegroupOffer(ctx, "ada", maize.ID, "maize"); !errors.Is(err, ErrConflict) {
		t.Fatalf("same product: %v", err)
	}
	got, err := m.RegroupOffer(ctx, "ada", maize.ID, " beans ")
	if err != nil || got.Product != "beans" || got.Category != "pulse" {
		t.Fatalf("regroup: %+v %v", got, err)
	}
	fresh, err := m.RegroupOffer(ctx, "ada", maize.ID, "millet")
	if err != nil || fresh.Product != "millet" || fresh.Category != "pulse" {
		t.Fatalf("a product with no category keeps the old one: %+v %v", fresh, err)
	}
}

func TestStockIsTheOwnersAndASecondSetReplacesTheFirst(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	p := approvedPoster(t, m, "ada", "Ada Farm")
	if _, err := m.SetStock(ctx, "bob", Stock{PosterID: p.ID, Product: "maize", Count: 5}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("not the owner: %v", err)
	}
	first, err := m.SetStock(ctx, "ada", Stock{PosterID: p.ID, Product: "maize", Count: 5})
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.SetStock(ctx, "ada", Stock{PosterID: p.ID, Product: "maize", Count: 2})
	if err != nil || second.ID != first.ID {
		t.Fatalf("a second set made a new row: %+v %+v %v", first, second, err)
	}
	if _, err := m.SetStock(ctx, "ada", Stock{PosterID: p.ID, Product: "beans", Count: 9}); err != nil {
		t.Fatal(err)
	}
	rows, err := m.GetStock(ctx, "ada", p.ID)
	if err != nil || len(rows) != 2 {
		t.Fatalf("stock rows: %+v %v", rows, err)
	}
	for _, r := range rows {
		if r.Product == "maize" && r.Count != 2 {
			t.Fatalf("maize count %v", r.Count)
		}
	}
}

func TestStockListIsPrivateAndAnswersNotFoundToEverybodyElse(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	p := approvedPoster(t, m, "ada", "Ada Farm")
	m.SetStock(ctx, "ada", Stock{PosterID: p.ID, Product: "maize", Count: 5})
	for _, caller := range []string{"bob", "", "admin"} {
		if _, err := m.GetStock(ctx, caller, p.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("caller %q: %v", caller, err)
		}
	}
	if _, err := m.GetStock(ctx, "ada", "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown poster: %v", err)
	}
}

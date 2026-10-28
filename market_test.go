package market_test

import (
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

func TestAnOrderKeepsThePriceItStartedWithAfterTheOfferChanges(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	o, err := m.CreateOffer(ctx, "ann", market.Offer{PosterID: p.ID, ProductWords: "beans", Price: 10})
	if err != nil {
		t.Fatal(err)
	}
	ord, err := m.BuyNow(ctx, "bea", o.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	o.Price = 99
	if _, err := m.UpdateOffer(ctx, "ann", o); err != nil {
		t.Fatal(err)
	}
	got, err := m.GetOrder(ctx, "bea", ord.ID)
	if err != nil || got.UnitPrice != 10 {
		t.Fatalf("price %v err %v", got.UnitPrice, err)
	}
	tr, err := m.OrderTrail(ctx, "bea", ord.ID)
	if err != nil || len(tr) != 1 {
		t.Fatal(tr, err)
	}
}

func TestAnAskIsMadeOncePerPosterPerRequest(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	r, _ := m.CreateRequest(ctx, "bea", market.Request{Wanted: "beans"})
	if _, err := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: r.ID, Question: "have any?"}); err != nil {
		t.Fatal(err)
	}
	_, err := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: r.ID, Question: "again?"})
	if !errors.Is(err, market.ErrAlreadyAsked) {
		t.Fatalf("want ErrAlreadyAsked, got %v", err)
	}
}

func TestAReplyNeverOpensAnOrderAndSilenceStaysWaiting(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	r, _ := m.CreateRequest(ctx, "bea", market.Request{Wanted: "beans"})
	a, _ := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: r.ID, Question: "have any?"})
	if got, _ := m.ListAsks(ctx, "bea"); got[0].Answer != models.AnswerWaiting {
		t.Fatal("silence must stay waiting")
	}
	if _, err := m.AnswerAsk(ctx, "ann", a.ID, models.AnswerYes, "", ""); err == nil {
		t.Fatal("a yes without a date must be refused")
	}
	if _, err := m.AnswerAsk(ctx, "ann", a.ID, models.AnswerYes, "2026-11-01", ""); err != nil {
		t.Fatal(err)
	}
	if orders, _ := m.ListOrders(ctx, "bea"); len(orders) != 0 {
		t.Fatal("a reply opened an order")
	}
	if ys, _ := m.ListYeses(ctx, "bea"); len(ys) != 1 {
		t.Fatal("the yes should be kept as a record")
	}
}

func TestOnlyThePosterAnswersHisAskAndMarksPaid(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	o, _ := m.CreateOffer(ctx, "ann", market.Offer{PosterID: p.ID, ProductWords: "beans", Price: 1})
	ord, _ := m.BuyNow(ctx, "bea", o.ID, 1)
	if _, err := m.MarkPaid(ctx, "bea", ord.ID); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("picker marked paid: %v", err)
	}
	if _, err := m.MarkPaid(ctx, "ann", ord.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.GetOrder(ctx, "cat", ord.ID); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("a stranger saw an order: %v", err)
	}
}

func TestStockIsPrivateToItsPoster(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	if _, err := m.SetStock(ctx, "ann", market.Stock{PosterID: p.ID, Product: "beans", Count: 5}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.GetStock(ctx, "bea", p.ID); err == nil {
		t.Fatal("a stranger read a stock list")
	}
	if _, err := m.SetStock(ctx, "bea", market.Stock{PosterID: p.ID, Product: "beans", Count: 0}); err == nil {
		t.Fatal("a stranger wrote a stock list")
	}
	if s, err := m.GetStock(ctx, "ann", p.ID); err != nil || len(s) != 1 || s[0].Count != 5 {
		t.Fatal(s, err)
	}
}

func TestAGroupOrderNeedsAMinimumAndShowsEachCallerOnlyWhatTheyMaySee(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	if _, err := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3}); err == nil {
		t.Fatal("a group order without a minimum must be refused")
	}
	g, err := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	if err != nil {
		t.Fatal(err)
	}
	m.JoinGroupOrder(ctx, "bea", g.ID, 4)
	m.JoinGroupOrder(ctx, "cat", g.ID, 6)

	pick, _ := m.GetGroupOrder(ctx, "bea", g.ID)
	if len(pick.Lines) != 1 || pick.Lines[0].Picker != "bea" || pick.Total != 10 || pick.Count != 2 || pick.Minimum != 10 {
		t.Fatalf("picker view %+v", pick)
	}
	coord, _ := m.GetGroupOrder(ctx, "coord", g.ID)
	post, _ := m.GetGroupOrder(ctx, "ann", g.ID)
	if len(coord.Lines) != 2 || len(post.Lines) != 2 {
		t.Fatalf("coordinator %d poster %d lines", len(coord.Lines), len(post.Lines))
	}
}

func TestARevisionKeepsThePreviousTermsAndEachAnswerIsKeptAgainstItsVersion(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	g, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	m.JoinGroupOrder(ctx, "bea", g.ID, 4)
	if _, err := m.AskRevision(ctx, "bea", g.ID, 5, 8); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("a picker revised terms: %v", err)
	}
	r, err := m.AskRevision(ctx, "ann", g.ID, 5, 8)
	if err != nil || r.TermsVersion != 2 || r.TermsHistory == "[]" {
		t.Fatal(r, err)
	}
	v, _ := m.GetGroupOrder(ctx, "bea", g.ID)
	if v.Lines[0].Answer != models.AnswerWaiting || v.Lines[0].AnsweredVersion != 1 {
		t.Fatalf("%+v", v.Lines[0])
	}
	l, err := m.AnswerRevision(ctx, "bea", g.ID, models.AnswerYes)
	if err != nil || l.AnsweredVersion != 2 {
		t.Fatal(l, err)
	}
}

func TestAnUnapprovedPosterCannotPostOrSell(t *testing.T) {
	m, ctx := newManager(t)
	p, _ := m.ApplyPoster(ctx, "ann", market.Poster{Name: "Ann"})
	if _, err := m.CreateOffer(ctx, "ann", market.Offer{PosterID: p.ID, ProductWords: "x", Price: 1}); !errors.Is(err, market.ErrPosterNotReady) {
		t.Fatalf("%v", err)
	}
}

func TestAnAwayPosterIsNotAskedAndIsAskableAgainWhenAvailable(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	if _, err := m.SetAvailability(ctx, "ann", p.ID, models.Away); err != nil {
		t.Fatal(err)
	}
	if _, err := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, Question: "any beans?"}); !errors.Is(err, market.ErrPosterAway) {
		t.Fatalf("%v", err)
	}
	if _, err := m.SetAvailability(ctx, "ann", p.ID, models.Available); err != nil {
		t.Fatal(err)
	}
	if _, err := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, Question: "any beans?"}); err != nil {
		t.Fatal(err)
	}
}

package market_test

import (
	"context"
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

type bidding struct {
	m      market.MarketManager
	ctx    context.Context
	notice market.Notice
	buyer  market.Poster
	rival  market.Poster
}

func newBidding(t *testing.T) bidding {
	t.Helper()
	m, ctx := newManager(t)
	m.UseAgents(func(_ context.Context, c string) bool { return c == "helper" })
	n, err := m.CreateNotice(ctx, "wanjiru", market.Notice{Wanted: "40 bags of maize", Quantity: 40})
	if err != nil {
		t.Fatal(err)
	}
	return bidding{m: m, ctx: ctx, notice: n, buyer: approvedPoster(t, m, ctx, "otieno"), rival: approvedPoster(t, m, ctx, "kamau")}
}

func (b bidding) quote(t *testing.T, caller string, p market.Poster, price float64) market.Quote {
	t.Helper()
	q, err := b.m.SendQuote(b.ctx, caller, b.notice.ID, market.Quote{PosterID: p.ID, Price: price})
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func TestAFarmerAcceptsTheCounteredPriceAfterTheBuyerRevisesToIt(t *testing.T) {
	b := newBidding(t)
	q := b.quote(t, "otieno", b.buyer, 3000)
	rival := b.quote(t, "kamau", b.rival, 3200)
	d, err := b.m.DraftCounter(b.ctx, "helper", q.ID, 3400, "Maize is dry and bagged")
	if err != nil || d.State != models.DraftWaiting {
		t.Fatal(d.State, err)
	}
	if got, _ := b.m.ListQuotes(b.ctx, "otieno", b.notice.ID); len(got) != 1 || got[0].CounterPrice != 0 {
		t.Fatalf("a draft must not reach the buyer: %+v", got)
	}
	sent, err := b.m.SendCounter(b.ctx, "wanjiru", d.ID)
	if err != nil || sent.State != models.QuoteCountered || sent.CounterPrice != 3400 {
		t.Fatal(sent.State, sent.CounterPrice, err)
	}
	revised, err := b.m.ReviseQuote(b.ctx, "otieno", q.ID, 3400, "Agreed")
	if err != nil || revised.State != models.QuoteOpen || revised.CounterPrice != 0 {
		t.Fatal(revised.State, err)
	}
	won, err := b.m.AcceptQuote(b.ctx, "wanjiru", q.ID)
	if err != nil || won.State != models.QuoteAccepted {
		t.Fatal(won.State, err)
	}
	all, _ := b.m.ListQuotes(b.ctx, "wanjiru", b.notice.ID)
	for _, got := range all {
		if got.ID == rival.ID && got.State != models.QuoteDeclined {
			t.Fatalf("the other quote must be declined: %+v", got)
		}
	}
	if _, err := b.m.AcceptQuote(b.ctx, "wanjiru", rival.ID); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("a sold notice takes no second acceptance, got %v", err)
	}
	if _, err := b.m.SendQuote(b.ctx, "kamau", b.notice.ID, market.Quote{PosterID: b.rival.ID, Price: 1}); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("a sold notice takes no quote, got %v", err)
	}
}

func TestOnlyTheAgentDraftsAndOnlyTheFarmerSendsAndAccepts(t *testing.T) {
	b := newBidding(t)
	q := b.quote(t, "otieno", b.buyer, 3000)
	for _, who := range []string{"wanjiru", "otieno", "kamau", ""} {
		if _, err := b.m.DraftCounter(b.ctx, who, q.ID, 3400, ""); !errors.Is(err, market.ErrForbidden) {
			t.Fatalf("%q must not draft, got %v", who, err)
		}
	}
	d, _ := b.m.DraftCounter(b.ctx, "helper", q.ID, 3400, "")
	for _, who := range []string{"helper", "otieno", "kamau"} {
		if _, err := b.m.SendCounter(b.ctx, who, d.ID); !errors.Is(err, market.ErrForbidden) {
			t.Fatalf("%q must not send the counter, got %v", who, err)
		}
		if _, err := b.m.AcceptQuote(b.ctx, who, q.ID); !errors.Is(err, market.ErrForbidden) {
			t.Fatalf("%q must not accept, got %v", who, err)
		}
	}
}

func TestWithNoAgentSuppliedNobodyDrafts(t *testing.T) {
	m, ctx := newManager(t)
	n, _ := m.CreateNotice(ctx, "wanjiru", market.Notice{Wanted: "maize"})
	p := approvedPoster(t, m, ctx, "otieno")
	q, _ := m.SendQuote(ctx, "otieno", n.ID, market.Quote{PosterID: p.ID, Price: 10})
	if _, err := m.DraftCounter(ctx, "helper", q.ID, 12, ""); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("want forbidden, got %v", err)
	}
}

func TestADraftIsSeenByTheFarmerAndItsAuthorOnly(t *testing.T) {
	b := newBidding(t)
	q := b.quote(t, "otieno", b.buyer, 3000)
	b.m.DraftCounter(b.ctx, "helper", q.ID, 3400, "")
	for who, want := range map[string]int{"wanjiru": 1, "helper": 1, "otieno": 0, "kamau": 0, "": 0} {
		if got, _ := b.m.ListDrafts(b.ctx, who, b.notice.ID); len(got) != want {
			t.Fatalf("%q sees %d drafts, want %d", who, len(got), want)
		}
	}
}

func TestABuyerSeesOnlyHisOwnQuoteAndTheFarmerAndTheAgentSeeAll(t *testing.T) {
	b := newBidding(t)
	b.quote(t, "otieno", b.buyer, 3000)
	b.quote(t, "kamau", b.rival, 3200)
	for who, want := range map[string]int{"wanjiru": 2, "helper": 2, "otieno": 1, "kamau": 1, "stranger": 0} {
		if got, _ := b.m.ListQuotes(b.ctx, who, b.notice.ID); len(got) != want {
			t.Fatalf("%q sees %d quotes, want %d", who, len(got), want)
		}
	}
}

func TestAPosterQuotesOnceAndOnlyOnAnOpenNotice(t *testing.T) {
	b := newBidding(t)
	b.quote(t, "otieno", b.buyer, 3000)
	if _, err := b.m.SendQuote(b.ctx, "otieno", b.notice.ID, market.Quote{PosterID: b.buyer.ID, Price: 2900}); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("a second quote must be refused, got %v", err)
	}
	if _, err := b.m.SendQuote(b.ctx, "kamau", b.notice.ID, market.Quote{PosterID: b.buyer.ID, Price: 1}); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("a quote for another's poster must be refused, got %v", err)
	}
	b.m.WithdrawNotice(b.ctx, "wanjiru", b.notice.ID)
	if _, err := b.m.SendQuote(b.ctx, "kamau", b.notice.ID, market.Quote{PosterID: b.rival.ID, Price: 1}); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("a withdrawn notice takes no quote, got %v", err)
	}
}

func TestACounterNeedsAnOpenQuoteAndAReviseNeedsACounter(t *testing.T) {
	b := newBidding(t)
	q := b.quote(t, "otieno", b.buyer, 3000)
	if _, err := b.m.ReviseQuote(b.ctx, "otieno", q.ID, 3100, ""); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("revising an uncountered quote must be refused, got %v", err)
	}
	d, _ := b.m.DraftCounter(b.ctx, "helper", q.ID, 3400, "")
	b.m.SendCounter(b.ctx, "wanjiru", d.ID)
	if _, err := b.m.DraftCounter(b.ctx, "helper", q.ID, 3500, ""); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("a countered quote takes no new draft, got %v", err)
	}
	if _, err := b.m.AcceptQuote(b.ctx, "wanjiru", q.ID); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("a countered quote waits for the buyer, got %v", err)
	}
	if _, err := b.m.SendCounter(b.ctx, "wanjiru", d.ID); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("a draft is sent once, got %v", err)
	}
}

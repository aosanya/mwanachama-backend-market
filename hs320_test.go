package market_test

import (
	"context"
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

func waitingRideAsk(t *testing.T) (market.MarketManager, market.Ask, func() context.Context) {
	t.Helper()
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	r, _ := m.CreateRequest(ctx, "bea", market.Request{Wanted: "a ride"})
	a, err := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: r.ID, Question: "can you come?"})
	if err != nil {
		t.Fatal(err)
	}
	return m, a, func() context.Context { return ctx }
}

func TestTheAskerWithdrawsAWaitingAskAndBothSidesSeeItWithdrawn(t *testing.T) {
	m, a, ctx := waitingRideAsk(t)
	got, err := m.WithdrawAsk(ctx(), "bea", a.ID)
	if err != nil || got.Answer != models.AnswerWithdrawn {
		t.Fatalf("withdraw: %+v %v", got, err)
	}
	for _, caller := range []string{"bea", "ann"} {
		asks, _ := m.ListAsks(ctx(), caller)
		if len(asks) != 1 || asks[0].Answer != models.AnswerWithdrawn {
			t.Fatalf("%s sees %+v", caller, asks)
		}
	}
}

func TestAStrangerAndThePosterCannotWithdrawAnAsk(t *testing.T) {
	m, a, ctx := waitingRideAsk(t)
	for _, caller := range []string{"cy", "ann", ""} {
		if _, err := m.WithdrawAsk(ctx(), caller, a.ID); !errors.Is(err, market.ErrNotFound) {
			t.Fatalf("%q withdrew: %v", caller, err)
		}
	}
	asks, _ := m.ListAsks(ctx(), "bea")
	if asks[0].Answer != models.AnswerWaiting {
		t.Fatalf("the ask changed: %+v", asks[0])
	}
}

func TestAnAnsweredAskCannotBeWithdrawn(t *testing.T) {
	for _, answer := range []string{models.AnswerYes, models.AnswerNo} {
		m, a, ctx := waitingRideAsk(t)
		date := ""
		if answer == models.AnswerYes {
			date = "2026-11-01"
		}
		if _, err := m.AnswerAsk(ctx(), "ann", a.ID, answer, date, ""); err != nil {
			t.Fatal(err)
		}
		if _, err := m.WithdrawAsk(ctx(), "bea", a.ID); !errors.Is(err, market.ErrConflict) {
			t.Fatalf("withdrew after %s: %v", answer, err)
		}
		asks, _ := m.ListAsks(ctx(), "bea")
		if asks[0].Answer != answer {
			t.Fatalf("the answer changed: %+v", asks[0])
		}
	}
}

func TestAWithdrawnAskCannotBeAnsweredOrWithdrawnAgain(t *testing.T) {
	m, a, ctx := waitingRideAsk(t)
	if _, err := m.WithdrawAsk(ctx(), "bea", a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.AnswerAsk(ctx(), "ann", a.ID, models.AnswerYes, "2026-11-01", ""); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("answered a withdrawn ask: %v", err)
	}
	if _, err := m.WithdrawAsk(ctx(), "bea", a.ID); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("withdrew twice: %v", err)
	}
	if ys, _ := m.ListYeses(ctx(), "bea"); len(ys) != 0 {
		t.Fatalf("a yes came from a withdrawn ask: %+v", ys)
	}
}

func TestAnUnknownAskIsNotFoundWhenWithdrawn(t *testing.T) {
	m, _, ctx := waitingRideAsk(t)
	if _, err := m.WithdrawAsk(ctx(), "bea", "nope"); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("%v", err)
	}
}

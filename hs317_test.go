package market_test

import (
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

func TestARiderYesCarriesTheMinutesHeIsAwayToBothSides(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	r, _ := m.CreateRequest(ctx, "bea", market.Request{Wanted: "a ride"})
	a, _ := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: r.ID, Question: "can you come?"})
	if _, err := m.AnswerAskWithMinutes(ctx, "ann", a.ID, models.AnswerYes, "2026-11-01", "", "", 0, 12); err != nil {
		t.Fatal(err)
	}
	for _, caller := range []string{"bea", "ann"} {
		ys, _ := m.ListYeses(ctx, caller)
		if len(ys) != 1 || ys[0].Minutes != 12 {
			t.Fatalf("%s sees %+v", caller, ys)
		}
	}
}

func TestMinutesAreNeverNegativeAndANoCarriesNone(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	r, _ := m.CreateRequest(ctx, "bea", market.Request{Wanted: "a ride"})
	a, _ := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: r.ID, Question: "can you come?"})
	if _, err := m.AnswerAskWithMinutes(ctx, "ann", a.ID, models.AnswerYes, "2026-11-01", "", "", 0, -3); !errors.Is(err, market.ErrInvalid) {
		t.Fatalf("negative minutes: %v", err)
	}
	if _, err := m.AnswerAskWithMinutes(ctx, "ann", a.ID, models.AnswerNo, "", "", "", 0, 5); !errors.Is(err, market.ErrInvalid) {
		t.Fatalf("a no with minutes: %v", err)
	}
	if _, err := m.AnswerAsk(ctx, "ann", a.ID, models.AnswerYes, "2026-11-01", ""); err != nil {
		t.Fatal(err)
	}
	ys, _ := m.ListYeses(ctx, "bea")
	if len(ys) != 1 || ys[0].Minutes != 0 {
		t.Fatalf("a yes without minutes: %+v", ys)
	}
}

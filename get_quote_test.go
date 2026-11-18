package market_test

import (
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
)

func TestOneQuoteIsReadByIdByThePickerAnAgentAndTheQuotesOwnPoster(t *testing.T) {
	b := newBidding(t)
	q := b.quote(t, "otieno", b.buyer, 3000)
	b.quote(t, "kamau", b.rival, 3200)
	for _, who := range []string{"wanjiru", "helper", "otieno"} {
		got, err := b.m.GetQuote(b.ctx, who, q.ID)
		if err != nil {
			t.Fatalf("%q must read the quote, got %v", who, err)
		}
		if got.ID != q.ID || got.PosterID != b.buyer.ID || got.Price != 3000 {
			t.Fatalf("%q read the wrong quote: %+v", who, got)
		}
	}
}

func TestAStrangerAndTheRivalPosterAreToldNotFoundForAQuoteTheyMayNotSee(t *testing.T) {
	b := newBidding(t)
	q := b.quote(t, "otieno", b.buyer, 3000)
	b.quote(t, "kamau", b.rival, 3200)
	for _, who := range []string{"stranger", "kamau", ""} {
		if _, err := b.m.GetQuote(b.ctx, who, q.ID); !errors.Is(err, market.ErrNotFound) {
			t.Fatalf("%q must be told not found, got %v", who, err)
		}
	}
}

func TestAnUnknownQuoteIdIsNotFoundForEveryone(t *testing.T) {
	b := newBidding(t)
	b.quote(t, "otieno", b.buyer, 3000)
	for _, who := range []string{"wanjiru", "helper", "otieno", "stranger", ""} {
		if _, err := b.m.GetQuote(b.ctx, who, "no-such-quote"); !errors.Is(err, market.ErrNotFound) {
			t.Fatalf("%q must be told not found, got %v", who, err)
		}
	}
}

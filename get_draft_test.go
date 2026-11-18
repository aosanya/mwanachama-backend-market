package market_test

import (
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

func draftedCounter(t *testing.T) (bidding, market.Quote, market.Draft) {
	t.Helper()
	b := newBidding(t)
	q := b.quote(t, "otieno", b.buyer, 3000)
	b.quote(t, "kamau", b.rival, 3200)
	d, err := b.m.DraftCounter(b.ctx, "helper", q.ID, 3400, "Maize is dry and bagged")
	if err != nil {
		t.Fatal(err)
	}
	return b, q, d
}

func TestThePickerAndTheDraftAuthorReadTheSameDraftById(t *testing.T) {
	b, q, d := draftedCounter(t)
	for _, who := range []string{"wanjiru", "helper"} {
		got, err := b.m.GetDraft(b.ctx, who, d.ID)
		if err != nil {
			t.Fatalf("%s: %v", who, err)
		}
		if got.ID != d.ID || got.Price != 3400 || got.Note != "Maize is dry and bagged" || got.State != models.DraftWaiting {
			t.Fatalf("%s read a different draft: %+v", who, got)
		}
		if got.QuoteID != q.ID || got.NoticeID != b.notice.ID {
			t.Fatalf("%s read a draft on the wrong quote or notice: %+v", who, got)
		}
	}
}

func TestAnyoneElseReadingADraftGetsTheSameNotFoundAsAMissingId(t *testing.T) {
	b, _, d := draftedCounter(t)
	for _, who := range []string{"otieno", "kamau", "stranger", ""} {
		if _, err := b.m.GetDraft(b.ctx, who, d.ID); !errors.Is(err, market.ErrNotFound) {
			t.Fatalf("%q reading a real draft: want not found, got %v", who, err)
		}
	}
	for _, who := range []string{"wanjiru", "helper"} {
		if _, err := b.m.GetDraft(b.ctx, who, "no-such-draft"); !errors.Is(err, market.ErrNotFound) {
			t.Fatalf("%q reading a missing id: want not found, got %v", who, err)
		}
	}
}

func TestASentDraftReadsBackAsSent(t *testing.T) {
	b, _, d := draftedCounter(t)
	if _, err := b.m.SendCounter(b.ctx, "wanjiru", d.ID); err != nil {
		t.Fatal(err)
	}
	got, err := b.m.GetDraft(b.ctx, "wanjiru", d.ID)
	if err != nil || got.State != models.DraftSent {
		t.Fatal(got.State, err)
	}
}

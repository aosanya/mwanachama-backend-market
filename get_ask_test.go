package market

import (
	"context"
	"errors"
	"testing"

	"github.com/aosanya/mwanachama-backend-market/models"
)

func TestGetAskIsSeenByItsPickerAndByTheOwnerOfTheAskedPosterOnly(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	ada := approvedPoster(t, m, "ada", "Ada Farm")
	approvedPoster(t, m, "bob", "Bob Shop")
	a := askOf(t, m, "cy", ada.ID, "have any maize?")
	own, err := m.GetAsk(ctx, "cy", a.ID)
	if err != nil || own.ID != a.ID || own.PickerName != "name-of-cy" {
		t.Fatalf("picker: %+v %v", own, err)
	}
	asked, err := m.GetAsk(ctx, "ada", a.ID)
	if err != nil || asked.ID != a.ID || asked.PosterID != ada.ID {
		t.Fatalf("poster owner: %+v %v", asked, err)
	}
	for _, caller := range []string{"bob", "dee", ""} {
		if _, err := m.GetAsk(ctx, caller, a.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("caller %q: %v", caller, err)
		}
	}
	if _, err := m.GetAsk(ctx, "cy", "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing ask: %v", err)
	}
}

func TestGetAskStillReturnsAnAskItsPickerWithdrew(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	ada := approvedPoster(t, m, "ada", "Ada Farm")
	a := askOf(t, m, "cy", ada.ID, "have any maize?")
	if _, err := m.WithdrawAsk(ctx, "cy", a.ID); err != nil {
		t.Fatal(err)
	}
	got, err := m.GetAsk(ctx, "cy", a.ID)
	if err != nil || got.Answer != models.AnswerWithdrawn {
		t.Fatalf("withdrawn ask: %+v %v", got, err)
	}
}

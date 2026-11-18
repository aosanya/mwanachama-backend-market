package market

import (
	"context"
	"errors"
	"testing"

	"github.com/aosanya/mwanachama-backend-market/models"
)

func pickerFullName(_ context.Context, id string) string {
	return map[string]string{"picker": "Pia Picker Otieno"}[id]
}

func TestGetNoticeReturnsAnOpenNoticeWithStateOpen(t *testing.T) {
	f := newAcceptFixture(t, 0)
	m := f.managers[0]
	m.UseNames(pickerFullName)
	n, err := m.GetNotice(context.Background(), "picker", f.noticeID)
	if err != nil || n.ID != f.noticeID || n.Wanted != "maize" || n.State != models.NoticeOpen {
		t.Fatalf("GetNotice answered %+v (%v), want the open maize notice", n, err)
	}
}

func TestGetNoticeStillReturnsAWithdrawnNoticeWithStateWithdrawn(t *testing.T) {
	f := newAcceptFixture(t, 0)
	ctx := context.Background()
	m := f.managers[0]
	if _, err := m.WithdrawNotice(ctx, "picker", f.noticeID); err != nil {
		t.Fatal(err)
	}
	n, err := m.GetNotice(ctx, "picker", f.noticeID)
	if err != nil || n.ID != f.noticeID || n.State != models.NoticeWithdrawn {
		t.Fatalf("GetNotice answered %+v (%v), want the withdrawn notice", n, err)
	}
}

func TestGetNoticeReturnsASoldNoticeWithStateSold(t *testing.T) {
	f := newAcceptFixture(t, 1)
	ctx := context.Background()
	m := f.managers[0]
	if _, err := m.AcceptQuote(ctx, "picker", f.quoteIDs[0]); err != nil {
		t.Fatal(err)
	}
	n, err := m.GetNotice(ctx, "picker", f.noticeID)
	if err != nil || n.ID != f.noticeID || n.State != models.NoticeSold {
		t.Fatalf("GetNotice answered %+v (%v), want the sold notice", n, err)
	}
}

func TestGetNoticeAnswersNotFoundForAMissingID(t *testing.T) {
	f := newAcceptFixture(t, 0)
	if _, err := f.managers[0].GetNotice(context.Background(), "picker", "no-such-notice"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetNotice answered %v, want ErrNotFound", err)
	}
}

func TestGetNoticeShowsThePickerTheirOwnIDAndFullName(t *testing.T) {
	f := newAcceptFixture(t, 0)
	m := f.managers[0]
	m.UseNames(pickerFullName)
	n, err := m.GetNotice(context.Background(), "picker", f.noticeID)
	if err != nil || n.Picker != "picker" || n.PickerName != "Pia Picker Otieno" {
		t.Fatalf("the picker saw %+v (%v), want own id and full name", n, err)
	}
}

func TestGetNoticeHidesThePickerIDAndShowsOnlyTheFirstNameToAnyoneElse(t *testing.T) {
	f := newAcceptFixture(t, 0)
	m := f.managers[0]
	m.UseNames(pickerFullName)
	for _, caller := range []string{"neighbour", ""} {
		n, err := m.GetNotice(context.Background(), caller, f.noticeID)
		if err != nil || n.Picker != "" || n.PickerName != "Pia" || n.Wanted != "maize" {
			t.Fatalf("caller %q saw %+v (%v), want no picker id and the name Pia", caller, n, err)
		}
	}
}

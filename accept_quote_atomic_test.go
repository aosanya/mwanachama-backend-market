package market

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-market/models"
)

var errInjected = errors.New("injected write failure")

func failQuoteWritesTo(t *testing.T, m *marketManager, state string) {
	t.Helper()
	table := m.st.Table(RoleQuote)
	name := "fail_quote_" + state + "_" + t.Name()
	err := m.db.Callback().Update().Before("gorm:update").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table != table {
			return
		}
		if set, ok := tx.Statement.Dest.(map[string]any); ok && fmt.Sprint(set["state"]) == state {
			tx.AddError(errInjected)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}

func assertNothingChanged(t *testing.T, f acceptFixture) {
	t.Helper()
	n, err := f.managers[0].noticeOf(context.Background(), f.noticeID)
	if err != nil || n.State != models.NoticeOpen {
		t.Fatalf("the notice is %q (%v), want open", n.State, err)
	}
	for id, state := range f.quoteStates(t) {
		if state != models.QuoteOpen {
			t.Fatalf("quote %s is %s, want open", id, state)
		}
	}
}

func TestAcceptQuoteLeavesTheNoticeOpenWhenDecliningTheOtherQuotesFails(t *testing.T) {
	f := newAcceptFixture(t, 3)
	failQuoteWritesTo(t, f.managers[0], models.QuoteDeclined)
	_, err := f.managers[0].AcceptQuote(context.Background(), "picker", f.quoteIDs[0])
	if !errors.Is(err, errInjected) {
		t.Fatalf("AcceptQuote answered %v, want the injected failure", err)
	}
	assertNothingChanged(t, f)
}

func TestAcceptQuoteLeavesTheNoticeOpenWhenMarkingTheWinnerAcceptedFails(t *testing.T) {
	f := newAcceptFixture(t, 3)
	failQuoteWritesTo(t, f.managers[0], models.QuoteAccepted)
	_, err := f.managers[0].AcceptQuote(context.Background(), "picker", f.quoteIDs[0])
	if !errors.Is(err, errInjected) {
		t.Fatalf("AcceptQuote answered %v, want the injected failure", err)
	}
	assertNothingChanged(t, f)
}

func TestAcceptQuoteAnswersAConflictAndLeavesTheNoticeOpenWhenTheQuoteChangedAfterItWasRead(t *testing.T) {
	f := newAcceptFixture(t, 2)
	ctx := context.Background()
	m, other := f.managers[0], f.managers[1]
	winner := f.quoteIDs[0]
	noticeTable := m.st.Table(RoleNotice)
	fired := false
	err := m.db.Callback().Update().Before("gorm:update").Register("counter_lands_first", func(tx *gorm.DB) {
		if tx.Statement.Table != noticeTable || fired {
			return
		}
		fired = true
		if err := other.q(ctx, RoleQuote).Where("id = ?", winner).Update("state", models.QuoteCountered).Error; err != nil {
			tx.AddError(err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.AcceptQuote(ctx, "picker", winner)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("AcceptQuote answered %v, want ErrConflict", err)
	}
	n, err := m.noticeOf(ctx, f.noticeID)
	if err != nil || n.State != models.NoticeOpen {
		t.Fatalf("the notice is %q (%v), want open", n.State, err)
	}
	states := f.quoteStates(t)
	if states[winner] != models.QuoteCountered || states[f.quoteIDs[1]] != models.QuoteOpen {
		t.Fatalf("quote states are %v, want the winner countered and the other open", states)
	}
}

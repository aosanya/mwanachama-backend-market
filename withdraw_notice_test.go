package market

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/aosanya/mwanachama-backend-market/models"
)

func TestWithdrawNoticeAnswersAConflictOnASoldNoticeAndLeavesItSold(t *testing.T) {
	f := newAcceptFixture(t, 1)
	ctx := context.Background()
	m := f.managers[0]
	if _, err := m.AcceptQuote(ctx, "picker", f.quoteIDs[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := m.WithdrawNotice(ctx, "picker", f.noticeID); !errors.Is(err, ErrConflict) {
		t.Fatalf("WithdrawNotice answered %v, want ErrConflict", err)
	}
	n, err := m.noticeOf(ctx, f.noticeID)
	if err != nil || n.State != models.NoticeSold {
		t.Fatalf("the notice is %q (%v), want sold", n.State, err)
	}
	if got := f.quoteStates(t)[f.quoteIDs[0]]; got != models.QuoteAccepted {
		t.Fatalf("the quote is %s, want accepted", got)
	}
}

func TestWithdrawNoticeAnswersAConflictOnAWithdrawnNotice(t *testing.T) {
	f := newAcceptFixture(t, 1)
	ctx := context.Background()
	m := f.managers[0]
	if _, err := m.WithdrawNotice(ctx, "picker", f.noticeID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.WithdrawNotice(ctx, "picker", f.noticeID); !errors.Is(err, ErrConflict) {
		t.Fatalf("the second WithdrawNotice answered %v, want ErrConflict", err)
	}
}

func TestAnAcceptAndAWithdrawAtOnceLeaveOneWinnerAndTheMatchingNoticeState(t *testing.T) {
	const rounds = 10
	for round := 0; round < rounds; round++ {
		f := newAcceptFixture(t, 1)
		ctx := context.Background()
		var wg sync.WaitGroup
		start := make(chan struct{})
		var acceptErr, withdrawErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, acceptErr = f.managers[0].AcceptQuote(ctx, "picker", f.quoteIDs[0])
		}()
		go func() {
			defer wg.Done()
			<-start
			_, withdrawErr = f.managers[1].WithdrawNotice(ctx, "picker", f.noticeID)
		}()
		close(start)
		wg.Wait()
		n, err := f.managers[0].noticeOf(ctx, f.noticeID)
		if err != nil {
			t.Fatal(err)
		}
		quote := f.quoteStates(t)[f.quoteIDs[0]]
		switch {
		case acceptErr == nil && withdrawErr == nil:
			t.Fatalf("round %d: the accept and the withdraw both answered success", round)
		case acceptErr == nil:
			if !errors.Is(withdrawErr, ErrConflict) || n.State != models.NoticeSold || quote != models.QuoteAccepted {
				t.Fatalf("round %d: accept won, withdraw %v, notice %s, quote %s", round, withdrawErr, n.State, quote)
			}
		case withdrawErr == nil:
			if !errors.Is(acceptErr, ErrConflict) || n.State != models.NoticeWithdrawn || quote != models.QuoteOpen {
				t.Fatalf("round %d: withdraw won, accept %v, notice %s, quote %s", round, acceptErr, n.State, quote)
			}
		default:
			t.Fatalf("round %d: neither won: accept %v, withdraw %v", round, acceptErr, withdrawErr)
		}
	}
}

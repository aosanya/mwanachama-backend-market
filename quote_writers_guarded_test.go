package market

import (
	"context"
	"errors"
	"sync"
	"testing"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-market/models"
)

type interleaved struct {
	acceptFixture
	ctx   context.Context
	first string
	other string
}

func newInterleaved(t *testing.T) interleaved {
	t.Helper()
	f := newAcceptFixture(t, 2)
	for _, m := range f.managers {
		m.UseAgents(func(_ context.Context, c string) bool { return c == "helper" })
	}
	return interleaved{acceptFixture: f, ctx: context.Background(), first: f.quoteIDs[0], other: f.quoteIDs[1]}
}

func (i interleaved) landsAfterTheNoticeIsRead(t *testing.T, change func() error) {
	t.Helper()
	m := i.managers[0]
	table := m.st.Table(RoleNotice)
	fired := false
	err := m.db.Callback().Query().After("gorm:query").Register("lands_after_notice_read", func(tx *gorm.DB) {
		if tx.Statement.Table != table || fired {
			return
		}
		fired = true
		if err := change(); err != nil {
			tx.AddError(err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (i interleaved) acceptsTheOther() error {
	_, err := i.managers[1].AcceptQuote(i.ctx, "picker", i.other)
	return err
}

func (i interleaved) declinesTheFirst() error {
	return i.managers[1].q(i.ctx, RoleQuote).Where("id = ?", i.first).Update("state", models.QuoteDeclined).Error
}

func (i interleaved) setQuoteState(t *testing.T, id, state string) {
	t.Helper()
	if err := i.managers[0].q(i.ctx, RoleQuote).Where("id = ?", id).Update("state", state).Error; err != nil {
		t.Fatal(err)
	}
}

func (i interleaved) waitingDraft(t *testing.T) Draft {
	t.Helper()
	d := Draft{ID: newID(), QuoteID: i.first, NoticeID: i.noticeID, Author: "helper", Price: 9, State: models.DraftWaiting, CreatedAt: i.managers[0].now()}
	if err := i.managers[0].insert(i.ctx, RoleDraft, d); err != nil {
		t.Fatal(err)
	}
	return d
}

func (i interleaved) drafts(t *testing.T) []Draft {
	t.Helper()
	m := i.managers[0]
	ds, err := listOf[Draft](m, m.q(i.ctx, RoleDraft).Where("notice_id = ?", i.noticeID), RoleDraft)
	if err != nil {
		t.Fatal(err)
	}
	return ds
}

func (i interleaved) assertSoldToTheOther(t *testing.T) {
	t.Helper()
	n, err := i.managers[0].noticeOf(i.ctx, i.noticeID)
	if err != nil || n.State != models.NoticeSold {
		t.Fatalf("the notice is %q (%v), want sold", n.State, err)
	}
	states := i.quoteStates(t)
	if len(states) != 2 || states[i.other] != models.QuoteAccepted || states[i.first] != models.QuoteDeclined {
		t.Fatalf("quote states are %v, want the other accepted, the first declined and nothing more", states)
	}
}

func TestSendQuoteAnswersAConflictWhenAnAcceptLandsFirst(t *testing.T) {
	i := newInterleaved(t)
	p := Poster{ID: newID(), Name: "Latecomer", Owner: "latecomer", Status: models.PosterApproved, Availability: models.Available, CreatedAt: i.managers[0].now()}
	if err := i.managers[0].insert(i.ctx, RolePoster, p); err != nil {
		t.Fatal(err)
	}
	i.landsAfterTheNoticeIsRead(t, i.acceptsTheOther)
	_, err := i.managers[0].SendQuote(i.ctx, "latecomer", i.noticeID, Quote{PosterID: p.ID, Price: 5})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("SendQuote answered %v, want ErrConflict", err)
	}
	i.assertSoldToTheOther(t)
}

func TestDraftCounterAnswersAConflictWhenAnAcceptLandsFirst(t *testing.T) {
	i := newInterleaved(t)
	i.landsAfterTheNoticeIsRead(t, i.acceptsTheOther)
	_, err := i.managers[0].DraftCounter(i.ctx, "helper", i.first, 9, "")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("DraftCounter answered %v, want ErrConflict", err)
	}
	if got := len(i.drafts(t)); got != 0 {
		t.Fatalf("%d drafts stored, want none", got)
	}
	i.assertSoldToTheOther(t)
}

func TestDraftCounterAnswersAConflictWhenTheQuoteIsDeclinedFirst(t *testing.T) {
	i := newInterleaved(t)
	i.landsAfterTheNoticeIsRead(t, i.declinesTheFirst)
	_, err := i.managers[0].DraftCounter(i.ctx, "helper", i.first, 9, "")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("DraftCounter answered %v, want ErrConflict", err)
	}
	if got := len(i.drafts(t)); got != 0 {
		t.Fatalf("%d drafts stored, want none", got)
	}
}

func TestSendCounterAnswersAConflictWhenAnAcceptLandsFirst(t *testing.T) {
	i := newInterleaved(t)
	d := i.waitingDraft(t)
	i.landsAfterTheNoticeIsRead(t, i.acceptsTheOther)
	_, err := i.managers[0].SendCounter(i.ctx, "picker", d.ID)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("SendCounter answered %v, want ErrConflict", err)
	}
	i.assertSoldToTheOther(t)
	if got := i.drafts(t); len(got) != 1 || got[0].State != models.DraftWaiting {
		t.Fatalf("drafts are %v, want the one draft still waiting", got)
	}
}

func TestSendCounterAnswersAConflictWhenTheQuoteIsDeclinedFirstAndTheNoticeIsStillOpen(t *testing.T) {
	i := newInterleaved(t)
	d := i.waitingDraft(t)
	i.landsAfterTheNoticeIsRead(t, i.declinesTheFirst)
	_, err := i.managers[0].SendCounter(i.ctx, "picker", d.ID)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("SendCounter answered %v, want ErrConflict", err)
	}
	if got := i.quoteStates(t)[i.first]; got != models.QuoteDeclined {
		t.Fatalf("the quote is %s, want declined", got)
	}
	if got := i.drafts(t); len(got) != 1 || got[0].State != models.DraftWaiting {
		t.Fatalf("drafts are %v, want the one draft still waiting", got)
	}
}

func TestReviseQuoteAnswersAConflictWhenAnAcceptLandsFirst(t *testing.T) {
	i := newInterleaved(t)
	i.setQuoteState(t, i.first, models.QuoteCountered)
	i.landsAfterTheNoticeIsRead(t, i.acceptsTheOther)
	_, err := i.managers[0].ReviseQuote(i.ctx, "supplier", i.first, 7, "")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("ReviseQuote answered %v, want ErrConflict", err)
	}
	i.assertSoldToTheOther(t)
}

func TestReviseQuoteAnswersAConflictWhenTheQuoteIsDeclinedFirstAndTheNoticeIsStillOpen(t *testing.T) {
	i := newInterleaved(t)
	i.setQuoteState(t, i.first, models.QuoteCountered)
	i.landsAfterTheNoticeIsRead(t, i.declinesTheFirst)
	_, err := i.managers[0].ReviseQuote(i.ctx, "supplier", i.first, 7, "")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("ReviseQuote answered %v, want ErrConflict", err)
	}
	if got := i.quoteStates(t)[i.first]; got != models.QuoteDeclined {
		t.Fatalf("the quote is %s, want declined", got)
	}
}

func TestQuoteWritersAtOnceWithAnAcceptLeaveOneAcceptedQuoteAndEveryOtherDeclined(t *testing.T) {
	const rounds, senders = 5, 4
	for round := 0; round < rounds; round++ {
		f := newAcceptFixture(t, 3)
		for _, m := range f.managers {
			m.UseAgents(func(_ context.Context, c string) bool { return c == "helper" })
		}
		ctx := context.Background()
		i := interleaved{acceptFixture: f, ctx: ctx, first: f.quoteIDs[1], other: f.quoteIDs[0]}
		i.setQuoteState(t, f.quoteIDs[2], models.QuoteCountered)
		draft := i.waitingDraft(t)
		posters := make([]Poster, senders)
		for k := range posters {
			posters[k] = Poster{ID: newID(), Name: "Latecomer", Owner: "latecomer", Status: models.PosterApproved, Availability: models.Available, CreatedAt: f.managers[0].now()}
			if err := f.managers[0].insert(ctx, RolePoster, posters[k]); err != nil {
				t.Fatal(err)
			}
		}
		start := make(chan struct{})
		var wg sync.WaitGroup
		errs := make(chan error, senders+4)
		run := func(m *marketManager, call func(*marketManager) error) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				errs <- call(m)
			}()
		}
		run(f.managers[0], func(m *marketManager) error {
			_, err := m.AcceptQuote(ctx, "picker", f.quoteIDs[0])
			return err
		})
		run(f.managers[1], func(m *marketManager) error {
			_, err := m.SendCounter(ctx, "picker", draft.ID)
			return err
		})
		run(f.managers[0], func(m *marketManager) error {
			_, err := m.ReviseQuote(ctx, "supplier", f.quoteIDs[2], 7, "")
			return err
		})
		run(f.managers[1], func(m *marketManager) error {
			_, err := m.DraftCounter(ctx, "helper", f.quoteIDs[1], 8, "")
			return err
		})
		for k := range posters {
			p := posters[k]
			run(f.managers[k%2], func(m *marketManager) error {
				_, err := m.SendQuote(ctx, "latecomer", f.noticeID, Quote{PosterID: p.ID, Price: 5})
				return err
			})
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil && !errors.Is(err, ErrConflict) {
				t.Fatalf("round %d: a writer answered %v, want success or ErrConflict", round, err)
			}
		}
		n, err := f.managers[0].noticeOf(ctx, f.noticeID)
		if err != nil || n.State != models.NoticeSold {
			t.Fatalf("round %d: the notice is %q (%v), want sold", round, n.State, err)
		}
		for id, state := range f.quoteStates(t) {
			want := models.QuoteDeclined
			if id == f.quoteIDs[0] {
				want = models.QuoteAccepted
			}
			if state != want {
				t.Fatalf("round %d: quote %s is %s, want %s", round, id, state, want)
			}
		}
	}
}

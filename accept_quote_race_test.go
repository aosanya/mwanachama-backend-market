package market

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aosanya/mwanachama-backend-market/models"
)

type acceptFixture struct {
	managers []*marketManager
	noticeID string
	quoteIDs []string
}

func newAcceptFixture(t *testing.T, quotes int) acceptFixture {
	t.Helper()
	s, err := SpecFor(quoteInstanceName())
	if err != nil {
		t.Fatal(err)
	}
	dsn := os.Getenv("MARKET_TEST_POSTGRES_DSN")
	if dsn == "" {
		dsn = filepath.Join(t.TempDir(), "accept.db") + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)"
	}
	managers := make([]*marketManager, 2)
	for i := range managers {
		cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
		var db *gorm.DB
		if os.Getenv("MARKET_TEST_POSTGRES_DSN") != "" {
			db, err = gorm.Open(postgres.Open(dsn), cfg)
		} else {
			db, err = gorm.Open(sqlite.Open(dsn), cfg)
		}
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if err := Provision(db, s); err != nil {
				t.Fatal(err)
			}
		}
		mgr, err := NewMarketManager(db, s)
		if err != nil {
			t.Fatal(err)
		}
		managers[i] = mgr.(*marketManager)
	}
	ctx := context.Background()
	m := managers[0]
	n := Notice{ID: newID(), Picker: "picker", Wanted: "maize", State: models.NoticeOpen, CreatedAt: m.now()}
	if err := m.insert(ctx, RoleNotice, n); err != nil {
		t.Fatal(err)
	}
	f := acceptFixture{managers: managers, noticeID: n.ID}
	for i := 0; i < quotes; i++ {
		p := Poster{ID: newID(), Name: "Supplier", Owner: "supplier", Status: models.PosterApproved, Availability: models.Available, CreatedAt: m.now()}
		if err := m.insert(ctx, RolePoster, p); err != nil {
			t.Fatal(err)
		}
		q := Quote{ID: newID(), NoticeID: n.ID, PosterID: p.ID, Price: float64(i + 1), State: models.QuoteOpen, CreatedAt: m.now()}
		if err := m.insert(ctx, RoleQuote, q); err != nil {
			t.Fatal(err)
		}
		f.quoteIDs = append(f.quoteIDs, q.ID)
	}
	return f
}

func (f acceptFixture) quoteStates(t *testing.T) map[string]string {
	t.Helper()
	qs, err := listOf[Quote](f.managers[0], f.managers[0].q(context.Background(), RoleQuote).Where("notice_id = ?", f.noticeID), RoleQuote)
	if err != nil {
		t.Fatal(err)
	}
	states := map[string]string{}
	for _, q := range qs {
		states[q.ID] = q.State
	}
	return states
}

func TestAcceptsOfDifferentQuotesAtOnceLeaveOneAcceptedQuoteAndASoldNotice(t *testing.T) {
	const quotes, rounds = 8, 5
	for round := 0; round < rounds; round++ {
		f := newAcceptFixture(t, quotes)
		var wg sync.WaitGroup
		start := make(chan struct{})
		errs := make([]error, quotes)
		for i, id := range f.quoteIDs {
			wg.Add(1)
			go func(i int, id string) {
				defer wg.Done()
				<-start
				_, errs[i] = f.managers[i%len(f.managers)].AcceptQuote(context.Background(), "picker", id)
			}(i, id)
		}
		close(start)
		wg.Wait()
		winner := ""
		for i, err := range errs {
			switch {
			case err == nil:
				if winner != "" {
					t.Fatalf("round %d: two accepts answered success", round)
				}
				winner = f.quoteIDs[i]
			case !errors.Is(err, ErrConflict):
				t.Fatalf("round %d: a lost accept answered %v, want ErrConflict", round, err)
			}
		}
		if winner == "" {
			t.Fatalf("round %d: no accept answered success", round)
		}
		accepted := 0
		for id, state := range f.quoteStates(t) {
			switch {
			case id == winner && state != models.QuoteAccepted:
				t.Fatalf("round %d: the winning quote is %s, want accepted", round, state)
			case id != winner && state != models.QuoteDeclined:
				t.Fatalf("round %d: a losing quote is %s, want declined", round, state)
			}
			if state == models.QuoteAccepted {
				accepted++
			}
		}
		if accepted != 1 {
			t.Fatalf("round %d: %d accepted quotes, want 1", round, accepted)
		}
		n, err := f.managers[0].noticeOf(context.Background(), f.noticeID)
		if err != nil || n.State != models.NoticeSold {
			t.Fatalf("round %d: the notice is %q (%v), want sold", round, n.State, err)
		}
	}
}

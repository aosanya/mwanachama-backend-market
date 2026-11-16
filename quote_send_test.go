package market

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aosanya/mwanachama-backend-market/models"
	"github.com/aosanya/mwanachama-backend-shared/gormutil"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

type quoteFixture struct {
	m        *marketManager
	spec     *spec.Spec
	noticeID string
	posterID string
	owner    string
}

func newQuoteFixture(t *testing.T) quoteFixture {
	t.Helper()
	db := openQuoteDB(t)
	s, err := SpecFor(quoteInstanceName())
	if err != nil {
		t.Fatal(err)
	}
	if err := Provision(db, s); err != nil {
		t.Fatal(err)
	}
	mgr, err := NewMarketManager(db, s)
	if err != nil {
		t.Fatal(err)
	}
	m := mgr.(*marketManager)
	ctx := context.Background()
	n := Notice{ID: newID(), Picker: "picker", Wanted: "maize", State: models.NoticeOpen, CreatedAt: m.now()}
	if err := m.insert(ctx, RoleNotice, n); err != nil {
		t.Fatal(err)
	}
	p := Poster{ID: newID(), Name: "Supplier", Owner: "supplier", Status: models.PosterApproved, Availability: models.Available, CreatedAt: m.now()}
	if err := m.insert(ctx, RolePoster, p); err != nil {
		t.Fatal(err)
	}
	return quoteFixture{m: m, spec: s, noticeID: n.ID, posterID: p.ID, owner: p.Owner}
}

func quoteInstanceName() string {
	if os.Getenv("MARKET_TEST_POSTGRES_DSN") == "" {
		return "quotetest"
	}
	return "q" + strings.ReplaceAll(newID(), "-", "")[:12]
}

func openQuoteDB(t *testing.T) *gorm.DB {
	t.Helper()
	cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	if dsn := os.Getenv("MARKET_TEST_POSTGRES_DSN"); dsn != "" {
		db, err := gorm.Open(postgres.Open(dsn), cfg)
		if err != nil {
			t.Fatal(err)
		}
		return db
	}
	dsn := filepath.Join(t.TempDir(), "market.db") + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := gorm.Open(sqlite.Open(dsn), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func (f quoteFixture) quotesOf(t *testing.T) []Quote {
	t.Helper()
	qs, err := listOf[Quote](f.m, f.m.q(context.Background(), RoleQuote).Where("notice_id = ? AND poster_id = ?", f.noticeID, f.posterID), RoleQuote)
	if err != nil {
		t.Fatal(err)
	}
	return qs
}

func (f quoteFixture) send(price float64) (Quote, error) {
	return f.m.SendQuote(context.Background(), f.owner, f.noticeID, Quote{PosterID: f.posterID, Price: price})
}

func TestAPosterHasOneQuoteOnANotice(t *testing.T) {
	f := newQuoteFixture(t)
	first := Quote{ID: newID(), NoticeID: f.noticeID, PosterID: f.posterID, Price: 10, State: models.QuoteOpen, CreatedAt: f.m.now()}
	if err := f.m.insert(context.Background(), RoleQuote, first); err != nil {
		t.Fatal(err)
	}
	second := first
	second.ID = newID()
	if err := f.m.insert(context.Background(), RoleQuote, second); !gormutil.IsUniqueViolation(err) {
		t.Fatalf("a second quote from the same poster on the same notice must be refused as a unique violation, got %v", err)
	}
}

func TestQuotesSentAtOnceLeaveOneQuotePerPosterAndNotice(t *testing.T) {
	const sendsEach, rounds = 8, 5
	for round := 0; round < rounds; round++ {
		f := newQuoteFixture(t)
		var wg sync.WaitGroup
		results := make(chan error, sendsEach)
		for i := 1; i <= sendsEach; i++ {
			wg.Add(1)
			go func(price float64) {
				defer wg.Done()
				_, err := f.send(price)
				results <- err
			}(float64(i))
		}
		wg.Wait()
		close(results)
		stored := 0
		for err := range results {
			switch {
			case err == nil:
				stored++
			case !errors.Is(err, ErrConflict):
				t.Fatalf("round %d: a send that did not store must answer ErrConflict, got %v", round, err)
			}
		}
		if got := len(f.quotesOf(t)); got != 1 || stored != 1 {
			t.Fatalf("round %d: %d sends at once stored %d quotes and answered %d successes, want 1 and 1", round, sendsEach, got, stored)
		}
	}
}

func TestASecondQuoteSentAfterTheFirstIsRefused(t *testing.T) {
	f := newQuoteFixture(t)
	if _, err := f.send(10); err != nil {
		t.Fatal(err)
	}
	if _, err := f.send(12); !errors.Is(err, ErrConflict) {
		t.Fatalf("a second quote must answer ErrConflict, got %v", err)
	}
	if got := len(f.quotesOf(t)); got != 1 {
		t.Fatalf("want 1 quote, got %d", got)
	}
}

func TestProvisionAgainDropsDuplicateQuotesAndKeepsTheIndex(t *testing.T) {
	f := newQuoteFixture(t)
	ctx := context.Background()
	s := f.spec
	if err := f.m.db.Exec("drop index if exists " + s.IndexFor(f.m.st.Object(RoleQuote), quoteIndexNamed(t, s, "notice_poster"))).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		q := Quote{ID: newID(), NoticeID: f.noticeID, PosterID: f.posterID, Price: float64(i + 1), State: models.QuoteOpen, CreatedAt: f.m.now()}
		if err := f.m.insert(ctx, RoleQuote, q); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := Provision(f.m.db, s); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(f.quotesOf(t)); got != 1 {
		t.Fatalf("provisioning must leave one quote for the poster on the notice, got %d", got)
	}
	again := Quote{ID: newID(), NoticeID: f.noticeID, PosterID: f.posterID, Price: 9, State: models.QuoteOpen, CreatedAt: f.m.now()}
	if err := f.m.insert(ctx, RoleQuote, again); !gormutil.IsUniqueViolation(err) {
		t.Fatalf("the index must be back after provisioning, got %v", err)
	}
}

func quoteIndexNamed(t *testing.T, s *spec.Spec, name string) spec.Index {
	t.Helper()
	for _, o := range s.Objects {
		if o.Role != RoleQuote {
			continue
		}
		for _, idx := range o.Indexes {
			if idx.Name == name {
				return idx
			}
		}
	}
	t.Fatalf("no index %s on quote", name)
	return spec.Index{}
}

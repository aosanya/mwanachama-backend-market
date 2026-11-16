package market

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aosanya/mwanachama-backend-market/models"
)

func managersOverOneOrderFile(t *testing.T) []*marketManager {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "orders.db") + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)"
	s, err := SpecFor("orderrace")
	if err != nil {
		t.Fatal(err)
	}
	managers := make([]*marketManager, 2)
	for i := range managers {
		db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
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
	return managers
}

func placedOrderFor(t *testing.T, m *marketManager, posterID string) Order {
	t.Helper()
	ctx := context.Background()
	if err := m.insert(ctx, RolePoster, Poster{ID: posterID, Owner: "seller", Status: models.PosterApproved, CreatedAt: m.now()}); err != nil {
		t.Fatal(err)
	}
	o := Order{ID: newID(), Picker: "buyer", PosterID: posterID, Quantity: 1, UnitPrice: 1, WorkflowVersion: 1,
		State: models.OrderPlaced, CreatedAt: m.now()}
	if err := m.insert(ctx, RoleOrder, o); err != nil {
		t.Fatal(err)
	}
	return o
}

func TestCancelAndMarkPaidAtOnceNeverReportACancelThatIsLost(t *testing.T) {
	managers := managersOverOneOrderFile(t)
	ctx := context.Background()
	for round := 0; round < 25; round++ {
		o := placedOrderFor(t, managers[0], "poster-"+newID())
		var wg sync.WaitGroup
		var cancelErr, paidErr error
		start := make(chan struct{})
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, cancelErr = managers[0].CancelOrder(ctx, "buyer", o.ID)
		}()
		go func() {
			defer wg.Done()
			<-start
			_, paidErr = managers[1].MarkPaid(ctx, "seller", o.ID)
		}()
		close(start)
		wg.Wait()
		for _, err := range []error{cancelErr, paidErr} {
			if err != nil && !errors.Is(err, ErrConflict) {
				t.Fatalf("round %d: a lost call answered %v, want nil or ErrConflict", round, err)
			}
		}
		var got Order
		if err := managers[0].get(ctx, RoleOrder, o.ID, &got); err != nil {
			t.Fatal(err)
		}
		if cancelErr == nil && got.State != models.OrderCancelled {
			t.Errorf("round %d: cancel answered success but the order is %s", round, got.State)
		}
		if paidErr == nil && !got.Paid {
			t.Errorf("round %d: mark paid answered success but the order is not paid", round)
		}
		if cancelErr != nil && paidErr != nil {
			t.Errorf("round %d: both calls were refused", round)
		}
	}
}

func TestMarkPaidOnACancelledOrderConflicts(t *testing.T) {
	m := managersOverOneOrderFile(t)[0]
	ctx := context.Background()
	o := placedOrderFor(t, m, "poster-"+newID())
	if _, err := m.CancelOrder(ctx, "buyer", o.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.MarkPaid(ctx, "seller", o.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("marking a cancelled order paid answered %v, want ErrConflict", err)
	}
	var got Order
	if err := m.get(ctx, RoleOrder, o.ID, &got); err != nil {
		t.Fatal(err)
	}
	if got.Paid {
		t.Error("a cancelled order was marked paid")
	}
}

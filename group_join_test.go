package market

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aosanya/mwanachama-backend-market/models"
	"github.com/aosanya/mwanachama-backend-shared/gormutil"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func newJoinFixture(t *testing.T) (*marketManager, string) {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "market.db") + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	s, err := SpecFor("jointest")
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
	g := GroupOrder{ID: newID(), Coordinator: "coordinator", PosterID: "poster", Minimum: 10, TermsVersion: 1,
		TermsHistory: "[]", State: models.GroupOpen, Direction: models.DirectionBuy, CreatedAt: m.now()}
	if err := m.insert(context.Background(), RoleGroupOrder, g); err != nil {
		t.Fatal(err)
	}
	return m, g.ID
}

func linesOf(t *testing.T, m *marketManager, groupID, picker string) []Line {
	t.Helper()
	ls, err := listOf[Line](m, m.q(context.Background(), RoleLine).Where("group_order_id = ? AND picker = ?", groupID, picker), RoleLine)
	if err != nil {
		t.Fatal(err)
	}
	return ls
}

func TestAPickerHasOneLineInAGroupOrder(t *testing.T) {
	m, groupID := newJoinFixture(t)
	ctx := context.Background()
	first := Line{ID: newID(), GroupOrderID: groupID, Picker: "ada", Quantity: 1, Answer: models.AnswerYes, AnsweredVersion: 1, CreatedAt: m.now()}
	if err := m.insert(ctx, RoleLine, first); err != nil {
		t.Fatal(err)
	}
	second := first
	second.ID = newID()
	if err := m.insert(ctx, RoleLine, second); !gormutil.IsUniqueViolation(err) {
		t.Fatalf("a second line for the same picker must be refused as a unique violation, got %v", err)
	}
}

func TestJoinsArrivingAtOnceLeaveOneLinePerPicker(t *testing.T) {
	m, groupID := newJoinFixture(t)
	ctx := context.Background()
	const joinsEach, rounds = 8, 5
	for round := 0; round < rounds; round++ {
		picker := "picker-" + newID()
		var wg sync.WaitGroup
		for i := 1; i <= joinsEach; i++ {
			wg.Add(1)
			go func(quantity float64) {
				defer wg.Done()
				if _, err := m.JoinGroupOrder(ctx, picker, groupID, quantity); err != nil {
					t.Errorf("join of %v failed: %v", quantity, err)
				}
			}(float64(i))
		}
		wg.Wait()
		if got := len(linesOf(t, m, groupID, picker)); got != 1 {
			t.Fatalf("round %d: %d joins at once left %d lines for one picker, want 1", round, joinsEach, got)
		}
	}
}

func TestAJoinThatLosesTheRaceUpdatesTheLineThatWon(t *testing.T) {
	m, groupID := newJoinFixture(t)
	ctx := context.Background()
	won := Line{ID: newID(), GroupOrderID: groupID, Picker: "ada", Quantity: 2, Answer: models.AnswerYes, AnsweredVersion: 1, CreatedAt: m.now()}
	if err := m.insert(ctx, RoleLine, won); err != nil {
		t.Fatal(err)
	}
	l, err := m.settleJoin(ctx, groupID, Line{ID: newID(), GroupOrderID: groupID, Picker: "ada", Quantity: 5, Answer: models.AnswerYes, AnsweredVersion: 1, CreatedAt: m.now()})
	if err != nil {
		t.Fatal(err)
	}
	ls := linesOf(t, m, groupID, "ada")
	if len(ls) != 1 || ls[0].ID != won.ID || ls[0].Quantity != 5 || l.ID != won.ID {
		t.Fatalf("the losing join must update the winning line to 5, got %+v returned %+v", ls, l)
	}
}

func TestProvisionAgainDropsDuplicateLinesAndKeepsTheIndex(t *testing.T) {
	m, groupID := newJoinFixture(t)
	ctx := context.Background()
	s, err := SpecFor("jointest")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.db.Exec("drop index if exists " + s.IndexFor(m.st.Object(RoleLine), indexNamed(t, s, "group_picker"))).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		l := Line{ID: newID(), GroupOrderID: groupID, Picker: "ada", Quantity: float64(i + 1), Answer: models.AnswerYes, AnsweredVersion: 1, CreatedAt: m.now()}
		if err := m.insert(ctx, RoleLine, l); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := Provision(m.db, s); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(linesOf(t, m, groupID, "ada")); got != 1 {
		t.Fatalf("provisioning must leave one line for the picker, got %d", got)
	}
	again := Line{ID: newID(), GroupOrderID: groupID, Picker: "ada", Quantity: 9, Answer: models.AnswerYes, AnsweredVersion: 1, CreatedAt: m.now()}
	if err := m.insert(ctx, RoleLine, again); !gormutil.IsUniqueViolation(err) {
		t.Fatalf("the index must be back after provisioning, got %v", err)
	}
}

func indexNamed(t *testing.T, s *spec.Spec, name string) spec.Index {
	t.Helper()
	for _, o := range s.Objects {
		if o.Role != RoleLine {
			continue
		}
		for _, idx := range o.Indexes {
			if idx.Name == name {
				return idx
			}
		}
	}
	t.Fatalf("no index %s on line", name)
	return spec.Index{}
}

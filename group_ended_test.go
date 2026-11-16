package market

import (
	"context"
	"testing"
	"time"

	"github.com/aosanya/mwanachama-backend-market/models"
)

func fixedClock(m *marketManager, at time.Time) string {
	m.clock = func() time.Time { return at }
	return m.now()
}

func TestACalledOffGroupOrderCarriesWhenItWasCalledOff(t *testing.T) {
	m, groupID := newJoinFixture(t)
	ctx := context.Background()
	stamp := fixedClock(m, time.Date(2026, 11, 3, 9, 30, 0, 0, time.UTC))

	called, err := m.CallOffGroupOrder(ctx, "coordinator", groupID)
	if err != nil {
		t.Fatal(err)
	}
	if called.CancelledAt != stamp || called.ClosedAt != "" {
		t.Fatalf("call off returned cancelled_at %q closed_at %q, want %q and empty", called.CancelledAt, called.ClosedAt, stamp)
	}

	listed, err := m.ListGroupOrdersBy(ctx, "coordinator", 0, "", true, models.GroupCancelled)
	if err != nil || len(listed) != 1 {
		t.Fatalf("listing cancelled group orders: %v %d", err, len(listed))
	}
	if listed[0].CancelledAt != stamp || listed[0].ClosedAt != "" {
		t.Fatalf("list returned cancelled_at %q closed_at %q, want %q and empty", listed[0].CancelledAt, listed[0].ClosedAt, stamp)
	}

	got, err := m.GetGroupOrder(ctx, "coordinator", groupID)
	if err != nil || got.CancelledAt != stamp {
		t.Fatalf("get returned cancelled_at %q, want %q (%v)", got.CancelledAt, stamp, err)
	}
}

func TestAClosedGroupOrderCarriesWhenItWasClosed(t *testing.T) {
	m, groupID := newJoinFixture(t)
	ctx := context.Background()
	stamp := fixedClock(m, time.Date(2026, 11, 4, 16, 0, 0, 0, time.UTC))

	var g GroupOrder
	if err := m.get(ctx, RoleGroupOrder, groupID, &g); err != nil {
		t.Fatal(err)
	}
	was := g
	g.State = models.GroupClosed
	if err := m.updateGroupFrom(ctx, was, &g); err != nil {
		t.Fatal(err)
	}
	if g.ClosedAt != stamp {
		t.Fatalf("closing stamped %q on the order, want %q", g.ClosedAt, stamp)
	}

	listed, err := m.ListGroupOrdersBy(ctx, "coordinator", 0, "", true, models.GroupClosed)
	if err != nil || len(listed) != 1 {
		t.Fatalf("listing closed group orders: %v %d", err, len(listed))
	}
	if listed[0].ClosedAt != stamp || listed[0].CancelledAt != "" {
		t.Fatalf("list returned closed_at %q cancelled_at %q, want %q and empty", listed[0].ClosedAt, listed[0].CancelledAt, stamp)
	}
}

func TestAnOpenGroupOrderCarriesNoEndTime(t *testing.T) {
	m, groupID := newJoinFixture(t)
	got, err := m.GetGroupOrder(context.Background(), "coordinator", groupID)
	if err != nil || got.ClosedAt != "" || got.CancelledAt != "" {
		t.Fatalf("open order carries closed_at %q cancelled_at %q (%v)", got.ClosedAt, got.CancelledAt, err)
	}
}

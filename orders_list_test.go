package market

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aosanya/mwanachama-backend-market/models"
)

func ordersListed(t *testing.T, m *marketManager, caller string, limit int, after, state string) []Order {
	t.Helper()
	got, err := m.ListOrders(context.Background(), caller, limit, after, state)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func idsOf(orders []Order) []string {
	ids := []string{}
	for _, o := range orders {
		ids = append(ids, o.ID)
	}
	return ids
}

func seedOrder(t *testing.T, m *marketManager, id, state, deliveredAt, cancelledAt string) {
	t.Helper()
	o := Order{ID: id, Picker: "buyer", PosterID: "poster", Quantity: 1, UnitPrice: 1, WorkflowVersion: 1,
		State: state, DeliveredAt: deliveredAt, CancelledAt: cancelledAt, CreatedAt: m.now()}
	if err := m.insert(context.Background(), RoleOrder, o); err != nil {
		t.Fatal(err)
	}
}

func newOrdersFixture(t *testing.T) *marketManager {
	t.Helper()
	m, _ := newJoinFixture(t)
	if err := m.insert(context.Background(), RolePoster, Poster{ID: "poster", Owner: "seller", Status: models.PosterApproved, CreatedAt: m.now()}); err != nil {
		t.Fatal(err)
	}
	seedOrder(t, m, "o1", models.OrderPlaced, "", "")
	seedOrder(t, m, "o2", models.OrderDelivered, "2026-11-01T10:00:00Z", "")
	seedOrder(t, m, "o3", models.OrderCancelled, "", "2026-11-02T10:00:00Z")
	seedOrder(t, m, "o4", models.OrderDelivered, "2026-11-03T10:00:00Z", "")
	return m
}

func TestDeliveredOrdersListCarryWhenTheyWereDelivered(t *testing.T) {
	m := newOrdersFixture(t)
	got := ordersListed(t, m, "buyer", 0, "", models.OrderDelivered)
	if fmt.Sprint(idsOf(got)) != "[o2 o4]" {
		t.Fatalf("delivered orders = %v", idsOf(got))
	}
	if got[0].DeliveredAt != "2026-11-01T10:00:00Z" || got[1].DeliveredAt != "2026-11-03T10:00:00Z" {
		t.Fatalf("delivered_at = %q, %q", got[0].DeliveredAt, got[1].DeliveredAt)
	}
}

func TestCancelledOrdersListCarryWhenTheyWereCancelled(t *testing.T) {
	m := newOrdersFixture(t)
	got := ordersListed(t, m, "seller", 0, "", models.OrderCancelled)
	if fmt.Sprint(idsOf(got)) != "[o3]" || got[0].CancelledAt != "2026-11-02T10:00:00Z" {
		t.Fatalf("cancelled orders = %+v", got)
	}
}

func TestOrdersListPagesByIDAfterTheFilter(t *testing.T) {
	m := newOrdersFixture(t)
	first := ordersListed(t, m, "buyer", 1, "", models.OrderDelivered)
	second := ordersListed(t, m, "buyer", 1, first[0].ID, models.OrderDelivered)
	third := ordersListed(t, m, "buyer", 1, second[0].ID, models.OrderDelivered)
	if fmt.Sprint(idsOf(first), idsOf(second), idsOf(third)) != "[o2] [o4] []" {
		t.Fatalf("pages = %v %v %v", idsOf(first), idsOf(second), idsOf(third))
	}
}

func TestOrdersListWithoutAStateStillShowsEveryState(t *testing.T) {
	m := newOrdersFixture(t)
	if got := ordersListed(t, m, "buyer", 0, "", ""); len(got) != 4 {
		t.Fatalf("orders = %v", idsOf(got))
	}
	if got := ordersListed(t, m, "stranger", 0, "", ""); len(got) != 0 {
		t.Fatalf("a stranger sees %v", idsOf(got))
	}
}

func TestOrdersListRefusesAnUnknownStateAndANegativePage(t *testing.T) {
	m := newOrdersFixture(t)
	if _, err := m.ListOrders(context.Background(), "buyer", 0, "", "lost"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown state error = %v", err)
	}
	if _, err := m.ListOrders(context.Background(), "buyer", -1, "", ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("negative page error = %v", err)
	}
}

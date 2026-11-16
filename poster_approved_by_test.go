package market_test

import "testing"

func TestAPosterListHidesWhoApprovedFromAnyoneWhoDoesNotDecidePosters(t *testing.T) {
	m, ctx := newManager(t)
	approvedPoster(t, m, ctx, "ann")

	seen, err := m.ListPostersFor(ctx, "bea", "")
	if err != nil || len(seen) != 1 {
		t.Fatal(seen, err)
	}
	if seen[0].ApprovedBy != "" {
		t.Fatalf("a resident read approved_by %q", seen[0].ApprovedBy)
	}

	decider, err := m.ListPostersFor(ctx, "coordinator", "")
	if err != nil || len(decider) != 1 || decider[0].ApprovedBy != "coordinator" {
		t.Fatalf("the one who decides posters must read approved_by: %+v %v", decider, err)
	}
}

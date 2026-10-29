package market_test

import (
	"context"
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

func groupWithTwoLines(t *testing.T) (market.MarketManager, context.Context, market.GroupOrder, market.Line, market.Line) {
	t.Helper()
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	g, err := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	if err != nil {
		t.Fatal(err)
	}
	a, _ := m.JoinGroupOrder(ctx, "bea", g.ID, 6)
	b, _ := m.JoinGroupOrder(ctx, "cy", g.ID, 5)
	return m, ctx, g, a, b
}

func TestOnlyTheCoordinatorSendsAnOpenGroupOrderAndNobodyJoinsAfter(t *testing.T) {
	m, ctx, g, _, _ := groupWithTwoLines(t)
	for _, caller := range []string{"bea", "ann", "coordinator", ""} {
		if _, err := m.SendGroupOrder(ctx, caller, g.ID); !errors.Is(err, market.ErrForbidden) {
			t.Fatalf("%q: want forbidden, got %v", caller, err)
		}
	}
	got, err := m.SendGroupOrder(ctx, "coord", g.ID)
	if err != nil || got.State != models.GroupSent {
		t.Fatalf("send: %+v %v", got, err)
	}
	if _, err := m.SendGroupOrder(ctx, "coord", g.ID); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("a second send: want conflict, got %v", err)
	}
	if _, err := m.JoinGroupOrder(ctx, "dee", g.ID, 1); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("join after send: want conflict, got %v", err)
	}
	if _, err := m.SendGroupOrder(ctx, "coord", "missing"); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestARevisingGroupOrderIsNotSent(t *testing.T) {
	m, ctx, g, _, _ := groupWithTwoLines(t)
	if _, err := m.AskRevision(ctx, "ann", g.ID, 4, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := m.SendGroupOrder(ctx, "coord", g.ID); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
}

func TestOnlyTheCoordinatorMarksALinePaidOnASentOrderAndItIsTrailed(t *testing.T) {
	m, ctx, g, a, b := groupWithTwoLines(t)
	if _, err := m.MarkLinePaid(ctx, "coord", g.ID, a.ID); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("before send: want conflict, got %v", err)
	}
	m.SendGroupOrder(ctx, "coord", g.ID)
	for _, caller := range []string{"bea", "ann", "coordinator"} {
		if _, err := m.MarkLinePaid(ctx, caller, g.ID, a.ID); !errors.Is(err, market.ErrForbidden) {
			t.Fatalf("%q: want forbidden, got %v", caller, err)
		}
	}
	got, err := m.MarkLinePaid(ctx, "coord", g.ID, a.ID)
	if err != nil || !got.Paid {
		t.Fatalf("paid: %+v %v", got, err)
	}
	if _, err := m.MarkLinePaid(ctx, "coord", g.ID, "missing"); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
	v, _ := m.GetGroupOrder(ctx, "cy", g.ID)
	if len(v.Lines) != 1 || v.Lines[0].ID != b.ID || v.Lines[0].Paid || v.Total != 11 || v.Count != 2 {
		t.Fatalf("cy sees only her own unpaid line plus the totals: %+v", v)
	}
	mine, _ := m.GetGroupOrder(ctx, "bea", g.ID)
	if len(mine.Lines) != 1 || !mine.Lines[0].Paid {
		t.Fatalf("bea sees her own line paid: %+v", mine.Lines)
	}
	tr, _ := m.GroupOrderTrail(ctx, "coord", g.ID)
	last := tr[len(tr)-1]
	if last.What != "marked a line paid" || last.Actor != "coord" {
		t.Fatalf("trail: %+v", tr)
	}
}

func TestALineOfAnotherOrderIsNotPaid(t *testing.T) {
	m, ctx, g, a, _ := groupWithTwoLines(t)
	other, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: g.PosterID, Price: 3, Minimum: 10})
	m.SendGroupOrder(ctx, "coord", other.ID)
	if _, err := m.MarkLinePaid(ctx, "coord", other.ID, a.ID); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("another order's line: want not found, got %v", err)
	}
}

func TestAReplacedGroupOrderWorkflowNamesWhoSendsAndWhoMarksLinesPaid(t *testing.T) {
	m, ctx := newManager(t)
	m.UseWorkflows(func(_ context.Context, name string, version int) (market.Workflow, bool, error) {
		if name != market.WorkflowGroupOrder {
			return market.Workflow{}, false, nil
		}
		return market.Workflow{Name: name, Version: 1, Steps: []market.Step{
			{Name: market.StepStart, Who: market.WhoCoordinator}, {Name: market.StepSent, Who: market.WhoCoordinator},
			{Name: market.StepLinePaid, Who: market.WhoPoster}}}, true, nil
	})
	p := approvedPoster(t, m, ctx, "ann")
	g, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	l, _ := m.JoinGroupOrder(ctx, "bea", g.ID, 6)
	m.SendGroupOrder(ctx, "coord", g.ID)
	if _, err := m.MarkLinePaid(ctx, "coord", g.ID, l.ID); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("coordinator must be refused, got %v", err)
	}
	if got, err := m.MarkLinePaid(ctx, "ann", g.ID, l.ID); err != nil || !got.Paid {
		t.Fatalf("poster marks it: %+v %v", got, err)
	}
}

func TestAGroupOrderKeepsItsStampedWorkflowForSendAndLinePaid(t *testing.T) {
	m, ctx := newManager(t)
	current := 1
	m.UseWorkflows(func(_ context.Context, name string, version int) (market.Workflow, bool, error) {
		if name != market.WorkflowGroupOrder {
			return market.Workflow{}, false, nil
		}
		n := current
		if version > 0 {
			n = version
		}
		who := market.WhoCoordinator
		if n >= 2 {
			who = market.WhoNone
		}
		return market.Workflow{Name: name, Version: n, Steps: []market.Step{
			{Name: market.StepStart, Who: market.WhoCoordinator}, {Name: market.StepSent, Who: who},
			{Name: market.StepLinePaid, Who: who}}}, true, nil
	})
	p := approvedPoster(t, m, ctx, "ann")
	old, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	l, _ := m.JoinGroupOrder(ctx, "bea", old.ID, 6)
	current = 2
	fresh, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	if _, err := m.SendGroupOrder(ctx, "coord", fresh.ID); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("version 2 sends nobody, got %v", err)
	}
	if _, err := m.SendGroupOrder(ctx, "coord", old.ID); err != nil {
		t.Fatalf("version 1 still lets the coordinator send: %v", err)
	}
	if got, err := m.MarkLinePaid(ctx, "coord", old.ID, l.ID); err != nil || !got.Paid {
		t.Fatalf("version 1 still lets the coordinator mark paid: %+v %v", got, err)
	}
}

func TestTheDefaultGroupOrderWorkflowHasTheSendAndLinePaidSteps(t *testing.T) {
	for _, w := range market.DefaultWorkflows() {
		if w.Name != market.WorkflowGroupOrder {
			continue
		}
		have := map[string]string{}
		for _, s := range w.Steps {
			have[s.Name] = s.Who
		}
		if have[market.StepSent] != market.WhoCoordinator || have[market.StepLinePaid] != market.WhoCoordinator {
			t.Fatalf("steps: %+v", w.Steps)
		}
		return
	}
	t.Fatal("no group order workflow")
}

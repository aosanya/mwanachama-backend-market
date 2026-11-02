package market_test

import (
	"context"
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

func TestTheDefaultGroupOrderWorkflowHasTheReopenedStepAndNoPaidStep(t *testing.T) {
	for _, w := range market.DefaultWorkflows() {
		if w.Name != market.WorkflowGroupOrder {
			continue
		}
		have := map[string]string{}
		for _, s := range w.Steps {
			have[s.Name] = s.Who
		}
		if have[market.StepReopened] != market.WhoCoordinator {
			t.Fatalf("steps: %+v", w.Steps)
		}
		if _, ok := have[market.StepPaid]; ok {
			t.Fatalf("the group order has no paid step: %+v", w.Steps)
		}
		return
	}
	t.Fatal("no group order workflow")
}

func TestOnlyTheCoordinatorTakesARevisingGroupOrderBackToOpen(t *testing.T) {
	m, ctx, g, _, _ := revisingWithTwoSilentLines(t)
	for _, caller := range []string{"bea", "ann", "coordinator", ""} {
		if _, err := m.ReopenGroupOrder(ctx, caller, g.ID); !errors.Is(err, market.ErrForbidden) {
			t.Fatalf("%q: want forbidden, got %v", caller, err)
		}
	}
	got, err := m.ReopenGroupOrder(ctx, "coord", g.ID)
	if err != nil || got.State != models.GroupOpen || got.TermsVersion != 2 {
		t.Fatalf("the coordinator reopens: %+v %v", got, err)
	}
	if _, err := m.SendGroupOrder(ctx, "coord", g.ID); err != nil {
		t.Fatalf("send is offered after the reopen: %v", err)
	}
}

func TestAGroupOrderThatIsNotRevisingIsNotReopened(t *testing.T) {
	m, ctx, g, _, _ := groupWithTwoLines(t)
	if _, err := m.ReopenGroupOrder(ctx, "coord", g.ID); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("an open order: want conflict, got %v", err)
	}
	if _, err := m.SendGroupOrder(ctx, "coord", g.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ReopenGroupOrder(ctx, "coord", g.ID); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("a sent order: want conflict, got %v", err)
	}
}

func TestAReopenIsWrittenInTheGroupOrdersTrail(t *testing.T) {
	m, ctx, g, _, _ := revisingWithTwoSilentLines(t)
	if _, err := m.ReopenGroupOrder(ctx, "coord", g.ID); err != nil {
		t.Fatal(err)
	}
	trail, err := m.GroupOrderTrail(ctx, "coord", g.ID)
	if err != nil {
		t.Fatal(err)
	}
	last := trail[len(trail)-1]
	if last.What != "took the revised order back to open" {
		t.Fatalf("trail: %+v", trail)
	}
}

func TestAReplacedGroupOrderWorkflowNamesWhoReopensOrIsRefusedWithoutTheStep(t *testing.T) {
	steps := func(extra ...market.Step) []market.Step {
		return append([]market.Step{{Name: market.StepStart, Who: market.WhoCoordinator}}, extra...)
	}
	cases := []struct {
		name   string
		extra  []market.Step
		caller string
		want   error
	}{
		{"poster", []market.Step{{Name: market.StepReopened, Who: market.WhoPoster}}, "ann", nil},
		{"poster refuses the coordinator", []market.Step{{Name: market.StepReopened, Who: market.WhoPoster}}, "coord", market.ErrForbidden},
		{"nobody", []market.Step{{Name: market.StepReopened, Who: market.WhoNone}}, "coord", market.ErrForbidden},
		{"no step", nil, "coord", market.ErrInvalid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, ctx := newManager(t)
			m.UseWorkflows(func(_ context.Context, name string, version int) (market.Workflow, bool, error) {
				if name != market.WorkflowGroupOrder {
					return market.Workflow{}, false, nil
				}
				return market.Workflow{Name: name, Version: 1, Steps: steps(c.extra...)}, true, nil
			})
			p := approvedPoster(t, m, ctx, "ann")
			g, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
			m.JoinGroupOrder(ctx, "bea", g.ID, 6)
			m.AskRevision(ctx, "ann", g.ID, 4, 10)
			_, err := m.ReopenGroupOrder(ctx, c.caller, g.ID)
			if c.want == nil && err != nil || c.want != nil && !errors.Is(err, c.want) {
				t.Fatalf("want %v, got %v", c.want, err)
			}
		})
	}
}

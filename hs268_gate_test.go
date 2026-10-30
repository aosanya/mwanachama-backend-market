package market_test

import (
	"context"
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

func revisingWithTwoSilentLines(t *testing.T) (market.MarketManager, context.Context, market.GroupOrder, market.Line, market.Line) {
	t.Helper()
	m, ctx, g, a, b := groupWithTwoLines(t)
	if _, err := m.AskRevision(ctx, "ann", g.ID, 4, 10); err != nil {
		t.Fatal(err)
	}
	return m, ctx, g, a, b
}

func TestOnlyTheCoordinatorDecidesForASilentLine(t *testing.T) {
	m, ctx, g, a, _ := revisingWithTwoSilentLines(t)
	for _, caller := range []string{"bea", "ann", "coordinator", ""} {
		if _, err := m.DecideSilentLine(ctx, caller, g.ID, a.ID, market.DecisionKeep); !errors.Is(err, market.ErrForbidden) {
			t.Fatalf("%q: want forbidden, got %v", caller, err)
		}
	}
	v, _ := m.GetGroupOrder(ctx, "coord", g.ID)
	for _, l := range v.Lines {
		if l.Answer != models.AnswerWaiting || l.DecidedByCoordinator {
			t.Fatalf("a refused decision changed %+v", l)
		}
	}
}

func TestKeepingASilentLineCountsItAtTheCurrentTermsAndMarksTheCoordinatorDecided(t *testing.T) {
	m, ctx, g, a, b := revisingWithTwoSilentLines(t)
	m.AnswerRevision(ctx, "cy", g.ID, models.AnswerNo)
	got, err := m.DecideSilentLine(ctx, "coord", g.ID, a.ID, market.DecisionKeep)
	if err != nil || got.Answer != models.AnswerYes || !got.DecidedByCoordinator || got.AnsweredVersion != 2 {
		t.Fatalf("keep: %+v %v", got, err)
	}
	v, _ := m.GetGroupOrder(ctx, "coord", g.ID)
	if v.Total != 6 || v.Count != 1 {
		t.Fatalf("the kept line counts: total %v count %d", v.Total, v.Count)
	}
	for _, l := range v.Lines {
		if l.ID == b.ID && l.DecidedByCoordinator {
			t.Fatalf("cy answered herself: %+v", l)
		}
	}
	tr, _ := m.GroupOrderTrail(ctx, "coord", g.ID)
	last := tr[len(tr)-1]
	if last.What != "kept a silent line" || last.Actor != "coord" {
		t.Fatalf("trail: %+v", tr)
	}
}

func TestTakingASilentLineOffRemovesItFromTheTotalAndCount(t *testing.T) {
	m, ctx, g, a, _ := revisingWithTwoSilentLines(t)
	got, err := m.DecideSilentLine(ctx, "coord", g.ID, a.ID, market.DecisionTakeOff)
	if err != nil || got.Answer != models.AnswerNo || !got.DecidedByCoordinator || got.AnsweredVersion != 2 {
		t.Fatalf("take off: %+v %v", got, err)
	}
	v, _ := m.GetGroupOrder(ctx, "coord", g.ID)
	if v.Total != 5 || v.Count != 1 {
		t.Fatalf("total %v count %d", v.Total, v.Count)
	}
	tr, _ := m.GroupOrderTrail(ctx, "coord", g.ID)
	if tr[len(tr)-1].What != "took a silent line off" {
		t.Fatalf("trail: %+v", tr)
	}
}

func TestOnlyASilentLineIsDecidedAndOnlyOnce(t *testing.T) {
	m, ctx, g, a, b := revisingWithTwoSilentLines(t)
	m.AnswerRevision(ctx, "cy", g.ID, models.AnswerYes)
	if _, err := m.DecideSilentLine(ctx, "coord", g.ID, b.ID, market.DecisionTakeOff); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("a line that answered: want conflict, got %v", err)
	}
	if _, err := m.DecideSilentLine(ctx, "coord", g.ID, a.ID, "later"); !errors.Is(err, market.ErrInvalid) {
		t.Fatalf("an unknown decision: want invalid, got %v", err)
	}
	if _, err := m.DecideSilentLine(ctx, "coord", g.ID, a.ID, ""); !errors.Is(err, market.ErrInvalid) {
		t.Fatalf("no decision: want invalid, got %v", err)
	}
	if _, err := m.DecideSilentLine(ctx, "coord", g.ID, a.ID, market.DecisionKeep); err != nil {
		t.Fatal(err)
	}
	if _, err := m.DecideSilentLine(ctx, "coord", g.ID, a.ID, market.DecisionTakeOff); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("a second decision: want conflict, got %v", err)
	}
	if _, err := m.DecideSilentLine(ctx, "coord", g.ID, "missing", market.DecisionKeep); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
	if _, err := m.DecideSilentLine(ctx, "coord", "missing", a.ID, market.DecisionKeep); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestALineOfAnotherGroupOrderIsNotDecided(t *testing.T) {
	m, ctx, g, a, _ := revisingWithTwoSilentLines(t)
	other, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: g.PosterID, Price: 3, Minimum: 10})
	if _, err := m.DecideSilentLine(ctx, "coord", other.ID, a.ID, market.DecisionKeep); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestNoLineIsDecidedOnASentOrCalledOffGroupOrder(t *testing.T) {
	m, ctx, g, a, _ := revisingWithTwoSilentLines(t)
	if _, err := m.CallOffGroupOrder(ctx, "coord", g.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.DecideSilentLine(ctx, "coord", g.ID, a.ID, market.DecisionKeep); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
}

func TestOnlyTheCoordinatorCallsAGroupOrderOffAtAnyStepAndOnlyOnce(t *testing.T) {
	steps := map[string]func(market.MarketManager, context.Context, market.GroupOrder){
		"open": func(market.MarketManager, context.Context, market.GroupOrder) {},
		"revising": func(m market.MarketManager, ctx context.Context, g market.GroupOrder) {
			m.AskRevision(ctx, "ann", g.ID, 4, 10)
		},
		"sent": func(m market.MarketManager, ctx context.Context, g market.GroupOrder) {
			m.SendGroupOrder(ctx, "coord", g.ID)
		},
	}
	for name, reach := range steps {
		t.Run(name, func(t *testing.T) {
			m, ctx, g, a, _ := groupWithTwoLines(t)
			reach(m, ctx, g)
			for _, caller := range []string{"bea", "ann", "coordinator", ""} {
				if _, err := m.CallOffGroupOrder(ctx, caller, g.ID); !errors.Is(err, market.ErrForbidden) {
					t.Fatalf("%q: want forbidden, got %v", caller, err)
				}
			}
			got, err := m.CallOffGroupOrder(ctx, "coord", g.ID)
			if err != nil || got.State != models.GroupCancelled {
				t.Fatalf("call off: %+v %v", got, err)
			}
			if _, err := m.CallOffGroupOrder(ctx, "coord", g.ID); !errors.Is(err, market.ErrConflict) {
				t.Fatalf("a second call off: want conflict, got %v", err)
			}
			if _, err := m.JoinGroupOrder(ctx, "dee", g.ID, 1); !errors.Is(err, market.ErrConflict) {
				t.Fatalf("join after call off: want conflict, got %v", err)
			}
			if _, err := m.MarkLinePaid(ctx, "coord", g.ID, a.ID); !errors.Is(err, market.ErrConflict) {
				t.Fatalf("pay after call off: want conflict, got %v", err)
			}
			tr, _ := m.GroupOrderTrail(ctx, "coord", g.ID)
			if tr[len(tr)-1].What != "called off" {
				t.Fatalf("trail: %+v", tr)
			}
		})
	}
	m, ctx, _, _, _ := groupWithTwoLines(t)
	if _, err := m.CallOffGroupOrder(ctx, "coord", "missing"); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestAReplacedGroupOrderWorkflowNamesWhoDecidesAndWhoCallsOff(t *testing.T) {
	m, ctx := newManager(t)
	m.UseWorkflows(func(_ context.Context, name string, version int) (market.Workflow, bool, error) {
		if name != market.WorkflowGroupOrder {
			return market.Workflow{}, false, nil
		}
		return market.Workflow{Name: name, Version: 1, Steps: []market.Step{
			{Name: market.StepStart, Who: market.WhoCoordinator}, {Name: market.StepDecided, Who: market.WhoPoster},
			{Name: market.StepCalledOff, Who: market.WhoNone}}}, true, nil
	})
	p := approvedPoster(t, m, ctx, "ann")
	g, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	l, _ := m.JoinGroupOrder(ctx, "bea", g.ID, 6)
	m.AskRevision(ctx, "ann", g.ID, 4, 10)
	if _, err := m.DecideSilentLine(ctx, "coord", g.ID, l.ID, market.DecisionKeep); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("the coordinator must be refused, got %v", err)
	}
	if got, err := m.DecideSilentLine(ctx, "ann", g.ID, l.ID, market.DecisionKeep); err != nil || got.Answer != models.AnswerYes {
		t.Fatalf("the poster decides: %+v %v", got, err)
	}
	if _, err := m.CallOffGroupOrder(ctx, "coord", g.ID); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("nobody calls off, got %v", err)
	}
}

func TestAGroupOrderKeepsItsStampedWorkflowForDecidedAndCalledOff(t *testing.T) {
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
			{Name: market.StepStart, Who: market.WhoCoordinator}, {Name: market.StepDecided, Who: who},
			{Name: market.StepCalledOff, Who: who}}}, true, nil
	})
	p := approvedPoster(t, m, ctx, "ann")
	old, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	l, _ := m.JoinGroupOrder(ctx, "bea", old.ID, 6)
	m.AskRevision(ctx, "ann", old.ID, 4, 10)
	current = 2
	fresh, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	if _, err := m.CallOffGroupOrder(ctx, "coord", fresh.ID); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("version 2 calls off nobody, got %v", err)
	}
	if _, err := m.DecideSilentLine(ctx, "coord", old.ID, l.ID, market.DecisionTakeOff); err != nil {
		t.Fatalf("version 1 still lets the coordinator decide: %v", err)
	}
	if _, err := m.CallOffGroupOrder(ctx, "coord", old.ID); err != nil {
		t.Fatalf("version 1 still lets the coordinator call off: %v", err)
	}
}

func TestTheDefaultGroupOrderWorkflowHasTheDecidedAndCalledOffSteps(t *testing.T) {
	for _, w := range market.DefaultWorkflows() {
		if w.Name != market.WorkflowGroupOrder {
			continue
		}
		have := map[string]string{}
		for _, s := range w.Steps {
			have[s.Name] = s.Who
		}
		if have[market.StepDecided] != market.WhoCoordinator || have[market.StepCalledOff] != market.WhoCoordinator {
			t.Fatalf("steps: %+v", w.Steps)
		}
		return
	}
	t.Fatal("no group order workflow")
}

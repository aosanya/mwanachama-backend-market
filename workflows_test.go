package market_test

import (
	"context"
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

type versioned struct {
	current    int
	cancel     *market.Workflow
	cancelFrom int
}

func (v *versioned) source(_ context.Context, name string, version int) (market.Workflow, bool, error) {
	n := v.current
	if version > 0 {
		n = version
	}
	switch name {
	case market.WorkflowBuyNow:
		return market.Workflow{Name: name, Version: n, Steps: []market.Step{
			{Name: market.StepAsked, Who: market.WhoPicker}, {Name: market.StepYes, Who: market.WhoPoster}, {Name: market.StepPaid, Who: market.WhoPoster}}}, true, nil
	case market.WorkflowGroupOrder:
		return market.Workflow{Name: name, Version: n, Steps: []market.Step{{Name: market.StepStart, Who: market.WhoCoordinator}}}, true, nil
	case market.WorkflowCancel:
		if v.cancel != nil && n >= v.cancelFrom {
			w := *v.cancel
			w.Version = n
			return w, true, nil
		}
	}
	return market.Workflow{}, false, nil
}

func placed(t *testing.T, m market.MarketManager, ctx context.Context) (market.Poster, market.Order) {
	t.Helper()
	p := approvedPoster(t, m, ctx, "ann")
	o, err := m.CreateOffer(ctx, "ann", market.Offer{PosterID: p.ID, ProductWords: "beans", Price: 10})
	if err != nil {
		t.Fatal(err)
	}
	ord, err := m.BuyNow(ctx, "bea", o.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	return p, ord
}

func TestOnlyACoordinatorApprovesAPoster(t *testing.T) {
	m, ctx := newManager(t)
	p, _ := m.ApplyPoster(ctx, "ann", market.Poster{Name: "Ann"})
	if _, err := m.ApprovePoster(ctx, "ann", p.ID); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("want forbidden, got %v", err)
	}
	if _, err := m.DeclinePoster(ctx, "", p.ID); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("want forbidden, got %v", err)
	}
}

func TestOnlyACoordinatorStartsAGroupOrder(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	_, err := m.StartGroupOrder(ctx, "bea", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	if !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("want forbidden, got %v", err)
	}
}

func TestAPosterAlonePerformsHisYesAndPaidSteps(t *testing.T) {
	m, ctx := newManager(t)
	p, ord := placed(t, m, ctx)
	r, _ := m.CreateRequest(ctx, "bea", market.Request{Wanted: "beans"})
	a, _ := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: r.ID, Question: "any?"})
	if _, err := m.AnswerAsk(ctx, "bea", a.ID, models.AnswerYes, "2026-11-01", ""); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("want forbidden, got %v", err)
	}
	if _, err := m.MarkPaid(ctx, "bea", ord.ID); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("want forbidden, got %v", err)
	}
}

func TestADefaultCancelIsAtOnceByThePickerAndClosesThePostersYes(t *testing.T) {
	m, ctx := newManager(t)
	p, ord := placed(t, m, ctx)
	r, _ := m.CreateRequest(ctx, "bea", market.Request{Wanted: "beans"})
	a, _ := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: r.ID, Question: "any?"})
	if _, err := m.AnswerAsk(ctx, "ann", a.ID, models.AnswerYes, "2026-11-01", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := m.CancelOrder(ctx, "ann", ord.ID); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("poster must not cancel by default, got %v", err)
	}
	got, err := m.CancelOrder(ctx, "bea", ord.ID)
	if err != nil || got.State != models.OrderCancelled {
		t.Fatal(got.State, err)
	}
	ys, _ := m.ListYeses(ctx, "bea")
	if len(ys) != 1 || ys[0].State != models.YesCalledOff {
		t.Fatalf("the open yes must close: %+v", ys)
	}
}

func TestAReplacedCancelWorkflowChangesWhoCancels(t *testing.T) {
	m, ctx := newManager(t)
	v := &versioned{current: 1, cancel: &market.Workflow{Name: market.WorkflowCancel, Steps: []market.Step{{Name: market.StepCancelled, Who: market.WhoPoster}}}}
	m.UseWorkflows(v.source)
	_, ord := placed(t, m, ctx)
	if _, err := m.CancelOrder(ctx, "bea", ord.ID); !errors.Is(err, market.ErrForbidden) {
		t.Fatalf("picker must be refused, got %v", err)
	}
	got, err := m.CancelOrder(ctx, "ann", ord.ID)
	if err != nil || got.State != models.OrderCancelled {
		t.Fatal(got.State, err)
	}
	ys, _ := m.ListYeses(ctx, "bea")
	for _, y := range ys {
		if y.State != models.YesOpen {
			t.Fatal("a replaced cancel that does not close yeses must leave them open")
		}
	}
}

func TestOrdersAndGroupOrdersAreStampedWithTheWorkflowVersion(t *testing.T) {
	m, ctx := newManager(t)
	v := &versioned{current: 3}
	m.UseWorkflows(v.source)
	p, ord := placed(t, m, ctx)
	if ord.WorkflowVersion != 3 {
		t.Fatalf("order version %d", ord.WorkflowVersion)
	}
	g, err := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	if err != nil || g.WorkflowVersion != 3 {
		t.Fatal(g.WorkflowVersion, err)
	}
}

func TestAnOrderKeepsItsStartedVersionWhenTheSourceMovesOn(t *testing.T) {
	m, ctx := newManager(t)
	v := &versioned{current: 1}
	m.UseWorkflows(v.source)
	_, ord := placed(t, m, ctx)
	v.current, v.cancelFrom = 2, 2
	v.cancel = &market.Workflow{Name: market.WorkflowCancel, Steps: []market.Step{{Name: market.StepCancelled, Who: market.WhoCoordinator}}}
	got, err := m.GetOrder(ctx, "bea", ord.ID)
	if err != nil || got.WorkflowVersion != 1 {
		t.Fatal(got.WorkflowVersion, err)
	}
	if _, err := m.CancelOrder(ctx, "bea", ord.ID); err != nil {
		t.Fatalf("version 1 cancel is still the picker's: %v", err)
	}
}

func TestAnUnusableWorkflowFromTheSourceIsRefused(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	o, _ := m.CreateOffer(ctx, "ann", market.Offer{PosterID: p.ID, ProductWords: "beans", Price: 10})
	m.UseWorkflows(func(_ context.Context, name string, _ int) (market.Workflow, bool, error) {
		return market.Workflow{Name: name, Version: 1, Steps: []market.Step{{Name: "x", Who: "anyone"}}}, true, nil
	})
	if _, err := m.BuyNow(ctx, "bea", o.ID, 1); !errors.Is(err, market.ErrInvalid) {
		t.Fatalf("want invalid, got %v", err)
	}
}

func TestAWorkflowThatNamesAnotherActorRefusesThePosterAtThatStep(t *testing.T) {
	for _, step := range []string{market.StepYes, market.StepPaid} {
		m, ctx := newManager(t)
		p, ord := placed(t, m, ctx)
		r, _ := m.CreateRequest(ctx, "bea", market.Request{Wanted: "beans"})
		a, _ := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: r.ID, Question: "any?"})
		m.UseWorkflows(func(_ context.Context, name string, _ int) (market.Workflow, bool, error) {
			if name != market.WorkflowBuyNow {
				return market.Workflow{}, false, nil
			}
			steps := []market.Step{{Name: market.StepAsked, Who: market.WhoPicker}, {Name: market.StepYes, Who: market.WhoPoster}, {Name: market.StepPaid, Who: market.WhoPoster}}
			for i := range steps {
				if steps[i].Name == step {
					steps[i].Who = market.WhoCoordinator
				}
			}
			return market.Workflow{Name: name, Version: 1, Steps: steps}, true, nil
		})
		var err error
		if step == market.StepYes {
			_, err = m.AnswerAsk(ctx, "ann", a.ID, models.AnswerYes, "2026-11-01", "")
		} else {
			_, err = m.MarkPaid(ctx, "ann", ord.ID)
		}
		if !errors.Is(err, market.ErrForbidden) {
			t.Fatalf("%s: want forbidden, got %v", step, err)
		}
	}
}

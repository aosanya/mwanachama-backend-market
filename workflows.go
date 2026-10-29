package market

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-market/models"
)

const (
	WhoPicker      = "picker"
	WhoPoster      = "poster"
	WhoCoordinator = "coordinator"
	WhoNone        = "none"

	WorkflowBuyNow        = "buy_now"
	WorkflowCancel        = "cancel"
	WorkflowGroupOrder    = "group_order"
	WorkflowApprovePoster = "approve_poster"

	StepAsked     = "asked"
	StepYes       = "yes"
	StepPaid      = "paid"
	StepCancelled = "cancelled"
	StepStart     = "start"
	StepApproved  = "approved"
)

type Step struct {
	Name      string `json:"name"`
	Who       string `json:"who"`
	ClosesYes bool   `json:"closes_yes,omitempty"`
}

type Workflow struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
	Steps   []Step `json:"steps"`
}

type WorkflowSource func(ctx context.Context, name string, version int) (Workflow, bool, error)

type CoordinatorCheck func(ctx context.Context, caller string) bool

func DefaultWorkflows() []Workflow {
	return []Workflow{
		{Name: WorkflowBuyNow, Version: 1, Steps: []Step{
			{Name: StepAsked, Who: WhoPicker}, {Name: StepYes, Who: WhoPoster}, {Name: StepPaid, Who: WhoPoster}}},
		{Name: WorkflowCancel, Version: 1, Steps: []Step{{Name: StepCancelled, Who: WhoPicker, ClosesYes: true}}},
		{Name: WorkflowGroupOrder, Version: 1, Steps: []Step{
			{Name: StepStart, Who: WhoCoordinator}, {Name: StepPaid, Who: WhoCoordinator}}},
		{Name: WorkflowApprovePoster, Version: 1, Steps: []Step{{Name: StepApproved, Who: WhoCoordinator}}},
	}
}

func defaultWorkflow(name string) (Workflow, bool) {
	for _, w := range DefaultWorkflows() {
		if w.Name == name {
			return w, true
		}
	}
	return Workflow{}, false
}

func (m *marketManager) UseWorkflows(s WorkflowSource) { m.source = s }

func (m *marketManager) UseNames(n NameOf) { m.names = n }

func (m *marketManager) UseCoordinators(c CoordinatorCheck) { m.coordinators = c }

type NameOf func(ctx context.Context, id string) string

func (m *marketManager) nameOf(ctx context.Context, id string) string {
	if m.names == nil || id == "" {
		return ""
	}
	return m.names(ctx, id)
}

func validWho(who string) bool {
	return who == WhoPicker || who == WhoPoster || who == WhoCoordinator || who == WhoNone
}

func (w Workflow) valid() bool {
	if w.Version < 1 || len(w.Steps) == 0 {
		return false
	}
	for _, s := range w.Steps {
		if s.Name == "" || !validWho(s.Who) {
			return false
		}
	}
	return true
}

func (m *marketManager) workflow(ctx context.Context, name string, version int) (Workflow, error) {
	if m.source != nil {
		w, found, err := m.source(ctx, name, version)
		if err != nil {
			return Workflow{}, err
		}
		if found {
			if w.Name != name || !w.valid() || (version > 0 && w.Version != version) {
				return Workflow{}, fmt.Errorf("%w: the workflow %q is not usable", ErrInvalid, name)
			}
			return w, nil
		}
	}
	w, ok := defaultWorkflow(name)
	if !ok {
		return w, fmt.Errorf("%w: no workflow %q", ErrInvalid, name)
	}
	return w, nil
}

type parties struct {
	picker      string
	posterID    string
	coordinator string
}

func (m *marketManager) gate(ctx context.Context, name string, version int, step, caller string, p parties) (Workflow, Step, error) {
	w, err := m.workflow(ctx, name, version)
	if err != nil {
		return w, Step{}, err
	}
	for _, s := range w.Steps {
		if s.Name != step {
			continue
		}
		if !m.isWho(ctx, s.Who, caller, p) {
			return w, s, ErrForbidden
		}
		return w, s, nil
	}
	return w, Step{}, fmt.Errorf("%w: workflow %q has no step %q", ErrInvalid, name, step)
}

func (m *marketManager) isWho(ctx context.Context, who, caller string, p parties) bool {
	if caller == "" {
		return false
	}
	switch who {
	case WhoPicker:
		return p.picker == caller
	case WhoPoster:
		_, err := m.posterOwnedBy(ctx, caller, p.posterID)
		return err == nil
	case WhoCoordinator:
		if p.coordinator != "" {
			return p.coordinator == caller
		}
		return m.coordinators != nil && m.coordinators(ctx, caller)
	}
	return false
}

func (m *marketManager) closeYeses(ctx context.Context, o Order) error {
	return m.q(ctx, RoleYes).Where("picker = ? AND poster_id = ? AND state = ?", o.Picker, o.PosterID, models.YesOpen).
		Update("state", models.YesCalledOff).Error
}

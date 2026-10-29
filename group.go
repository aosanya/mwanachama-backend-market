package market

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aosanya/mwanachama-backend-market/models"
)

type terms struct {
	Version int     `json:"version"`
	Price   float64 `json:"price"`
	Minimum float64 `json:"minimum"`
}

func (m *marketManager) StartGroupOrder(ctx context.Context, caller string, g GroupOrder) (GroupOrder, error) {
	if g.Minimum <= 0 {
		return g, fmt.Errorf("%w: a group order needs a minimum", ErrInvalid)
	}
	var p Poster
	if err := m.get(ctx, RolePoster, g.PosterID, &p); err != nil {
		return g, err
	}
	if p.Status != models.PosterApproved {
		return g, ErrPosterNotReady
	}
	wf, _, err := m.gate(ctx, WorkflowGroupOrder, 0, StepStart, caller, parties{})
	if err != nil {
		return g, err
	}
	g.ID, g.Coordinator, g.TermsVersion, g.TermsHistory = newID(), caller, 1, "[]"
	g.WorkflowVersion, g.State, g.CreatedAt = wf.Version, models.GroupOpen, m.now()
	if g.Direction == "" {
		g.Direction = models.DirectionBuy
	}
	if err := m.checks(RoleGroupOrder, g); err != nil {
		return g, err
	}
	return g, m.insert(ctx, RoleGroupOrder, g)
}

func (m *marketManager) openGroup(ctx context.Context, id string) (GroupOrder, error) {
	var g GroupOrder
	if err := m.get(ctx, RoleGroupOrder, id, &g); err != nil {
		return g, err
	}
	if g.State == models.GroupClosed || g.State == models.GroupCancelled {
		return g, ErrConflict
	}
	return g, nil
}

func (m *marketManager) myLine(ctx context.Context, groupID, caller string) (*Line, error) {
	ls, err := listOf[Line](m, m.q(ctx, RoleLine).Where("group_order_id = ? AND picker = ?", groupID, caller), RoleLine)
	if err != nil || len(ls) == 0 {
		return nil, err
	}
	return &ls[0], nil
}

func (m *marketManager) JoinGroupOrder(ctx context.Context, caller, id string, quantity float64) (Line, error) {
	g, err := m.openGroup(ctx, id)
	if err != nil {
		return Line{}, err
	}
	if quantity <= 0 || caller == "" {
		return Line{}, fmt.Errorf("%w: a line needs a picker and a positive quantity", ErrInvalid)
	}
	if l, err := m.myLine(ctx, id, caller); err != nil {
		return Line{}, err
	} else if l != nil {
		l.Quantity = quantity
		return *l, m.update(ctx, RoleLine, l.ID, *l)
	}
	l := Line{ID: newID(), GroupOrderID: id, Picker: caller, Quantity: quantity,
		Answer: models.AnswerYes, AnsweredVersion: g.TermsVersion, CreatedAt: m.now()}
	if err := m.checks(RoleLine, l); err != nil {
		return l, err
	}
	return l, m.insert(ctx, RoleLine, l)
}

func (m *marketManager) AskRevision(ctx context.Context, caller, id string, price, minimum float64) (GroupOrder, error) {
	g, err := m.openGroup(ctx, id)
	if err != nil {
		return g, err
	}
	if _, err := m.posterOwnedBy(ctx, caller, g.PosterID); err != nil {
		return g, err
	}
	if minimum <= 0 {
		return g, fmt.Errorf("%w: a group order needs a minimum", ErrInvalid)
	}
	var history []terms
	_ = json.Unmarshal([]byte(g.TermsHistory), &history)
	history = append(history, terms{g.TermsVersion, g.Price, g.Minimum})
	raw, _ := json.Marshal(history)
	g.TermsHistory, g.TermsVersion = string(raw), g.TermsVersion+1
	g.Price, g.Minimum, g.State = price, minimum, models.GroupRevising
	if err := m.update(ctx, RoleGroupOrder, id, g); err != nil {
		return g, err
	}
	return g, m.q(ctx, RoleLine).Where("group_order_id = ?", id).Update("answer", models.AnswerWaiting).Error
}

func (m *marketManager) AnswerRevision(ctx context.Context, caller, id, answer string) (Line, error) {
	g, err := m.openGroup(ctx, id)
	if err != nil {
		return Line{}, err
	}
	if answer != models.AnswerYes && answer != models.AnswerNo {
		return Line{}, fmt.Errorf("%w: answer yes or no", ErrInvalid)
	}
	l, err := m.myLine(ctx, id, caller)
	if err != nil {
		return Line{}, err
	}
	if l == nil {
		return Line{}, ErrNotFound
	}
	l.Answer, l.AnsweredVersion = answer, g.TermsVersion
	return *l, m.update(ctx, RoleLine, l.ID, *l)
}

func (m *marketManager) view(ctx context.Context, caller string, g GroupOrder) (GroupOrderView, error) {
	lines, err := listOf[Line](m, m.q(ctx, RoleLine).Where("group_order_id = ?", g.ID), RoleLine)
	if err != nil {
		return GroupOrderView{}, err
	}
	v := GroupOrderView{GroupOrder: g, Lines: []Line{}}
	for _, l := range lines {
		if l.Answer != models.AnswerNo {
			v.Total += l.Quantity
			v.Count++
		}
	}
	_, posterErr := m.posterOwnedBy(ctx, caller, g.PosterID)
	seesAll := (g.Coordinator == caller && caller != "") || posterErr == nil || m.isWho(ctx, WhoCoordinator, caller, parties{})
	for _, l := range lines {
		if seesAll || (l.Picker == caller && caller != "") {
			v.Lines = append(v.Lines, l)
		}
	}
	return v, nil
}

func (m *marketManager) GetGroupOrder(ctx context.Context, caller, id string) (GroupOrderView, error) {
	var g GroupOrder
	if err := m.get(ctx, RoleGroupOrder, id, &g); err != nil {
		return GroupOrderView{}, err
	}
	return m.view(ctx, caller, g)
}

func (m *marketManager) ListGroupOrders(ctx context.Context, caller string) ([]GroupOrderView, error) {
	gs, err := listOf[GroupOrder](m, m.q(ctx, RoleGroupOrder), RoleGroupOrder)
	if err != nil {
		return nil, err
	}
	out := []GroupOrderView{}
	for _, g := range gs {
		v, err := m.view(ctx, caller, g)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

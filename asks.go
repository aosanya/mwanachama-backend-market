package market

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-market/models"
)

func (m *marketManager) CreateRequest(ctx context.Context, caller string, r Request) (Request, error) {
	r.ID, r.Picker, r.CreatedAt = newID(), caller, m.now()
	if err := m.checks(RoleRequest, r); err != nil {
		return r, err
	}
	return r, m.insert(ctx, RoleRequest, r)
}

func (m *marketManager) CreateAsk(ctx context.Context, caller string, a Ask) (Ask, error) {
	var p Poster
	if err := m.get(ctx, RolePoster, a.PosterID, &p); err != nil {
		return a, err
	}
	if p.Availability == models.Away {
		return a, ErrPosterAway
	}
	a.ID, a.Picker, a.Answer, a.CreatedAt, a.AnsweredAt = newID(), caller, models.AnswerWaiting, m.now(), ""
	if err := m.checks(RoleAsk, a); err != nil {
		return a, err
	}
	n, err := m.count(ctx, RoleAsk, "poster_id = ? AND picker = ? AND request_id = ? AND offer_id = ?", a.PosterID, caller, a.RequestID, a.OfferID)
	if err != nil {
		return a, err
	}
	if n > 0 {
		return a, ErrAlreadyAsked
	}
	return a, m.insert(ctx, RoleAsk, a)
}

func (m *marketManager) AnswerAsk(ctx context.Context, caller, askID, answer, date, note string) (Ask, error) {
	var a Ask
	if err := m.get(ctx, RoleAsk, askID, &a); err != nil {
		return a, err
	}
	if _, err := m.posterOwnedBy(ctx, caller, a.PosterID); err != nil {
		return a, err
	}
	if _, _, err := m.gate(ctx, WorkflowBuyNow, 0, StepYes, caller, parties{picker: a.Picker, posterID: a.PosterID}); err != nil {
		return a, err
	}
	if answer != models.AnswerYes && answer != models.AnswerNo {
		return a, fmt.Errorf("%w: an ask is answered yes or no, silence stays waiting", ErrInvalid)
	}
	if a.Answer != models.AnswerWaiting {
		return a, ErrConflict
	}
	if answer == models.AnswerYes {
		y := Yes{Date: date, Note: note}
		if y.Date == "" {
			return a, fmt.Errorf("%w: a yes carries a date", ErrInvalid)
		}
		y.ID, y.AskID, y.PosterID, y.Picker = newID(), a.ID, a.PosterID, a.Picker
		y.State, y.CreatedAt = models.YesOpen, m.now()
		if err := m.checks(RoleYes, y); err != nil {
			return a, err
		}
		if err := m.insert(ctx, RoleYes, y); err != nil {
			return a, err
		}
	}
	a.Answer, a.AnsweredAt = answer, m.now()
	a.PickerName = m.nameOf(ctx, a.Picker)
	return a, m.update(ctx, RoleAsk, a.ID, a)
}

func (m *marketManager) ListAsks(ctx context.Context, caller string) ([]Ask, error) {
	all, err := listOf[Ask](m, m.q(ctx, RoleAsk), RoleAsk)
	if err != nil {
		return nil, err
	}
	owned, err := m.ownedPosters(ctx, caller)
	if err != nil {
		return nil, err
	}
	var out []Ask
	for _, a := range all {
		if a.Picker == caller || owned[a.PosterID] {
			a.PickerName = m.nameOf(ctx, a.Picker)
			out = append(out, a)
		}
	}
	return out, nil
}

func (m *marketManager) ListYeses(ctx context.Context, caller string) ([]Yes, error) {
	all, err := listOf[Yes](m, m.q(ctx, RoleYes), RoleYes)
	if err != nil {
		return nil, err
	}
	owned, err := m.ownedPosters(ctx, caller)
	if err != nil {
		return nil, err
	}
	var out []Yes
	for _, y := range all {
		if y.Picker == caller || owned[y.PosterID] {
			out = append(out, y)
		}
	}
	return out, nil
}

func (m *marketManager) ownedPosters(ctx context.Context, caller string) (map[string]bool, error) {
	ps, err := listOf[Poster](m, m.q(ctx, RolePoster).Where("owner = ?", caller), RolePoster)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, p := range ps {
		out[p.ID] = caller != ""
	}
	return out, nil
}

func (m *marketManager) CreateNotice(ctx context.Context, caller string, n Notice) (Notice, error) {
	n.ID, n.Picker, n.State, n.CreatedAt = newID(), caller, models.NoticeOpen, m.now()
	if err := m.checks(RoleNotice, n); err != nil {
		return n, err
	}
	return n, m.insert(ctx, RoleNotice, n)
}

func (m *marketManager) WithdrawNotice(ctx context.Context, caller, id string) (Notice, error) {
	var n Notice
	if err := m.get(ctx, RoleNotice, id, &n); err != nil {
		return n, err
	}
	if n.Picker != caller {
		return n, ErrNotFound
	}
	n.State = models.NoticeWithdrawn
	return n, m.update(ctx, RoleNotice, id, n)
}

func (m *marketManager) ListNotices(ctx context.Context) ([]Notice, error) {
	return listOf[Notice](m, m.q(ctx, RoleNotice).Where("state = ?", models.NoticeOpen), RoleNotice)
}

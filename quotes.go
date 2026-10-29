package market

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-market/models"
)

func (m *marketManager) openNotice(ctx context.Context, id string) (Notice, error) {
	var n Notice
	if err := m.get(ctx, RoleNotice, id, &n); err != nil {
		return n, err
	}
	if n.State != models.NoticeOpen {
		return n, ErrConflict
	}
	return n, nil
}

func (m *marketManager) quoteOn(ctx context.Context, id string) (Quote, Notice, error) {
	var q Quote
	var n Notice
	if err := m.get(ctx, RoleQuote, id, &q); err != nil {
		return q, n, err
	}
	return q, n, m.get(ctx, RoleNotice, q.NoticeID, &n)
}

func (m *marketManager) SendQuote(ctx context.Context, caller, noticeID string, q Quote) (Quote, error) {
	n, err := m.openNotice(ctx, noticeID)
	if err != nil {
		return q, err
	}
	p, err := m.posterOwnedBy(ctx, caller, q.PosterID)
	if err != nil {
		return q, err
	}
	if _, _, err := m.gate(ctx, WorkflowBidding, 0, StepQuoted, caller, parties{posterID: p.ID}); err != nil {
		return q, err
	}
	if p.Status != models.PosterApproved {
		return q, ErrPosterNotReady
	}
	if p.Availability == models.Away {
		return q, ErrPosterAway
	}
	standing, err := m.count(ctx, RoleQuote, "notice_id = ? AND poster_id = ?", noticeID, p.ID)
	if err != nil {
		return q, err
	}
	if standing > 0 {
		return q, ErrConflict
	}
	q.ID, q.NoticeID, q.Picker, q.CounterPrice = newID(), noticeID, n.Picker, 0
	q.State, q.CreatedAt = models.QuoteOpen, m.now()
	if q.Price <= 0 {
		return q, fmt.Errorf("%w: a quote carries a price", ErrInvalid)
	}
	if err := m.checks(RoleQuote, q); err != nil {
		return q, err
	}
	return q, m.insert(ctx, RoleQuote, q)
}

func (m *marketManager) ListQuotes(ctx context.Context, caller, noticeID string) ([]Quote, error) {
	n, err := m.noticeOf(ctx, noticeID)
	if err != nil {
		return nil, err
	}
	all, err := listOf[Quote](m, m.q(ctx, RoleQuote).Where("notice_id = ?", noticeID), RoleQuote)
	if err != nil {
		return nil, err
	}
	if caller != "" && (n.Picker == caller || m.isWho(ctx, WhoAgent, caller, parties{})) {
		return all, nil
	}
	owned, err := m.ownedPosters(ctx, caller)
	if err != nil {
		return nil, err
	}
	var out []Quote
	for _, q := range all {
		if owned[q.PosterID] {
			out = append(out, q)
		}
	}
	return out, nil
}

func (m *marketManager) noticeOf(ctx context.Context, id string) (Notice, error) {
	var n Notice
	return n, m.get(ctx, RoleNotice, id, &n)
}

func (m *marketManager) DraftCounter(ctx context.Context, caller, quoteID string, price float64, note string) (Draft, error) {
	q, n, err := m.quoteOn(ctx, quoteID)
	if err != nil {
		return Draft{}, err
	}
	if _, _, err := m.gate(ctx, WorkflowBidding, 0, StepDrafted, caller, parties{picker: n.Picker, posterID: q.PosterID}); err != nil {
		return Draft{}, err
	}
	if q.State != models.QuoteOpen || n.State != models.NoticeOpen {
		return Draft{}, ErrConflict
	}
	if price <= 0 {
		return Draft{}, fmt.Errorf("%w: a counter carries a price", ErrInvalid)
	}
	d := Draft{ID: newID(), QuoteID: q.ID, NoticeID: n.ID, Author: caller, Price: price, Note: note, State: models.DraftWaiting, CreatedAt: m.now()}
	if err := m.checks(RoleDraft, d); err != nil {
		return d, err
	}
	return d, m.insert(ctx, RoleDraft, d)
}

func (m *marketManager) ListDrafts(ctx context.Context, caller, noticeID string) ([]Draft, error) {
	n, err := m.noticeOf(ctx, noticeID)
	if err != nil {
		return nil, err
	}
	all, err := listOf[Draft](m, m.q(ctx, RoleDraft).Where("notice_id = ?", noticeID), RoleDraft)
	if err != nil {
		return nil, err
	}
	var out []Draft
	for _, d := range all {
		if caller != "" && (n.Picker == caller || d.Author == caller) {
			out = append(out, d)
		}
	}
	return out, nil
}

func (m *marketManager) SendCounter(ctx context.Context, caller, draftID string) (Quote, error) {
	var d Draft
	if err := m.get(ctx, RoleDraft, draftID, &d); err != nil {
		return Quote{}, err
	}
	q, n, err := m.quoteOn(ctx, d.QuoteID)
	if err != nil {
		return q, err
	}
	if _, _, err := m.gate(ctx, WorkflowBidding, 0, StepCountered, caller, parties{picker: n.Picker, posterID: q.PosterID}); err != nil {
		return q, err
	}
	if d.State != models.DraftWaiting || q.State != models.QuoteOpen || n.State != models.NoticeOpen {
		return q, ErrConflict
	}
	d.State = models.DraftSent
	if err := m.update(ctx, RoleDraft, d.ID, d); err != nil {
		return q, err
	}
	q.CounterPrice, q.State = d.Price, models.QuoteCountered
	return q, m.update(ctx, RoleQuote, q.ID, q)
}

func (m *marketManager) ReviseQuote(ctx context.Context, caller, quoteID string, price float64, note string) (Quote, error) {
	q, n, err := m.quoteOn(ctx, quoteID)
	if err != nil {
		return q, err
	}
	if _, _, err := m.gate(ctx, WorkflowBidding, 0, StepQuoted, caller, parties{picker: n.Picker, posterID: q.PosterID}); err != nil {
		return q, err
	}
	if q.State != models.QuoteCountered || n.State != models.NoticeOpen {
		return q, ErrConflict
	}
	if price <= 0 {
		return q, fmt.Errorf("%w: a quote carries a price", ErrInvalid)
	}
	q.Price, q.Note, q.CounterPrice, q.State = price, note, 0, models.QuoteOpen
	return q, m.update(ctx, RoleQuote, q.ID, q)
}

func (m *marketManager) AcceptQuote(ctx context.Context, caller, quoteID string) (Quote, error) {
	q, n, err := m.quoteOn(ctx, quoteID)
	if err != nil {
		return q, err
	}
	if _, _, err := m.gate(ctx, WorkflowBidding, 0, StepAccepted, caller, parties{picker: n.Picker, posterID: q.PosterID}); err != nil {
		return q, err
	}
	if q.State != models.QuoteOpen || n.State != models.NoticeOpen {
		return q, ErrConflict
	}
	if err := m.q(ctx, RoleQuote).Where("notice_id = ? AND id <> ?", n.ID, q.ID).
		Update("state", models.QuoteDeclined).Error; err != nil {
		return q, err
	}
	n.State = models.NoticeSold
	if err := m.update(ctx, RoleNotice, n.ID, n); err != nil {
		return q, err
	}
	q.State = models.QuoteAccepted
	return q, m.update(ctx, RoleQuote, q.ID, q)
}

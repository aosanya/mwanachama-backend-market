package market

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-market/models"
	"github.com/aosanya/mwanachama-backend-shared/gormutil"
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
	q.ID, q.NoticeID, q.Picker, q.CounterPrice = newID(), noticeID, n.Picker, 0
	q.State, q.CreatedAt = models.QuoteOpen, m.now()
	if q.Price <= 0 {
		return q, fmt.Errorf("%w: a quote carries a price", ErrInvalid)
	}
	if err := m.checks(RoleQuote, q); err != nil {
		return q, err
	}
	err = m.inTransaction(ctx, func(tx *marketManager) error {
		if err := tx.holdOpenNotice(ctx, noticeID); err != nil {
			return err
		}
		return tx.insert(ctx, RoleQuote, q)
	})
	if gormutil.IsUniqueViolation(err) {
		return q, ErrConflict
	}
	return q, err
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

func (m *marketManager) GetQuote(ctx context.Context, caller, quoteID string) (Quote, error) {
	q, n, err := m.quoteOn(ctx, quoteID)
	if err != nil {
		return Quote{}, err
	}
	if caller != "" && (n.Picker == caller || m.isWho(ctx, WhoAgent, caller, parties{})) {
		return q, nil
	}
	owned, err := m.ownedPosters(ctx, caller)
	if err != nil {
		return Quote{}, err
	}
	if !owned[q.PosterID] {
		return Quote{}, ErrNotFound
	}
	return q, nil
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
	err = m.inTransaction(ctx, func(tx *marketManager) error {
		if err := tx.holdOpenNotice(ctx, n.ID); err != nil {
			return err
		}
		if err := tx.holdWhileIn(ctx, RoleQuote, q.ID, models.QuoteOpen); err != nil {
			return err
		}
		return tx.insert(ctx, RoleDraft, d)
	})
	return d, err
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

func (m *marketManager) GetDraft(ctx context.Context, caller, draftID string) (Draft, error) {
	var d Draft
	if err := m.get(ctx, RoleDraft, draftID, &d); err != nil {
		return Draft{}, err
	}
	n, err := m.noticeOf(ctx, d.NoticeID)
	if err != nil {
		return Draft{}, err
	}
	if caller == "" || (n.Picker != caller && d.Author != caller) {
		return Draft{}, ErrNotFound
	}
	return d, nil
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
	sent, countered := d, q
	sent.State = models.DraftSent
	countered.CounterPrice, countered.State = d.Price, models.QuoteCountered
	err = m.inTransaction(ctx, func(tx *marketManager) error {
		if err := tx.holdOpenNotice(ctx, n.ID); err != nil {
			return err
		}
		if err := tx.updateWhileIn(ctx, RoleDraft, d.ID, models.DraftWaiting, sent); err != nil {
			return err
		}
		return tx.updateWhileIn(ctx, RoleQuote, q.ID, models.QuoteOpen, countered)
	})
	if err != nil {
		return q, err
	}
	return countered, nil
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
	revised := q
	revised.Price, revised.Note, revised.CounterPrice, revised.State = price, note, 0, models.QuoteOpen
	err = m.inTransaction(ctx, func(tx *marketManager) error {
		if err := tx.holdOpenNotice(ctx, n.ID); err != nil {
			return err
		}
		return tx.updateWhileIn(ctx, RoleQuote, q.ID, models.QuoteCountered, revised)
	})
	if err != nil {
		return q, err
	}
	return revised, nil
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
	accepted := q
	accepted.State = models.QuoteAccepted
	err = m.inTransaction(ctx, func(tx *marketManager) error {
		if err := tx.claimNotice(ctx, n.ID); err != nil {
			return err
		}
		if err := tx.q(ctx, RoleQuote).Where("notice_id = ? AND id <> ?", n.ID, q.ID).
			Update("state", models.QuoteDeclined).Error; err != nil {
			return err
		}
		return tx.updateWhileIn(ctx, RoleQuote, q.ID, models.QuoteOpen, accepted)
	})
	if err != nil {
		return q, err
	}
	return accepted, nil
}

func (m *marketManager) claimNotice(ctx context.Context, id string) error {
	return m.moveOpenNotice(ctx, id, models.NoticeSold)
}

func (m *marketManager) holdOpenNotice(ctx context.Context, id string) error {
	return m.moveOpenNotice(ctx, id, models.NoticeOpen)
}

func (m *marketManager) moveOpenNotice(ctx context.Context, id, state string) error {
	res := m.q(ctx, RoleNotice).Where("id = ? AND state = ?", id, models.NoticeOpen).Update("state", state)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrConflict
	}
	return nil
}

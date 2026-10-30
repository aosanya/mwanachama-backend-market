package market

import (
	"context"
	"fmt"
	"strings"

	"github.com/aosanya/mwanachama-backend-market/models"
)

func (m *marketManager) ApplyPoster(ctx context.Context, caller string, p Poster) (Poster, error) {
	p.ID, p.Owner, p.ApprovedBy = newID(), caller, ""
	p.Status, p.Availability, p.CreatedAt = models.PosterPending, models.Available, m.now()
	if caller == "" {
		return p, fmt.Errorf("%w: a poster applies as somebody", ErrInvalid)
	}
	if err := m.checks(RolePoster, p); err != nil {
		return p, err
	}
	if err := m.insert(ctx, RolePoster, p); err != nil {
		return p, err
	}
	return m.named(ctx, p), nil
}

func (m *marketManager) named(ctx context.Context, p Poster) Poster {
	p.OwnerName = m.nameOf(ctx, p.Owner)
	p.ApprovedByName = m.nameOf(ctx, p.ApprovedBy)
	return p
}

func (m *marketManager) ListPosters(ctx context.Context, status string) ([]Poster, error) {
	q := m.q(ctx, RolePoster)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	ps, err := listOf[Poster](m, q, RolePoster)
	for i := range ps {
		ps[i] = m.named(ctx, ps[i])
	}
	return ps, err
}

func (m *marketManager) setStatus(ctx context.Context, caller, id, status string) (Poster, error) {
	var p Poster
	if err := m.get(ctx, RolePoster, id, &p); err != nil {
		return p, err
	}
	if _, _, err := m.gate(ctx, WorkflowApprovePoster, 0, StepApproved, caller, parties{}); err != nil {
		return p, err
	}
	p.Status = status
	if status == models.PosterApproved {
		p.ApprovedBy = caller
	}
	if err := m.update(ctx, RolePoster, id, p); err != nil {
		return p, err
	}
	return m.named(ctx, p), nil
}

func (m *marketManager) ApprovePoster(ctx context.Context, caller, id string) (Poster, error) {
	return m.setStatus(ctx, caller, id, models.PosterApproved)
}

func (m *marketManager) DeclinePoster(ctx context.Context, caller, id string) (Poster, error) {
	return m.setStatus(ctx, caller, id, models.PosterDeclined)
}

func (m *marketManager) RemovePoster(ctx context.Context, caller, id string) (Poster, error) {
	return m.setStatus(ctx, caller, id, models.PosterRemoved)
}

func (m *marketManager) SetAvailability(ctx context.Context, caller, id, availability string) (Poster, error) {
	p, err := m.posterOwnedBy(ctx, caller, id)
	if err != nil {
		return p, err
	}
	p.Availability = availability
	if err := m.checks(RolePoster, p); err != nil {
		return p, err
	}
	if err := m.update(ctx, RolePoster, id, p); err != nil {
		return p, err
	}
	return m.named(ctx, p), nil
}

func (m *marketManager) CreateOffer(ctx context.Context, caller string, o Offer) (Offer, error) {
	p, err := m.posterOwnedBy(ctx, caller, o.PosterID)
	if err != nil {
		return o, err
	}
	if p.Status != models.PosterApproved {
		return o, ErrPosterNotReady
	}
	o.ID, o.PostedAt = newID(), m.now()
	if err := m.checks(RoleOffer, o); err != nil {
		return o, err
	}
	return o, m.insert(ctx, RoleOffer, o)
}

func (m *marketManager) UpdateOffer(ctx context.Context, caller string, o Offer) (Offer, error) {
	var old Offer
	if err := m.get(ctx, RoleOffer, o.ID, &old); err != nil {
		return o, err
	}
	if _, err := m.posterOwnedBy(ctx, caller, old.PosterID); err != nil {
		return o, err
	}
	o.PosterID, o.PostedAt = old.PosterID, m.now()
	if err := m.checks(RoleOffer, o); err != nil {
		return o, err
	}
	return o, m.update(ctx, RoleOffer, o.ID, o)
}

func (m *marketManager) RegroupOffer(ctx context.Context, caller, offerID, product string) (Offer, error) {
	if caller == "" {
		return Offer{}, ErrForbidden
	}
	product = strings.TrimSpace(product)
	if product == "" {
		return Offer{}, fmt.Errorf("%w: name the product to group the offer under", ErrInvalid)
	}
	var o Offer
	if err := m.get(ctx, RoleOffer, offerID, &o); err != nil {
		return o, err
	}
	if o.Product == product {
		return o, fmt.Errorf("%w: the offer is already under %s", ErrConflict, product)
	}
	o.Product = product
	return o, m.update(ctx, RoleOffer, o.ID, o)
}

func (m *marketManager) ListOffers(ctx context.Context, posterID string) ([]Offer, error) {
	q := m.q(ctx, RoleOffer)
	if posterID != "" {
		q = q.Where("poster_id = ?", posterID)
	}
	return listOf[Offer](m, q, RoleOffer)
}

func (m *marketManager) SetStock(ctx context.Context, caller string, s Stock) (Stock, error) {
	if _, err := m.posterOwnedBy(ctx, caller, s.PosterID); err != nil {
		return s, err
	}
	if err := m.checks(RoleStock, s); err != nil {
		return s, err
	}
	var rows []Stock
	rows, err := listOf[Stock](m, m.q(ctx, RoleStock).Where("poster_id = ? AND product = ?", s.PosterID, s.Product), RoleStock)
	if err != nil {
		return s, err
	}
	if len(rows) > 0 {
		s.ID = rows[0].ID
		return s, m.update(ctx, RoleStock, s.ID, s)
	}
	s.ID = newID()
	return s, m.insert(ctx, RoleStock, s)
}

func (m *marketManager) GetStock(ctx context.Context, caller, posterID string) ([]Stock, error) {
	if _, err := m.posterOwnedBy(ctx, caller, posterID); err != nil {
		return nil, ErrNotFound
	}
	return listOf[Stock](m, m.q(ctx, RoleStock).Where("poster_id = ?", posterID), RoleStock)
}

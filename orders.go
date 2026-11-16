package market

import (
	"context"
	"fmt"
	"gorm.io/gorm/clause"

	"github.com/aosanya/mwanachama-backend-market/models"
)

const (
	defaultOrderPage = 50
	largestOrderPage = 200
)

func unitPrice(o Offer, quantity float64) float64 {
	if o.BulkPrice > 0 && o.BulkMin > 0 && quantity >= o.BulkMin {
		return o.BulkPrice
	}
	return o.Price
}

func (m *marketManager) BuyNow(ctx context.Context, caller, offerID string, quantity float64, yesID string) (Order, error) {
	var o Offer
	if err := m.get(ctx, RoleOffer, offerID, &o); err != nil {
		return Order{}, err
	}
	if quantity <= 0 || caller == "" {
		return Order{}, fmt.Errorf("%w: an order needs a picker and a positive quantity", ErrInvalid)
	}
	var p Poster
	if err := m.get(ctx, RolePoster, o.PosterID, &p); err != nil {
		return Order{}, err
	}
	if p.Status != models.PosterApproved {
		return Order{}, ErrPosterNotReady
	}
	wf, err := m.workflow(ctx, WorkflowBuyNow, 0)
	if err != nil {
		return Order{}, err
	}
	var promised string
	if yesID != "" {
		var y Yes
		if err := m.get(ctx, RoleYes, yesID, &y); err != nil || y.Picker != caller {
			return Order{}, ErrNotFound
		}
		if y.State != models.YesOpen || y.PosterID != o.PosterID {
			return Order{}, ErrConflict
		}
		promised = y.Date
	}
	ord := Order{ID: newID(), Picker: caller, PosterID: o.PosterID, OfferID: o.ID, Product: o.Product,
		Quantity: quantity, UnitPrice: unitPrice(o, quantity), WorkflowVersion: wf.Version,
		State: models.OrderPlaced, YesID: yesID, PromisedDate: promised, CreatedAt: m.now()}
	if ord.Product == "" {
		ord.Product = o.ProductWords
	}
	if err := m.checks(RoleOrder, ord); err != nil {
		return ord, err
	}
	if err := m.insert(ctx, RoleOrder, ord); err != nil {
		return ord, err
	}
	ord.PickerName = m.nameOf(ctx, ord.Picker)
	return ord, m.note(ctx, ord.ID, caller, "placed")
}

func (m *marketManager) visibleOrder(ctx context.Context, caller, id string) (Order, bool, error) {
	var o Order
	if err := m.get(ctx, RoleOrder, id, &o); err != nil {
		return o, false, err
	}
	if o.Picker == caller && caller != "" {
		return o, false, nil
	}
	if _, err := m.posterOwnedBy(ctx, caller, o.PosterID); err == nil {
		return o, true, nil
	}
	return o, false, ErrNotFound
}

func (m *marketManager) setWhilePlaced(ctx context.Context, id string, columns map[string]any) error {
	res := m.q(ctx, RoleOrder).Where("id = ? AND state = ?", id, models.OrderPlaced).Updates(columns)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrConflict
	}
	return nil
}

func (m *marketManager) CancelOrder(ctx context.Context, caller, id string) (Order, error) {
	o, _, err := m.visibleOrder(ctx, caller, id)
	if err != nil {
		return o, err
	}
	_, step, err := m.gate(ctx, WorkflowCancel, o.WorkflowVersion, StepCancelled, caller, parties{picker: o.Picker, posterID: o.PosterID})
	if err != nil {
		return o, err
	}
	if o.State != models.OrderPlaced {
		return o, ErrConflict
	}
	cancelledAt := m.now()
	if err := m.setWhilePlaced(ctx, id, map[string]any{"state": models.OrderCancelled, "cancelled_at": cancelledAt}); err != nil {
		return o, err
	}
	o.State = models.OrderCancelled
	o.CancelledAt = cancelledAt
	if step.ClosesYes {
		if err := m.closeYeses(ctx, o); err != nil {
			return o, err
		}
	}
	return o, m.note(ctx, id, caller, "cancelled")
}

func (m *marketManager) MarkPaid(ctx context.Context, caller, id string) (Order, error) {
	o, byPoster, err := m.visibleOrder(ctx, caller, id)
	if err != nil {
		return o, err
	}
	if !byPoster {
		return o, ErrForbidden
	}
	if _, _, err := m.gate(ctx, WorkflowBuyNow, o.WorkflowVersion, StepPaid, caller, parties{picker: o.Picker, posterID: o.PosterID}); err != nil {
		return o, err
	}
	if o.State != models.OrderPlaced {
		return o, ErrConflict
	}
	if err := m.setWhilePlaced(ctx, id, map[string]any{"paid": true}); err != nil {
		return o, err
	}
	o.Paid = true
	return o, m.note(ctx, id, caller, "marked paid")
}

func (m *marketManager) GetOrder(ctx context.Context, caller, id string) (Order, error) {
	o, _, err := m.visibleOrder(ctx, caller, id)
	o.PickerName = m.nameOf(ctx, o.Picker)
	return o, err
}

func knownOrderState(state string) bool {
	switch state {
	case models.OrderPlaced, models.OrderDelivered, models.OrderCancelled:
		return true
	}
	return false
}

func (m *marketManager) ListOrders(ctx context.Context, caller string, limit int, after, state string) ([]Order, error) {
	if limit < 0 {
		return nil, fmt.Errorf("%w: a page is not negative", ErrInvalid)
	}
	if state != "" && !knownOrderState(state) {
		return nil, fmt.Errorf("%w: %q is not an order state", ErrInvalid, state)
	}
	if limit == 0 {
		limit = defaultOrderPage
	}
	if limit > largestOrderPage {
		limit = largestOrderPage
	}
	if caller == "" {
		return []Order{}, nil
	}
	q := m.q(ctx, RoleOrder).Order("id").Limit(limit).
		Where("(picker = ? OR poster_id IN (?))", caller, m.q(ctx, RolePoster).Select("id").Where("owner = ?", caller))
	if after != "" {
		q = q.Where("id > ?", after)
	}
	if state != "" {
		q = q.Where("state = ?", state)
	}
	out, err := listOf[Order](m, q, RoleOrder)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].PickerName = m.nameOf(ctx, out[i].Picker)
	}
	return out, nil
}

func (m *marketManager) OrderTrail(ctx context.Context, caller, orderID string) ([]Trail, error) {
	var o Order
	if err := m.get(ctx, RoleOrder, orderID, &o); err != nil {
		return nil, err
	}
	ownTrail := o.Picker == caller && caller != ""
	if !ownTrail && !m.isWho(ctx, WhoCoordinator, caller, parties{}) {
		return nil, ErrNotFound
	}
	rows, err := listOf[Trail](m, m.q(ctx, RoleTrail).Where("order_id = ?", orderID).Order(clause.OrderByColumn{Column: clause.Column{Name: "at"}}), RoleTrail)
	for i := range rows {
		rows[i].ActorName = m.nameOf(ctx, rows[i].Actor)
	}
	return rows, err
}

package market

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-market/models"
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
	o.State = models.OrderCancelled
	if err := m.update(ctx, RoleOrder, id, o); err != nil {
		return o, err
	}
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
	o.Paid = true
	if err := m.update(ctx, RoleOrder, id, o); err != nil {
		return o, err
	}
	return o, m.note(ctx, id, caller, "marked paid")
}

func (m *marketManager) GetOrder(ctx context.Context, caller, id string) (Order, error) {
	o, _, err := m.visibleOrder(ctx, caller, id)
	o.PickerName = m.nameOf(ctx, o.Picker)
	return o, err
}

func (m *marketManager) ListOrders(ctx context.Context, caller string) ([]Order, error) {
	owned, err := m.ownedPosters(ctx, caller)
	if err != nil {
		return nil, err
	}
	all, err := listOf[Order](m, m.q(ctx, RoleOrder), RoleOrder)
	if err != nil {
		return nil, err
	}
	var out []Order
	for _, o := range all {
		if (o.Picker == caller && caller != "") || owned[o.PosterID] {
			o.PickerName = m.nameOf(ctx, o.Picker)
			out = append(out, o)
		}
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
	return listOf[Trail](m, m.q(ctx, RoleTrail).Where("order_id = ?", orderID), RoleTrail)
}

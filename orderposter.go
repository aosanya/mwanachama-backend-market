package market

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-market/models"
)

type OrderPoster = models.OrderPoster

const (
	OrderPosterFromGroupOrder = "group_order"
	OrderPosterOnlyApproved   = "only_approved_poster"
)

func (m *marketManager) GroupOrderPoster(ctx context.Context, groupOrderID string) (OrderPoster, error) {
	if groupOrderID != "" {
		var g GroupOrder
		if err := m.get(ctx, RoleGroupOrder, groupOrderID, &g); err != nil {
			return OrderPoster{}, err
		}
		var p Poster
		if err := m.get(ctx, RolePoster, g.PosterID, &p); err != nil {
			return OrderPoster{}, err
		}
		return OrderPoster{PosterID: p.ID, PosterName: p.Name, Source: OrderPosterFromGroupOrder}, nil
	}
	approved, err := m.ListPosters(ctx, models.PosterApproved)
	if err != nil {
		return OrderPoster{}, err
	}
	switch len(approved) {
	case 0:
		return OrderPoster{}, ErrPosterNotReady
	case 1:
		return OrderPoster{PosterID: approved[0].ID, PosterName: approved[0].Name, Source: OrderPosterOnlyApproved}, nil
	}
	return OrderPoster{}, fmt.Errorf("%w: %d posters are approved, so name the poster_id", ErrInvalid, len(approved))
}

func (m *marketManager) posterWhenUnnamed(ctx context.Context, posterID string) (string, error) {
	if posterID != "" {
		return posterID, nil
	}
	s, err := m.GroupOrderPoster(ctx, "")
	return s.PosterID, err
}

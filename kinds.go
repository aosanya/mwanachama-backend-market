package market

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

const (
	defaultKinds = 3
	largestKinds = 10
)

func (m *marketManager) categoryOf(ctx context.Context, product string) string {
	if product == "" {
		return ""
	}
	rows, err := listOf[Offer](m, m.q(ctx, RoleOffer).Where("product = ? AND category <> ''", product).Order("id").Limit(1), RoleOffer)
	if err != nil || len(rows) == 0 {
		return ""
	}
	return rows[0].Category
}

func (m *marketManager) CategoriseProduct(ctx context.Context, caller, product, category string) ([]Offer, error) {
	if caller == "" {
		return nil, ErrForbidden
	}
	product, category = strings.TrimSpace(product), strings.TrimSpace(category)
	if product == "" || category == "" {
		return nil, fmt.Errorf("%w: name the product and the category to put it under", ErrInvalid)
	}
	offers, err := listOf[Offer](m, m.q(ctx, RoleOffer).Where("product = ?", product).Order("id"), RoleOffer)
	if err != nil {
		return nil, err
	}
	if len(offers) == 0 {
		return nil, ErrNotFound
	}
	if offers[0].Category == category && allUnder(offers, category) {
		return offers, fmt.Errorf("%w: %s is already under %s", ErrConflict, product, category)
	}
	for i := range offers {
		offers[i].Category = category
		if err := m.update(ctx, RoleOffer, offers[i].ID, offers[i]); err != nil {
			return nil, err
		}
	}
	return offers, nil
}

func allUnder(offers []Offer, category string) bool {
	for _, o := range offers {
		if o.Category != category {
			return false
		}
	}
	return true
}

type tally struct {
	asked    int
	offers   int
	products map[string]int
}

func (m *marketManager) ListKinds(ctx context.Context, limit int) ([]Kind, error) {
	if limit < 0 {
		return nil, fmt.Errorf("%w: a count is not negative", ErrInvalid)
	}
	if limit == 0 {
		limit = defaultKinds
	}
	if limit > largestKinds {
		limit = largestKinds
	}
	offers, err := listOf[Offer](m, m.q(ctx, RoleOffer).Where("category <> ''").Order("id"), RoleOffer)
	if err != nil {
		return nil, err
	}
	byProduct := map[string]string{}
	byOffer := map[string]string{}
	tallies := map[string]*tally{}
	for _, o := range offers {
		if _, ok := tallies[o.Category]; !ok {
			tallies[o.Category] = &tally{products: map[string]int{}}
		}
		tallies[o.Category].offers++
		byOffer[o.ID] = o.Product
		if _, ok := byProduct[o.Product]; !ok && o.Product != "" {
			byProduct[o.Product] = o.Category
		}
	}
	asks, err := listOf[Ask](m, m.q(ctx, RoleAsk), RoleAsk)
	if err != nil {
		return nil, err
	}
	for _, a := range asks {
		product := a.Product
		if product == "" {
			product = byOffer[a.OfferID]
		}
		countDemand(tallies, byProduct, product)
	}
	orders, err := listOf[Order](m, m.q(ctx, RoleOrder), RoleOrder)
	if err != nil {
		return nil, err
	}
	for _, o := range orders {
		countDemand(tallies, byProduct, o.Product)
	}
	return rankKinds(tallies, limit), nil
}

func countDemand(tallies map[string]*tally, byProduct map[string]string, product string) {
	category, ok := byProduct[product]
	if !ok {
		return
	}
	t := tallies[category]
	t.asked++
	t.products[product]++
}

func rankKinds(tallies map[string]*tally, limit int) []Kind {
	kinds := make([]Kind, 0, len(tallies))
	for category, t := range tallies {
		kinds = append(kinds, Kind{Category: category, Asked: t.asked, Offers: t.offers, Product: topProduct(t.products)})
	}
	sort.Slice(kinds, func(i, j int) bool {
		a, b := kinds[i], kinds[j]
		if a.Asked != b.Asked {
			return a.Asked > b.Asked
		}
		if a.Offers != b.Offers {
			return a.Offers > b.Offers
		}
		return a.Category < b.Category
	})
	if len(kinds) > limit {
		kinds = kinds[:limit]
	}
	return kinds
}

func topProduct(counts map[string]int) string {
	best, most := "", 0
	for product, n := range counts {
		if n > most || (n == most && product < best) {
			best, most = product, n
		}
	}
	return best
}

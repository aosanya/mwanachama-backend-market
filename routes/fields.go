package routes

import (
	"fmt"
	"maps"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/spec"

	market "github.com/aosanya/mwanachama-backend-market"
)

var carriers = map[string]string{
	market.RolePoster:     "Poster",
	market.RoleOffer:      "Offer",
	market.RoleRequest:    "Request",
	market.RoleAsk:        "Ask",
	market.RoleNotice:     "Notice",
	market.RoleQuote:      "Quote",
	market.RoleDraft:      "Draft",
	market.RoleYes:        "Yes",
	market.RoleOrder:      "Order",
	market.RoleGroupOrder: "GroupOrder",
	market.RoleLine:       "Line",
	market.RoleStock:      "Stock",
	market.RoleTrail:      "Trail",
}

var serverOwned = map[string]bool{
	"Poster.owner":                true,
	"Poster.approved_by":          true,
	"Poster.owner_name":           true,
	"Poster.approved_by_name":     true,
	"Poster.status":               true,
	"Poster.availability":         true,
	"Request.picker":              true,
	"Request.picker_name":         true,
	"Ask.picker":                  true,
	"Ask.picker_name":             true,
	"Ask.answer":                  true,
	"Notice.picker":               true,
	"Notice.picker_name":          true,
	"Notice.state":                true,
	"Quote.notice_id":             true,
	"Quote.picker":                true,
	"Quote.counter_price":         true,
	"Quote.state":                 true,
	"GroupOrder.coordinator":      true,
	"GroupOrder.terms_version":    true,
	"GroupOrder.terms_history":    true,
	"GroupOrder.workflow_version": true,
	"GroupOrder.state":            true,
}

var namedOnlyWhenSeveral = map[string]bool{
	"GroupOrder.poster_id": true,
}

func Fields() (map[string]dispatch.FieldDoc, error) {
	b, err := market.Blueprint()
	if err != nil {
		return nil, err
	}
	out := map[string]dispatch.FieldDoc{}
	for _, role := range b.Roles() {
		carrier, named := carriers[role]
		if !named {
			return nil, fmt.Errorf("market tools: role %q has no type to describe", role)
		}
		o, ok := b.Object(role)
		if !ok {
			return nil, fmt.Errorf("market tools: the blueprint declares no %q", role)
		}
		for _, f := range o.Fields {
			key := carrier + "." + f.Name
			out[key] = dispatch.FieldDoc{
				Description: f.Description,
				Required:    f.Required && !namedOnlyWhenSeveral[key],
				ReadOnly:    f.Primary || f.Type == spec.TypeTimestamp,
				Values:      f.Values,
			}
		}
	}
	for key := range serverOwned {
		doc := out[key]
		doc.ReadOnly = true
		out[key] = doc
	}
	return out, nil
}

func updateOfferFields(fields map[string]dispatch.FieldDoc) map[string]dispatch.FieldDoc {
	out := maps.Clone(fields)
	doc := out["Offer.poster_id"]
	doc.Required = false
	out["Offer.poster_id"] = doc
	return out
}

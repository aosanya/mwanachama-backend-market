package routes

import (
	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/httpwire"

	market "github.com/aosanya/mwanachama-backend-market"
)

type Route = httpwire.Route

type Mount = dispatch.Mount

type Tool = dispatch.Tool

var Sentinels = map[string]error{
	"ErrInvalid":        market.ErrInvalid,
	"ErrConflict":       market.ErrConflict,
	"ErrNotFound":       market.ErrNotFound,
	"ErrForbidden":      market.ErrForbidden,
	"ErrAlreadyAsked":   market.ErrAlreadyAsked,
	"ErrPosterNotReady": market.ErrPosterNotReady,
	"ErrPosterAway":     market.ErrPosterAway,
}

var AnonymousActions = []string{}

var Table = dispatch.NewTable(market.Operations(), Sentinels, AnonymousActions...)

func Build(mm market.MarketManager) ([]Route, error) { return Table.Build(mm, Mount{}) }

func BuildFor(mm market.MarketManager, m Mount) ([]Route, error) { return Table.Build(mm, m) }

func Routes(mm market.MarketManager) []Route { return Table.Routes(mm, Mount{}) }

func RoutesFor(mm market.MarketManager, m Mount) []Route { return Table.Routes(mm, m) }

func Split(mm market.MarketManager, m Mount) dispatch.Split { return Table.Split(mm, m) }

const updateOfferOperation = "update_offer"

func BuildTools(mm market.MarketManager, m Mount) ([]Tool, error) {
	s, err := dispatch.Parse(market.Operations())
	if err != nil {
		return nil, err
	}
	fields, err := Fields()
	if err != nil {
		return nil, err
	}
	tools, err := dispatch.Tools(s, toolDeps(mm, m, fields))
	if err != nil {
		return nil, err
	}
	changes, err := dispatch.Tools(onlyOperation(s, updateOfferOperation), toolDeps(mm, m, updateOfferFields(fields)))
	if err != nil {
		return nil, err
	}
	for i := range tools {
		if tools[i].Name == changes[0].Name {
			tools[i] = changes[0]
		}
	}
	return tools, nil
}

func toolDeps(mm market.MarketManager, m Mount, fields map[string]dispatch.FieldDoc) dispatch.Deps {
	return dispatch.Deps{
		Manager: mm, Errors: Sentinels, Fields: fields, Authorize: m.Authorize, Caller: m.Caller, Device: m.Device,
	}
}

func onlyOperation(s *dispatch.Spec, name string) *dispatch.Spec {
	return &dispatch.Spec{Base: s.Base, Operations: map[string]dispatch.Operation{name: s.Operations[name]}, Errors: s.Errors}
}

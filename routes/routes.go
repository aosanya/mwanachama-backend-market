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
}

var AnonymousActions = []string{}

var Table = dispatch.NewTable(market.Operations(), Sentinels, AnonymousActions...)

func Build(mm market.MarketManager) ([]Route, error) { return Table.Build(mm, Mount{}) }

func BuildFor(mm market.MarketManager, m Mount) ([]Route, error) { return Table.Build(mm, m) }

func Routes(mm market.MarketManager) []Route { return Table.Routes(mm, Mount{}) }

func RoutesFor(mm market.MarketManager, m Mount) []Route { return Table.Routes(mm, m) }

func Split(mm market.MarketManager, m Mount) dispatch.Split { return Table.Split(mm, m) }

func BuildTools(mm market.MarketManager, m Mount) ([]Tool, error) {
	s, err := dispatch.Parse(market.Operations())
	if err != nil {
		return nil, err
	}
	return dispatch.Tools(s, dispatch.Deps{
		Manager: mm, Errors: Sentinels, Authorize: m.Authorize, Caller: m.Caller, Device: m.Device,
	})
}

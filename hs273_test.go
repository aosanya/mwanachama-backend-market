package market_test

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

const inputsBuyNowVersion = 5

func mountedMarket(t *testing.T, db *gorm.DB, mount string) market.MarketManager {
	t.Helper()
	s, err := market.SpecForMount("app", mount)
	if err != nil {
		t.Fatal(err)
	}
	if err := market.Provision(db, s); err != nil {
		t.Fatal(err)
	}
	m, err := market.NewMarketManager(db, s)
	if err != nil {
		t.Fatal(err)
	}
	m.UseCoordinators(func(_ context.Context, c string) bool { return c == "coordinator" })
	return m
}

func TestTwoMarketsInOneAgencyAreSeparate(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()
	inputs := mountedMarket(t, db, "inputs")
	harvest := mountedMarket(t, db, "harvest")
	inputs.UseWorkflows(func(_ context.Context, name string, version int) (market.Workflow, bool, error) {
		if name != market.WorkflowBuyNow || (version != 0 && version != inputsBuyNowVersion) {
			return market.Workflow{}, false, nil
		}
		return market.Workflow{Name: name, Version: inputsBuyNowVersion, Steps: []market.Step{
			{Name: market.StepAsked, Who: market.WhoPicker}, {Name: market.StepYes, Who: market.WhoPoster}, {Name: market.StepPaid, Who: market.WhoPoster}}}, true, nil
	})

	poster := approvedPoster(t, inputs, ctx, "ann")
	offer, err := inputs.CreateOffer(ctx, "ann", market.Offer{PosterID: poster.ID, ProductWords: "maharagwe", UnitWords: "kg", Price: 120, Product: "beans"})
	if err != nil {
		t.Fatal(err)
	}
	order, err := inputs.BuyNow(ctx, "bea", offer.ID, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	group, err := inputs.StartGroupOrder(ctx, "coordinator", market.GroupOrder{PosterID: poster.ID, Price: 3, Minimum: 10})
	if err != nil {
		t.Fatal(err)
	}

	posters, _ := harvest.ListPosters(ctx, "")
	offers, _ := harvest.ListOffers(ctx, "")
	orders, _ := harvest.ListOrders(ctx, "bea")
	groups, _ := harvest.ListGroupOrders(ctx, "coordinator", 0, "")
	if len(posters) != 0 || len(offers) != 0 || len(orders) != 0 || len(groups) != 0 {
		t.Fatalf("Harvest sees Inputs' posters %d, offers %d, orders %d, group orders %d", len(posters), len(offers), len(orders), len(groups))
	}
	if _, err := harvest.GetOrder(ctx, "bea", order.ID); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("Harvest read Inputs' order: %v", err)
	}
	if _, err := harvest.GetGroupOrder(ctx, "coordinator", group.ID); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("Harvest read Inputs' group order: %v", err)
	}

	waiting, err := harvest.ApplyPoster(ctx, "ann", market.Poster{Name: "Poster ann"})
	if err != nil || waiting.Status == models.PosterApproved {
		t.Fatalf("the same owner applying in Harvest: %+v %v", waiting, err)
	}
	if _, err := harvest.ApprovePoster(ctx, "coordinator", poster.ID); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("approving Inputs' poster from Harvest: %v", err)
	}
	inInputs, _ := inputs.ListPosters(ctx, models.PosterApproved)
	inHarvest, _ := harvest.ListPosters(ctx, models.PosterApproved)
	if len(inInputs) != 1 || len(inHarvest) != 0 {
		t.Fatalf("approved posters: Inputs %d, Harvest %d, want 1 and 0", len(inInputs), len(inHarvest))
	}

	if _, err := harvest.ApprovePoster(ctx, "coordinator", waiting.ID); err != nil {
		t.Fatal(err)
	}
	harvestOffer, err := harvest.CreateOffer(ctx, "ann", market.Offer{PosterID: waiting.ID, ProductWords: "maharagwe", UnitWords: "kg", Price: 120, Product: "beans"})
	if err != nil {
		t.Fatal(err)
	}
	harvestOrder, err := harvest.BuyNow(ctx, "bea", harvestOffer.ID, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if order.WorkflowVersion != inputsBuyNowVersion || harvestOrder.WorkflowVersion != 1 {
		t.Fatalf("workflow versions: Inputs %d, Harvest %d, want %d and 1", order.WorkflowVersion, harvestOrder.WorkflowVersion, inputsBuyNowVersion)
	}
	if got, _ := inputs.ListOrders(ctx, "bea"); len(got) != 1 || got[0].ID != order.ID {
		t.Fatalf("Inputs' orders after Harvest took one: %+v", got)
	}
}

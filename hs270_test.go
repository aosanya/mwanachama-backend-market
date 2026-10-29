package market_test

import (
	"context"
	"errors"
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

func upperNames(_ context.Context, id string) string { return "Name of " + id }

func TestAnOrderMadeFromAYesKeepsTheYesAndItsPromisedDate(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	o, _ := m.CreateOffer(ctx, "ann", market.Offer{PosterID: p.ID, ProductWords: "beans", Price: 10})
	r, _ := m.CreateRequest(ctx, "bea", market.Request{Wanted: "beans"})
	a, _ := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: r.ID, Question: "have any?"})
	if _, err := m.AnswerAsk(ctx, "ann", a.ID, models.AnswerYes, "2026-11-01", ""); err != nil {
		t.Fatal(err)
	}
	ys, _ := m.ListYeses(ctx, "bea")
	ord, err := m.BuyNow(ctx, "bea", o.ID, 2, ys[0].ID)
	if err != nil || ord.YesID != ys[0].ID || ord.PromisedDate != "2026-11-01" {
		t.Fatalf("order %+v err %v", ord, err)
	}
	got, _ := m.GetOrder(ctx, "bea", ord.ID)
	list, _ := m.ListOrders(ctx, "ann")
	if got.YesID != ys[0].ID || got.PromisedDate != "2026-11-01" || len(list) != 1 || list[0].PromisedDate != "2026-11-01" {
		t.Fatalf("get %+v list %+v", got, list)
	}
	if _, err := m.BuyNow(ctx, "cat", o.ID, 1, ys[0].ID); !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("another picker used the yes: %v", err)
	}
	plain, _ := m.BuyNow(ctx, "bea", o.ID, 1, "")
	if plain.YesID != "" || plain.PromisedDate != "" {
		t.Fatalf("plain order %+v", plain)
	}
}

func TestReadsCarryThePickersNameWhenTheHostSuppliesOne(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	o, _ := m.CreateOffer(ctx, "ann", market.Offer{PosterID: p.ID, ProductWords: "beans", Price: 10})
	r, _ := m.CreateRequest(ctx, "bea", market.Request{Wanted: "beans"})
	m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: r.ID, Question: "have any?"})
	ord, _ := m.BuyNow(ctx, "bea", o.ID, 2, "")
	g, _ := m.StartGroupOrder(ctx, "coord", market.GroupOrder{PosterID: p.ID, Price: 3, Minimum: 10})
	m.JoinGroupOrder(ctx, "bea", g.ID, 4)

	asks, _ := m.ListAsks(ctx, "ann")
	orders, _ := m.ListOrders(ctx, "ann")
	view, _ := m.GetGroupOrder(ctx, "coord", g.ID)
	if asks[0].PickerName != "" || orders[0].PickerName != "" || view.Lines[0].PickerName != "" {
		t.Fatal("no names were supplied")
	}
	m.UseNames(upperNames)
	asks, _ = m.ListAsks(ctx, "ann")
	orders, _ = m.ListOrders(ctx, "ann")
	got, _ := m.GetOrder(ctx, "bea", ord.ID)
	view, _ = m.GetGroupOrder(ctx, "coord", g.ID)
	want := "Name of bea"
	if asks[0].PickerName != want || orders[0].PickerName != want || got.PickerName != want || view.Lines[0].PickerName != want {
		t.Fatalf("%q %q %q %q", asks[0].PickerName, orders[0].PickerName, got.PickerName, view.Lines[0].PickerName)
	}
}

func TestPosterReadsCarryTheOwnerAndApproverNamesWhenTheHostSuppliesOne(t *testing.T) {
	m, ctx := newManager(t)
	p := approvedPoster(t, m, ctx, "ann")
	applied, _ := m.ApplyPoster(ctx, "cat", market.Poster{Name: "Cat's farm"})
	if p.OwnerName != "" || applied.OwnerName != "" {
		t.Fatal("no names were supplied")
	}
	list, _ := m.ListPosters(ctx, "")
	for _, x := range list {
		if x.OwnerName != "" || x.ApprovedByName != "" {
			t.Fatalf("no names were supplied: %+v", x)
		}
	}
	m.UseNames(upperNames)
	list, _ = m.ListPosters(ctx, "")
	byOwner := map[string]market.Poster{}
	for _, x := range list {
		byOwner[x.Owner] = x
	}
	if byOwner["ann"].OwnerName != "Name of ann" || byOwner["ann"].ApprovedByName != "Name of "+byOwner["ann"].ApprovedBy || byOwner["ann"].ApprovedBy == "" {
		t.Fatalf("approved %+v", byOwner["ann"])
	}
	if byOwner["cat"].OwnerName != "Name of cat" || byOwner["cat"].ApprovedByName != "" {
		t.Fatalf("waiting %+v", byOwner["cat"])
	}
	got, err := m.SetAvailability(ctx, "ann", p.ID, models.Available)
	if err != nil || got.OwnerName != "Name of ann" {
		t.Fatalf("set availability %+v %v", got, err)
	}
	again, err := m.ApplyPoster(ctx, "dee", market.Poster{Name: "Dee"})
	if err != nil || again.OwnerName != "Name of dee" {
		t.Fatalf("apply %+v %v", again, err)
	}
}

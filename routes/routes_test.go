package routes_test

import (
	"context"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/routes"
)

func manager(t *testing.T) market.MarketManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	s, _ := market.SpecFor("app")
	if err := market.Provision(db, s); err != nil {
		t.Fatal(err)
	}
	m, err := market.NewMarketManager(db, s)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestTheDeclaredTableBuildsEveryRoute(t *testing.T) {
	rs, err := routes.Build(manager(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 31 {
		t.Fatalf("want 31 routes, got %d", len(rs))
	}
}

func TestEveryAddressIsGatedByDefault(t *testing.T) {
	if len(routes.AnonymousActions) != 0 {
		t.Fatal("nothing in the market is anonymous")
	}
}

func TestTheDeclaredToolsCarryTheNamesCallersUse(t *testing.T) {
	tools, err := routes.BuildTools(manager(t), routes.Mount{})
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, tl := range tools {
		have[tl.Name] = true
	}
	for _, n := range strings.Fields(`market_apply_poster market_list_posters market_list_offers market_create_offer market_update_offer market_set_stock market_list_asks market_answer_ask market_buy_now market_get_order market_list_orders market_cancel_order market_mark_paid market_list_group_orders market_get_group_order market_join_group_order market_ask_revision market_answer_revision market_create_wanted market_ask market_create_notice market_withdraw_notice market_approve_poster market_decline_poster`) {
		if !have[n] {
			t.Errorf("missing tool %s", n)
		}
	}
	_ = context.Background
}

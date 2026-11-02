package routes_test

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-market/routes"
)

type calledTools struct {
	t    *testing.T
	by   map[string]routes.Tool
	user string
}

func newCalledTools(t *testing.T) *calledTools {
	t.Helper()
	c := &calledTools{t: t, by: map[string]routes.Tool{}}
	tools, err := routes.BuildTools(manager(t), routes.Mount{Caller: func(context.Context) string { return c.user }})
	if err != nil {
		t.Fatal(err)
	}
	for _, tl := range tools {
		c.by[tl.Name] = tl
	}
	return c
}

func (c *calledTools) as(user string) *calledTools {
	c.user = user
	return c
}

func (c *calledTools) call(name string, args map[string]any) (map[string]any, error) {
	c.t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		c.t.Fatal(err)
	}
	out, err := c.by[name].Invoke(context.Background(), raw)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		c.t.Fatal(err)
	}
	body := map[string]any{}
	if err := json.Unmarshal(encoded, &body); err != nil {
		c.t.Fatal(err)
	}
	return body, nil
}

func (c *calledTools) mustCall(name string, args map[string]any) map[string]any {
	c.t.Helper()
	body, err := c.call(name, args)
	if err != nil {
		c.t.Fatalf("%s: %v", name, err)
	}
	return body
}

func (c *calledTools) schema(name string) (properties map[string]map[string]any, required []string) {
	c.t.Helper()
	tl, ok := c.by[name]
	if !ok {
		c.t.Fatalf("no tool %s", name)
	}
	var decoded struct {
		Properties map[string]map[string]any `json:"properties"`
		Required   []string                  `json:"required"`
	}
	if err := json.Unmarshal(tl.InputSchema, &decoded); err != nil {
		c.t.Fatal(err)
	}
	return decoded.Properties, decoded.Required
}

func (c *calledTools) approvedPoster(owner string) string {
	c.t.Helper()
	applied := c.as(owner).mustCall("market_apply_poster", map[string]any{"name": "Ann Seeds"})
	id := applied["id"].(string)
	c.as("coordinator").mustCall("market_approve_poster", map[string]any{"posterID": id})
	return id
}

type wholeRecordShape struct {
	tool     string
	required []string
	kept     []string
	dropped  []string
}

var wholeRecordShapes = []wholeRecordShape{
	{
		tool:     "market_apply_poster",
		required: []string{"name"},
		kept:     []string{"name", "area", "posts"},
		dropped:  []string{"id", "owner", "approved_by", "owner_name", "approved_by_name", "status", "availability", "created_at"},
	},
	{
		tool:     "market_create_offer",
		required: []string{"poster_id", "product_words"},
		kept:     []string{"poster_id", "product_words", "unit_words", "price", "bulk_min", "bulk_price", "product", "category"},
		dropped:  []string{"id", "posted_at"},
	},
	{
		tool:     "market_update_offer",
		required: []string{"offerID", "product_words"},
		kept:     []string{"offerID", "poster_id", "product_words", "unit_words", "price", "bulk_min", "bulk_price", "product", "category"},
		dropped:  []string{"id", "posted_at"},
	},
	{
		tool:     "market_create_wanted",
		required: []string{"wanted"},
		kept:     []string{"wanted", "quantity", "needed_by"},
		dropped:  []string{"id", "picker", "picker_name", "created_at"},
	},
	{
		tool:     "market_ask",
		required: []string{"poster_id", "question"},
		kept:     []string{"poster_id", "request_id", "offer_id", "kind", "product", "question", "detail"},
		dropped:  []string{"id", "picker", "picker_name", "answer", "created_at", "answered_at"},
	},
	{
		tool:     "market_create_notice",
		required: []string{"wanted"},
		kept:     []string{"wanted", "quantity", "needed_by"},
		dropped:  []string{"id", "picker", "picker_name", "state", "created_at"},
	},
	{
		tool:     "market_send_quote",
		required: []string{"noticeID", "poster_id", "price"},
		kept:     []string{"noticeID", "poster_id", "price", "note"},
		dropped:  []string{"id", "notice_id", "picker", "counter_price", "state", "created_at"},
	},
	{
		tool:     "market_set_stock",
		required: []string{"poster_id", "product"},
		kept:     []string{"poster_id", "product", "count"},
		dropped:  []string{"id"},
	},
	{
		tool:     "market_start_group_order",
		required: []string{"minimum"},
		kept:     []string{"poster_id", "product", "price", "minimum", "direction"},
		dropped:  []string{"id", "coordinator", "terms_version", "terms_history", "workflow_version", "state", "created_at"},
	},
}

func TestHS541_EveryToolThatTakesAWholeRecordRequiresWhatItsCodeRequiresAndAsksForNoServerOwnedField(t *testing.T) {
	c := newCalledTools(t)
	for _, want := range wholeRecordShapes {
		t.Run(want.tool, func(t *testing.T) {
			c.t = t
			properties, required := c.schema(want.tool)
			sort.Strings(required)
			wanted := append([]string(nil), want.required...)
			sort.Strings(wanted)
			if strings.Join(required, ",") != strings.Join(wanted, ",") {
				t.Fatalf("required %v, want %v", required, wanted)
			}
			for _, key := range want.kept {
				if _, ok := properties[key]; !ok {
					t.Errorf("the schema lost the input %q", key)
				}
			}
			for _, key := range want.dropped {
				if _, ok := properties[key]; ok {
					t.Errorf("the schema asks for %q, which the server sets", key)
				}
			}
		})
	}
}

func TestHS541_StartingAGroupOrderWithoutAMinimumIsRefusedWithTheSchemasError(t *testing.T) {
	c := newCalledTools(t)
	posterID := c.approvedPoster("ann")
	_, err := c.as("coordinator").call("market_start_group_order", map[string]any{"poster_id": posterID, "product": "beans", "price": 100})
	if err == nil || err.Error() != `market_start_group_order needs "minimum"` {
		t.Fatalf("a group order with no minimum: %v", err)
	}
}

func TestHS541_AServerOwnedFieldIsRefusedAsAnArgumentAndTheServerStillSetsIt(t *testing.T) {
	c := newCalledTools(t)
	posterID := c.approvedPoster("ann")
	for _, field := range []string{"id", "state", "coordinator", "created_at", "terms_version", "workflow_version", "terms_history"} {
		_, err := c.as("coordinator").call("market_start_group_order", map[string]any{"poster_id": posterID, "minimum": 50, field: "x"})
		if err == nil || err.Error() != `market_start_group_order takes no argument "`+field+`"` {
			t.Errorf("a group order sent with %s: %v", field, err)
		}
	}
	started := c.as("coordinator").mustCall("market_start_group_order", map[string]any{"poster_id": posterID, "product": "beans", "price": 100, "minimum": 50})
	if started["id"] == "" || started["state"] != "open" || started["coordinator"] != "coordinator" || started["terms_version"] != 1.0 || started["created_at"] == "" {
		t.Fatalf("the server did not set what it owns: %v", started)
	}
	if started["workflow_version"] == 0.0 {
		t.Fatalf("the group order kept no workflow version: %v", started)
	}
}

func TestHS541_AnOfferIsRefusedItsServerOwnedFieldsAndStillGetsThem(t *testing.T) {
	c := newCalledTools(t)
	posterID := c.approvedPoster("ann")
	for _, field := range []string{"id", "posted_at"} {
		_, err := c.as("ann").call("market_create_offer", map[string]any{"poster_id": posterID, "product_words": "beans", "price": 4, field: "x"})
		if err == nil || err.Error() != `market_create_offer takes no argument "`+field+`"` {
			t.Errorf("an offer sent with %s: %v", field, err)
		}
	}
	for _, missing := range []string{"poster_id", "product_words"} {
		args := map[string]any{"poster_id": posterID, "product_words": "beans", "price": 4}
		delete(args, missing)
		_, err := c.as("ann").call("market_create_offer", args)
		if err == nil || err.Error() != `market_create_offer needs "`+missing+`"` {
			t.Errorf("an offer with no %s: %v", missing, err)
		}
	}
	offer := c.as("ann").mustCall("market_create_offer", map[string]any{"poster_id": posterID, "product_words": "beans", "price": 4})
	if offer["id"] == "" || offer["posted_at"] == "" {
		t.Fatalf("the server did not set what it owns: %v", offer)
	}
}

func TestHS541_ChangingAnOfferStillTakesThePosterTheAppSendsButDoesNotRequireIt(t *testing.T) {
	c := newCalledTools(t)
	posterID := c.approvedPoster("ann")
	offer := c.as("ann").mustCall("market_create_offer", map[string]any{"poster_id": posterID, "product_words": "beans", "price": 4})
	offerID := offer["id"].(string)

	withPoster := c.as("ann").mustCall("market_update_offer", map[string]any{"offerID": offerID, "poster_id": posterID, "product_words": "beans", "price": 5})
	if withPoster["price"] != 5.0 || withPoster["poster_id"] != posterID {
		t.Fatalf("a change that names the poster: %v", withPoster)
	}
	withoutPoster := c.as("ann").mustCall("market_update_offer", map[string]any{"offerID": offerID, "product_words": "beans", "price": 6})
	if withoutPoster["price"] != 6.0 || withoutPoster["poster_id"] != posterID {
		t.Fatalf("a change that leaves the poster out: %v", withoutPoster)
	}
	if _, err := c.as("ann").call("market_update_offer", map[string]any{"offerID": offerID, "price": 7}); err == nil || err.Error() != `market_update_offer needs "product_words"` {
		t.Fatalf("a change with no words: %v", err)
	}
}

func TestHS541_ANoticeAndAWantAreRefusedWhatTheServerSetsAndStillGetIt(t *testing.T) {
	c := newCalledTools(t)
	for _, tool := range []string{"market_create_wanted", "market_create_notice"} {
		_, err := c.as("bea").call(tool, map[string]any{"wanted": "a ride to Bondo", "picker": "mallory"})
		if err == nil || err.Error() != tool+` takes no argument "picker"` {
			t.Errorf("%s sent with picker: %v", tool, err)
		}
		_, err = c.as("bea").call(tool, map[string]any{"quantity": 2})
		if err == nil || err.Error() != tool+` needs "wanted"` {
			t.Errorf("%s with nothing wanted: %v", tool, err)
		}
		made := c.as("bea").mustCall(tool, map[string]any{"wanted": "a ride to Bondo"})
		if made["picker"] != "bea" || made["id"] == "" {
			t.Errorf("%s did not set what it owns: %v", tool, made)
		}
	}
}

func TestHS541_AGroupOrderStartedWithNoPosterNamedBuysFromTheOnlyApprovedOne(t *testing.T) {
	c := newCalledTools(t)
	posterID := c.approvedPoster("ann")
	started := c.as("coordinator").mustCall("market_start_group_order", map[string]any{"product": "beans", "price": 100, "minimum": 50})
	if started["poster_id"] != posterID {
		t.Fatalf("the group order buys from %v, want %v", started["poster_id"], posterID)
	}
}

package market_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

type askWorld struct {
	m        market.MarketManager
	ctx      context.Context
	statemts *[]string
	first    market.Request
	second   market.Request
	firstAsk market.Ask
	nextAsk  market.Ask
}

func newAskWorld(t *testing.T) askWorld {
	t.Helper()
	db := newDB(t)
	var statements []string
	if err := db.Callback().Query().After("gorm:query").Register("record_sql", func(tx *gorm.DB) {
		statements = append(statements, tx.Statement.SQL.String())
	}); err != nil {
		t.Fatal(err)
	}
	s, err := market.SpecFor("app")
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
	ctx := context.Background()
	p := approvedPoster(t, m, ctx, "ann")
	first, err := m.CreateRequest(ctx, "bea", market.Request{Wanted: "hybrid maize seed"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.CreateRequest(ctx, "bea", market.Request{Wanted: "a ride"})
	if err != nil {
		t.Fatal(err)
	}
	firstAsk, err := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: first.ID, Question: "do you hold it?"})
	if err != nil {
		t.Fatal(err)
	}
	nextAsk, err := m.CreateAsk(ctx, "bea", market.Ask{PosterID: p.ID, RequestID: second.ID, Question: "can you come?"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.AnswerAsk(ctx, "ann", firstAsk.ID, models.AnswerYes, "2026-11-03", ""); err != nil {
		t.Fatal(err)
	}
	return askWorld{m: m, ctx: ctx, statemts: &statements, first: first, second: second, firstAsk: firstAsk, nextAsk: nextAsk}
}

func idsOfAsks(asks []market.Ask) []string {
	var ids []string
	for _, a := range asks {
		ids = append(ids, a.ID)
	}
	return ids
}

func TestTheAskListTakesTheRequestAndTheAnswerInTheQuery(t *testing.T) {
	w := newAskWorld(t)
	cases := []struct {
		name, caller, request, answer string
		want                          []string
	}{
		{"everything the asker has", "bea", "", "", []string{w.firstAsk.ID, w.nextAsk.ID}},
		{"one request", "bea", w.first.ID, "", []string{w.firstAsk.ID}},
		{"only the waiting ones", "bea", "", models.AnswerWaiting, []string{w.nextAsk.ID}},
		{"only the yes ones", "bea", "", models.AnswerYes, []string{w.firstAsk.ID}},
		{"a request and an answer that no ask has", "bea", w.first.ID, models.AnswerWaiting, nil},
		{"the poster's owner sees the asks put to the poster", "ann", w.second.ID, "", []string{w.nextAsk.ID}},
		{"a stranger sees none", "cy", w.first.ID, "", nil},
		{"an unknown request is an empty list", "bea", "nope", "", nil},
	}
	for _, c := range cases {
		got, err := w.m.ListAsksBy(w.ctx, c.caller, c.request, c.answer)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if strings.Join(idsOfAsks(got), ",") != strings.Join(c.want, ",") {
			t.Fatalf("%s: got %v want %v", c.name, idsOfAsks(got), c.want)
		}
	}
}

func TestTheAskListRefusesAnAnswerThatIsNotOne(t *testing.T) {
	w := newAskWorld(t)
	if _, err := w.m.ListAsksBy(w.ctx, "bea", "", "maybe"); !errors.Is(err, market.ErrInvalid) {
		t.Fatalf("an unknown answer: %v", err)
	}
}

func TestTheAskListWithNeitherFilterHoldsEveryAskOfTheCaller(t *testing.T) {
	w := newAskWorld(t)
	got, err := w.m.ListAsksBy(w.ctx, "bea", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("bea sees %v", idsOfAsks(got))
	}
}

func TestTheAskFiltersAreInTheStatementAndNotInGo(t *testing.T) {
	w := newAskWorld(t)
	*w.statemts = nil
	if _, err := w.m.ListAsksBy(w.ctx, "bea", w.first.ID, models.AnswerYes); err != nil {
		t.Fatal(err)
	}
	var asked string
	for _, s := range *w.statemts {
		if strings.HasPrefix(s, "SELECT *") {
			asked = s
		}
	}
	for _, column := range []string{"request_id", "answer", "picker"} {
		if !strings.Contains(asked, column) {
			t.Fatalf("the ask statement does not filter on %s: %s", column, asked)
		}
	}
}

func TestTheYesListTakesTheRequestThroughItsAskAndTheState(t *testing.T) {
	w := newAskWorld(t)
	cases := []struct {
		name, caller, request, state string
		want                         int
	}{
		{"everything the asker has", "bea", "", "", 1},
		{"the request the yes answers", "bea", w.first.ID, "", 1},
		{"a request the yes does not answer", "bea", w.second.ID, "", 0},
		{"an open yes", "bea", "", models.YesOpen, 1},
		{"a delivered yes", "bea", "", models.YesDelivered, 0},
		{"the poster's owner sees the yes", "ann", w.first.ID, models.YesOpen, 1},
		{"a stranger sees none", "cy", w.first.ID, "", 0},
		{"an unknown request is an empty list", "bea", "nope", "", 0},
	}
	for _, c := range cases {
		got, err := w.m.ListYesesBy(w.ctx, c.caller, c.request, c.state)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if len(got) != c.want {
			t.Fatalf("%s: got %d want %d", c.name, len(got), c.want)
		}
		if c.want == 1 && got[0].AskID != w.firstAsk.ID {
			t.Fatalf("%s: the yes is for ask %s", c.name, got[0].AskID)
		}
	}
}

func TestTheYesListRefusesAStateThatIsNotOne(t *testing.T) {
	w := newAskWorld(t)
	if _, err := w.m.ListYesesBy(w.ctx, "bea", "", "maybe"); !errors.Is(err, market.ErrInvalid) {
		t.Fatalf("an unknown state: %v", err)
	}
}

func TestTheYesFiltersAreInTheStatementAndNotInGo(t *testing.T) {
	w := newAskWorld(t)
	*w.statemts = nil
	if _, err := w.m.ListYesesBy(w.ctx, "bea", w.first.ID, models.YesOpen); err != nil {
		t.Fatal(err)
	}
	var asked string
	for _, s := range *w.statemts {
		if strings.HasPrefix(s, "SELECT *") {
			asked = s
		}
	}
	for _, column := range []string{"ask_id", "request_id", "state", "picker"} {
		if !strings.Contains(asked, column) {
			t.Fatalf("the yes statement does not filter on %s: %s", column, asked)
		}
	}
}

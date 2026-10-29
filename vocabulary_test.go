package market_test

import (
	"testing"

	market "github.com/aosanya/mwanachama-backend-market"
	"github.com/aosanya/mwanachama-backend-market/models"
)

func heldVocabularies() map[string][]string {
	answers := []string{models.AnswerWaiting, models.AnswerYes, models.AnswerNo}
	return map[string][]string{
		"poster.status":         {models.PosterPending, models.PosterApproved, models.PosterDeclined, models.PosterRemoved},
		"poster.availability":   {models.Available, models.Away},
		"ask.answer":            append(append([]string{}, answers...), models.AnswerWithdrawn),
		"line.answer":           answers,
		"notice.state":          {models.NoticeOpen, models.NoticeWithdrawn, models.NoticeSold},
		"quote.state":           {models.QuoteOpen, models.QuoteCountered, models.QuoteAccepted, models.QuoteDeclined},
		"draft.state":           {models.DraftWaiting, models.DraftSent},
		"yes.state":             {models.YesOpen, models.YesDelivered, models.YesCalledOff},
		"order.state":           {models.OrderPlaced, models.OrderCancelled, models.OrderDelivered},
		"group_order.state":     {models.GroupOpen, models.GroupRevising, models.GroupClosed, models.GroupCancelled},
		"group_order.direction": {models.DirectionBuy, models.DirectionSell},
	}
}

func holdsValue(values []string, s string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}

func TestVocabularyMatchesTheBlueprint(t *testing.T) {
	b, err := market.Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}
	held := heldVocabularies()
	declared := map[string][]string{}
	for _, o := range b.Objects {
		for _, f := range o.Fields {
			if len(f.Values) > 0 {
				declared[o.Role+"."+f.Name] = f.Values
			}
		}
	}
	for at, values := range declared {
		if _, ok := held[at]; !ok {
			t.Errorf("the blueprint declares values for %s and models/vocabulary.go holds none", at)
			continue
		}
		for _, v := range values {
			if !holdsValue(held[at], v) {
				t.Errorf("the blueprint declares %s = %q and Go holds no constant for it", at, v)
			}
		}
	}
	for at, values := range held {
		if _, ok := declared[at]; !ok {
			t.Errorf("models/vocabulary.go holds constants for %s and the blueprint declares no values there", at)
			continue
		}
		for _, v := range values {
			if !holdsValue(declared[at], v) {
				t.Errorf("Go holds %s = %q and the blueprint does not declare it", at, v)
			}
		}
	}
}

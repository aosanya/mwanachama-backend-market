package market

import (
	"sort"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"
)

var markedReferences = map[string]string{
	"posterID":     "market.poster",
	"poster_id":    "market.poster",
	"offerID":      "market.offer",
	"offer_id":     "market.offer",
	"requestID":    "market.request",
	"askID":        "market.ask",
	"noticeID":     "market.notice",
	"quoteID":      "market.quote",
	"draftID":      "market.draft",
	"yes_id":       "market.yes",
	"orderID":      "market.order",
	"groupOrderID": "market.group_order",
	"lineID":       "market.line",
}

func TestEveryArgumentNamingAThingAnOperatorWorksWithIsMarkedWithWhatItRefersTo(t *testing.T) {
	s, err := dispatch.Parse(Operations())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	marked := map[string]int{}
	for _, name := range sortedOperationNames(s) {
		for _, a := range s.Operations[name].Args {
			want, isReference := markedReferences[a.As]
			isReference = isReference && !a.Ignored
			if isReference && a.Refers != want {
				t.Errorf("%s: argument %s refers %q, want %q", name, a.As, a.Refers, want)
			}
			if !isReference && a.Refers != "" {
				t.Errorf("%s: argument %s refers %q but is not one of the things an operator works with", name, a.As, a.Refers)
			}
			if isReference {
				marked[a.As]++
			}
		}
	}
	for as := range markedReferences {
		if marked[as] == 0 {
			t.Errorf("no operation declares %s, so nothing proves it is marked", as)
		}
	}
}

func TestReferencesListsTheMarkedArgumentsAndNoCallerOrWholeRequest(t *testing.T) {
	s, err := dispatch.Parse(Operations())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	references := s.References()
	if len(references) < 38 {
		t.Fatalf("References lists %d arguments, want every poster, offer, request, ask, notice, quote, draft, yes, order, group order and line id the file binds one at a time", len(references))
	}
	for _, r := range references {
		if markedReferences[r.As] != r.Refers {
			t.Errorf("%s.%s is listed as referring %q", r.Operation, r.As, r.Refers)
		}
		if r.As == "caller" || r.As == "" {
			t.Errorf("%s binds %q, which is the session or a whole request, not a reference", r.Operation, r.As)
		}
	}
}

func sortedOperationNames(s *dispatch.Spec) []string {
	names := make([]string, 0, len(s.Operations))
	for name := range s.Operations {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

package market

import (
	"context"
	"errors"
	"testing"

	"github.com/aosanya/mwanachama-backend-market/models"
)

func askOf(t *testing.T, m *marketManager, picker, posterID, question string) Ask {
	t.Helper()
	a, err := m.CreateAsk(context.Background(), picker, Ask{PosterID: posterID, Question: question})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestCreateRequestTakesPickerAndTimeFromTheCallerNotTheBody(t *testing.T) {
	m := newPosterFixture(t)
	r, err := m.CreateRequest(context.Background(), "cy", Request{ID: "forged", Picker: "mallory", Wanted: "maize", CreatedAt: "1999"})
	if err != nil {
		t.Fatal(err)
	}
	if r.ID == "forged" || r.Picker != "cy" || r.CreatedAt == "1999" {
		t.Fatalf("server-owned fields came from the body: %+v", r)
	}
}

func TestCreateRequestWithNothingWantedIsRefused(t *testing.T) {
	m := newPosterFixture(t)
	if _, err := m.CreateRequest(context.Background(), "cy", Request{}); err == nil {
		t.Fatal("a request wanting nothing was accepted")
	}
}

func TestGetRequestIsSeenByItsPickerAndByPostersAskedAboutItOnly(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	ada := approvedPoster(t, m, "ada", "Ada Farm")
	approvedPoster(t, m, "bob", "Bob Shop")
	r, err := m.CreateRequest(ctx, "cy", Request{Wanted: "maize"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.CreateAsk(ctx, "cy", Ask{PosterID: ada.ID, RequestID: r.ID, Question: "have any?"}); err != nil {
		t.Fatal(err)
	}
	own, err := m.GetRequest(ctx, "cy", r.ID)
	if err != nil || own.PickerName != "name-of-cy" {
		t.Fatalf("picker: %+v %v", own, err)
	}
	asked, err := m.GetRequest(ctx, "ada", r.ID)
	if err != nil || asked.PickerName != "name-of-cy" {
		t.Fatalf("asked poster: %+v %v", asked, err)
	}
	for _, caller := range []string{"bob", "dee", ""} {
		if _, err := m.GetRequest(ctx, caller, r.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("caller %q: %v", caller, err)
		}
	}
	if _, err := m.GetRequest(ctx, "cy", "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing request: %v", err)
	}
}

func TestCreateAskStartsWaitingAndIgnoresClaimedAnswerAndPicker(t *testing.T) {
	m := newPosterFixture(t)
	p := approvedPoster(t, m, "ada", "Ada Farm")
	a, err := m.CreateAsk(context.Background(), "cy", Ask{ID: "forged", PosterID: p.ID, Picker: "mallory", Question: "any maize?",
		Answer: models.AnswerYes, AnsweredAt: "1999"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == "forged" || a.Picker != "cy" || a.Answer != models.AnswerWaiting || a.AnsweredAt != "" {
		t.Fatalf("server-owned fields came from the body: %+v", a)
	}
}

func TestCreateAskRefusesUnknownPosterAwayPosterAndRepeat(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	p := approvedPoster(t, m, "ada", "Ada Farm")
	if _, err := m.CreateAsk(ctx, "cy", Ask{PosterID: "missing", Question: "q"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown poster: %v", err)
	}
	askOf(t, m, "cy", p.ID, "first")
	if _, err := m.CreateAsk(ctx, "cy", Ask{PosterID: p.ID, Question: "again"}); !errors.Is(err, ErrAlreadyAsked) {
		t.Fatalf("repeat: %v", err)
	}
	if _, err := m.CreateAsk(ctx, "dee", Ask{PosterID: p.ID, Question: "another picker"}); err != nil {
		t.Fatalf("another picker was refused: %v", err)
	}
	if _, err := m.SetAvailability(ctx, "ada", p.ID, models.Away); err != nil {
		t.Fatal(err)
	}
	if _, err := m.CreateAsk(ctx, "eve", Ask{PosterID: p.ID, Question: "q"}); !errors.Is(err, ErrPosterAway) {
		t.Fatalf("away poster: %v", err)
	}
}

func TestOnlyThePostersOwnerAnswersAnAsk(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	p := approvedPoster(t, m, "ada", "Ada Farm")
	a := askOf(t, m, "cy", p.ID, "q")
	for _, caller := range []string{"cy", "bob", ""} {
		if _, err := m.AnswerAsk(ctx, caller, a.ID, models.AnswerNo, "", ""); !errors.Is(err, ErrForbidden) {
			t.Fatalf("answer by %q: %v", caller, err)
		}
	}
	if _, err := m.AnswerAsk(ctx, "ada", "missing", models.AnswerNo, "", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown ask: %v", err)
	}
}

func TestAnAskIsAnsweredOnlyYesOrNoAndOnlyOnce(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	p := approvedPoster(t, m, "ada", "Ada Farm")
	a := askOf(t, m, "cy", p.ID, "q")
	for _, bad := range []string{"", models.AnswerWaiting, models.AnswerWithdrawn, "maybe"} {
		if _, err := m.AnswerAsk(ctx, "ada", a.ID, bad, "2026-12-01", ""); !errors.Is(err, ErrInvalid) {
			t.Fatalf("answer %q: %v", bad, err)
		}
	}
	got, err := m.AnswerAsk(ctx, "ada", a.ID, models.AnswerNo, "", "")
	if err != nil || got.Answer != models.AnswerNo || got.AnsweredAt == "" || got.PickerName != "name-of-cy" {
		t.Fatalf("no: %+v %v", got, err)
	}
	if _, err := m.AnswerAsk(ctx, "ada", a.ID, models.AnswerYes, "2026-12-01", ""); !errors.Is(err, ErrConflict) {
		t.Fatalf("second answer: %v", err)
	}
	yeses, err := m.ListYesesBy(ctx, "cy", "", "")
	if err != nil || len(yeses) != 0 {
		t.Fatalf("a no left a yes behind: %+v %v", yeses, err)
	}
}

func TestAYesNeedsADateAndAFreshOneRecordsTheOffer(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	p := approvedPoster(t, m, "ada", "Ada Farm")
	offer, err := m.CreateOffer(ctx, "ada", Offer{PosterID: p.ID, ProductWords: "maize", Price: 10})
	if err != nil {
		t.Fatal(err)
	}
	a := askOf(t, m, "cy", p.ID, "q")
	if _, err := m.AnswerAsk(ctx, "ada", a.ID, models.AnswerYes, "", "soon"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("yes without a date: %v", err)
	}
	got, err := m.AnswerAskWithMinutes(ctx, "ada", a.ID, models.AnswerYes, "2026-12-01", "bring a bag", offer.ID, 9.5, 30)
	if err != nil || got.Answer != models.AnswerYes {
		t.Fatalf("yes: %+v %v", got, err)
	}
	yeses, err := m.ListYesesBy(ctx, "cy", "", models.YesOpen)
	if err != nil || len(yeses) != 1 {
		t.Fatalf("yeses: %+v %v", yeses, err)
	}
	y := yeses[0]
	if y.AskID != a.ID || y.Picker != "cy" || y.PosterID != p.ID || y.OfferID != offer.ID || y.Price != 9.5 ||
		y.Minutes != 30 || y.Date != "2026-12-01" || y.Note != "bring a bag" || y.PickerName != "name-of-cy" {
		t.Fatalf("yes record: %+v", y)
	}
}

func TestAnAnswerRefusesWhatDoesNotBelongToIt(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	ada := approvedPoster(t, m, "ada", "Ada Farm")
	bob := approvedPoster(t, m, "bob", "Bob Shop")
	bobsOffer, err := m.CreateOffer(ctx, "bob", Offer{PosterID: bob.ID, ProductWords: "beans", Price: 5})
	if err != nil {
		t.Fatal(err)
	}
	adasOffer, err := m.CreateOffer(ctx, "ada", Offer{PosterID: ada.ID, ProductWords: "maize", Price: 5})
	if err != nil {
		t.Fatal(err)
	}
	a := askOf(t, m, "cy", ada.ID, "q")
	cases := map[string]func() error{
		"another poster's offer": func() error {
			_, err := m.AnswerAskWithOffer(ctx, "ada", a.ID, models.AnswerYes, "2026-12-01", "", bobsOffer.ID, 5)
			return err
		},
		"unknown offer": func() error {
			_, err := m.AnswerAskWithOffer(ctx, "ada", a.ID, models.AnswerYes, "2026-12-01", "", "missing", 5)
			return err
		},
		"negative price": func() error {
			_, err := m.AnswerAskWithOffer(ctx, "ada", a.ID, models.AnswerYes, "2026-12-01", "", adasOffer.ID, -1)
			return err
		},
		"negative minutes": func() error {
			_, err := m.AnswerAskWithMinutes(ctx, "ada", a.ID, models.AnswerYes, "2026-12-01", "", "", 0, -5)
			return err
		},
		"no with a price": func() error {
			_, err := m.AnswerAskWithOffer(ctx, "ada", a.ID, models.AnswerNo, "", "", "", 3)
			return err
		},
		"no with an offer": func() error {
			_, err := m.AnswerAskWithOffer(ctx, "ada", a.ID, models.AnswerNo, "", "", adasOffer.ID, 0)
			return err
		},
		"no with minutes": func() error {
			_, err := m.AnswerAskWithMinutes(ctx, "ada", a.ID, models.AnswerNo, "", "", "", 0, 10)
			return err
		},
	}
	for name, call := range cases {
		if err := call(); !errors.Is(err, ErrInvalid) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	still, err := m.ListAsksBy(ctx, "cy", "", models.AnswerWaiting)
	if err != nil || len(still) != 1 {
		t.Fatalf("a refused answer changed the ask: %+v %v", still, err)
	}
}

func TestOnlyThePickerWithdrawsAndOnlyAWaitingAsk(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	p := approvedPoster(t, m, "ada", "Ada Farm")
	a := askOf(t, m, "cy", p.ID, "q")
	for _, caller := range []string{"ada", "dee", ""} {
		if _, err := m.WithdrawAsk(ctx, caller, a.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("withdraw by %q: %v", caller, err)
		}
	}
	got, err := m.WithdrawAsk(ctx, "cy", a.ID)
	if err != nil || got.Answer != models.AnswerWithdrawn {
		t.Fatalf("withdraw: %+v %v", got, err)
	}
	if _, err := m.WithdrawAsk(ctx, "cy", a.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("second withdraw: %v", err)
	}
	if _, err := m.AnswerAsk(ctx, "ada", a.ID, models.AnswerNo, "", ""); !errors.Is(err, ErrConflict) {
		t.Fatalf("answering a withdrawn ask: %v", err)
	}
}

func TestListAsksByShowsPickersTheirOwnAndPostersTheirsAndNobodyElse(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	ada := approvedPoster(t, m, "ada", "Ada Farm")
	bob := approvedPoster(t, m, "bob", "Bob Shop")
	askOf(t, m, "cy", ada.ID, "to ada")
	askOf(t, m, "cy", bob.ID, "to bob")
	askOf(t, m, "dee", ada.ID, "dee to ada")
	counts := map[string]int{"cy": 2, "dee": 1, "ada": 2, "bob": 1, "eve": 0, "": 0}
	for caller, want := range counts {
		got, err := m.ListAsksBy(ctx, caller, "", "")
		if err != nil || len(got) != want {
			t.Fatalf("caller %q saw %d asks (%v), want %d", caller, len(got), err, want)
		}
	}
	got, _ := m.ListAsksBy(ctx, "ada", "", "")
	for _, a := range got {
		if a.PickerName != "name-of-"+a.Picker {
			t.Fatalf("picker not named: %+v", a)
		}
	}
}

func TestListAsksByFiltersByRequestAndAnswerAndRefusesAnUnknownAnswer(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	ada := approvedPoster(t, m, "ada", "Ada Farm")
	bob := approvedPoster(t, m, "bob", "Bob Shop")
	r, err := m.CreateRequest(ctx, "cy", Request{Wanted: "maize"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.CreateAsk(ctx, "cy", Ask{PosterID: ada.ID, RequestID: r.ID, Question: "q"}); err != nil {
		t.Fatal(err)
	}
	other := askOf(t, m, "cy", bob.ID, "loose")
	if _, err := m.AnswerAsk(ctx, "bob", other.ID, models.AnswerNo, "", ""); err != nil {
		t.Fatal(err)
	}
	byRequest, err := m.ListAsksBy(ctx, "cy", r.ID, "")
	if err != nil || len(byRequest) != 1 || byRequest[0].RequestID != r.ID {
		t.Fatalf("by request: %+v %v", byRequest, err)
	}
	answered, err := m.ListAsksBy(ctx, "cy", "", models.AnswerNo)
	if err != nil || len(answered) != 1 || answered[0].ID != other.ID {
		t.Fatalf("by answer: %+v %v", answered, err)
	}
	if _, err := m.ListAsksBy(ctx, "cy", "", "maybe"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown answer: %v", err)
	}
}

func TestListYesesByFiltersByStateAndRequestAndShowsOnlyTheCallersOwn(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	ada := approvedPoster(t, m, "ada", "Ada Farm")
	r, err := m.CreateRequest(ctx, "cy", Request{Wanted: "maize"})
	if err != nil {
		t.Fatal(err)
	}
	withRequest, err := m.CreateAsk(ctx, "cy", Ask{PosterID: ada.ID, RequestID: r.ID, Question: "q"})
	if err != nil {
		t.Fatal(err)
	}
	loose := askOf(t, m, "dee", ada.ID, "loose")
	for _, a := range []Ask{withRequest, loose} {
		if _, err := m.AnswerAsk(ctx, "ada", a.ID, models.AnswerYes, "2026-12-01", ""); err != nil {
			t.Fatal(err)
		}
	}
	byRequest, err := m.ListYesesBy(ctx, "ada", r.ID, "")
	if err != nil || len(byRequest) != 1 || byRequest[0].AskID != withRequest.ID {
		t.Fatalf("by request: %+v %v", byRequest, err)
	}
	if mine, _ := m.ListYesesBy(ctx, "dee", "", models.YesOpen); len(mine) != 1 || mine[0].AskID != loose.ID {
		t.Fatalf("dee saw %+v", mine)
	}
	if none, _ := m.ListYesesBy(ctx, "eve", "", ""); len(none) != 0 {
		t.Fatalf("a stranger saw yeses: %+v", none)
	}
	if none, _ := m.ListYesesBy(ctx, "ada", "", models.YesDelivered); len(none) != 0 {
		t.Fatalf("delivered filter matched open yeses: %+v", none)
	}
	if _, err := m.ListYesesBy(ctx, "ada", "", "lost"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown state: %v", err)
	}
}

func TestNoticesAreOpenToAllButNameOnlyTheirOwnPickerInFull(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	n, err := m.CreateNotice(ctx, "cy", Notice{ID: "forged", Picker: "mallory", Wanted: "maize", State: models.NoticeWithdrawn})
	if err != nil {
		t.Fatal(err)
	}
	if n.ID == "forged" || n.Picker != "cy" || n.State != models.NoticeOpen {
		t.Fatalf("server-owned fields came from the body: %+v", n)
	}
	if _, err := m.CreateNotice(ctx, "cy", Notice{}); err == nil {
		t.Fatal("a notice wanting nothing was accepted")
	}
	if _, err := m.CreateNotice(ctx, "dee", Notice{Wanted: "beans"}); err != nil {
		t.Fatal(err)
	}
	own, err := m.ListNoticesFor(ctx, "cy")
	if err != nil || len(own) != 2 {
		t.Fatalf("own: %+v %v", own, err)
	}
	for _, x := range own {
		switch x.Wanted {
		case "maize":
			if x.Picker != "cy" || x.PickerName != "name-of-cy" {
				t.Fatalf("own notice hidden from its picker: %+v", x)
			}
		case "beans":
			if x.Picker != "" || x.PickerName != "name-of-dee" {
				t.Fatalf("somebody else's notice leaked its picker: %+v", x)
			}
		}
	}
	for _, x := range mustNotices(t, m, "") {
		if x.Picker != "" {
			t.Fatalf("anonymous reader saw a picker id: %+v", x)
		}
	}
}

func mustNotices(t *testing.T, m *marketManager, caller string) []Notice {
	t.Helper()
	got, err := m.ListNoticesFor(context.Background(), caller)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestOnlyThePickerWithdrawsANoticeAndItLeavesTheOpenList(t *testing.T) {
	m := newPosterFixture(t)
	ctx := context.Background()
	n, err := m.CreateNotice(ctx, "cy", Notice{Wanted: "maize"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.WithdrawNotice(ctx, "dee", n.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("withdraw by another: %v", err)
	}
	if _, err := m.WithdrawNotice(ctx, "cy", "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("withdraw unknown: %v", err)
	}
	got, err := m.WithdrawNotice(ctx, "cy", n.ID)
	if err != nil || got.State != models.NoticeWithdrawn {
		t.Fatalf("withdraw: %+v %v", got, err)
	}
	if open, err := m.ListNotices(ctx); err != nil || len(open) != 0 {
		t.Fatalf("a withdrawn notice is still listed: %+v %v", open, err)
	}
}

func TestFirstWordOfAName(t *testing.T) {
	cases := map[string]string{"Ada Lovelace": "Ada", "  Bob  ": "Bob", "Cy": "Cy", "": "", "   ": ""}
	for in, want := range cases {
		if got := firstWord(in); got != want {
			t.Fatalf("firstWord(%q) = %q, want %q", in, got, want)
		}
	}
}

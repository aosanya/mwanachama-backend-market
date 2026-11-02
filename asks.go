package market

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-market/models"
)

func (m *marketManager) CreateRequest(ctx context.Context, caller string, r Request) (Request, error) {
	r.ID, r.Picker, r.CreatedAt = newID(), caller, m.now()
	if err := m.checks(RoleRequest, r); err != nil {
		return r, err
	}
	return r, m.insert(ctx, RoleRequest, r)
}

func (m *marketManager) GetRequest(ctx context.Context, caller, id string) (Request, error) {
	var r Request
	if err := m.get(ctx, RoleRequest, id, &r); err != nil {
		return r, err
	}
	if caller != "" && r.Picker == caller {
		r.PickerName = m.nameOf(ctx, r.Picker)
		return r, nil
	}
	owned, err := m.ownedPosters(ctx, caller)
	if err != nil {
		return r, err
	}
	asks, err := listOf[Ask](m, m.q(ctx, RoleAsk).Where("request_id = ?", id), RoleAsk)
	if err != nil {
		return r, err
	}
	for _, a := range asks {
		if owned[a.PosterID] {
			r.PickerName = m.nameOf(ctx, r.Picker)
			return r, nil
		}
	}
	return r, ErrNotFound
}

func (m *marketManager) CreateAsk(ctx context.Context, caller string, a Ask) (Ask, error) {
	var p Poster
	if err := m.get(ctx, RolePoster, a.PosterID, &p); err != nil {
		return a, err
	}
	if p.Availability == models.Away {
		return a, ErrPosterAway
	}
	a.ID, a.Picker, a.Answer, a.CreatedAt, a.AnsweredAt = newID(), caller, models.AnswerWaiting, m.now(), ""
	if err := m.checks(RoleAsk, a); err != nil {
		return a, err
	}
	n, err := m.count(ctx, RoleAsk, "poster_id = ? AND picker = ? AND request_id = ? AND offer_id = ?", a.PosterID, caller, a.RequestID, a.OfferID)
	if err != nil {
		return a, err
	}
	if n > 0 {
		return a, ErrAlreadyAsked
	}
	return a, m.insert(ctx, RoleAsk, a)
}

func (m *marketManager) AnswerAsk(ctx context.Context, caller, askID, answer, date, note string) (Ask, error) {
	return m.AnswerAskWithOffer(ctx, caller, askID, answer, date, note, "", 0)
}

func (m *marketManager) quotedOffer(ctx context.Context, posterID, offerID string, price float64) error {
	if price < 0 {
		return fmt.Errorf("%w: a price is not negative", ErrInvalid)
	}
	if offerID == "" {
		return nil
	}
	var o Offer
	if err := m.get(ctx, RoleOffer, offerID, &o); err != nil || o.PosterID != posterID {
		return fmt.Errorf("%w: a yes points only at the answering poster's own offer", ErrInvalid)
	}
	return nil
}

func (m *marketManager) AnswerAskWithOffer(ctx context.Context, caller, askID, answer, date, note, offerID string, price float64) (Ask, error) {
	return m.AnswerAskWithMinutes(ctx, caller, askID, answer, date, note, offerID, price, 0)
}

func (m *marketManager) AnswerAskWithMinutes(ctx context.Context, caller, askID, answer, date, note, offerID string, price float64, minutes int) (Ask, error) {
	var a Ask
	if err := m.get(ctx, RoleAsk, askID, &a); err != nil {
		return a, err
	}
	if _, err := m.posterOwnedBy(ctx, caller, a.PosterID); err != nil {
		return a, err
	}
	if _, _, err := m.gate(ctx, WorkflowBuyNow, 0, StepYes, caller, parties{picker: a.Picker, posterID: a.PosterID}); err != nil {
		return a, err
	}
	if answer != models.AnswerYes && answer != models.AnswerNo {
		return a, fmt.Errorf("%w: an ask is answered yes or no, silence stays waiting", ErrInvalid)
	}
	if a.Answer != models.AnswerWaiting {
		return a, ErrConflict
	}
	if answer == models.AnswerNo && (offerID != "" || price != 0 || minutes != 0) {
		return a, fmt.Errorf("%w: a no carries no offer, price or minutes", ErrInvalid)
	}
	if minutes < 0 {
		return a, fmt.Errorf("%w: minutes are not negative", ErrInvalid)
	}
	if answer == models.AnswerYes {
		if err := m.quotedOffer(ctx, a.PosterID, offerID, price); err != nil {
			return a, err
		}
		y := Yes{Date: date, Note: note, OfferID: offerID, Price: price, Minutes: minutes}
		if y.Date == "" {
			return a, fmt.Errorf("%w: a yes carries a date", ErrInvalid)
		}
		y.ID, y.AskID, y.PosterID, y.Picker = newID(), a.ID, a.PosterID, a.Picker
		y.State, y.CreatedAt = models.YesOpen, m.now()
		if err := m.checks(RoleYes, y); err != nil {
			return a, err
		}
		if err := m.insert(ctx, RoleYes, y); err != nil {
			return a, err
		}
	}
	a.Answer, a.AnsweredAt = answer, m.now()
	a.PickerName = m.nameOf(ctx, a.Picker)
	return a, m.update(ctx, RoleAsk, a.ID, a)
}

func (m *marketManager) WithdrawAsk(ctx context.Context, caller, id string) (Ask, error) {
	var a Ask
	if err := m.get(ctx, RoleAsk, id, &a); err != nil {
		return a, err
	}
	if caller == "" || a.Picker != caller {
		return a, ErrNotFound
	}
	if a.Answer != models.AnswerWaiting {
		return a, fmt.Errorf("%w: only an ask that is still waiting can be withdrawn", ErrConflict)
	}
	a.Answer = models.AnswerWithdrawn
	a.PickerName = m.nameOf(ctx, a.Picker)
	return a, m.update(ctx, RoleAsk, a.ID, a)
}

func (m *marketManager) ListAsks(ctx context.Context, caller string) ([]Ask, error) {
	return m.ListAsksBy(ctx, caller, "", "")
}

func knownAnswer(answer string) bool {
	switch answer {
	case models.AnswerWaiting, models.AnswerYes, models.AnswerNo, models.AnswerWithdrawn:
		return true
	}
	return false
}

func knownYesState(state string) bool {
	switch state {
	case models.YesOpen, models.YesDelivered, models.YesCalledOff:
		return true
	}
	return false
}

func (m *marketManager) callersOwn(ctx context.Context, q *gorm.DB, caller string) *gorm.DB {
	if caller == "" {
		return q.Where("picker = ?", caller)
	}
	return q.Where("(picker = ? OR poster_id IN (?))", caller, m.q(ctx, RolePoster).Select("id").Where("owner = ?", caller))
}

func (m *marketManager) ListAsksBy(ctx context.Context, caller, requestID, answer string) ([]Ask, error) {
	if answer != "" && !knownAnswer(answer) {
		return nil, fmt.Errorf("%w: %q is not an answer", ErrInvalid, answer)
	}
	q := m.callersOwn(ctx, m.q(ctx, RoleAsk), caller)
	if requestID != "" {
		q = q.Where("request_id = ?", requestID)
	}
	if answer != "" {
		q = q.Where("answer = ?", answer)
	}
	asks, err := listOf[Ask](m, q, RoleAsk)
	if err != nil {
		return nil, err
	}
	for i := range asks {
		asks[i].PickerName = m.nameOf(ctx, asks[i].Picker)
	}
	return asks, nil
}

func (m *marketManager) ListYeses(ctx context.Context, caller string) ([]Yes, error) {
	return m.ListYesesBy(ctx, caller, "", "")
}

func (m *marketManager) ListYesesBy(ctx context.Context, caller, requestID, state string) ([]Yes, error) {
	if state != "" && !knownYesState(state) {
		return nil, fmt.Errorf("%w: %q is not a state of a yes", ErrInvalid, state)
	}
	q := m.callersOwn(ctx, m.q(ctx, RoleYes), caller)
	if requestID != "" {
		q = q.Where("ask_id IN (?)", m.q(ctx, RoleAsk).Select("id").Where("request_id = ?", requestID))
	}
	if state != "" {
		q = q.Where("state = ?", state)
	}
	yeses, err := listOf[Yes](m, q, RoleYes)
	if err != nil {
		return nil, err
	}
	for i := range yeses {
		yeses[i].PickerName = m.nameOf(ctx, yeses[i].Picker)
	}
	return yeses, nil
}

func (m *marketManager) ownedPosters(ctx context.Context, caller string) (map[string]bool, error) {
	ps, err := listOf[Poster](m, m.q(ctx, RolePoster).Where("owner = ?", caller), RolePoster)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, p := range ps {
		out[p.ID] = caller != ""
	}
	return out, nil
}

func (m *marketManager) CreateNotice(ctx context.Context, caller string, n Notice) (Notice, error) {
	n.ID, n.Picker, n.State, n.CreatedAt = newID(), caller, models.NoticeOpen, m.now()
	if err := m.checks(RoleNotice, n); err != nil {
		return n, err
	}
	return n, m.insert(ctx, RoleNotice, n)
}

func (m *marketManager) WithdrawNotice(ctx context.Context, caller, id string) (Notice, error) {
	var n Notice
	if err := m.get(ctx, RoleNotice, id, &n); err != nil {
		return n, err
	}
	if n.Picker != caller {
		return n, ErrNotFound
	}
	n.State = models.NoticeWithdrawn
	return n, m.update(ctx, RoleNotice, id, n)
}

func (m *marketManager) ListNotices(ctx context.Context) ([]Notice, error) {
	return listOf[Notice](m, m.q(ctx, RoleNotice).Where("state = ?", models.NoticeOpen), RoleNotice)
}

func (m *marketManager) ListNoticesFor(ctx context.Context, caller string) ([]Notice, error) {
	notices, err := m.ListNotices(ctx)
	if err != nil {
		return nil, err
	}
	for i := range notices {
		name := m.nameOf(ctx, notices[i].Picker)
		if caller != "" && notices[i].Picker == caller {
			notices[i].PickerName = name
			continue
		}
		notices[i].Picker, notices[i].PickerName = "", firstWord(name)
	}
	return notices, nil
}

func firstWord(name string) string {
	if words := strings.Fields(name); len(words) > 0 {
		return words[0]
	}
	return ""
}

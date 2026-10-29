package market

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-market/models"
)

const stampLayout = "2006-01-02T15:04:05.000000Z07:00"

type Poster = models.Poster
type Offer = models.Offer
type Request = models.Request
type Ask = models.Ask
type Notice = models.Notice
type Quote = models.Quote
type Draft = models.Draft
type Yes = models.Yes
type Order = models.Order
type GroupOrder = models.GroupOrder
type Line = models.Line
type Stock = models.Stock
type Trail = models.Trail
type GroupOrderView = models.GroupOrderView

type MarketManager interface {
	UseWorkflows(s WorkflowSource)
	UseCoordinators(c CoordinatorCheck)
	UseAgents(a AgentCheck)
	UseNames(n NameOf)
	ForRun(runID string) (MarketManager, error)

	ApplyPoster(ctx context.Context, caller string, p Poster) (Poster, error)
	ListPosters(ctx context.Context, status string) ([]Poster, error)
	ApprovePoster(ctx context.Context, caller, id string) (Poster, error)
	DeclinePoster(ctx context.Context, caller, id string) (Poster, error)
	RemovePoster(ctx context.Context, caller, id string) (Poster, error)
	SetAvailability(ctx context.Context, caller, id, availability string) (Poster, error)

	CreateOffer(ctx context.Context, caller string, o Offer) (Offer, error)
	UpdateOffer(ctx context.Context, caller string, o Offer) (Offer, error)
	ListOffers(ctx context.Context, posterID string) ([]Offer, error)

	CreateRequest(ctx context.Context, caller string, r Request) (Request, error)
	GetRequest(ctx context.Context, caller, id string) (Request, error)
	CreateAsk(ctx context.Context, caller string, a Ask) (Ask, error)
	AnswerAsk(ctx context.Context, caller, askID, answer, date, note string) (Ask, error)
	AnswerAskWithOffer(ctx context.Context, caller, askID, answer, date, note, offerID string, price float64) (Ask, error)
	ListAsks(ctx context.Context, caller string) ([]Ask, error)
	ListYeses(ctx context.Context, caller string) ([]Yes, error)

	CreateNotice(ctx context.Context, caller string, n Notice) (Notice, error)
	WithdrawNotice(ctx context.Context, caller, id string) (Notice, error)
	ListNotices(ctx context.Context) ([]Notice, error)

	SendQuote(ctx context.Context, caller, noticeID string, q Quote) (Quote, error)
	ListQuotes(ctx context.Context, caller, noticeID string) ([]Quote, error)
	DraftCounter(ctx context.Context, caller, quoteID string, price float64, note string) (Draft, error)
	ListDrafts(ctx context.Context, caller, noticeID string) ([]Draft, error)
	SendCounter(ctx context.Context, caller, draftID string) (Quote, error)
	ReviseQuote(ctx context.Context, caller, quoteID string, price float64, note string) (Quote, error)
	AcceptQuote(ctx context.Context, caller, quoteID string) (Quote, error)

	BuyNow(ctx context.Context, caller, offerID string, quantity float64, yesID string) (Order, error)
	CancelOrder(ctx context.Context, caller, id string) (Order, error)
	MarkPaid(ctx context.Context, caller, id string) (Order, error)
	GetOrder(ctx context.Context, caller, id string) (Order, error)
	ListOrders(ctx context.Context, caller string) ([]Order, error)
	OrderTrail(ctx context.Context, caller, orderID string) ([]Trail, error)

	SetStock(ctx context.Context, caller string, s Stock) (Stock, error)
	GetStock(ctx context.Context, caller, posterID string) ([]Stock, error)

	StartGroupOrder(ctx context.Context, caller string, g GroupOrder) (GroupOrder, error)
	JoinGroupOrder(ctx context.Context, caller, id string, quantity float64) (Line, error)
	AskRevision(ctx context.Context, caller, id string, price, minimum float64) (GroupOrder, error)
	AnswerRevision(ctx context.Context, caller, id, answer string) (Line, error)
	PromiseGroupOrder(ctx context.Context, caller, id, date, note string) (GroupOrder, error)
	ListGroupOrders(ctx context.Context, caller string, limit int, after string) ([]GroupOrderView, error)
	GroupOrderTrail(ctx context.Context, caller, id string) ([]Trail, error)
	GetGroupOrder(ctx context.Context, caller, id string) (GroupOrderView, error)
}

type marketManager struct {
	db    *gorm.DB
	st    *store
	clock func() time.Time

	source       WorkflowSource
	coordinators CoordinatorCheck
	agents       AgentCheck
	names        NameOf
}

func NewMarketManager(db *gorm.DB, s *spec.Spec) (MarketManager, error) {
	st, err := newStore(db, s, map[string]any{
		RolePoster:     models.Poster{},
		RoleOffer:      models.Offer{},
		RoleRequest:    models.Request{},
		RoleAsk:        models.Ask{},
		RoleNotice:     models.Notice{},
		RoleQuote:      models.Quote{},
		RoleDraft:      models.Draft{},
		RoleYes:        models.Yes{},
		RoleOrder:      models.Order{},
		RoleGroupOrder: models.GroupOrder{},
		RoleLine:       models.Line{},
		RoleStock:      models.Stock{},
		RoleTrail:      models.Trail{},
	})
	if err != nil {
		return nil, fmt.Errorf("NewMarketManager: %w", err)
	}
	return &marketManager{db: db, st: st, clock: time.Now}, nil
}

func (m *marketManager) now() string { return m.clock().UTC().Format(stampLayout) }

func (m *marketManager) q(ctx context.Context, role string) *gorm.DB {
	return m.st.Query(ctx, role)
}

func (m *marketManager) checks(role string, v any) error { return check(m.st.Object(role), v) }

func (m *marketManager) insert(ctx context.Context, role string, v any) error {
	row, err := m.st.Row(role, v)
	if err != nil {
		return err
	}
	return m.q(ctx, role).Create(row).Error
}

func (m *marketManager) update(ctx context.Context, role, id string, v any) error {
	row, err := encode(m.st.Object(role), v)
	if err != nil {
		return err
	}
	return m.q(ctx, role).Where("id = ?", id).Updates(row).Error
}

func (m *marketManager) get(ctx context.Context, role, id string, out any) error {
	var rows []map[string]any
	if err := m.q(ctx, role).Where("id = ?", id).Limit(1).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return ErrNotFound
	}
	return decode(m.st.Object(role), rows[0], out)
}

func listOf[T any](m *marketManager, q *gorm.DB, role string) ([]T, error) {
	return specstore.List[T](m.st, q, role)
}

func (m *marketManager) count(ctx context.Context, role, where string, args ...any) (int, error) {
	var n int64
	err := m.q(ctx, role).Where(where, args...).Count(&n).Error
	return int(n), err
}

func (m *marketManager) posterOwnedBy(ctx context.Context, caller, posterID string) (Poster, error) {
	var p Poster
	if err := m.get(ctx, RolePoster, posterID, &p); err != nil {
		return p, err
	}
	if caller == "" || p.Owner != caller {
		return p, ErrForbidden
	}
	return p, nil
}

func (m *marketManager) note(ctx context.Context, orderID, actor, what string) error {
	return m.insert(ctx, RoleTrail, Trail{ID: newID(), OrderID: orderID, Actor: actor, What: what, At: m.now()})
}

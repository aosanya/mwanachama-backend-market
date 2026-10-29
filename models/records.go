package models

type Poster struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Area         string `json:"area,omitempty"`
	Posts        string `json:"posts,omitempty"`
	Owner        string `json:"owner,omitempty"`
	ApprovedBy   string `json:"approved_by,omitempty"`
	Status       string `json:"status"`
	Availability string `json:"availability"`
	CreatedAt    string `json:"created_at"`
}

type Offer struct {
	ID           string  `json:"id"`
	PosterID     string  `json:"poster_id"`
	ProductWords string  `json:"product_words"`
	UnitWords    string  `json:"unit_words,omitempty"`
	Price        float64 `json:"price"`
	BulkMin      float64 `json:"bulk_min,omitempty"`
	BulkPrice    float64 `json:"bulk_price,omitempty"`
	Product      string  `json:"product,omitempty"`
	PostedAt     string  `json:"posted_at"`
}

type Request struct {
	ID        string  `json:"id"`
	Picker    string  `json:"picker"`
	Wanted    string  `json:"wanted"`
	Quantity  float64 `json:"quantity,omitempty"`
	NeededBy  string  `json:"needed_by,omitempty"`
	CreatedAt string  `json:"created_at"`
}

type Ask struct {
	ID         string `json:"id"`
	PosterID   string `json:"poster_id"`
	RequestID  string `json:"request_id,omitempty"`
	OfferID    string `json:"offer_id,omitempty"`
	Picker     string `json:"picker,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Question   string `json:"question"`
	Answer     string `json:"answer"`
	Detail     string `json:"detail,omitempty"`
	PickerName string `json:"picker_name,omitempty" spec:"-"`
	CreatedAt  string `json:"created_at"`
	AnsweredAt string `json:"answered_at,omitempty"`
}

type Notice struct {
	ID        string  `json:"id"`
	Picker    string  `json:"picker"`
	Wanted    string  `json:"wanted"`
	Quantity  float64 `json:"quantity,omitempty"`
	NeededBy  string  `json:"needed_by,omitempty"`
	State     string  `json:"state"`
	CreatedAt string  `json:"created_at"`
}

type Yes struct {
	ID        string  `json:"id"`
	AskID     string  `json:"ask_id"`
	PosterID  string  `json:"poster_id,omitempty"`
	Picker    string  `json:"picker,omitempty"`
	What      string  `json:"what,omitempty"`
	Quantity  float64 `json:"quantity,omitempty"`
	Date      string  `json:"date,omitempty"`
	Note      string  `json:"note,omitempty"`
	State     string  `json:"state"`
	CreatedAt string  `json:"created_at"`
}

type Order struct {
	ID              string  `json:"id"`
	Picker          string  `json:"picker"`
	PosterID        string  `json:"poster_id"`
	OfferID         string  `json:"offer_id,omitempty"`
	Product         string  `json:"product,omitempty"`
	Quantity        float64 `json:"quantity"`
	UnitPrice       float64 `json:"unit_price"`
	WorkflowVersion int     `json:"workflow_version"`
	State           string  `json:"state"`
	Paid            bool    `json:"paid"`
	YesID           string  `json:"yes_id,omitempty"`
	PromisedDate    string  `json:"promised_date,omitempty"`
	PickerName      string  `json:"picker_name,omitempty" spec:"-"`
	CreatedAt       string  `json:"created_at"`
}

type GroupOrder struct {
	ID              string  `json:"id"`
	PosterID        string  `json:"poster_id"`
	Coordinator     string  `json:"coordinator,omitempty"`
	Product         string  `json:"product,omitempty"`
	Price           float64 `json:"price"`
	Minimum         float64 `json:"minimum"`
	TermsVersion    int     `json:"terms_version"`
	TermsHistory    string  `json:"terms_history,omitempty"`
	WorkflowVersion int     `json:"workflow_version"`
	State           string  `json:"state"`
	Direction       string  `json:"direction"`
	CreatedAt       string  `json:"created_at"`
}

type Line struct {
	ID                   string  `json:"id"`
	GroupOrderID         string  `json:"group_order_id"`
	Picker               string  `json:"picker,omitempty"`
	Quantity             float64 `json:"quantity"`
	Answer               string  `json:"answer"`
	AnsweredVersion      int     `json:"answered_version"`
	DecidedByCoordinator bool    `json:"decided_by_coordinator"`
	Paid                 bool    `json:"paid"`
	PickerName           string  `json:"picker_name,omitempty" spec:"-"`
	CreatedAt            string  `json:"created_at"`
}

type Stock struct {
	ID       string  `json:"id"`
	PosterID string  `json:"poster_id"`
	Product  string  `json:"product"`
	Count    float64 `json:"count"`
}

type Trail struct {
	ID      string `json:"id"`
	OrderID string `json:"order_id"`
	Actor   string `json:"actor,omitempty"`
	What    string `json:"what"`
	At      string `json:"at"`
}

type GroupOrderView struct {
	GroupOrder
	Total float64 `json:"total"`
	Count int     `json:"count"`
	Lines []Line  `json:"lines"`
}

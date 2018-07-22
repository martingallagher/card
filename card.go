package card

import (
	"github.com/cockroachdb/apd"
	"github.com/pkg/errors"
)

// Account request types.
const (
	Load Operation = iota
	Authorize
	Capture
	Reverse
	Refund
)

// Compile-time verification of Card interface implementation for the Account struct.
var _ Card = (*Account)(nil)

// Account method errors.
var (
	ErrUnderflow        = errors.New("requested amount exceeds available amount")
	ErrMerchantNotFound = errors.New("merchant record not found")
)

// Operation represents a transaction operation.
type Operation uint8

func (op Operation) String() string {
	switch op {
	case Load:
		return "LOAD"
	case Authorize:
		return "AUTHORIZE"
	case Capture:
		return "CAPTURE"
	case Reverse:
		return "REVERSE"
	case Refund:
		return "REFUND"
	}

	return "UNKNOWN"
}

// Card represents the prepaid card account interface.
type Card interface {
	Loader
	Authorizer
	Capturer
	Reverser
	Refunder
	Balancer
}

// Loader defines the account loader interface.
type Loader interface {
	Load(amount *apd.Decimal) error
}

// Authorizer defines the account authorization request interface.
type Authorizer interface {
	Authorize(merchantID int, amount *apd.Decimal) error
}

// Capturer defines the account loader interface.
type Capturer interface {
	Capture(merchantID int, amount *apd.Decimal) error
}

// Reverser defines the reverse authorization interface.
type Reverser interface {
	Reverse(merchantID int, amount *apd.Decimal) error
}

// Refunder defines the refund interface.
type Refunder interface {
	Refund(merchantID int, amount *apd.Decimal) error
}

// Balancer defines the account balance interface.
type Balancer interface {
	Balance() (*Balance, error)
}

// Account represents a prepaid card account.
type Account struct {
	ID           int               `json:"id"`
	Available    *apd.Decimal      `json:"available"`
	Blocked      *apd.Decimal      `json:"blocked"`
	Merchants    map[int]*Merchant `json:"merchants,omitempty"`
	Transactions []Transaction     `json:"transactions,omitempty"`
}

// Merchant represents a merchant.
type Merchant struct {
	Available *apd.Decimal `json:"available"`
	Captured  *apd.Decimal `json:"captured"`
}

// Transaction represents a prepaid card transaction.
type Transaction struct {
	Type       Operation    `json:"type"`
	MerchantID *int         `json:"merchantID,omitempty"`
	Amount     *apd.Decimal `json:"amount"`
}

// Balance represents a prepaid card balance.
type Balance struct {
	Total     *apd.Decimal
	Available *apd.Decimal
	Blocked   *apd.Decimal
}

// NewAccount returns a new account instance.
func NewAccount(id int) *Account {
	return &Account{
		ID:        id,
		Available: apd.New(0, 0),
		Blocked:   apd.New(0, 0),
	}
}

func getContext() *apd.Context {
	// Comply with GAAP decimal precision
	return apd.BaseContext.WithPrecision(16)
}

// Load loads the given amount to the account.
func (a *Account) Load(amount *apd.Decimal) error {
	_, err := getContext().Add(a.Available, a.Available, amount)

	if err != nil {
		return err
	}

	a.Transactions = append(a.Transactions, Transaction{Load, nil, amount})

	return err
}

// Authorize authorizes the given amount to the given merchant.
func (a *Account) Authorize(merchantID int, amount *apd.Decimal) error {
	if a.Available.Cmp(amount) < 0 {
		return ErrUnderflow
	}

	ctx := getContext()
	_, err := ctx.Sub(a.Available, a.Available, amount)

	if err != nil {
		return err
	}

	_, err = ctx.Add(a.Blocked, a.Blocked, amount)

	if err != nil {
		return err
	}

	m, exists := a.Merchants[merchantID]

	if !exists {
		if a.Merchants == nil {
			a.Merchants = map[int]*Merchant{}
		}

		a.Merchants[merchantID] = &Merchant{apd.New(0, 0), apd.New(0, 0)}
		m = a.Merchants[merchantID]
	}

	_, err = ctx.Add(m.Available, m.Available, amount)

	if err != nil {
		return err
	}

	a.Transactions = append(a.Transactions, Transaction{Authorize, &merchantID, amount})

	return err
}

// Capture captures the given amount for the given merchant.
func (a *Account) Capture(merchantID int, amount *apd.Decimal) error {
	m, exists := a.Merchants[merchantID]

	if !exists {
		return errors.Wrapf(ErrMerchantNotFound, "ID: %d", merchantID)
	}

	if m.Available.Cmp(amount) < 0 {
		return ErrUnderflow
	}

	ctx := getContext()
	_, err := ctx.Sub(m.Available, m.Available, amount)

	if err != nil {
		return err
	}

	_, err = ctx.Add(m.Captured, m.Captured, amount)

	if err != nil {
		return err
	}

	_, err = ctx.Sub(a.Blocked, a.Blocked, amount)

	if err != nil {
		return err
	}

	a.Transactions = append(a.Transactions, Transaction{Capture, &merchantID, amount})

	return nil
}

// Reverse reverses the given amount from the given merchant.
func (a *Account) Reverse(merchantID int, amount *apd.Decimal) error {
	m, exists := a.Merchants[merchantID]

	if !exists {
		return errors.Wrapf(ErrMerchantNotFound, "ID: %d", merchantID)
	}

	if m.Available.Cmp(amount) < 0 {
		return ErrUnderflow
	}

	ctx := getContext()
	_, err := ctx.Sub(m.Available, m.Available, amount)

	if err != nil {
		return err
	}

	_, err = ctx.Sub(a.Blocked, a.Blocked, amount)

	if err != nil {
		return err
	}

	_, err = ctx.Add(a.Available, a.Available, amount)

	if err != nil {
		return err
	}

	a.Transactions = append(a.Transactions, Transaction{Reverse, &merchantID, amount})

	return nil
}

// Refund refunds the given amount from the given merchant.
func (a *Account) Refund(merchantID int, amount *apd.Decimal) error {
	m, exists := a.Merchants[merchantID]

	if !exists {
		return errors.Wrapf(ErrMerchantNotFound, "ID: %d", merchantID)
	}

	if m.Captured.Cmp(amount) < 0 {
		return ErrUnderflow
	}

	ctx := getContext()
	_, err := ctx.Sub(m.Captured, m.Captured, amount)

	if err != nil {
		return err
	}

	_, err = ctx.Add(a.Available, a.Available, amount)

	if err != nil {
		return err
	}

	a.Transactions = append(a.Transactions, Transaction{Refund, &merchantID, amount})

	return nil
}

// Balance returns the account balance.
func (a *Account) Balance() (*Balance, error) {
	total := apd.New(0, 0)
	_, err := getContext().Add(total, a.Available, a.Blocked)

	if err != nil {
		return nil, err
	}

	return &Balance{
		Total:     total,
		Available: a.Available,
		Blocked:   a.Blocked,
	}, nil
}

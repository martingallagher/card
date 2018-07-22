package card_test

import (
	"testing"

	"github.com/cockroachdb/apd"
	. "github.com/martingallagher/card"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

const merchantID = 1

func decimalFromString(s string) *apd.Decimal {
	d, _, err := apd.NewFromString(s)

	if err != nil {
		panic(err)
	}

	return d
}

func TestLoad(t *testing.T) {
	account := NewAccount(0)
	zero := apd.New(0, 0)
	tests := []struct {
		amount *apd.Decimal
		total  *apd.Decimal
	}{
		{decimalFromString("10.5"), decimalFromString("10.5")},
		{decimalFromString("10.5"), decimalFromString("21.0")},
		{decimalFromString("33.33"), decimalFromString("54.33")},
	}

	for i, v := range tests {
		require.NoError(t, account.Load(v.amount))
		require.Len(t, account.Transactions, i+1)

		balance, err := account.Balance()

		require.NoError(t, err)
		require.Equal(t, v.total, balance.Total)
		require.Equal(t, v.total, balance.Available)
		require.Equal(t, zero, balance.Blocked)
	}
}

func TestAuthorize(t *testing.T) {
	account := NewAccount(0)

	t.Run("Load amount", func(t *testing.T) {
		require.NoError(t, account.Load(decimalFromString("112.34")))
		require.Len(t, account.Transactions, 1)
	})

	t.Run("Authorize £25.33", func(t *testing.T) {
		amount := decimalFromString("25.33")

		require.NoError(t, account.Authorize(merchantID, amount))

		balance, err := account.Balance()

		require.NoError(t, err)
		require.Equal(t, decimalFromString("87.01"), balance.Available)
		require.Equal(t, amount, balance.Blocked)
		require.Equal(t, amount, account.Merchants[merchantID].Available)
		require.Len(t, account.Transactions, 2)
	})

	t.Run("Authorize £5", func(t *testing.T) {
		require.NoError(t, account.Authorize(merchantID, apd.New(5, 0)))

		balance, err := account.Balance()

		require.NoError(t, err)
		require.Equal(t, decimalFromString("82.01"), balance.Available)

		expected := decimalFromString("30.33")

		require.Equal(t, expected, balance.Blocked)
		require.Equal(t, expected, account.Merchants[merchantID].Available)
		require.Len(t, account.Transactions, 3)
	})

	t.Run("Attempt to load amount exceeding available amount", func(t *testing.T) {
		require.Equal(t, ErrUnderflow, account.Authorize(merchantID, decimalFromString("82.02")))
		require.Len(t, account.Transactions, 3)
	})
}

func TestCapture(t *testing.T) {
	account := NewAccount(0)

	require.NoError(t, account.Load(apd.New(10, 0)))
	require.NoError(t, account.Authorize(merchantID, apd.New(2, 0)))

	t.Run("Capture £1", func(t *testing.T) {
		require.NoError(t, account.Capture(merchantID, apd.New(1, 0)))

		balance, err := account.Balance()

		require.NoError(t, err)
		require.Equal(t, apd.New(8, 0), balance.Available)
		require.Equal(t, apd.New(1, 0), balance.Blocked)
		require.Equal(t, apd.New(9, 0), balance.Total)
	})

	t.Run("Invalid merchant ID", func(t *testing.T) {
		require.Equal(t, ErrMerchantNotFound, errors.Cause(account.Capture(0, nil)))
	})

	t.Run("Attempt to capture amount exceeding merchant available amount", func(t *testing.T) {
		require.Equal(t, ErrUnderflow, account.Capture(merchantID, apd.New(2, 0)))
	})

	require.Len(t, account.Transactions, 3)
}

func loadAndAuthorize(t *testing.T, account *Account) {
	amount := decimalFromString("9999.99")

	require.NoError(t, account.Load(amount))

	authorize := decimalFromString("333.33")

	require.NoError(t, account.Authorize(merchantID, authorize))
	require.Equal(t, authorize, account.Merchants[merchantID].Available)

	balance, err := account.Balance()

	require.NoError(t, err)
	require.Equal(t, decimalFromString("9666.66"), balance.Available)
	require.Equal(t, authorize, balance.Blocked)
	require.Equal(t, amount, balance.Total)
}

func TestReverse(t *testing.T) {
	account := NewAccount(0)

	loadAndAuthorize(t, account)

	t.Run("Invalid merchant ID", func(t *testing.T) {
		require.Equal(t, ErrMerchantNotFound, errors.Cause(account.Reverse(0, nil)))
	})

	t.Run("Reverse £66.66", func(t *testing.T) {
		require.NoError(t, account.Reverse(merchantID, decimalFromString("66.66")))

		balance, err := account.Balance()

		require.NoError(t, err)
		require.Equal(t, decimalFromString("9733.32"), balance.Available)
		require.Equal(t, decimalFromString("266.67"), balance.Blocked)
	})

	t.Run("Attempt to reverse invalid sum", func(t *testing.T) {
		require.Equal(t, ErrUnderflow, account.Reverse(merchantID, decimalFromString("500.50")))
	})

	require.Len(t, account.Transactions, 3)
}

func TestRefund(t *testing.T) {
	account := NewAccount(0)

	loadAndAuthorize(t, account)

	t.Run("Invalid merchant ID", func(t *testing.T) {
		require.Equal(t, ErrMerchantNotFound, errors.Cause(account.Refund(0, nil)))
	})

	t.Run("Capture and refund", func(t *testing.T) {
		capture := decimalFromString("100.00")

		require.NoError(t, account.Capture(merchantID, capture))
		require.Equal(t, decimalFromString("233.33"), account.Merchants[merchantID].Available)
		require.Equal(t, capture, account.Merchants[merchantID].Captured)

		balance, err := account.Balance()

		require.NoError(t, err)
		require.Equal(t, decimalFromString("9666.66"), balance.Available)
		require.Equal(t, decimalFromString("233.33"), balance.Blocked)
		require.NoError(t, account.Refund(merchantID, decimalFromString("50")))

		balance, err = account.Balance()

		require.NoError(t, err)
		require.Equal(t, decimalFromString("9716.66"), balance.Available)
		require.Equal(t, decimalFromString("233.33"), balance.Blocked)
	})

	t.Run("Attempt to refund invalid amount", func(t *testing.T) {
		require.Equal(t, ErrUnderflow, account.Capture(merchantID, decimalFromString("233.34")))
	})

	require.Len(t, account.Transactions, 4)
}

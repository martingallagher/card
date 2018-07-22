package card_test

import (
	"testing"

	. "github.com/martingallagher/card"
	"github.com/stretchr/testify/require"
)

func TestStatement(t *testing.T) {
	account := NewAccount(0)

	require.NoError(t, account.Load(decimalFromString("915.75")))
	require.NoError(t, account.Authorize(1, decimalFromString("15.00")))
	require.NoError(t, account.Capture(1, decimalFromString("5")))
	require.NoError(t, account.Capture(1, decimalFromString("5")))
	require.NoError(t, account.Reverse(1, decimalFromString("2.5")))
	require.NoError(t, account.Refund(1, decimalFromString("10")))
	require.NoError(t, account.Capture(1, decimalFromString("2.5")))

	statement, err := account.Statement()

	require.NoError(t, err)

	const expected = `Available:                           913.25
Blocked:                               0.00
Total:                               913.25

-------------------------------------------
 ID     | Type      | Merchant | Amount
-------------------------------------------
 0      | LOAD      |          |    915.75
 1      | AUTHORIZE | 1        |     15.00
 2      | CAPTURE   | 1        |      5.00
 3      | CAPTURE   | 1        |      5.00
 4      | REVERSE   | 1        |      2.50
 5      | REFUND    | 1        |     10.00
 6      | CAPTURE   | 1        |      2.50
-------------------------------------------`

	require.Equal(t, expected, statement)
}

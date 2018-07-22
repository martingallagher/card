package card

import (
	"fmt"
	"strconv"
	"strings"
)

// Statement generates an account statement.
func (a *Account) Statement() (string, error) {
	balance, err := a.Balance()

	if err != nil {
		return "", err
	}

	available, err := balance.Available.Float64()

	if err != nil {
		return "", err
	}

	blocked, err := balance.Blocked.Float64()

	if err != nil {
		return "", err
	}

	total, err := balance.Total.Float64()

	if err != nil {
		return "", err
	}

	_ = balance

	var (
		sb   strings.Builder
		line = strings.Repeat("-", 43)
	)

	fmt.Fprintf(&sb, `Available: %32.2f
Blocked: %34.2f
Total: %36.2f

%[4]s
 ID     | Type      | Merchant | Amount
%[4]s`, available, blocked, total, line)

	if len(a.Transactions) == 0 {
		sb.WriteString("\n          *** NO TRANSACTIONS ***")

		return sb.String(), nil
	}

	sb.WriteByte('\n')

	for i, v := range a.Transactions {
		var merchant string

		if v.MerchantID != nil {
			merchant = strconv.Itoa(*v.MerchantID)
		}

		f, err := v.Amount.Float64()

		if err != nil {
			return "", err
		}

		fmt.Fprintf(&sb, " %-6d | %-9s | %-8s | %9.2f\n", i, v.Type, merchant, f)
	}

	sb.WriteString(line)

	return sb.String(), nil
}

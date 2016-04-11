package statement

import (
	"strconv"
	"time"
)

type Dollar struct {
	Cents    int64
	HasValue bool
}

func NewDollar(number float64) Dollar {
	return Dollar{int64(number*100 + 0.5), true}
}

func (d Dollar) MarshalJSON() ([]byte, error) {
	if !d.HasValue {
		return []byte("null"), nil
	}
	num := strconv.FormatFloat(float64(d.Cents)/100, 'f', 2, 64)
	return []byte(num), nil
}

type Statement struct {
	Entity               string
	ASXCode              string
	AccountHolders       []string
	PaymentDate          time.Time
	TotalPayment         Dollar
	FrankingCredit       Dollar
	UnfrankedAmount      Dollar
	FrankedAmount        Dollar
	WithholdingTax       Dollar
	SharesAllotted       int
	CostOfSharesAllotted Dollar
	TotalShares          int
}

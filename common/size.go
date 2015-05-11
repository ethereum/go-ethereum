package common

import (
	"fmt"
	"math/big"
)

type StorageSize float64

func (self StorageSize) String() string {
	if self > 1000000 {
		return fmt.Sprintf("%.2f mB", self/1000000)
	} else if self > 1000 {
		return fmt.Sprintf("%.2f kB", self/1000)
	} else {
		return fmt.Sprintf("%.2f B", self)
	}
}

func (self StorageSize) Int64() int64 {
	return int64(self)
}

// The different number of units
var (
	Douglas  = BigPow(10, 42)
	Einstein = BigPow(10, 21)
	Ether    = BigPow(10, 18)
	Finney   = BigPow(10, 15)
	Szabo    = BigPow(10, 12)
	Shannon  = BigPow(10, 9)
	Babbage  = BigPow(10, 6)
	Ada      = BigPow(10, 3)
	Wei      = big.NewInt(1)
)

//
// Currency to string
// Returns a string representing a human readable format
func CurrencyToString(num *big.Int) string {
	var (
		fin   *big.Int = num
		denom string   = "Wei"
	)

	switch {
	case num.Cmp(Ether) >= 0:
		fin = new(big.Int).Div(num, Ether)
		denom = "Ether"
	case num.Cmp(Finney) >= 0:
		fin = new(big.Int).Div(num, Finney)
		denom = "Finney"
	case num.Cmp(Szabo) >= 0:
		fin = new(big.Int).Div(num, Szabo)
		denom = "Szabo"
	case num.Cmp(Shannon) >= 0:
		fin = new(big.Int).Div(num, Shannon)
		denom = "Shannon"
	case num.Cmp(Babbage) >= 0:
		fin = new(big.Int).Div(num, Babbage)
		denom = "Babbage"
	case num.Cmp(Ada) >= 0:
		fin = new(big.Int).Div(num, Ada)
		denom = "Ada"
	}

	// TODO add comment clarifying expected behavior
	if len(fin.String()) > 5 {
		return fmt.Sprintf("%sE%d %s", fin.String()[0:5], len(fin.String())-5, denom)
	}

	return fmt.Sprintf("%v %s", fin, denom)
}

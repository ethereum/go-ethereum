package ethutil

import (
	"fmt"
	"math/big"
)

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
	switch {
	case num.Cmp(Douglas) >= 0:
		return fmt.Sprintf("%v Douglas", new(big.Int).Div(num, Douglas))
	case num.Cmp(Einstein) >= 0:
		return fmt.Sprintf("%v Einstein", new(big.Int).Div(num, Einstein))
	case num.Cmp(Ether) >= 0:
		return fmt.Sprintf("%v Ether", new(big.Int).Div(num, Ether))
	case num.Cmp(Finney) >= 0:
		return fmt.Sprintf("%v Finney", new(big.Int).Div(num, Finney))
	case num.Cmp(Szabo) >= 0:
		return fmt.Sprintf("%v Szabo", new(big.Int).Div(num, Szabo))
	case num.Cmp(Shannon) >= 0:
		return fmt.Sprintf("%v Shannon", new(big.Int).Div(num, Shannon))
	case num.Cmp(Babbage) >= 0:
		return fmt.Sprintf("%v Babbage", new(big.Int).Div(num, Babbage))
	case num.Cmp(Ada) >= 0:
		return fmt.Sprintf("%v Ada", new(big.Int).Div(num, Ada))
	}

	return fmt.Sprintf("%v Wei", num)
}

// Common big integers often used
var (
	Big1   = big.NewInt(1)
	Big2   = big.NewInt(1)
	Big0   = big.NewInt(0)
	Big32  = big.NewInt(32)
	Big256 = big.NewInt(0xff)
)

// Creates an ethereum address given the bytes and the nonce
func CreateAddress(b []byte, nonce *big.Int) []byte {
	addrBytes := append(b, nonce.Bytes()...)

	return Sha3Bin(addrBytes)[12:]
}

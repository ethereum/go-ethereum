package ethutil

import (
	"fmt"
	"math/big"
)

var (
	Ether  = BigPow(10, 18)
	Finney = BigPow(10, 15)
	Szabo  = BigPow(10, 12)
	Vito   = BigPow(10, 9)
	Turing = BigPow(10, 6)
	Eins   = BigPow(10, 3)
	Wei    = big.NewInt(1)
)

func CurrencyToString(num *big.Int) string {
	switch {
	case num.Cmp(Ether) >= 0:
		return fmt.Sprintf("%v Ether", new(big.Int).Div(num, Ether))
	case num.Cmp(Finney) >= 0:
		return fmt.Sprintf("%v Finney", new(big.Int).Div(num, Finney))
	case num.Cmp(Szabo) >= 0:
		return fmt.Sprintf("%v Szabo", new(big.Int).Div(num, Szabo))
	case num.Cmp(Vito) >= 0:
		return fmt.Sprintf("%v Vito", new(big.Int).Div(num, Vito))
	case num.Cmp(Turing) >= 0:
		return fmt.Sprintf("%v Turing", new(big.Int).Div(num, Turing))
	case num.Cmp(Eins) >= 0:
		return fmt.Sprintf("%v Eins", new(big.Int).Div(num, Eins))
	}

	return fmt.Sprintf("%v Wei", num)
}

var (
	Big1   = big.NewInt(1)
	Big2   = big.NewInt(1)
	Big0   = big.NewInt(0)
	Big32  = big.NewInt(32)
	Big256 = big.NewInt(0xff)
)

func CreateAddress(b []byte, nonce *big.Int) []byte {
	addrBytes := append(b, nonce.Bytes()...)

	return Sha3Bin(addrBytes)[12:]
}

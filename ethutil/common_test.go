package ethutil

import (
	"fmt"
	"math/big"
	"testing"
)

func TestCommon(t *testing.T) {
	fmt.Println(CurrencyToString(BigPow(10, 19)))
	fmt.Println(CurrencyToString(BigPow(10, 16)))
	fmt.Println(CurrencyToString(BigPow(10, 13)))
	fmt.Println(CurrencyToString(BigPow(10, 10)))
	fmt.Println(CurrencyToString(BigPow(10, 7)))
	fmt.Println(CurrencyToString(BigPow(10, 4)))
	fmt.Println(CurrencyToString(big.NewInt(10)))
}

package ethutil

import (
	"fmt"
	"math/big"
	"runtime"
	"time"
)

func IsWindows() bool {
	return runtime.GOOS == "windows"
}

func WindonizePath(path string) string {
	if string(path[0]) == "/" && IsWindows() {
		path = path[1:]
	}
	return path
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
	case num.Cmp(Douglas) >= 0:
		fin = new(big.Int).Div(num, Douglas)
		denom = "Douglas"
	case num.Cmp(Einstein) >= 0:
		fin = new(big.Int).Div(num, Einstein)
		denom = "Einstein"
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

// Common big integers often used
var (
	Big1     = big.NewInt(1)
	Big2     = big.NewInt(2)
	Big3     = big.NewInt(3)
	Big0     = big.NewInt(0)
	BigTrue  = Big1
	BigFalse = Big0
	Big32    = big.NewInt(32)
	Big256   = big.NewInt(0xff)
	Big257   = big.NewInt(257)
)

func Bench(pre string, cb func()) {
	start := time.Now()
	cb()
	fmt.Println(pre, ": took:", time.Since(start))
}

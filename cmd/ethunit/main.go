// Copyright 2025 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// ethunit converts values between Ethereum denominations (wei, gwei, ether).
package main

import (
	"flag"
	"fmt"
	"math/big"
	"os"
	"strings"
)

var (
	fromUnit = flag.String("from", "wei", "source denomination (wei, gwei, ether)")
	toUnit   = flag.String("to", "ether", "target denomination (wei, gwei, ether)")
)

// unitMultipliers maps denomination names to their wei multipliers.
var unitMultipliers = map[string]*big.Int{
	"wei":   new(big.Int).SetUint64(1),
	"gwei":  new(big.Int).SetUint64(1e9),
	"ether": new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
}

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "[-from unit] [-to unit] <value>")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, `
Converts a value between Ethereum denominations.

Supported units: wei, gwei, ether

Examples:
  ethunit -from wei -to ether 1000000000000000000
  ethunit -from ether -to gwei 1.5
  ethunit -from gwei -to wei 100`)
	}
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	result, err := convert(flag.Arg(0), *fromUnit, *toUnit)
	if err != nil {
		die(err)
	}
	fmt.Println(result)
}

// convert converts a decimal string value from one denomination to another.
func convert(value, from, to string) (string, error) {
	from = strings.ToLower(from)
	to = strings.ToLower(to)

	fromMul, ok := unitMultipliers[from]
	if !ok {
		return "", fmt.Errorf("unknown source unit %q", from)
	}
	toMul, ok := unitMultipliers[to]
	if !ok {
		return "", fmt.Errorf("unknown target unit %q", to)
	}

	// Parse input as a decimal number. We support both integer and fractional
	// values by splitting on the decimal point and scaling to wei.
	weiValue, err := parseToWei(value, fromMul)
	if err != nil {
		return "", err
	}

	return formatFromWei(weiValue, toMul), nil
}

// parseToWei parses a decimal string and returns the value in wei.
func parseToWei(s string, multiplier *big.Int) (*big.Int, error) {
	parts := strings.SplitN(s, ".", 2)
	if len(parts) == 1 {
		// Integer value.
		intPart, ok := new(big.Int).SetString(parts[0], 10)
		if !ok {
			return nil, fmt.Errorf("invalid number %q", s)
		}
		return intPart.Mul(intPart, multiplier), nil
	}
	// Fractional value: combine integer and fractional parts into a single
	// integer, then divide out the fractional scaling.
	intPart, ok := new(big.Int).SetString(parts[0], 10)
	if !ok {
		return nil, fmt.Errorf("invalid number %q", s)
	}
	fracStr := parts[1]
	// Remove trailing zeros for cleanliness, but track original length for scale.
	fracLen := len(fracStr)
	fracPart, ok := new(big.Int).SetString(fracStr, 10)
	if !ok {
		return nil, fmt.Errorf("invalid number %q", s)
	}
	// scale = 10^fracLen
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(fracLen)), nil)

	// weiValue = intPart * multiplier + fracPart * multiplier / scale
	weiFromInt := new(big.Int).Mul(intPart, multiplier)
	weiFromFrac := new(big.Int).Mul(fracPart, multiplier)
	weiFromFrac.Div(weiFromFrac, scale)

	return weiFromInt.Add(weiFromInt, weiFromFrac), nil
}

// formatFromWei formats a wei value into the target denomination.
func formatFromWei(wei *big.Int, divisor *big.Int) string {
	if divisor.Cmp(big.NewInt(1)) == 0 {
		return wei.String()
	}
	whole := new(big.Int).Div(wei, divisor)
	remainder := new(big.Int).Mod(wei, divisor)

	if remainder.Sign() == 0 {
		return whole.String()
	}
	// Format remainder with leading zeros to match divisor magnitude.
	divisorStr := divisor.String()
	fracStr := fmt.Sprintf("%0*s", len(divisorStr)-1, remainder.String())
	fracStr = strings.TrimRight(fracStr, "0")
	return whole.String() + "." + fracStr
}

func die(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

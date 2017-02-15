// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Spec at https://github.com/ethereum/wiki/wiki/ICAP:-Inter-exchange-Client-Address-Protocol

package common

import (
	"errors"
	"math/big"
	"strconv"
	"strings"
)

var (
	Base36Chars          = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	ICAPLengthError      = errors.New("Invalid ICAP length")
	ICAPEncodingError    = errors.New("Invalid ICAP encoding")
	ICAPChecksumError    = errors.New("Invalid ICAP checksum")
	ICAPCountryCodeError = errors.New("Invalid ICAP country code")
	ICAPAssetIdentError  = errors.New("Invalid ICAP asset identifier")
	ICAPInstCodeError    = errors.New("Invalid ICAP institution code")
	ICAPClientIdentError = errors.New("Invalid ICAP client identifier")
)

func ICAPToAddress(s string) (Address, error) {
	switch len(s) {
	case 35: // "XE" + 2 digit checksum + 31 base-36 chars of address
		return parseICAP(s)
	case 34: // "XE" + 2 digit checksum + 30 base-36 chars of address
		return parseICAP(s)
	case 20: // "XE" + 2 digit checksum + 3-char asset identifier +
		// 4-char institution identifier + 9-char institution client identifier
		return parseIndirectICAP(s)
	default:
		return Address{}, ICAPLengthError
	}
}

func parseICAP(s string) (Address, error) {
	if !strings.HasPrefix(s, "XE") {
		return Address{}, ICAPCountryCodeError
	}
	if err := validCheckSum(s); err != nil {
		return Address{}, err
	}
	// checksum is ISO13616, Ethereum address is base-36
	bigAddr, _ := new(big.Int).SetString(s[4:], 36)
	return BigToAddress(bigAddr), nil
}

func parseIndirectICAP(s string) (Address, error) {
	if !strings.HasPrefix(s, "XE") {
		return Address{}, ICAPCountryCodeError
	}
	if s[4:7] != "ETH" {
		return Address{}, ICAPAssetIdentError
	}
	if err := validCheckSum(s); err != nil {
		return Address{}, err
	}
	// TODO: integrate with ICAP namereg
	return Address{}, errors.New("not implemented")
}

func AddressToICAP(a Address) (string, error) {
	enc := base36Encode(a.Big())
	// zero padd encoded address to Direct ICAP length if needed
	if len(enc) < 30 {
		enc = join(strings.Repeat("0", 30-len(enc)), enc)
	}
	icap := join("XE", checkDigits(enc), enc)
	return icap, nil
}

// TODO: integrate with ICAP namereg when it's available
func AddressToIndirectICAP(a Address, instCode string) (string, error) {
	// return addressToIndirectICAP(a, instCode)
	return "", errors.New("not implemented")
}

func addressToIndirectICAP(a Address, instCode string) (string, error) {
	// TODO: add addressToClientIdent which grabs client ident from ICAP namereg
	//clientIdent := addressToClientIdent(a)
	clientIdent := "todo"
	return clientIdentToIndirectICAP(instCode, clientIdent)
}

func clientIdentToIndirectICAP(instCode, clientIdent string) (string, error) {
	if len(instCode) != 4 || !validBase36(instCode) {
		return "", ICAPInstCodeError
	}
	if len(clientIdent) != 9 || !validBase36(instCode) {
		return "", ICAPClientIdentError
	}

	// currently ETH is only valid asset identifier
	s := join("ETH", instCode, clientIdent)
	return join("XE", checkDigits(s), s), nil
}

// https://en.wikipedia.org/wiki/International_Bank_Account_Number#Validating_the_IBAN
func validCheckSum(s string) error {
	s = join(s[4:], s[:4])
	expanded, err := iso13616Expand(s)
	if err != nil {
		return err
	}
	checkSumNum, _ := new(big.Int).SetString(expanded, 10)
	if checkSumNum.Mod(checkSumNum, Big97).Cmp(Big1) != 0 {
		return ICAPChecksumError
	}
	return nil
}

func checkDigits(s string) string {
	expanded, _ := iso13616Expand(strings.Join([]string{s, "XE00"}, ""))
	num, _ := new(big.Int).SetString(expanded, 10)
	num.Sub(Big98, num.Mod(num, Big97))

	checkDigits := num.String()
	// zero padd checksum
	if len(checkDigits) == 1 {
		checkDigits = join("0", checkDigits)
	}
	return checkDigits
}

// not base-36, but expansion to decimal literal: A = 10, B = 11, ... Z = 35
func iso13616Expand(s string) (string, error) {
	var parts []string
	if !validBase36(s) {
		return "", ICAPEncodingError
	}
	for _, c := range s {
		i := uint64(c)
		if i >= 65 {
			parts = append(parts, strconv.FormatUint(uint64(c)-55, 10))
		} else {
			parts = append(parts, string(c))
		}
	}
	return join(parts...), nil
}

func base36Encode(i *big.Int) string {
	var chars []rune
	x := new(big.Int)
	for {
		x.Mod(i, Big36)
		chars = append(chars, rune(Base36Chars[x.Uint64()]))
		i.Div(i, Big36)
		if i.Cmp(Big0) == 0 {
			break
		}
	}
	// reverse slice
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
	return string(chars)
}

func validBase36(s string) bool {
	for _, c := range s {
		i := uint64(c)
		// 0-9 or A-Z
		if i < 48 || (i > 57 && i < 65) || i > 90 {
			return false
		}
	}
	return true
}

func join(s ...string) string {
	return strings.Join(s, "")
}

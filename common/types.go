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

package common

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"strconv"

	"github.com/XinFinOrg/XDPoSChain/common/hexutil"
	"golang.org/x/crypto/sha3"
)

const (
	HashLength                       = 32
	AddressLength                    = 20
	BlockSigners                     = "xdc0000000000000000000000000000000000000089"
	MasternodeVotingSMC              = "xdc0000000000000000000000000000000000000088"
	RandomizeSMC                     = "xdc0000000000000000000000000000000000000090"
	FoudationAddr                    = "xdc0000000000000000000000000000000000000068"
	TeamAddr                         = "xdc0000000000000000000000000000000000000099"
	XDCXAddr                         = "xdc0000000000000000000000000000000000000091"
	TradingStateAddr                 = "xdc0000000000000000000000000000000000000092"
	XDCXLendingAddress               = "xdc0000000000000000000000000000000000000093"
	XDCXLendingFinalizedTradeAddress = "xdc0000000000000000000000000000000000000094"
	XDCNativeAddress                 = "xdc0000000000000000000000000000000000000001"
	LendingLockAddress               = "xdc0000000000000000000000000000000000000011"
	VoteMethod                       = "0x6dd7d8ea"
	UnvoteMethod                     = "0x02aa9be2"
	ProposeMethod                    = "0x01267951"
	ResignMethod                     = "0xae6e43f5"
	SignMethod                       = "0xe341eaa4"
	XDCXApplyMethod                  = "0xc6b32f34"
	XDCZApplyMethod                  = "0xc6b32f34"
)

var (
	// MaxHash represents the maximum possible hash value.
	MaxHash = HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	BlockSignersBinary                     = HexToAddress("0x0000000000000000000000000000000000000089")
	MasternodeVotingSMCBinary              = HexToAddress("0x0000000000000000000000000000000000000088")
	RandomizeSMCBinary                     = HexToAddress("0x0000000000000000000000000000000000000090")
	FoudationAddrBinary                    = HexToAddress("0x0000000000000000000000000000000000000068")
	TeamAddrBinary                         = HexToAddress("0x0000000000000000000000000000000000000099")
	XDCXAddrBinary                         = HexToAddress("0x0000000000000000000000000000000000000091")
	TradingStateAddrBinary                 = HexToAddress("0x0000000000000000000000000000000000000092")
	XDCXLendingAddressBinary               = HexToAddress("0x0000000000000000000000000000000000000093")
	XDCXLendingFinalizedTradeAddressBinary = HexToAddress("0x0000000000000000000000000000000000000094")
	XDCNativeAddressBinary                 = HexToAddress("0x0000000000000000000000000000000000000001")
	LendingLockAddressBinary               = HexToAddress("0x0000000000000000000000000000000000000011")
)

var (
	hashT    = reflect.TypeOf(Hash{})
	addressT = reflect.TypeOf(Address{})
)

// Hash represents the 32 byte Keccak256 hash of arbitrary data.
type Hash [HashLength]byte

type Vote struct {
	Masternode Address
	Voter      Address
}

// BytesToHash sets b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

func StringToHash(s string) Hash { return BytesToHash([]byte(s)) }

// BigToHash sets byte representation of b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BigToHash(b *big.Int) Hash { return BytesToHash(b.Bytes()) }

func Uint64ToHash(b uint64) Hash { return BytesToHash(new(big.Int).SetUint64(b).Bytes()) }

// HexToHash sets byte representation of s to hash.
// If b is larger than len(h), b will be cropped from the left.
func HexToHash(s string) Hash { return BytesToHash(FromHex(s)) }

// Cmp compares two hashes.
func (h Hash) Cmp(other Hash) int {
	return bytes.Compare(h[:], other[:])
}

// IsZero returns if a Hash is empty
func (h Hash) IsZero() bool { return h == Hash{} }

// Str get the string representation of the underlying hash
func (h Hash) Str() string { return string(h[:]) }

// Bytes gets the byte representation of the underlying hash.
func (h Hash) Bytes() []byte { return h[:] }

// Big converts a hash to a big integer.
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }

// Hex converts a hash to a hex string.
func (h Hash) Hex() string { return hexutil.Encode(h[:]) }

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (h Hash) TerminalString() string {
	return fmt.Sprintf("%x..%x", h[:3], h[29:])
}

// String implements the stringer interface and is used also by the logger when
// doing full logging into a file.
func (h Hash) String() string {
	return h.Hex()
}

// Format implements fmt.Formatter.
// Hash supports the %v, %s, %q, %x, %X and %d format verbs.
func (h Hash) Format(s fmt.State, c rune) {
	hexb := make([]byte, 2+len(h)*2)
	copy(hexb, "0x")
	hex.Encode(hexb[2:], h[:])

	switch c {
	case 'x', 'X':
		if !s.Flag('#') {
			hexb = hexb[2:]
		}
		if c == 'X' {
			hexb = bytes.ToUpper(hexb)
		}
		fallthrough
	case 'v', 's':
		s.Write(hexb)
	case 'q':
		q := []byte{'"'}
		s.Write(q)
		s.Write(hexb)
		s.Write(q)
	case 'd':
		fmt.Fprint(s, ([len(h)]byte)(h))
	default:
		fmt.Fprintf(s, "%%!%c(hash=%x)", c, h)
	}
}

// UnmarshalText parses a hash in hex syntax.
func (h *Hash) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Hash", input, h[:])
}

// UnmarshalJSON parses a hash in hex syntax.
func (h *Hash) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(hashT, input, h[:])
}

// MarshalText returns the hex representation of h.
func (h Hash) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

// Generate implements testing/quick.Generator.
func (h Hash) Generate(rand *rand.Rand, size int) reflect.Value {
	m := rand.Intn(len(h))
	for i := len(h) - 1; i > m; i-- {
		h[i] = byte(rand.Uint32())
	}
	return reflect.ValueOf(h)
}

// UnprefixedHash allows marshaling a Hash without 0x prefix.
type UnprefixedHash Hash

// UnmarshalText decodes the hash from hex. The 0x prefix is optional.
func (h *UnprefixedHash) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedUnprefixedText("UnprefixedHash", input, h[:])
}

// MarshalText encodes the hash as hex.
func (h UnprefixedHash) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h[:])), nil
}

/////////// Address

// Address represents the 20 byte address of an Ethereum account.
type Address [AddressLength]byte

// BytesToAddress returns Address with value b.
// If b is larger than len(h), b will be cropped from the left.
func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

func StringToAddress(s string) Address { return BytesToAddress([]byte(s)) }

// BigToAddress returns Address with byte values of b.
// If b is larger than len(h), b will be cropped from the left.
func BigToAddress(b *big.Int) Address { return BytesToAddress(b.Bytes()) }

// HexToAddress returns Address with byte values of s.
// If s is larger than len(h), s will be cropped from the left.
func HexToAddress(s string) Address { return BytesToAddress(FromHex(s)) }

// IsHexAddress verifies whether a string can represent a valid hex-encoded
// Ethereum address or not.
func IsHexAddress(s string) bool {
	if has0xPrefix(s) {
		s = s[2:]
	} else if hasXdcPrefix(s) {
		s = s[3:]
	}
	return len(s) == 2*AddressLength && isHex(s)
}

// IsZero returns if a address is empty
func (a Address) IsZero() bool { return a == Address{} }

// Str gets the string representation of the underlying address
func (a Address) Str() string { return string(a[:]) }

// Bytes gets the string representation of the underlying address.
func (a Address) Bytes() []byte { return a[:] }

// Hash converts an address to a hash by left-padding it with zeros.
func (a Address) Hash() Hash { return BytesToHash(a[:]) }

// Big converts an address to a big integer.
func (a Address) Big() *big.Int { return new(big.Int).SetBytes(a[:]) }

// Hex returns an EIP55-compliant hex string representation of the address.
func (a Address) Hex() string {
	checksumed := a.checksumHex()
	return "xdc" + string(checksumed[2:])
}

// String implements fmt.Stringer.
func (a Address) String() string {
	return a.Hex()
}

func (a *Address) checksumHex() []byte {
	buf := a.hex()

	// compute checksum
	sha := sha3.NewLegacyKeccak256()
	sha.Write(buf[2:])
	hash := sha.Sum(nil)
	for i := 2; i < len(buf); i++ {
		hashByte := hash[(i-2)/2]
		if i%2 == 0 {
			hashByte = hashByte >> 4
		} else {
			hashByte &= 0xf
		}
		if buf[i] > '9' && hashByte > 7 {
			buf[i] -= 32
		}
	}
	return buf[:]
}

func (a Address) hex() []byte {
	var buf [len(a)*2 + 2]byte
	copy(buf[:2], "0x")
	hex.Encode(buf[2:], a[:])
	return buf[:]
}

// Format implements fmt.Formatter.
// Address supports the %v, %s, %q, %x, %X and %d format verbs.
func (a Address) Format(s fmt.State, c rune) {
	switch c {
	case 'v', 's':
		s.Write(a.checksumHex())
	case 'q':
		q := []byte{'"'}
		s.Write(q)
		s.Write(a.checksumHex())
		s.Write(q)
	case 'x', 'X':
		// %x disables the checksum.
		hex := a.hex()
		if !s.Flag('#') {
			hex = hex[2:]
		}
		if c == 'X' {
			hex = bytes.ToUpper(hex)
		}
		s.Write(hex)
	case 'd':
		fmt.Fprint(s, ([len(a)]byte)(a))
	default:
		fmt.Fprintf(s, "%%!%c(address=%x)", c, a)
	}
}

// SetBytes sets the address to the value of b.
// If b is larger than len(a) it will panic.
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}

// MarshalText returns the hex representation of a.
func (a Address) MarshalText() ([]byte, error) {
	// Handle '0x' or 'xdc' prefix here.
	if Enable0xPrefix {
		return hexutil.Bytes(a[:]).MarshalText()
	} else {
		return hexutil.Bytes(a[:]).MarshalXDCText()
	}
}

// UnmarshalText parses a hash in hex syntax.
func (a *Address) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Address", input, a[:])
}

// UnmarshalJSON parses a hash in hex syntax.
func (a *Address) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(addressT, input, a[:])
}

// UnprefixedAddress allows marshaling an Address without 0x prefix.
type UnprefixedAddress Address

// UnmarshalText decodes the address from hex. The 0x prefix is optional.
func (a *UnprefixedAddress) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedUnprefixedText("UnprefixedAddress", input, a[:])
}

// MarshalText encodes the address as hex.
func (a UnprefixedAddress) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(a[:])), nil
}

// RemoveItemFromArray extracts validators from byte array.
func RemoveItemFromArray(array []Address, items []Address) []Address {
	// Create newArray to stop append change array value
	newArray := make([]Address, len(array))
	copy(newArray, array)

	if len(items) == 0 {
		return newArray
	}

	for _, item := range items {
		for i := len(newArray) - 1; i >= 0; i-- {
			if newArray[i] == item {
				newArray = append(newArray[:i], newArray[i+1:]...)
			}
		}
	}

	return newArray
}

// ExtractAddressToBytes extracts validators from byte array.
func ExtractAddressToBytes(penalties []Address) []byte {
	data := []byte{}
	for _, signer := range penalties {
		data = append(data, signer[:]...)
	}
	return data
}

func ExtractAddressFromBytes(bytePenalties []byte) []Address {
	if bytePenalties != nil && len(bytePenalties) < AddressLength {
		return []Address{}
	}
	penalties := make([]Address, len(bytePenalties)/AddressLength)
	for i := 0; i < len(penalties); i++ {
		copy(penalties[i][:], bytePenalties[i*AddressLength:])
	}
	return penalties
}

// AddressEIP55 is an alias of Address with a customized json marshaller
type AddressEIP55 Address

// String returns the hex representation of the address in the manner of EIP55.
func (addr AddressEIP55) String() string {
	return Address(addr).Hex()
}

// MarshalJSON marshals the address in the manner of EIP55.
func (addr AddressEIP55) MarshalJSON() ([]byte, error) {
	return json.Marshal(addr.String())
}

type Decimal uint64

func isString(input []byte) bool {
	return len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"'
}

// UnmarshalJSON parses a hash in hex syntax.
func (d *Decimal) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return &json.UnmarshalTypeError{Value: "non-string", Type: reflect.TypeOf(uint64(0))}
	}
	if i, err := strconv.ParseUint(string(input[1:len(input)-1]), 10, 64); err == nil {
		*d = Decimal(i)
		return nil
	} else {
		return err
	}
}

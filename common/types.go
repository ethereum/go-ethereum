package common

import "math/big"

const (
	hashLength    = 32
	addressLength = 20
)

type (
	Hash    [hashLength]byte
	Address [addressLength]byte
)

var (
	zeroHash    Hash
	zeroAddress Address
)

func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}
func StringToHash(s string) Hash { return BytesToHash([]byte(s)) }
func BigToHash(b *big.Int) Hash  { return BytesToHash(b.Bytes()) }

// Don't use the default 'String' method in case we want to overwrite

// Get the string representation of the underlying hash
func (h Hash) Str() string   { return string(h[:]) }
func (h Hash) Bytes() []byte { return h[:] }
func (h Hash) Big() *big.Int { return Bytes2Big(h[:]) }

// Sets the hash to the value of b. If b is larger than len(h) it will panic
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-hashLength:]
	}

	// reverse loop
	for i := len(b) - 1; i >= 0; i-- {
		h[i] = b[i]
	}
}

// Set string `s` to h. If s is larger than len(h) it will panic
func (h *Hash) SetString(s string) { h.SetBytes([]byte(s)) }

// Sets h to other
func (h *Hash) Set(other Hash) {
	for i, v := range other {
		h[i] = v
	}
}

/////////// Address
func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}
func StringToAddress(s string) Address { return BytesToAddress([]byte(s)) }
func BigToAddress(b *big.Int) Address  { return BytesToAddress(b.Bytes()) }

// Get the string representation of the underlying address
func (a Address) Str() string   { return string(a[:]) }
func (a Address) Bytes() []byte { return a[:] }
func (a Address) Big() *big.Int { return Bytes2Big(a[:]) }

// Sets the address to the value of b. If b is larger than len(a) it will panic
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-addressLength:]
	}

	// reverse loop
	for i := len(b) - 1; i >= 0; i-- {
		a[i] = b[i]
	}
}

// Set string `s` to a. If s is larger than len(a) it will panic
func (a *Address) SetString(s string) { a.SetBytes([]byte(s)) }

// Sets a to other
func (a *Address) Set(other Address) {
	for i, v := range other {
		a[i] = v
	}
}

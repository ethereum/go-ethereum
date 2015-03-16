package common

type (
	Hash    [32]byte
	Address [20]byte
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

func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}
func StringToAddress(s string) Address { return BytesToAddress([]byte(s)) }

// Don't use the default 'String' method in case we want to overwrite

// Get the string representation of the underlying hash
func (h Hash) Str() string {
	return string(h[:])
}

// Sets the hash to the value of b. If b is larger than len(h) it will panic
func (h Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		panic("unable to set bytes. too big")
	}

	// reverse loop
	for i := len(b); i >= 0; i-- {
		h[i] = b[i]
	}
}

// Set string `s` to h. If s is larger than len(h) it will panic
func (h Hash) SetString(s string) { h.SetBytes([]byte(s)) }

// Sets h to other
func (h Hash) Set(other Hash) {
	for i, v := range other {
		h[i] = v
	}
}

// Get the string representation of the underlying address
func (a Address) Str() string {
	return string(a[:])
}

// Sets the address to the value of b. If b is larger than len(a) it will panic
func (a Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		panic("unable to set bytes. too big")
	}

	// reverse loop
	for i := len(b); i >= 0; i-- {
		a[i] = b[i]
	}
}

// Set string `s` to a. If s is larger than len(a) it will panic
func (a Address) SetString(s string) { a.SetBytes([]byte(s)) }

// Sets a to other
func (a Address) Set(other Address) {
	for i, v := range other {
		a[i] = v
	}
}

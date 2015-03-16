package common

type (
	Hash    [32]byte
	Address [20]byte
)

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

func (h Hash) SetString(s string) { h.SetBytes([]byte(s)) }

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
func (a Address) SetString(s string) { h.SetBytes([]byte(a)) }

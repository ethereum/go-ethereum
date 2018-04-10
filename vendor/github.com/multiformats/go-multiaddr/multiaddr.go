package multiaddr

import (
	"bytes"
	"fmt"
	"log"
	"strings"
)

// multiaddr is the data structure representing a Multiaddr
type multiaddr struct {
	bytes []byte
}

// NewMultiaddr parses and validates an input string, returning a *Multiaddr
func NewMultiaddr(s string) (a Multiaddr, err error) {
	defer func() {
		if e := recover(); e != nil {
			log.Printf("Panic in NewMultiaddr on input %q: %s", s, e)
			err = fmt.Errorf("%v", e)
		}
	}()
	b, err := stringToBytes(s)
	if err != nil {
		return nil, err
	}
	return &multiaddr{bytes: b}, nil
}

// NewMultiaddrBytes initializes a Multiaddr from a byte representation.
// It validates it as an input string.
func NewMultiaddrBytes(b []byte) (a Multiaddr, err error) {
	defer func() {
		if e := recover(); e != nil {
			log.Printf("Panic in NewMultiaddrBytes on input %q: %s", b, e)
			err = fmt.Errorf("%v", e)
		}
	}()

	if err := validateBytes(b); err != nil {
		return nil, err
	}

	return &multiaddr{bytes: b}, nil
}

// Equal tests whether two multiaddrs are equal
func (m *multiaddr) Equal(m2 Multiaddr) bool {
	return bytes.Equal(m.bytes, m2.Bytes())
}

// Bytes returns the []byte representation of this Multiaddr
func (m *multiaddr) Bytes() []byte {
	// consider returning copy to prevent changing underneath us?
	cpy := make([]byte, len(m.bytes))
	copy(cpy, m.bytes)
	return cpy
}

// String returns the string representation of a Multiaddr
func (m *multiaddr) String() string {
	s, err := bytesToString(m.bytes)
	if err != nil {
		panic("multiaddr failed to convert back to string. corrupted?")
	}
	return s
}

// Protocols returns the list of protocols this Multiaddr has.
// will panic in case we access bytes incorrectly.
func (m *multiaddr) Protocols() []Protocol {
	ps := make([]Protocol, 0, 8)
	b := m.bytes
	for len(b) > 0 {
		code, n, err := ReadVarintCode(b)
		if err != nil {
			panic(err)
		}

		p := ProtocolWithCode(code)
		if p.Code == 0 {
			// this is a panic (and not returning err) because this should've been
			// caught on constructing the Multiaddr
			panic(fmt.Errorf("no protocol with code %d", b[0]))
		}
		ps = append(ps, p)
		b = b[n:]

		size, err := sizeForAddr(p, b)
		if err != nil {
			panic(err)
		}

		b = b[size:]
	}
	return ps
}

// Encapsulate wraps a given Multiaddr, returning the resulting joined Multiaddr
func (m *multiaddr) Encapsulate(o Multiaddr) Multiaddr {
	mb := m.bytes
	ob := o.Bytes()

	b := make([]byte, len(mb)+len(ob))
	copy(b, mb)
	copy(b[len(mb):], ob)
	return &multiaddr{bytes: b}
}

// Decapsulate unwraps Multiaddr up until the given Multiaddr is found.
func (m *multiaddr) Decapsulate(o Multiaddr) Multiaddr {
	s1 := m.String()
	s2 := o.String()
	i := strings.LastIndex(s1, s2)
	if i < 0 {
		// if multiaddr not contained, returns a copy.
		cpy := make([]byte, len(m.bytes))
		copy(cpy, m.bytes)
		return &multiaddr{bytes: cpy}
	}

	ma, err := NewMultiaddr(s1[:i])
	if err != nil {
		panic("Multiaddr.Decapsulate incorrect byte boundaries.")
	}
	return ma
}

var ErrProtocolNotFound = fmt.Errorf("protocol not found in multiaddr")

func (m *multiaddr) ValueForProtocol(code int) (string, error) {
	for _, sub := range Split(m) {
		p := sub.Protocols()[0]
		if p.Code == code {
			if p.Size == 0 {
				return "", nil
			}
			return strings.SplitN(sub.String(), "/", 3)[2], nil
		}
	}

	return "", ErrProtocolNotFound
}

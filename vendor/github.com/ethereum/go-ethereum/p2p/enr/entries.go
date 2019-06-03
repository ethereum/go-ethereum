// Copyright 2017 The go-ethereum Authors
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

package enr

import (
	"fmt"
	"io"
	"net"

	"github.com/ethereum/go-ethereum/rlp"
)

// Entry is implemented by known node record entry types.
//
// To define a new entry that is to be included in a node record,
// create a Go type that satisfies this interface. The type should
// also implement rlp.Decoder if additional checks are needed on the value.
type Entry interface {
	ENRKey() string
}

type generic struct {
	key   string
	value interface{}
}

func (g generic) ENRKey() string { return g.key }

func (g generic) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, g.value)
}

func (g *generic) DecodeRLP(s *rlp.Stream) error {
	return s.Decode(g.value)
}

// WithEntry wraps any value with a key name. It can be used to set and load arbitrary values
// in a record. The value v must be supported by rlp. To use WithEntry with Load, the value
// must be a pointer.
func WithEntry(k string, v interface{}) Entry {
	return &generic{key: k, value: v}
}

// TCP is the "tcp" key, which holds the TCP port of the node.
type TCP uint16

func (v TCP) ENRKey() string { return "tcp" }

// UDP is the "udp" key, which holds the UDP port of the node.
type UDP uint16

func (v UDP) ENRKey() string { return "udp" }

// ID is the "id" key, which holds the name of the identity scheme.
type ID string

const IDv4 = ID("v4") // the default identity scheme

func (v ID) ENRKey() string { return "id" }

// IP is the "ip" key, which holds the IP address of the node.
type IP net.IP

func (v IP) ENRKey() string { return "ip" }

// EncodeRLP implements rlp.Encoder.
func (v IP) EncodeRLP(w io.Writer) error {
	if ip4 := net.IP(v).To4(); ip4 != nil {
		return rlp.Encode(w, ip4)
	}
	return rlp.Encode(w, net.IP(v))
}

// DecodeRLP implements rlp.Decoder.
func (v *IP) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode((*net.IP)(v)); err != nil {
		return err
	}
	if len(*v) != 4 && len(*v) != 16 {
		return fmt.Errorf("invalid IP address, want 4 or 16 bytes: %v", *v)
	}
	return nil
}

// KeyError is an error related to a key.
type KeyError struct {
	Key string
	Err error
}

// Error implements error.
func (err *KeyError) Error() string {
	if err.Err == errNotFound {
		return fmt.Sprintf("missing ENR key %q", err.Key)
	}
	return fmt.Sprintf("ENR key %q: %v", err.Key, err.Err)
}

// IsNotFound reports whether the given error means that a key/value pair is
// missing from a record.
func IsNotFound(err error) bool {
	kerr, ok := err.(*KeyError)
	return ok && kerr.Err == errNotFound
}

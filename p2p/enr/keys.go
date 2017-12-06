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

package enr

import (
	"crypto/ecdsa"
	"fmt"
	"io"
	"net"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/rlp"
)

type generic struct {
	key   string
	value interface{}
}

func (g generic) ENRKey() string {
	return g.key
}

func (g generic) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, g.value)
}

func (g *generic) DecodeRLP(s *rlp.Stream) error {
	return s.Decode(g.value)
}

// WithKey returns a new Key that can be set in a Record.
// v must implement the rlp.Encoder and rlp.Decoder interface.
func WithKey(k string, v interface{}) Key {
	return &generic{key: k, value: v}
}

// DiscPort represents an UDP port for discovery v5.
type DiscPort uint16

// ENRKey returns the node record key for an UDP port for discovery.
func (DiscPort) ENRKey() string {
	return "discv5"
}

const ID_SECP256k1_KECCAK = "secp256k1-keccak" // identity scheme identifier

// ID is the name of the identity scheme, e.g. "secp256k1-keccak".
type ID string

// ENRKey returns the node record key for its identity scheme.
func (ID) ENRKey() string {
	return "id"
}

// IP4 represents an 4-byte IPv4 address in a node record.
type IP4 net.IP

// ENRKey returns the node record key for an IPv4 address.
func (IP4) ENRKey() string {
	return "ip4"
}

// EncodeRLP implements rlp.Encoder.
func (v IP4) EncodeRLP(w io.Writer) error {
	ip4 := net.IP(v).To4()
	if ip4 == nil {
		return fmt.Errorf("invalid IPv4 address: %v", v)
	}
	return rlp.Encode(w, ip4)
}

// DecodeRLP implements rlp.Decoder.
func (v *IP4) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode((*net.IP)(v)); err != nil {
		return err
	}
	if len(*v) != 4 {
		return fmt.Errorf("invalid IPv4 address, want 4 bytes: %v", *v)
	}
	return nil
}

// IP6 represents an 16-byte IPv6 address in a node record.
type IP6 net.IP

// ENRKey returns the node record key for an IPv6 address.
func (IP6) ENRKey() string {
	return "ip6"
}

// EncodeRLP implements rlp.Encoder.
func (v IP6) EncodeRLP(w io.Writer) error {
	ip6 := net.IP(v)
	return rlp.Encode(w, ip6)
}

// DecodeRLP implements rlp.Decoder.
func (v *IP6) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode((*net.IP)(v)); err != nil {
		return err
	}
	if len(*v) != 16 {
		return fmt.Errorf("invalid IPv6 address, want 16 bytes: %v", *v)
	}
	return nil
}

// Secp256k1 is compressed secp256k1 public key.
type Secp256k1 ecdsa.PublicKey

// ENRKey returns the node record key for the secp256k1 public key.
func (Secp256k1) ENRKey() string {
	return "secp256k1"
}

// EncodeRLP implements rlp.Encoder.
func (v Secp256k1) EncodeRLP(w io.Writer) error {
	pk := btcec.PublicKey(v)

	return rlp.Encode(w, pk.SerializeCompressed())
}

// DecodeRLP implements rlp.Decoder.
func (v *Secp256k1) DecodeRLP(s *rlp.Stream) error {
	buf := make([]byte, 33)
	if err := s.Decode(&buf); err != nil {
		return err
	}

	pk, err := btcec.ParsePubKey(buf, btcec.S256())
	if err != nil {
		return err
	}

	*v = (Secp256k1)(*pk)

	return nil
}

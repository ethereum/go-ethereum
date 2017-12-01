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
	"fmt"
	"io"
	"net"

	"github.com/ethereum/go-ethereum/rlp"
)

// IP6 represents an 16-byte IPv6 address in a node record.
type IP6 net.IP

// ENRKey returns the node record key for an IPv6 address.
func (IP6) ENRKey() string {
	return "ip6"
}

func (v IP6) EncodeRLP(w io.Writer) error {
	ip6 := net.IP(v)
	return rlp.Encode(w, ip6)
}

func (v *IP6) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode((*net.IP)(v)); err != nil {
		return err
	}
	if len(*v) != 16 {
		return fmt.Errorf("invalid IPv6 address, want 16 bytes: %v", *v)
	}
	return nil
}

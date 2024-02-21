// Copyright 2024 The go-ethereum Authors
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
package types

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// PublicKey is a BLS12-381 public key used by validators.
type PublicKey [48]byte

// Exit represents an EIP-7002 exit request from Source for the validator
// associate with PublicKey.
type Exit struct {
	Source    common.Address `json:"source"`
	PublicKey PublicKey      `json:"pubkey"`
}

// Exits implements DerivableList for exits.
type Exits []*Exit

// Len returns the length of s.
func (s Exits) Len() int { return len(s) }

// EncodeIndex encodes the i'th exit to w.
func (s Exits) EncodeIndex(i int, w *bytes.Buffer) {
	rlp.Encode(w, s[i])
}

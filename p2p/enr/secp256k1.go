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
	"io"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/rlp"
)

type Secp256k1 ecdsa.PublicKey

func (Secp256k1) ENRKey() string {
	return "secp256k1"
}

func (v Secp256k1) EncodeRLP(w io.Writer) error {
	pk := btcec.PublicKey(v)

	return rlp.Encode(w, pk.SerializeCompressed())
}

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

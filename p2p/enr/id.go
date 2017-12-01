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
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

type ID string

func (ID) ENRKey() string {
	return "id"
}

func (v ID) EncodeRLP(w io.Writer) error {
	id := string(v)
	return rlp.Encode(w, id)
}

func (v *ID) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode((*string)(v)); err != nil {
		return err
	}
	return nil
}

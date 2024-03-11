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

//go:generate go run github.com/fjl/gencodec -type InclusionList gen_inclusion_rlp.go
//go:generate go run ../../rlp/rlpgen -type InclusionList -out gen_inculsion.go

type InclusionListEntry struct {
	Address common.Address
	Nonce   uint64
}

type InclusionListSummary struct {
	Slot          uint64
	ProposerIndex uint64
	ParentHash    common.Hash
	Summary       []*InclusionListEntry
}

// InclusionList represents a validator InclusionList from the consensus layer.
type InclusionList struct {
	List []*InclusionListEntry
}

// Len returns the length of s.
func (s InclusionList) Len() int { return len(s.List) }

// EncodeIndex encodes the i'th InclusionList to w. Note that this does not check for errors
// because we assume that *InclusionList will only ever contain valid InclusionLists that were either
// constructed by decoding or via public API in this package.
func (s InclusionList) EncodeIndex(i int, w *bytes.Buffer) {
	rlp.Encode(w, s.List[i])
}

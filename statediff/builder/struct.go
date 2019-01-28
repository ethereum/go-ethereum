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

// Contains a batch of utility type declarations used by the tests. As the node
// operates on unique types, a lot of them are needed to check various features.

package builder

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type StateDiff struct {
	BlockNumber     int64                          `json:"blockNumber"      gencodec:"required"`
	BlockHash       common.Hash                    `json:"blockHash"        gencodec:"required"`
	CreatedAccounts map[common.Address]AccountDiff `json:"createdAccounts"  gencodec:"required"`
	DeletedAccounts map[common.Address]AccountDiff `json:"deletedAccounts"  gencodec:"required"`
	UpdatedAccounts map[common.Address]AccountDiff `json:"updatedAccounts"  gencodec:"required"`

	encoded []byte
	err     error
}

func (self *StateDiff) ensureEncoded() {
	if self.encoded == nil && self.err == nil {
		self.encoded, self.err = json.Marshal(self)
	}
}

// Implement Encoder interface for StateDiff
func (sd *StateDiff) Length() int {
	sd.ensureEncoded()
	return len(sd.encoded)
}

// Implement Encoder interface for StateDiff
func (sd *StateDiff) Encode() ([]byte, error) {
	sd.ensureEncoded()
	return sd.encoded, sd.err
}

type AccountDiff struct {
	Nonce        DiffUint64             `json:"nonce"         gencodec:"required"`
	Balance      DiffBigInt             `json:"balance"       gencodec:"required"`
	CodeHash     string                 `json:"codeHash"      gencodec:"required"`
	ContractRoot DiffString             `json:"contractRoot"  gencodec:"required"`
	Storage      map[string]DiffStorage `json:"storage"       gencodec:"required"`
}

type DiffStorage struct {
	Key *string `json:"key" gencodec:"optional"`
	Value *string `json:"value"  gencodec:"optional"`
}
type DiffString struct {
	Value *string `json:"value"  gencodec:"optional"`
}
type DiffUint64 struct {
	Value *uint64 `json:"value"  gencodec:"optional"`
}
type DiffBigInt struct {
	Value *big.Int `json:"value"  gencodec:"optional"`
}

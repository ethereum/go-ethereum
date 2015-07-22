// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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

package state

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

type Log struct {
	Address common.Address
	Topics  []common.Hash
	Data    []byte
	Number  uint64

	TxHash    common.Hash
	TxIndex   uint
	BlockHash common.Hash
	Index     uint
}

func NewLog(address common.Address, topics []common.Hash, data []byte, number uint64) *Log {
	return &Log{Address: address, Topics: topics, Data: data, Number: number}
}

func (self *Log) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{self.Address, self.Topics, self.Data})
}

func (self *Log) String() string {
	return fmt.Sprintf(`log: %x %x %x %x %d %x %d`, self.Address, self.Topics, self.Data, self.TxHash, self.TxIndex, self.BlockHash, self.Index)
}

type Logs []*Log

type LogForStorage Log

func (self *LogForStorage) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{
		self.Address,
		self.Topics,
		self.Data,
		self.Number,
		self.TxHash,
		self.TxIndex,
		self.BlockHash,
		self.Index,
	})
}

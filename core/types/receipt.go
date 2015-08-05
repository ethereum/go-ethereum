// Copyright 2014 The go-ethereum Authors
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
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/rlp"
)

type Receipt struct {
	PostState         []byte
	CumulativeGasUsed *big.Int
	Bloom             Bloom
	TxHash            common.Hash
	ContractAddress   common.Address
	logs              state.Logs
	GasUsed           *big.Int
}

func NewReceipt(root []byte, cumalativeGasUsed *big.Int) *Receipt {
	return &Receipt{PostState: common.CopyBytes(root), CumulativeGasUsed: new(big.Int).Set(cumalativeGasUsed)}
}

func (self *Receipt) SetLogs(logs state.Logs) {
	self.logs = logs
}

func (self *Receipt) Logs() state.Logs {
	return self.logs
}

func (self *Receipt) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{self.PostState, self.CumulativeGasUsed, self.Bloom, self.logs})
}

func (self *Receipt) DecodeRLP(s *rlp.Stream) error {
	var r struct {
		PostState         []byte
		CumulativeGasUsed *big.Int
		Bloom             Bloom
		TxHash            common.Hash
		ContractAddress   common.Address
		Logs              state.Logs
		GasUsed           *big.Int
	}
	if err := s.Decode(&r); err != nil {
		return err
	}
	self.PostState, self.CumulativeGasUsed, self.Bloom, self.TxHash, self.ContractAddress, self.logs, self.GasUsed = r.PostState, r.CumulativeGasUsed, r.Bloom, r.TxHash, r.ContractAddress, r.Logs, r.GasUsed

	return nil
}

type ReceiptForStorage Receipt

func (self *ReceiptForStorage) EncodeRLP(w io.Writer) error {
	storageLogs := make([]*state.LogForStorage, len(self.logs))
	for i, log := range self.logs {
		storageLogs[i] = (*state.LogForStorage)(log)
	}
	return rlp.Encode(w, []interface{}{self.PostState, self.CumulativeGasUsed, self.Bloom, self.TxHash, self.ContractAddress, storageLogs, self.GasUsed})
}

func (self *Receipt) RlpEncode() []byte {
	bytes, err := rlp.EncodeToBytes(self)
	if err != nil {
		fmt.Println("TMP -- RECEIPT ENCODE ERROR", err)
	}
	return bytes
}

func (self *Receipt) Cmp(other *Receipt) bool {
	if bytes.Compare(self.PostState, other.PostState) != 0 {
		return false
	}

	return true
}

func (self *Receipt) String() string {
	return fmt.Sprintf("receipt{med=%x cgas=%v bloom=%x logs=%v}", self.PostState, self.CumulativeGasUsed, self.Bloom, self.logs)
}

type Receipts []*Receipt

func (self Receipts) RlpEncode() []byte {
	bytes, err := rlp.EncodeToBytes(self)
	if err != nil {
		fmt.Println("TMP -- RECEIPTS ENCODE ERROR", err)
	}
	return bytes
}

func (self Receipts) Len() int            { return len(self) }
func (self Receipts) GetRlp(i int) []byte { return common.Rlp(self[i]) }

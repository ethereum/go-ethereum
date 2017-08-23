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
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

//go:generate gencodec -type Receipt -field-override receiptMarshaling -out gen_receipt_json.go

const (
	receiptStatusSuccessful = byte(0x01)
	receiptStatusFailed     = byte(0x00)
)

// Receipt represents the results of a transaction.
type Receipt struct {
	// Consensus fields
	PostState         []byte   `json:"root"`
	Failed            bool     `json:"failed"`
	CumulativeGasUsed *big.Int `json:"cumulativeGasUsed" gencodec:"required"`
	Bloom             Bloom    `json:"logsBloom"         gencodec:"required"`
	Logs              []*Log   `json:"logs"              gencodec:"required"`

	// Implementation fields (don't reorder!)
	TxHash          common.Hash    `json:"transactionHash" gencodec:"required"`
	ContractAddress common.Address `json:"contractAddress"`
	GasUsed         *big.Int       `json:"gasUsed" gencodec:"required"`
}

type receiptMarshaling struct {
	PostState         hexutil.Bytes
	CumulativeGasUsed *hexutil.Big
	GasUsed           *hexutil.Big
}

// homesteadReceiptRLP contains the receipt's Homestead consensus fields, used
// during RLP serialization.
type homesteadReceiptRLP struct {
	PostState         []byte
	CumulativeGasUsed *big.Int
	Bloom             Bloom
	Logs              []*Log
}

// metropolisReceiptRLP contains the receipt's Metropolis consensus fields, used
// during RLP serialization.
type metropolisReceiptRLP struct {
	Status            byte
	CumulativeGasUsed *big.Int
	Bloom             Bloom
	Logs              []*Log
}

// NewReceipt creates a barebone transaction receipt, copying the init fields.
func NewReceipt(root []byte, failed bool, cumulativeGasUsed *big.Int) *Receipt {
	return &Receipt{PostState: common.CopyBytes(root), Failed: failed, CumulativeGasUsed: new(big.Int).Set(cumulativeGasUsed)}
}

// EncodeRLP implements rlp.Encoder, and flattens the consensus fields of a receipt
// into an RLP stream. If no post state is present, metropolis fork is assumed.
func (r *Receipt) EncodeRLP(w io.Writer) error {
	if r.PostState == nil {
		status := receiptStatusSuccessful
		if r.Failed {
			status = receiptStatusFailed
		}
		return rlp.Encode(w, &metropolisReceiptRLP{status, r.CumulativeGasUsed, r.Bloom, r.Logs})
	}
	return rlp.Encode(w, &homesteadReceiptRLP{r.PostState, r.CumulativeGasUsed, r.Bloom, r.Logs})
}

// DecodeRLP implements rlp.Decoder, and loads the consensus fields of a receipt
// from an RLP stream.
func (r *Receipt) DecodeRLP(s *rlp.Stream) error {
	// Load the raw bytes since we have multiple possible formats
	raw, err := s.Raw()
	if err != nil {
		return err
	}
	content, _, err := rlp.SplitList(raw)
	if err != nil {
		return err
	}
	kind, cnt, _, err := rlp.Split(content)
	if err != nil {
		return err
	}
	// Deserialize based on the first component type.
	switch {
	case kind == rlp.Byte || (kind == rlp.String && len(cnt) == 0):
		// The first component of metropolis receipts is Byte (0x01), or the empty
		// string (0x80, decoded as a byte with 0x00 value).
		var metro metropolisReceiptRLP
		if err := rlp.DecodeBytes(raw, &metro); err != nil {
			return err
		}
		switch metro.Status {
		case receiptStatusSuccessful:
			r.Failed = false
		case receiptStatusFailed:
			r.Failed = true
		default:
			return fmt.Errorf("invalid status byte: 0x%x", metro.Status)
		}
		r.CumulativeGasUsed = metro.CumulativeGasUsed
		r.Bloom = metro.Bloom
		r.Logs = metro.Logs
		return nil

	case kind == rlp.String:
		// The first component of homestead receipts is non-empty String.
		var home homesteadReceiptRLP
		if err := rlp.DecodeBytes(raw, &home); err != nil {
			return err
		}
		r.PostState = home.PostState[:]
		r.CumulativeGasUsed = home.CumulativeGasUsed
		r.Bloom = home.Bloom
		r.Logs = home.Logs
		return nil

	default:
		return fmt.Errorf("invalid first receipt component: %v", kind)
	}
}

// String implements the Stringer interface.
func (r *Receipt) String() string {
	if r.PostState == nil {
		return fmt.Sprintf("receipt{failed=%t cgas=%v bloom=%x logs=%v}", r.Failed, r.CumulativeGasUsed, r.Bloom, r.Logs)
	}
	return fmt.Sprintf("receipt{med=%x cgas=%v bloom=%x logs=%v}", r.PostState, r.CumulativeGasUsed, r.Bloom, r.Logs)
}

// ReceiptForStorage is a wrapper around a Receipt that flattens and parses the
// entire content of a receipt, as opposed to only the consensus fields originally.
type ReceiptForStorage Receipt

// EncodeRLP implements rlp.Encoder, and flattens all content fields of a receipt
// into an RLP stream.
func (r *ReceiptForStorage) EncodeRLP(w io.Writer) error {
	logs := make([]*LogForStorage, len(r.Logs))
	for i, log := range r.Logs {
		logs[i] = (*LogForStorage)(log)
	}
	return rlp.Encode(w, []interface{}{r.PostState, r.Failed, r.CumulativeGasUsed, r.Bloom, r.TxHash, r.ContractAddress, logs, r.GasUsed})
}

// DecodeRLP implements rlp.Decoder, and loads both consensus and implementation
// fields of a receipt from an RLP stream.
func (r *ReceiptForStorage) DecodeRLP(s *rlp.Stream) error {
	var receipt struct {
		PostState         []byte
		Failed            bool
		CumulativeGasUsed *big.Int
		Bloom             Bloom
		TxHash            common.Hash
		ContractAddress   common.Address
		Logs              []*LogForStorage
		GasUsed           *big.Int
	}
	if err := s.Decode(&receipt); err != nil {
		return err
	}
	// Assign the consensus fields
	r.PostState, r.Failed, r.CumulativeGasUsed, r.Bloom = receipt.PostState, receipt.Failed, receipt.CumulativeGasUsed, receipt.Bloom
	r.Logs = make([]*Log, len(receipt.Logs))
	for i, log := range receipt.Logs {
		r.Logs[i] = (*Log)(log)
	}
	// Assign the implementation fields
	r.TxHash, r.ContractAddress, r.GasUsed = receipt.TxHash, receipt.ContractAddress, receipt.GasUsed

	return nil
}

// Receipts is a wrapper around a Receipt array to implement DerivableList.
type Receipts []*Receipt

// Len returns the number of receipts in this list.
func (r Receipts) Len() int { return len(r) }

// GetRlp returns the RLP encoding of one receipt from the list.
func (r Receipts) GetRlp(i int) []byte {
	bytes, err := rlp.EncodeToBytes(r[i])
	if err != nil {
		panic(err)
	}
	return bytes
}

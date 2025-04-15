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

package eth

import (
	"bytes"
	"fmt"
	"io"
	"iter"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// Receipt is the eth/69 network encoding of a receipt.
type Receipt struct {
	TxType            byte
	PostStateOrStatus []byte
	GasUsed           uint64
	Logs              rlp.RawValue
}

func (r *Receipt) DecodeRLP(s *rlp.Stream) error {
	if _, err := s.List(); err != nil {
		return err
	}
	txtype, err := s.Uint8()
	if err != nil {
		return fmt.Errorf("invalid txType: %w", err)
	}
	postStateOrStatus, err := s.Bytes()
	if err != nil {
		return fmt.Errorf("invalid postStateOrStatus: %w", err)
	}
	gasUsed, err := s.Uint64()
	if err != nil {
		return fmt.Errorf("invalid gasUsed: %w", err)
	}
	logs, err := s.Raw()
	if err != nil {
		return fmt.Errorf("invalid logs: %w", err)
	}
	*r = Receipt{
		TxType:            txtype,
		PostStateOrStatus: postStateOrStatus,
		GasUsed:           gasUsed,
		Logs:              logs,
	}
	return s.ListEnd()
}

// EncodeRLP writes the network encoding of receipts.
func (r *Receipt) EncodeRLP(_w io.Writer) error {
	w := rlp.NewEncoderBuffer(_w)
	list := w.List()
	w.WriteUint64(uint64(r.TxType))
	w.WriteBytes(r.PostStateOrStatus)
	w.WriteUint64(r.GasUsed)
	w.Write(r.Logs)
	w.ListEnd(list)
	return w.Flush()
}

// encodeStorage writes the network encoding of receipts.
func (r *Receipt) encodeStorage(w *rlp.EncoderBuffer) {
	list := w.List()
	w.WriteBytes(r.PostStateOrStatus)
	w.WriteUint64(r.GasUsed)
	w.Write(r.Logs)
	w.ListEnd(list)
}

// bloom computes the bloom filter of the receipt.
// Note this doesn't check the validity of encoding, and will produce an invalid filter
// for invalid input. This is acceptable for the purpose of this function, which is
// recomputing the receipt hash.
func (r *Receipt) bloom(buffer *[6]byte) types.Bloom {
	var b types.Bloom
	logsIter, err := rlp.NewListIterator(r.Logs)
	if err != nil {
		return b
	}
	for logsIter.Next() {
		log := logsIter.Value()
		address, log, _ := rlp.SplitString(log)
		b.AddWithBuffer(address, buffer)
		topicsIter, err := rlp.NewListIterator(log)
		if err != nil {
			return b
		}
		for topicsIter.Next() {
			b.AddWithBuffer(topicsIter.Value(), buffer)
		}
	}
	return b
}

// ReceiptList69 is the block receipt list as downloaded by eth/69.
// This implements types.DerivableList for validation purposes.
type ReceiptList69 struct {
	buf   *receiptListBuffers
	items []Receipt
}

type receiptListBuffers struct {
	enc   rlp.EncoderBuffer
	bloom [6]byte
}

// Len returns the length of the list.
func (rl *ReceiptList69) Len() int {
	return len(rl.items)
}

// EncodeIndex implements types.DerivableList.
func (rl *ReceiptList69) EncodeIndex(i int, b *bytes.Buffer) {
	if rl.buf == nil {
		rl.buf = new(receiptListBuffers)
	}

	var (
		r     = &rl.items[i]
		bloom = r.bloom(&rl.buf.bloom)
		w     = &rl.buf.enc
	)
	// encode receipt list: [postStateOrStatus, gasUsed, bloom, logs]
	w.Reset(b)
	l := w.List()
	w.WriteBytes(r.PostStateOrStatus)
	w.WriteUint64(r.GasUsed)
	w.WriteBytes(bloom[:])
	w.Write(r.Logs)
	w.ListEnd(l)
	if err := w.Flush(); err != nil {
		return
	}
	// if this is a legacy transaction receipt, we are done.
	if r.TxType == 0 {
		return
	}
	// Otherwise it's a typed transaction receipt, which has the type prefix and
	// the inner list as a byte-array: tx-type || rlp(list).
	// Since b contains the correct inner list, we can reuse its content.
	w.Reset(b)
	w.WriteUint64(uint64(r.TxType))
	w.WriteBytes(b.Bytes())
	w.Flush()
}

// DecodeRLP decodes a list receipts from the network format.
func (rl *ReceiptList69) DecodeRLP(s *rlp.Stream) error {
	if _, err := s.List(); err != nil {
		return err
	}
	for i := 0; s.MoreDataInList(); i++ {
		var item Receipt
		err := item.DecodeRLP(s)
		if err != nil {
			return fmt.Errorf("receipt %d: %v", i, err)
		}
		rl.items = append(rl.items, item)
	}
	return s.ListEnd()
}

// toStorageReceiptsRLP encodes the receipts for storage into the database.
func (rl *ReceiptList69) toStorageReceiptsRLP() rlp.RawValue {
	if rl.buf == nil {
		rl.buf = new(receiptListBuffers)
	}

	var (
		out bytes.Buffer
		w   = &rl.buf.enc
	)
	w.Reset(&out)
	outer := w.List()
	for _, receipts := range rl.items {
		receipts.encodeStorage(w)
	}
	w.ListEnd(outer)
	w.Flush()
	return out.Bytes()
}

// blockReceiptsToNetwork takes a slice of rlp-encoded receipts, and transactions,
// and applies the type-encoding on the receipts (for non-legacy receipts).
// e.g. for non-legacy receipts: receipt-data -> {tx-type || receipt-data}
func blockReceiptsToNetwork(blockReceipts, blockBody rlp.RawValue) ([]byte, error) {
	txTypesIter, err := txTypesInBody(blockBody)
	if err != nil {
		return nil, fmt.Errorf("invalid block body: %v", err)
	}
	nextTxType, stopTxTypes := iter.Pull(txTypesIter)
	defer stopTxTypes()

	var (
		out   bytes.Buffer
		enc   = rlp.NewEncoderBuffer(&out)
		it, _ = rlp.NewListIterator(blockReceipts)
	)
	outer := enc.List()
	for i := 0; it.Next(); i++ {
		txType, _ := nextTxType()
		content, _, _ := rlp.SplitList(it.Value())
		receiptList := enc.List()
		enc.WriteUint64(uint64(txType))
		enc.Write(content)
		enc.ListEnd(receiptList)
	}
	enc.ListEnd(outer)
	enc.Flush()
	return out.Bytes(), nil
}

// txTypesInBody parses the transactions list of an encoded block body,
// returning just the types.
func txTypesInBody(body rlp.RawValue) (iter.Seq[byte], error) {
	bodyFields, _, err := rlp.SplitList(body)
	if err != nil {
		return nil, err
	}
	txsIter, err := rlp.NewListIterator(bodyFields)
	if err != nil {
		return nil, err
	}
	return func(yield func(byte) bool) {
		for txsIter.Next() {
			var txType byte
			switch k, content, _, _ := rlp.Split(txsIter.Value()); k {
			case rlp.List:
				txType = 0
			case rlp.String:
				if len(content) > 0 {
					txType = content[0]
				}
			}
			if !yield(txType) {
				return
			}
		}
	}, nil
}

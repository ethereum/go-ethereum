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
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// Receipt is the representation of receipts for networking purposes.
type Receipt struct {
	TxType            byte
	PostStateOrStatus []byte
	GasUsed           uint64
	Logs              rlp.RawValue
}

func newReceipt(tr *types.Receipt) Receipt {
	r := Receipt{TxType: tr.Type, GasUsed: tr.CumulativeGasUsed}
	if tr.PostState != nil {
		r.PostStateOrStatus = tr.PostState
	} else {
		r.PostStateOrStatus = new(big.Int).SetUint64(tr.Status).Bytes()
	}
	r.Logs, _ = rlp.EncodeToBytes(tr.Logs)
	return r
}

// encodeForHash encodes a receipt for the block receiptsRoot derivation.
func (r *Receipt) encodeForHash(bloomBuf *[6]byte, out *bytes.Buffer) {
	// For typed receipts, add the tx type.
	if r.TxType != 0 {
		out.WriteByte(r.TxType)
	}
	// Encode list = [postStateOrStatus, gasUsed, bloom, logs].
	w := rlp.NewEncoderBuffer(out)
	l := w.List()
	w.WriteBytes(r.PostStateOrStatus)
	w.WriteUint64(r.GasUsed)
	bloom := r.bloom(bloomBuf)
	w.WriteBytes(bloom[:])
	w.Write(r.Logs)
	w.ListEnd(l)
	w.Flush()
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
		log, _, _ := rlp.SplitList(logsIter.Value())
		address, log, _ := rlp.SplitString(log)
		b.AddWithBuffer(address, buffer)
		topicsIter, err := rlp.NewListIterator(log)
		if err != nil {
			return b
		}
		for topicsIter.Next() {
			topic, _, _ := rlp.SplitString(topicsIter.Value())
			b.AddWithBuffer(topic, buffer)
		}
	}
	return b
}

// decode assigns the fields of r by decoding the network format.
func (r *Receipt) decode(input []byte) error {
	input, _, err := rlp.SplitList(input)
	if err != nil {
		return fmt.Errorf("inner list: %v", err)
	}

	// txType
	var txType uint64
	txType, input, err = rlp.SplitUint64(input)
	if err != nil {
		return fmt.Errorf("invalid txType: %w", err)
	}
	if txType > 0x7f {
		return fmt.Errorf("invalid txType: too large")
	}
	r.TxType = byte(txType)

	// status
	r.PostStateOrStatus, input, err = rlp.SplitString(input)
	if err != nil {
		return fmt.Errorf("invalid postStateOrStatus: %w", err)
	}
	if len(r.PostStateOrStatus) > 1 && len(r.PostStateOrStatus) != 32 {
		return fmt.Errorf("invalid postStateOrStatus length %d", len(r.PostStateOrStatus))
	}

	// gas
	r.GasUsed, input, err = rlp.SplitUint64(input)
	if err != nil {
		return fmt.Errorf("invalid gasUsed: %w", err)
	}

	// logs
	_, rest, err := rlp.SplitList(input)
	if err != nil {
		return fmt.Errorf("invalid logs: %w", err)
	}
	if len(rest) != 0 {
		return fmt.Errorf("junk at end of receipt")
	}
	r.Logs = input
	return nil
}

// ReceiptList is the block receipt list as downloaded by eth/69.
type ReceiptList struct {
	items rlp.RawList[Receipt]
}

// NewReceiptList creates a receipt list.
// This is slow, and exists for testing purposes.
func NewReceiptList(trs []*types.Receipt) *ReceiptList {
	rl := new(ReceiptList)
	for _, tr := range trs {
		r := newReceipt(tr)
		encoded, _ := rlp.EncodeToBytes(&r)
		rl.items.AppendRaw(encoded)
	}
	return rl
}

// DecodeRLP decodes a list receipts from the network format.
func (rl *ReceiptList) DecodeRLP(s *rlp.Stream) error {
	return rl.items.DecodeRLP(s)
}

// EncodeRLP encodes the list into the network format of eth/69.
func (rl *ReceiptList) EncodeRLP(w io.Writer) error {
	return rl.items.EncodeRLP(w)
}

// EncodeForStorage encodes a list of receipts for the database.
// It only strips the first element (TxType) from each receipt's
// raw RLP without the actual decoding and re-encoding.
func (rl *ReceiptList) EncodeForStorage() (rlp.RawValue, error) {
	var out bytes.Buffer
	w := rlp.NewEncoderBuffer(&out)
	outer := w.List()
	it := rl.items.ContentIterator()
	for it.Next() {
		content, _, err := rlp.SplitList(it.Value())
		if err != nil {
			return nil, fmt.Errorf("bad receipt: %v", err)
		}
		_, _, rest, err := rlp.Split(content)
		if err != nil {
			return nil, fmt.Errorf("bad receipt: %v", err)
		}
		inner := w.List()
		w.Write(rest)
		w.ListEnd(inner)
	}
	if it.Err() != nil {
		return nil, fmt.Errorf("bad list: %v", it.Err())
	}
	w.ListEnd(outer)
	w.Flush()
	return out.Bytes(), nil
}

// Derivable returns a DerivableList, which can be used to decode
func (rl *ReceiptList) Derivable() types.DerivableList {
	var bloomBuf [6]byte
	return newDerivableRawList(&rl.items, func(data []byte, outbuf *bytes.Buffer) {
		var r Receipt
		if r.decode(data) == nil {
			r.encodeForHash(&bloomBuf, outbuf)
		}
	})
}

// blockReceiptsToNetwork takes a slice of rlp-encoded receipts, and transactions,
// and re-encodes them for the network protocol.
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

// txTypesInBody parses the transactions list of an encoded block body, returning just the types.
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

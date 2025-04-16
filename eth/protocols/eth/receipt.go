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

// This is just a sanity limit for the size of a single receipt.
const maxReceiptSize = 16 * 1024 * 1024

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

// decode68 parses a receipt in the eth/68 network encoding.
func (r *Receipt) decode68(buf *receiptListBuffers, s *rlp.Stream) error {
	k, size, err := s.Kind()
	if err != nil {
		return err
	}

	*r = Receipt{}
	if k == rlp.List {
		// Legacy receipt.
		return r.decodeInnerList(s, false, true)
	}
	// Typed receipt.
	if size < 2 || size > maxReceiptSize {
		return fmt.Errorf("invalid receipt size %d", size)
	}
	buf.tmp.Reset()
	buf.tmp.Grow(int(size))
	payload := buf.tmp.Bytes()[:int(size)]
	if err := s.ReadBytes(payload); err != nil {
		return err
	}
	r.TxType = payload[0]
	s2 := rlp.NewStream(bytes.NewReader(payload[1:]), 0)
	return r.decodeInnerList(s2, false, true)
}

// decode69 parses a receipt in the eth/69 network encoding.
func (r *Receipt) decode69(s *rlp.Stream) error {
	*r = Receipt{}
	return r.decodeInnerList(s, true, false)
}

// decodeDatabase parses a receipt in the basic database encoding.
func (r *Receipt) decodeDatabase(txType byte, s *rlp.Stream) error {
	*r = Receipt{TxType: txType}
	return r.decodeInnerList(s, false, false)
}

func (r *Receipt) decodeInnerList(s *rlp.Stream, readTxType, readBloom bool) error {
	_, err := s.List()
	if err != nil {
		return err
	}
	if readTxType {
		r.TxType, err = s.Uint8()
		if err != nil {
			return fmt.Errorf("invalid txType: %w", err)
		}
	}
	r.PostStateOrStatus, err = s.Bytes()
	if err != nil {
		return fmt.Errorf("invalid postStateOrStatus: %w", err)
	}
	r.GasUsed, err = s.Uint64()
	if err != nil {
		return fmt.Errorf("invalid gasUsed: %w", err)
	}
	if readBloom {
		var b types.Bloom
		if err := s.ReadBytes(b[:]); err != nil {
			return fmt.Errorf("invalid bloom: %v", err)
		}
	}
	r.Logs, err = s.Raw()
	if err != nil {
		return fmt.Errorf("invalid logs: %w", err)
	}
	return s.ListEnd()
}

// encodeForStorage produces the the storage encoding, i.e. the result matches
// the RLP encoding of types.ReceiptForStorage.
func (r *Receipt) encodeForStorage(w *rlp.EncoderBuffer) {
	list := w.List()
	w.WriteBytes(r.PostStateOrStatus)
	w.WriteUint64(r.GasUsed)
	w.Write(r.Logs)
	w.ListEnd(list)
}

// encodeForNetwork68 produces the eth/68 network protocol encoding of a receipt.
// Note this recomputes the bloom filter of the receipt.
func (r *Receipt) encodeForNetwork68(buf *receiptListBuffers, w *rlp.EncoderBuffer) {
	writeInner := func(w *rlp.EncoderBuffer) {
		list := w.List()
		w.WriteBytes(r.PostStateOrStatus)
		w.WriteUint64(r.GasUsed)
		bloom := r.bloom(&buf.bloom)
		w.WriteBytes(bloom[:])
		w.Write(r.Logs)
		w.ListEnd(list)
	}

	if r.TxType == 0 {
		writeInner(w)
	} else {
		buf.tmp.Reset()
		buf.tmp.WriteByte(r.TxType)
		buf.enc.Reset(&buf.tmp)
		writeInner(&buf.enc)
		buf.enc.Flush()
		w.WriteBytes(buf.tmp.Bytes())
	}
}

// encodeForNetwork69 produces the eth/69 network protocol encoding of a receipt.
func (r *Receipt) encodeForNetwork69(w *rlp.EncoderBuffer) {
	list := w.List()
	w.WriteUint64(uint64(r.TxType))
	w.WriteBytes(r.PostStateOrStatus)
	w.WriteUint64(r.GasUsed)
	w.Write(r.Logs)
	w.ListEnd(list)
}

// encodeForHash encodes a receipt for the block receiptsRoot derivation.
func (r *Receipt) encodeForHash(buf *receiptListBuffers, out *bytes.Buffer) {
	// For typed receipts, add the tx type.
	if r.TxType != 0 {
		out.WriteByte(r.TxType)
	}
	// Encode list = [postStateOrStatus, gasUsed, bloom, logs].
	w := &buf.enc
	w.Reset(out)
	l := w.List()
	w.WriteBytes(r.PostStateOrStatus)
	w.WriteUint64(r.GasUsed)
	bloom := r.bloom(&buf.bloom)
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

type receiptListBuffers struct {
	enc   rlp.EncoderBuffer
	bloom [6]byte
	tmp   bytes.Buffer
}

func initBuffers(buf **receiptListBuffers) {
	if *buf == nil {
		*buf = new(receiptListBuffers)
	}
}

// encodeForStorage encodes a list of receipts for the database.
func (buf *receiptListBuffers) encodeForStorage(rs []Receipt) rlp.RawValue {
	var out bytes.Buffer
	w := &buf.enc
	w.Reset(&out)
	outer := w.List()
	for _, receipts := range rs {
		receipts.encodeForStorage(w)
	}
	w.ListEnd(outer)
	w.Flush()
	return out.Bytes()
}

// ReceiptList68 is a block receipt list as downloaded by eth/68.
// This also implements types.DerivableList for validation purposes.
type ReceiptList68 struct {
	buf   *receiptListBuffers
	items []Receipt
}

// NewReceiptList68 creates a receipt list.
// This is slow, and exists for testing purposes.
func NewReceiptList68(trs []*types.Receipt) *ReceiptList68 {
	rl := &ReceiptList68{items: make([]Receipt, len(trs))}
	for i, tr := range trs {
		rl.items[i] = newReceipt(tr)
	}
	return rl
}

func blockReceiptsToNetwork68(blockReceipts, blockBody rlp.RawValue) ([]byte, error) {
	txTypesIter, err := txTypesInBody(blockBody)
	if err != nil {
		return nil, fmt.Errorf("invalid block body: %v", err)
	}
	nextTxType, stopTxTypes := iter.Pull(txTypesIter)
	defer stopTxTypes()

	var (
		out bytes.Buffer
		buf receiptListBuffers
	)
	blockReceiptIter, _ := rlp.NewListIterator(blockReceipts)
	innerReader := bytes.NewReader(nil)
	innerStream := rlp.NewStream(innerReader, 0)
	w := rlp.NewEncoderBuffer(&out)
	outer := w.List()
	for i := 0; blockReceiptIter.Next(); i++ {
		content := blockReceiptIter.Value()
		innerReader.Reset(content)
		innerStream.Reset(innerReader, uint64(len(content)))
		var r Receipt
		txType, _ := nextTxType()
		if err := r.decodeDatabase(txType, innerStream); err != nil {
			return nil, fmt.Errorf("invalid database receipt %d: %v", i, err)
		}
		r.encodeForNetwork68(&buf, &w)
	}
	w.ListEnd(outer)
	w.Flush()
	return out.Bytes(), nil
}

// setBuffers implements ReceiptsList.
func (rl *ReceiptList68) setBuffers(buf *receiptListBuffers) {
	rl.buf = buf
}

// EncodeForStorage encodes the receipts for storage into the database.
func (rl *ReceiptList68) EncodeForStorage() rlp.RawValue {
	initBuffers(&rl.buf)
	return rl.buf.encodeForStorage(rl.items)
}

// Len implements types.DerivableList.
func (rl *ReceiptList68) Len() int {
	return len(rl.items)
}

// EncodeIndex implements types.DerivableList.
func (rl *ReceiptList68) EncodeIndex(i int, out *bytes.Buffer) {
	initBuffers(&rl.buf)
	rl.items[i].encodeForHash(rl.buf, out)
}

// DecodeRLP decodes a list of receipts from the network format.
func (rl *ReceiptList68) DecodeRLP(s *rlp.Stream) error {
	initBuffers(&rl.buf)
	if _, err := s.List(); err != nil {
		return err
	}
	for i := 0; s.MoreDataInList(); i++ {
		var item Receipt
		err := item.decode68(rl.buf, s)
		if err != nil {
			return fmt.Errorf("receipt %d: %v", i, err)
		}
		rl.items = append(rl.items, item)
	}
	return s.ListEnd()
}

// EncodeRLP encodes the list into the network format of eth/68.
func (rl *ReceiptList68) EncodeRLP(_w io.Writer) error {
	initBuffers(&rl.buf)
	w := rlp.NewEncoderBuffer(_w)
	outer := w.List()
	for i := range rl.items {
		rl.items[i].encodeForNetwork68(rl.buf, &w)
	}
	w.ListEnd(outer)
	return w.Flush()
}

// ReceiptList69 is the block receipt list as downloaded by eth/69.
// This implements types.DerivableList for validation purposes.
type ReceiptList69 struct {
	buf   *receiptListBuffers
	items []Receipt
}

// NewReceiptList69 creates a receipt list.
// This is slow, and exists for testing purposes.
func NewReceiptList69(trs []*types.Receipt) *ReceiptList69 {
	rl := &ReceiptList69{items: make([]Receipt, len(trs))}
	for i, tr := range trs {
		rl.items[i] = newReceipt(tr)
	}
	return rl
}

// setBuffers implements ReceiptsList.
func (rl *ReceiptList69) setBuffers(buf *receiptListBuffers) {
	rl.buf = buf
}

// EncodeForStorage encodes the receipts for storage into the database.
func (rl *ReceiptList69) EncodeForStorage() rlp.RawValue {
	initBuffers(&rl.buf)
	return rl.buf.encodeForStorage(rl.items)
}

// Len implements types.DerivableList.
func (rl *ReceiptList69) Len() int {
	return len(rl.items)
}

// EncodeIndex implements types.DerivableList.
func (rl *ReceiptList69) EncodeIndex(i int, out *bytes.Buffer) {
	initBuffers(&rl.buf)
	rl.items[i].encodeForHash(rl.buf, out)
}

// DecodeRLP decodes a list receipts from the network format.
func (rl *ReceiptList69) DecodeRLP(s *rlp.Stream) error {
	if _, err := s.List(); err != nil {
		return err
	}
	for i := 0; s.MoreDataInList(); i++ {
		var item Receipt
		err := item.decode69(s)
		if err != nil {
			return fmt.Errorf("receipt %d: %v", i, err)
		}
		rl.items = append(rl.items, item)
	}
	return s.ListEnd()
}

// EncodeRLP encodes the list into the network format of eth/69.
func (rl *ReceiptList69) EncodeRLP(_w io.Writer) error {
	w := rlp.NewEncoderBuffer(_w)
	outer := w.List()
	for i := range rl.items {
		rl.items[i].encodeForNetwork69(&w)
	}
	w.ListEnd(outer)
	return w.Flush()
}

// blockReceiptsToNetwork69 takes a slice of rlp-encoded receipts, and transactions,
// and applies the type-encoding on the receipts (for non-legacy receipts).
// e.g. for non-legacy receipts: receipt-data -> {tx-type || receipt-data}
func blockReceiptsToNetwork69(blockReceipts, blockBody rlp.RawValue) ([]byte, error) {
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

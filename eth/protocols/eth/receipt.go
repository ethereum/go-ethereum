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
	"errors"
	"io"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

type receiptForNetwork types.Receipt

func (r *receiptForNetwork) DecodeRLP(s *rlp.Stream) error {
	kind, size, err := s.Kind()
	if err != nil {
		return err
	}
	if kind != rlp.List {
		return errors.New("invalid receipt")
	}
	if size == 4 {
		var rec = new(struct {
			Type    byte
			Status  uint64
			GasUsed uint64
			Logs    []*types.Log
		})
		if err := s.Decode(rec); err != nil {
			return err
		}
		r.Type = rec.Type
		r.Status = rec.Status
		r.GasUsed = rec.GasUsed
		r.Logs = rec.Logs
	} else {
		s.Decode(&r)
	}
	return nil
}

func (r *receiptForNetwork) EncodeRLP(_w io.Writer) error {
	data := &types.ReceiptForStorage{Status: r.Status, CumulativeGasUsed: r.GasUsed, Logs: r.Logs}
	if r.Type == types.LegacyTxType {
		return rlp.Encode(_w, data)
	}
	w := rlp.NewEncoderBuffer(_w)
	outerList := w.List()
	w.Write([]byte{r.Type})
	if r.Status == types.ReceiptStatusSuccessful {
		w.Write([]byte{0x01})
	} else {
		w.Write([]byte{0x00})
	}
	w.WriteUint64(r.GasUsed)
	logList := w.List()
	for _, log := range r.Logs {
		if err := log.EncodeRLP(w); err != nil {
			return err
		}
	}
	w.ListEnd(logList)
	w.ListEnd(outerList)
	return w.Flush()
}

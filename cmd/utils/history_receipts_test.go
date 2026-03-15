// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func makeTestReceipt(typ uint8) *types.Receipt {
	r := &types.Receipt{
		Type:              typ,
		Status:            types.ReceiptStatusSuccessful,
		CumulativeGasUsed: 42_000,
		Logs: []*types.Log{
			{
				Address: common.HexToAddress("0x1"),
				Topics:  []common.Hash{common.HexToHash("0x2"), common.HexToHash("0x3")},
				Data:    []byte{0xde, 0xad, 0xbe, 0xef},
			},
		},
	}
	r.Bloom = types.CreateBloom(r)
	return r
}

func TestImportHistory_ConvertSlimReceiptsToStorage(t *testing.T) {
	tests := []struct {
		name     string
		receipts types.Receipts
	}{
		{
			name:     "typed-single",
			receipts: types.Receipts{makeTestReceipt(types.DynamicFeeTxType)},
		},
		{
			name:     "legacy-single",
			receipts: types.Receipts{makeTestReceipt(types.LegacyTxType)},
		},
		{
			name: "mixed-multiple",
			receipts: types.Receipts{
				makeTestReceipt(types.LegacyTxType),
				makeTestReceipt(types.DynamicFeeTxType),
			},
		},
		{
			name:     "empty",
			receipts: types.Receipts{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rawSlimReceipts := make([]*types.SlimReceipt, len(tc.receipts))
			for i, receipt := range tc.receipts {
				rawSlimReceipts[i] = (*types.SlimReceipt)(receipt)
			}
			rawSlim, err := rlp.EncodeToBytes(rawSlimReceipts)
			if err != nil {
				t.Fatalf("failed to encode slim receipts: %v", err)
			}
			got, err := convertReceiptsToStorage(rawSlim, eraReceiptFormatSlim, len(tc.receipts))
			if err != nil {
				t.Fatalf("conversion failed: %v", err)
			}
			want := types.EncodeBlockReceiptLists([]types.Receipts{tc.receipts})[0]
			if !bytes.Equal(got, want) {
				t.Fatalf("converted storage receipts mismatch\ngot:  %x\nwant: %x", got, want)
			}
		})
	}
}

func TestImportHistory_ConvertSlimReceiptsToStorageErrors(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		want string
	}{
		{
			name: "invalid-rlp",
			raw:  []byte{0xff},
			want: "invalid block receipts list",
		},
		{
			name: "too-few-fields",
			raw:  []byte{0xc4, 0xc3, 0x01, 0x02, 0x03}, // [[1,2,3]]
			want: "want 4",
		},
		{
			name: "too-many-fields",
			raw:  []byte{0xc6, 0xc5, 0x01, 0x02, 0x03, 0x04, 0x05}, // [[1,2,3,4,5]]
			want: "too many fields",
		},
		{
			name: "tx-receipt-count-mismatch",
			raw:  []byte{0xc5, 0xc4, 0x01, 0x01, 0x02, 0xc0}, // [[1,1,2,[]]]
			want: "tx/receipt count mismatch",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := convertReceiptsToStorage(tc.raw, eraReceiptFormatSlim, 2)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

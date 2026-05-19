// Copyright 2026 The go-ethereum Authors
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

package stateless

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestWitnessToExtWitnessOrdersFields(t *testing.T) {
	witness := &Witness{
		Headers: []*types.Header{testHeader(3), testHeader(2), testHeader(1)},
		Codes: map[string]struct{}{
			string([]byte{0x02}):       {},
			string([]byte{0x01, 0xff}): {},
			string([]byte{0x01}):       {},
		},
		State: map[string]struct{}{
			string([]byte{0xff}): {},
			string([]byte{0x00}): {},
			string([]byte{0x7f}): {},
		},
	}
	ext := witness.ToExtWitness()

	checkHeaderNumbers(t, ext.Headers, []uint64{1, 2, 3})
	checkBytes(t, "codes", ext.Codes, [][]byte{
		{0x01},
		{0x01, 0xff},
		{0x02},
	})
	checkBytes(t, "state", ext.State, [][]byte{
		{0x00},
		{0x7f},
		{0xff},
	})
}

func TestWitnessFromExtWitnessNormalizesHeaderOrder(t *testing.T) {
	tests := []struct {
		name    string
		headers []*types.Header
	}{
		{
			name:    "spec ordered",
			headers: []*types.Header{testHeader(1), testHeader(2), testHeader(3)},
		},
		{
			name:    "legacy internal ordered",
			headers: []*types.Header{testHeader(3), testHeader(2), testHeader(1)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var witness Witness
			if err := witness.FromExtWitness(&ExtWitness{Headers: tt.headers}); err != nil {
				t.Fatalf("FromExtWitness returned error: %v", err)
			}
			checkHeaderNumbers(t, witness.Headers, []uint64{3, 2, 1})
			if root := witness.Root(); root != testHeaderRoot(3) {
				t.Fatalf("root mismatch: have %s, want %s", root, testHeaderRoot(3))
			}
		})
	}
}

func TestWitnessFromExtWitnessRejectsEmptyHeaders(t *testing.T) {
	var witness Witness
	if err := witness.FromExtWitness(&ExtWitness{}); err == nil {
		t.Fatal("expected empty witness error")
	}
}

func testHeader(number uint64) *types.Header {
	return &types.Header{
		Number: new(big.Int).SetUint64(number),
		Root:   testHeaderRoot(number),
	}
}

func testHeaderRoot(number uint64) common.Hash {
	return common.Hash{byte(number)}
}

func checkHeaderNumbers(t *testing.T, headers []*types.Header, want []uint64) {
	t.Helper()
	if len(headers) != len(want) {
		t.Fatalf("header count mismatch: have %d, want %d", len(headers), len(want))
	}
	for i, header := range headers {
		if header.Number.Uint64() != want[i] {
			t.Fatalf("header %d number mismatch: have %d, want %d", i, header.Number.Uint64(), want[i])
		}
	}
}

func checkBytes(t *testing.T, name string, got []hexutil.Bytes, want [][]byte) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s count mismatch: have %d, want %d", name, len(got), len(want))
	}
	for i := range got {
		if !bytes.Equal(got[i], want[i]) {
			t.Fatalf("%s %d mismatch: have %x, want %x", name, i, got[i], want[i])
		}
	}
}

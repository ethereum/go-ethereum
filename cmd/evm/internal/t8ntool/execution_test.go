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

package t8ntool

import (
	"bytes"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func mustEncodeHeader(t *testing.T, header *types.Header) []byte {
	t.Helper()

	encoded, err := rlp.EncodeToBytes(header)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestConvertExecutionWitnessOrdersForEEST(t *testing.T) {
	parent := &types.Header{Number: big.NewInt(2)}
	grandparent := &types.Header{Number: big.NewInt(1)}
	witness := &stateless.Witness{
		Headers: []*types.Header{parent, grandparent},
		Codes: map[string]struct{}{
			string([]byte{0x02}): {},
			string([]byte{0x01}): {},
		},
		State: map[string]struct{}{
			string([]byte{0x20}): {},
			string([]byte{0x10}): {},
		},
	}

	got, err := convertExecutionWitness(witness)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.State[0], []byte{0x10}) || !bytes.Equal(got.State[1], []byte{0x20}) {
		t.Fatalf("state not sorted lexicographically: %#x", got.State)
	}
	if !bytes.Equal(got.Codes[0], []byte{0x01}) || !bytes.Equal(got.Codes[1], []byte{0x02}) {
		t.Fatalf("codes not sorted lexicographically: %#x", got.Codes)
	}
	wantHeaders := [][]byte{
		mustEncodeHeader(t, grandparent),
		mustEncodeHeader(t, parent),
	}
	for i, want := range wantHeaders {
		if !bytes.Equal(got.Headers[i], want) {
			t.Fatalf("header %d mismatch: have %#x want %#x", i, got.Headers[i], want)
		}
	}
}

func TestT8NHeaderReaderValidatesHash(t *testing.T) {
	header := &types.Header{Number: big.NewInt(7)}
	reader, err := newT8nHeaderReader(map[gethmath.HexOrDecimal64]hexutil.Bytes{
		7: mustEncodeHeader(t, header),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := reader.GetHeader(header.Hash(), 7); got == nil || got.Hash() != header.Hash() {
		t.Fatalf("failed to retrieve known header: %v", got)
	}
	if got := reader.GetHeader(common.Hash{0x01}, 7); got != nil {
		t.Fatalf("retrieved header for mismatched hash: %v", got)
	}
	if err := reader.Err(); err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("unexpected reader error: %v", err)
	}
}

func TestT8NHeaderReaderReportsMissingAncestor(t *testing.T) {
	reader, err := newT8nHeaderReader(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := reader.GetHeader(common.Hash{}, 1); got != nil {
		t.Fatalf("retrieved missing header: %v", got)
	}
	if err := reader.Err(); err == nil || !strings.Contains(err.Error(), "missing blockHeaders[1]") {
		t.Fatalf("unexpected reader error: %v", err)
	}
}

func TestT8NHeaderReaderRejectsNumberMismatch(t *testing.T) {
	header := &types.Header{Number: big.NewInt(2)}
	_, err := newT8nHeaderReader(map[gethmath.HexOrDecimal64]hexutil.Bytes{
		1: mustEncodeHeader(t, header),
	})
	if err == nil || !strings.Contains(err.Error(), "decodes to header number") {
		t.Fatalf("unexpected error: %v", err)
	}
}

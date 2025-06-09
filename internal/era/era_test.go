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

package era

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

type testchain struct {
	headers  [][]byte
	bodies   [][]byte
	receipts [][]byte
	tds      []*big.Int
}

func TestEra1Builder(t *testing.T) {
	t.Parallel()

	// Get temp directory.
	f, err := os.CreateTemp(t.TempDir(), "era1-test")
	if err != nil {
		t.Fatalf("error creating temp file: %v", err)
	}
	defer f.Close()

	var (
		builder = NewBuilder(f)
		chain   = testchain{}
	)
	for i := 0; i < 128; i++ {
		chain.headers = append(chain.headers, mustEncode(&types.Header{Number: big.NewInt(int64(i))}))
		chain.bodies = append(chain.bodies, mustEncode(&types.Body{Transactions: []*types.Transaction{types.NewTransaction(0, common.Address{byte(i)}, nil, 0, nil, nil)}}))
		chain.receipts = append(chain.receipts, mustEncode(&types.Receipts{{CumulativeGasUsed: uint64(i)}}))
		chain.tds = append(chain.tds, big.NewInt(int64(i)))
	}

	// Write blocks to Era1.
	for i := 0; i < len(chain.headers); i++ {
		var (
			header   = chain.headers[i]
			body     = chain.bodies[i]
			receipts = chain.receipts[i]
			hash     = common.Hash{byte(i)}
			td       = chain.tds[i]
		)
		if err = builder.AddRLP(header, body, receipts, uint64(i), hash, td, big.NewInt(1)); err != nil {
			t.Fatalf("error adding entry: %v", err)
		}
	}

	// Finalize Era1.
	if _, err := builder.Finalize(); err != nil {
		t.Fatalf("error finalizing era1: %v", err)
	}

	// Verify Era1 contents.
	e, err := Open(f.Name())
	if err != nil {
		t.Fatalf("failed to open era: %v", err)
	}
	defer e.Close()
	it, err := NewRawIterator(e)
	if err != nil {
		t.Fatalf("failed to make iterator: %s", err)
	}
	for i := uint64(0); i < uint64(len(chain.headers)); i++ {
		if !it.Next() {
			t.Fatalf("expected more entries")
		}
		if it.Error() != nil {
			t.Fatalf("unexpected error %v", it.Error())
		}
		// Check headers.
		rawHeader, err := io.ReadAll(it.Header)
		if err != nil {
			t.Fatalf("error reading header from iterator: %v", err)
		}
		if !bytes.Equal(rawHeader, chain.headers[i]) {
			t.Fatalf("mismatched header: want %s, got %s", chain.headers[i], rawHeader)
		}
		header, err := e.GetHeaderByNumber(i)
		if err != nil {
			t.Fatalf("error reading header: %v", err)
		}
		encHeader, err := rlp.EncodeToBytes(header)
		if err != nil {
			t.Fatalf("error encoding header: %v", err)
		}
		if !bytes.Equal(encHeader, chain.headers[i]) {
			t.Fatalf("mismatched header: want %s, got %s", chain.headers[i], encHeader)
		}

		// Check bodies.
		body, err := io.ReadAll(it.Body)
		if err != nil {
			t.Fatalf("error reading body: %v", err)
		}
		if !bytes.Equal(body, chain.bodies[i]) {
			t.Fatalf("mismatched body: want %s, got %s", chain.bodies[i], body)
		}

		// Check receipts.
		rawReceipts, err := io.ReadAll(it.Receipts)
		if err != nil {
			t.Fatalf("error reading receipts from iterator: %v", err)
		}
		if !bytes.Equal(rawReceipts, chain.receipts[i]) {
			t.Fatalf("mismatched receipts: want %s, got %s", chain.receipts[i], rawReceipts)
		}
		receipts, err := e.GetReceiptsByNumber(i)
		if err != nil {
			t.Fatalf("error reading receipts: %v", err)
		}
		encReceipts, err := rlp.EncodeToBytes(receipts)
		if err != nil {
			t.Fatalf("error encoding receipts: %v", err)
		}
		if !bytes.Equal(encReceipts, chain.receipts[i]) {
			t.Fatalf("mismatched receipts: want %s, got %s", chain.receipts[i], encReceipts)
		}

		// Check total difficulty.
		rawTd, err := io.ReadAll(it.TotalDifficulty)
		if err != nil {
			t.Fatalf("error reading td: %v", err)
		}
		td := new(big.Int).SetBytes(reverseOrder(rawTd))
		if td.Cmp(chain.tds[i]) != 0 {
			t.Fatalf("mismatched tds: want %s, got %s", chain.tds[i], td)
		}
	}
}

func TestEraFilename(t *testing.T) {
	t.Parallel()

	for i, tt := range []struct {
		network  string
		epoch    int
		root     common.Hash
		expected string
	}{
		{"mainnet", 1, common.Hash{1}, "mainnet-00001-01000000.era1"},
	} {
		got := Filename(tt.network, tt.epoch, tt.root)
		if tt.expected != got {
			t.Errorf("test %d: invalid filename: want %s, got %s", i, tt.expected, got)
		}
	}
}

func mustEncode(obj any) []byte {
	b, err := rlp.EncodeToBytes(obj)
	if err != nil {
		panic(fmt.Sprintf("failed in encode obj: %v", err))
	}
	return b
}

// Copyright 2025 The go-ethereum Authors
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

package era2

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
	"github.com/klauspost/compress/snappy"
)

type testchain struct {
	headers  []types.Header
	bodies   []types.Body
	receipts []types.Receipts
	tds      []*big.Int
}

func TestEra2Builder(t *testing.T) {
	t.Parallel()
	// Get temp directory.
	f, err := os.CreateTemp(t.TempDir(), "era2-test")
	if err != nil {
		t.Fatalf("error creating temp file: %v", err)
	}
	defer f.Close()

	var (
		builder = NewBuilder(f)
		chain   = testchain{}
	)
	for i := 0; i < 128; i++ {
		chain.headers = append(chain.headers, types.Header{Number: big.NewInt(int64(i))})
		chain.bodies = append(chain.bodies, types.Body{Transactions: []*types.Transaction{types.NewTransaction(0, common.Address{byte(i)}, nil, 0, nil, nil)}})
		chain.receipts = append(chain.receipts, types.Receipts{{CumulativeGasUsed: uint64(i)}})
		chain.tds = append(chain.tds, big.NewInt(int64(i)))
	}

	// Write blocks to Era1.
	for i := 0; i < len(chain.headers); i++ {
		var (
			header   = chain.headers[i]
			body     = chain.bodies[i]
			receipts = chain.receipts[i]
			td       = chain.tds[i]
		)
		if err = builder.Add(header, body, receipts, td, nil); err != nil {
			t.Fatalf("error adding entry: %v", err)
		}
	}

	// Finalize Era1.
	if _, err := builder.Finalize(); err != nil {
		t.Fatalf("error finalizing era1: %v", err)
	}

	// 3. open reader
	t.Logf("filename: %s", f.Name())
	era, err := Open(f.Name())
	if err != nil {
		t.Fatalf("open era: %v", err)
	}
	defer era.Close()

	hdrs, err := era.GetHeaders(0, 128)
	if err != nil {
		t.Fatalf("BatchHeaders full: %v", err)
	}
	bods, err := era.GetBodies(0, 128)
	if err != nil {
		t.Fatalf("BatchBodies full: %v", err)
	}
	recs, err := era.GetReceipts(0, 128)
	if err != nil {
		t.Fatalf("BatchReceipts full: %v", err)
	}

	for i := 0; i < 128; i++ {
		if hdrs[i].Hash() != chain.headers[i].Hash() {
			t.Fatalf("batch header %d mismatch", i)
		}
		if !bytes.Equal(
			mustEncode(chain.bodies[i]),
			mustEncode(bods[i]),
		) {
			t.Fatalf("batch body %d mismatch", i)
		}
		if !bytes.Equal(
			mustEncode(chain.receipts[i]),
			mustEncode(recs[i]),
		) {
			t.Fatalf("batch receipts %d mismatch", i)
		}
	}

	const start, cnt = 50, 10
	winHdrs, err := era.GetHeaders(start, cnt)
	if err != nil {
		t.Fatalf("BatchHeaders window: %v", err)
	}
	for j := 0; j < cnt; j++ {
		idx := start + j
		if winHdrs[j].Hash() != chain.headers[idx].Hash() {
			t.Fatalf("window header %d mismatch", idx)
		}
	}

	for i := 0; i < 128; i++ {
		bn := uint64(i)
		gotBlock, err := era.GetBlockByNumber(bn)
		if err != nil {
			t.Fatalf("get block %d: %v", i, err)
		}
		if chain.headers[i].Hash() != gotBlock.Header().Hash() {
			t.Fatalf("header %d mismatch", i)
		}
		if !bytes.Equal(mustEncode(chain.bodies[i]), mustEncode(gotBlock.Body())) {
			t.Fatalf("body %d mismatch", i)
		}

		rawBody, err := era.GetRawBodyFrameByNumber(bn)
		if err != nil {
			t.Fatalf("raw body %d: %v", i, err)
		}
		decBody, err := io.ReadAll(
			snappy.NewReader(bytes.NewReader(rawBody)),
		)
		if err != nil {
			t.Fatalf("snappy decode body %d: %v", i, err)
		}
		if !bytes.Equal(decBody, mustEncode(chain.bodies[i])) {
			t.Fatalf("body frame %d mismatch", i)
		}

		rawRcpt, err := era.GetRawReceiptsFrameByNumber(bn)
		if err != nil {
			t.Fatalf("raw receipts %d: %v", i, err)
		}
		decRcpt, err := io.ReadAll(
			snappy.NewReader(bytes.NewReader(rawRcpt)),
		)
		if err != nil {
			t.Fatalf("snappy decode receipts %d: %v", i, err)
		}
		if !bytes.Equal(decRcpt, mustEncode(chain.receipts[i])) {
			t.Fatalf("receipts frame %d mismatch", i)
		}

		td, err := era.getTD(bn)
		if err != nil {
			t.Fatalf("getTD %d: %v", i, err)
		}
		if td.Cmp(chain.tds[i]) != 0 {
			t.Fatalf("td %d mismatch: want %v got %v", i, chain.tds[i], td)
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

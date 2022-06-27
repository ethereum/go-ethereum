// Copyright 2021 The go-ethereum Authors
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

package rawdb

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestTxTrace(t *testing.T) {
	db := NewMemoryDatabase()

	testCases := []struct {
		name         string
		txHash       common.Hash
		expectedData []byte
	}{
		{
			name:         "test1",
			txHash:       common.Hash{},
			expectedData: []byte{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := ReadTxTrace(db, tc.txHash)
			if !bytes.Equal(tc.expectedData, data) {
				t.Fatalf("Unexpected tx trace data returned")
			}
		})
	}

	txHashStr := "0x5d763978db1d0aedce8cc7c97389fccd1be95e17da337aeec0dd8f8ff2417726"
	txHash := common.HexToHash(txHashStr)
	WriteTxTrace(db, txHash, []byte("hello world"))
	data := ReadTxTrace(db, txHash)
	if !bytes.Equal(data, []byte("hello world")) {
		t.Fatalf("Unexpected tx trace data returned")
	}
}

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

package eth

import (
	"compress/gzip"
	"errors"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

var errLateClose = errors.New("injected late gzip close failure")

type lateFailGzip struct {
	inner  io.Writer
	closed bool
}

func (w *lateFailGzip) Write(p []byte) (int, error) {
	return w.inner.Write(p)
}

func (w *lateFailGzip) Close() error {
	w.closed = true
	return errLateClose
}

func TestAdminExportChainReportsGzipCloseFailure(t *testing.T) {
	genesis := &core.Genesis{
		Config: params.TestChainConfig,
		Alloc: types.GenesisAlloc{
			{1}: {Balance: big.NewInt(params.Ether)},
		},
	}
	chain := newTestBlockChain(t, 1, genesis, nil)
	defer chain.Stop()

	var injected *lateFailGzip
	oldFactory := newAdminGzipWriter
	newAdminGzipWriter = func(dst io.Writer) io.WriteCloser {
		injected = &lateFailGzip{inner: gzip.NewWriter(dst)}
		return injected
	}
	defer func() { newAdminGzipWriter = oldFactory }()

	path := filepath.Join(t.TempDir(), "chain.gz")
	result, err := NewAdminAPI(&Ethereum{blockchain: chain}).ExportChain(path, nil, nil)
	if result {
		t.Fatal("ExportChain returned success after gzip finalization failed")
	}
	if !errors.Is(err, errLateClose) {
		t.Fatalf("ExportChain error = %v, want %v", err, errLateClose)
	}
	if injected == nil || !injected.closed {
		t.Fatal("fixture did not execute the failing Close")
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err == nil {
		_, err = io.ReadAll(zr)
		_ = zr.Close()
	}
	if err == nil {
		t.Fatal("gzip archive unexpectedly valid")
	}
}

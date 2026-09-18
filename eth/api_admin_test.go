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
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

var errGzipClose = errors.New("gzip close failure")

type closeErrorWriter struct {
	io.Writer
}

func (closeErrorWriter) Close() error {
	return errGzipClose
}

func TestAdminExportChainGzipCloseError(t *testing.T) {
	genesis := &core.Genesis{
		Config: params.TestChainConfig,
		Alloc: types.GenesisAlloc{
			{1}: {Balance: big.NewInt(params.Ether)},
		},
	}
	chain := newTestBlockChain(t, 1, genesis, nil)
	defer chain.Stop()

	api := NewAdminAPI(&Ethereum{blockchain: chain})
	path := filepath.Join(t.TempDir(), "chain.gz")
	success, err := api.exportChain(path, nil, nil, func(dst io.Writer) io.WriteCloser {
		return closeErrorWriter{Writer: gzip.NewWriter(dst)}
	})
	if success {
		t.Fatal("export succeeded despite gzip close failure")
	}
	if !errors.Is(err, errGzipClose) {
		t.Fatalf("unexpected error: %v", err)
	}
}

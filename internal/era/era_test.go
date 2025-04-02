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
	"io"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
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
		chain.headers = append(chain.headers, []byte{byte('h'), byte(i)})
		chain.bodies = append(chain.bodies, []byte{byte('b'), byte(i)})
		chain.receipts = append(chain.receipts, []byte{byte('r'), byte(i)})
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
		header, err := io.ReadAll(it.Header)
		if err != nil {
			t.Fatalf("error reading header: %v", err)
		}
		if !bytes.Equal(header, chain.headers[i]) {
			t.Fatalf("mismatched header: want %s, got %s", chain.headers[i], header)
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
		receipts, err := io.ReadAll(it.Receipts)
		if err != nil {
			t.Fatalf("error reading receipts: %v", err)
		}
		if !bytes.Equal(receipts, chain.receipts[i]) {
			t.Fatalf("mismatched receipts: want %s, got %s", chain.receipts[i], receipts)
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

func genTestChain(t *testing.T) (*core.Genesis, []*types.Block, []types.Receipts) {
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	var (
		address = crypto.PubkeyToAddress(privateKey.PublicKey)
		genesis = &core.Genesis{
			Config: params.TestChainConfig,
			Alloc: types.GenesisAlloc{
				address: {
					Balance: big.NewInt(10000000000000000), // 10 ETH
				},
			},
			GasLimit:   1000000,
			Difficulty: big.NewInt(1),
		}
	)
	_, chain, receipts := core.GenerateChainWithGenesis(genesis, ethash.NewFaker(), 3, func(i int, gen *core.BlockGen) {
		// Add a transfer transaction
		to := common.HexToAddress("0x5678")
		tx := types.NewTx(&types.LegacyTx{
			Nonce:    uint64(i * 2),
			To:       &to,
			Value:    big.NewInt(100000000000000), // 0.1 ETH
			Gas:      21000,
			GasPrice: big.NewInt(1000000000), // 1 Gwei
		})
		signedTx, err := types.SignTx(tx, types.NewEIP155Signer(genesis.Config.ChainID), privateKey)
		require.NoError(t, err)
		gen.AddTx(signedTx)

		// Add a contract creation transaction
		tx = types.NewTx(&types.LegacyTx{
			Nonce:    uint64(i*2 + 1),
			Value:    big.NewInt(0),
			Gas:      100000,
			GasPrice: big.NewInt(1000000000), // 1 Gwei
			// Simple contract that returns empty data
			Data: []byte{
				0x60, 0x00, // PUSH1 0
				0x60, 0x00, // PUSH1 0
				0x52,       // MSTORE
				0x60, 0x20, // PUSH1 32
				0x60, 0x00, // PUSH1 0
				0xf3, // RETURN
			},
		})
		signedTx, err = types.SignTx(tx, types.NewEIP155Signer(genesis.Config.ChainID), privateKey)
		require.NoError(t, err)
		gen.AddTx(signedTx)
	})
	return genesis, chain, receipts
}

// exportChain creates a temporary era file with the given chain.
func exportChain(t *testing.T, genesis *core.Genesis, chain []*types.Block, receipts []types.Receipts) (string, string) {
	var (
		tmpDir   = t.TempDir()
		fileName = "test.era1"
	)
	tmpFile, err := os.Create(filepath.Join(tmpDir, fileName))
	require.NoError(t, err)

	builder := NewBuilder(tmpFile)
	// Add blocks to era
	for i, block := range chain {
		td := new(big.Int).Add(genesis.Difficulty, big.NewInt(int64(i+1)))
		headerData, err := rlp.EncodeToBytes(block.Header())
		require.NoError(t, err)
		bodyData, err := rlp.EncodeToBytes(block.Body())
		require.NoError(t, err)
		receiptsData, err := rlp.EncodeToBytes(receipts[i])
		require.NoError(t, err)

		err = builder.AddRLP(headerData, bodyData, receiptsData, block.NumberU64(), block.Hash(), td, block.Difficulty())
		require.NoError(t, err)
	}

	_, err = builder.Finalize()
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpDir, fileName
}

func TestEraFunctions(t *testing.T) {
	genesis, blocks, receipts := genTestChain(t)
	testdir, filename := exportChain(t, genesis, blocks, receipts)
	defer os.RemoveAll(testdir)
	// Open the era file
	era, err := Open(filepath.Join(testdir, filename))
	require.NoError(t, err)
	defer era.Close()

	t.Run("GetHeaderByNumber", func(t *testing.T) {
		header, err := era.GetHeaderByNumber(era.Start())
		require.NoError(t, err)
		haveJson, err := rlp.EncodeToBytes(header)
		require.NoError(t, err)
		wantJson, err := rlp.EncodeToBytes(blocks[0].Header())
		require.NoError(t, err)
		require.Equal(t, wantJson, haveJson)

		header, err = era.GetHeaderByNumber(era.Start() + era.Count() - 1)
		require.NoError(t, err)
		haveJson, err = rlp.EncodeToBytes(header)
		require.NoError(t, err)
		wantJson, err = rlp.EncodeToBytes(blocks[2].Header())
		require.NoError(t, err)
		require.Equal(t, wantJson, haveJson)
	})

	t.Run("GetBlockByNumber", func(t *testing.T) {
		block, err := era.GetBlockByNumber(era.Start())
		require.NoError(t, err)
		haveJson, err := rlp.EncodeToBytes(block)
		require.NoError(t, err)
		wantJson, err := rlp.EncodeToBytes(blocks[0])
		require.NoError(t, err)
		require.Equal(t, wantJson, haveJson)

		block, err = era.GetBlockByNumber(era.Start() + 1)
		require.NoError(t, err)
		haveJson, err = rlp.EncodeToBytes(block)
		require.NoError(t, err)
		wantJson, err = rlp.EncodeToBytes(blocks[1])
		require.NoError(t, err)
		require.Equal(t, wantJson, haveJson)
	})

	t.Run("GetReceipts", func(t *testing.T) {
		rcpts, err := era.GetReceipts(era.Start())
		require.NoError(t, err)
		require.Equal(t, 2, len(rcpts)) // Should have 2 receipts

		require.Equal(t, receipts[0][0].CumulativeGasUsed, rcpts[0].CumulativeGasUsed)
		require.Equal(t, receipts[0][1].CumulativeGasUsed, rcpts[1].CumulativeGasUsed)
		require.Equal(t, receipts[0][0].Status, rcpts[0].Status)
		require.Equal(t, receipts[0][1].Status, rcpts[0].Status)
	})

	t.Run("OutOfBounds", func(t *testing.T) {
		_, err := era.GetHeaderByNumber(era.Start() - 1)
		require.Error(t, err)
		require.Equal(t, "out-of-bounds", err.Error())

		_, err = era.GetHeaderByNumber(era.Start() + era.Count())
		require.Error(t, err)
		require.Equal(t, "out-of-bounds", err.Error())
	})
}

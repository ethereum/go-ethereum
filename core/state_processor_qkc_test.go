// Copyright 2026-2027 The QuarkChain Authors
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

package core

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func TestCheckDepositOrder(t *testing.T) {
	var (
		d = types.NewTx(&types.DepositTx{Value: new(uint256.Int), GasPrice: new(uint256.Int)})
		n = types.NewTx(&types.LegacyTx{GasPrice: new(big.Int), Value: new(big.Int)})
	)
	for _, tt := range []struct {
		txs types.Transactions
		ok  bool
	}{
		{nil, true},
		{types.Transactions{n, n}, true},
		{types.Transactions{d, d}, true},
		{types.Transactions{d, d, n, n}, true},
		{types.Transactions{n, d}, false},
		{types.Transactions{d, n, d}, false},
	} {
		var kinds []byte
		for _, tx := range tt.txs {
			kinds = append(kinds, tx.Type())
		}
		if err := checkDepositOrder(tt.txs); (err == nil) != tt.ok || (err != nil && !errors.Is(err, ErrDepositOrder)) {
			t.Errorf("tx types %x: error %v, want ok %v", kinds, err, tt.ok)
		}
	}
}

// TestDepositAfterTransaction checks that a block running a deposit after a
// transaction is invalid.
func TestDepositAfterTransaction(t *testing.T) {
	var (
		config = *params.MergedTestChainConfig
		engine = beacon.New(ethash.NewFaker())
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		sender = crypto.PubkeyToAddress(key.PublicKey)
		to     = common.HexToAddress("0xe0a")
	)
	config.QuarkChain = &params.QuarkChainConfig{ShardID: 1, ShardIDs: []uint32{0, 1}}
	gspec := &Genesis{
		Config: &config,
		Alloc:  withSystemContracts(types.GenesisAlloc{sender: {Balance: big.NewInt(params.Ether)}}),
	}
	signer := types.LatestSigner(gspec.Config)
	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		b.AddTx(types.MustSignNewTx(key, signer, &types.DynamicFeeTx{
			ChainID:   gspec.Config.ChainID,
			To:        &to,
			Value:     big.NewInt(1),
			Gas:       params.TxGas,
			GasFeeCap: newGwei(5),
			GasTipCap: big.NewInt(2),
		}))
		b.AddTx(types.NewTx(&types.DepositTx{
			From:     common.HexToAddress("0xa1"),
			To:       to,
			Value:    uint256.NewInt(1),
			GasPrice: uint256.NewInt(params.GWei),
			Gas:      params.XShardSendGas,
			IsMove:   true,
		}))
	})
	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	defer chain.Stop()
	if _, err := chain.InsertChain(blocks); !errors.Is(err, ErrDepositOrder) {
		t.Errorf("inserted a deposit after a transaction, error %v", err)
	}
}

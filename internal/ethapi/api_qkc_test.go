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

package ethapi

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// TestRPCDepositTransaction checks the RPC representation of a cross-shard
// deposit, and that clients decode it back to the deposit.
func TestRPCDepositTransaction(t *testing.T) {
	config := *params.MergedTestChainConfig
	config.QuarkChain = &params.QuarkChainConfig{ShardID: 1, ShardIDs: []uint32{0, 1}}
	deposit := &types.DepositTx{
		Version:         types.DepositTxVersion,
		SourceBlockHash: common.Hash{0xaa},
		SourceLogIndex:  3,
		From:            common.Address{0x01},
		To:              &common.Address{0x02},
		Value:           uint256.NewInt(1000),
		GasPrice:        uint256.NewInt(10),
		Gas:             50000,
		IsMove:          true,
		Data:            []byte{0xca, 0xfe},
	}
	tx := types.NewTx(deposit)
	// The gas price of the deposit is below the base fee.
	rpcTx := newRPCTransaction(tx, common.Hash{0xbb}, 1, 1, 0, big.NewInt(1000), &config)

	if rpcTx.From != deposit.From || rpcTx.GasPrice.ToInt().Cmp(big.NewInt(10)) != 0 || uint64(rpcTx.Type) != types.DepositTxType {
		t.Errorf("sender %v, gas price %v, type %d, want the ones of the deposit", rpcTx.From, rpcTx.GasPrice, rpcTx.Type)
	}
	if rpcTx.Version == nil || uint64(*rpcTx.Version) != uint64(deposit.Version) ||
		rpcTx.SourceBlockHash == nil || *rpcTx.SourceBlockHash != deposit.SourceBlockHash ||
		rpcTx.SourceLogIndex == nil || uint64(*rpcTx.SourceLogIndex) != deposit.SourceLogIndex ||
		rpcTx.IsMove == nil || !*rpcTx.IsMove {
		t.Errorf("deposit fields: version %v, source block %v, log index %v, move %v", rpcTx.Version, rpcTx.SourceBlockHash, rpcTx.SourceLogIndex, rpcTx.IsMove)
	}
	if rpcTx.ChainID != nil || rpcTx.GasFeeCap != nil || rpcTx.YParity != nil {
		t.Errorf("unexpected fields: chain id %v, fee cap %v, y parity %v", rpcTx.ChainID, rpcTx.GasFeeCap, rpcTx.YParity)
	}
	enc, err := json.Marshal(rpcTx)
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}
	dec := new(types.Transaction)
	if err := json.Unmarshal(enc, dec); err != nil {
		t.Fatalf("failed to decode %s: %v", enc, err)
	}
	if dec.Hash() != tx.Hash() {
		t.Errorf("decoded transaction %v, want %v", dec.Hash(), tx.Hash())
	}
}

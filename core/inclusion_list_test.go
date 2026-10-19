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

package core

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// newInclusionListTestEnv returns an Amsterdam test chain with Bogota active
// from genesis.
func newInclusionListTestEnv(extra types.GenesisAlloc) *balTestEnv {
	env := newBALTestEnv(extra)
	env.cfg.BogotaTime = new(uint64)
	return env
}

func inclusionListTestKey(t *testing.T, name string) (*ecdsa.PrivateKey, common.Address) {
	t.Helper()
	key, err := crypto.ToECDSA(crypto.Keccak256([]byte(name)))
	if err != nil {
		t.Fatal(err)
	}
	return key, crypto.PubkeyToAddress(key.PublicKey)
}

func signInclusionListTx(t *testing.T, signer types.Signer, key *ecdsa.PrivateKey, nonce, gas uint64, feeCap *big.Int, value int64) *types.Transaction {
	t.Helper()
	tx, err := types.SignNewTx(key, signer, &types.DynamicFeeTx{
		ChainID:   signer.ChainID(),
		Nonce:     nonce,
		GasTipCap: common.Big0,
		GasFeeCap: feeCap,
		Gas:       gas,
		To:        &common.Address{0xaa},
		Value:     big.NewInt(value),
	})
	if err != nil {
		t.Fatal(err)
	}
	return tx
}

func encodeInclusionList(t *testing.T, txs ...*types.Transaction) [][]byte {
	t.Helper()
	list := make([][]byte, len(txs))
	for i, tx := range txs {
		enc, err := tx.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		list[i] = enc
	}
	return list
}

// TestCheckInclusionList checks which inclusion list transactions count as
// ones the block could have included, against a fixed post-transaction state.
func TestCheckInclusionList(t *testing.T) {
	env := newInclusionListTestEnv(nil)
	bc, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, beacon.New(ethash.NewFaker()), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer bc.Stop()

	var (
		signer = types.LatestSigner(env.cfg)
		feeCap = big.NewInt(2 * params.InitialBaseFee)
		header = &types.Header{
			Number:        big.NewInt(1),
			GasLimit:      60_000_000,
			BaseFee:       big.NewInt(params.InitialBaseFee),
			Difficulty:    common.Big0,
			ExcessBlobGas: new(uint64),
		}
		bobKey, bob   = inclusionListTestKey(t, "bob")
		daveKey, dave = inclusionListTestKey(t, "dave")
	)
	blob, err := types.SignNewTx(bobKey, signer, &types.BlobTx{
		ChainID:    uint256.MustFromBig(env.cfg.ChainID),
		GasFeeCap:  uint256.MustFromBig(feeCap),
		Gas:        21000,
		To:         common.Address{0xaa},
		BlobFeeCap: uint256.NewInt(params.InitialBaseFee),
		BlobHashes: []common.Hash{{0x01}},
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name              string
		list              [][]byte
		gasExec, gasState uint64
		want              bool
	}{
		{name: "undecodable", list: [][]byte{{0x02, 0xc0}}, want: true},
		{name: "blob transaction", list: encodeInclusionList(t, blob), want: true},

		// The check is the block's own transaction validity, not just the
		// nonce and balance check of the EIP text.
		{name: "fee cap below base fee", list: encodeInclusionList(t, signInclusionListTx(t, signer, bobKey, 0, 21000, big.NewInt(params.InitialBaseFee-1), 1)), want: true},
		{name: "gas below intrinsic", list: encodeInclusionList(t, signInclusionListTx(t, signer, bobKey, 0, 20999, feeCap, 1)), want: true},

		// Each gas dimension has its own room. The execution reservation is
		// capped at MaxTxGas, so 30M gas fits after 40M of a 60M block.
		{name: "room in both dimensions", list: encodeInclusionList(t, signInclusionListTx(t, signer, bobKey, 0, 30_000_000, feeCap, 1)), gasExec: 40_000_000, gasState: 10_000_000, want: false},

		// The first transaction buys gas before failing the intrinsic gas
		// check. The second one needs all of dave's balance, so it only
		// passes if that purchase was undone.
		{name: "checks are independent", list: encodeInclusionList(t,
			signInclusionListTx(t, signer, daveKey, 0, 20000, feeCap, 1),
			signInclusionListTx(t, signer, daveKey, 0, 21000, feeCap, 1),
		), want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
			if err != nil {
				t.Fatal(err)
			}
			statedb.SetBalance(bob, uint256.NewInt(params.Ether), tracing.BalanceChangeUnspecified)
			statedb.SetBalance(dave, uint256.NewInt(21000*2*params.InitialBaseFee+1), tracing.BalanceChangeUnspecified)

			gp := &GasPool{initial: header.GasLimit, cumulativeExecution: tc.gasExec, cumulativeState: tc.gasState}
			if got := checkInclusionList(env.cfg, bc, header, nil, statedb, gp, tc.list); got != tc.want {
				t.Errorf("satisfied = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestCheckInclusionListBlockState checks that a block's inclusion list is
// checked against the state after its last transaction and before its
// withdrawals.
func TestCheckInclusionListBlockState(t *testing.T) {
	var (
		aliceKey, alice = inclusionListTestKey(t, "alice")
		carolKey, carol = inclusionListTestKey(t, "carol")
		env             = newInclusionListTestEnv(types.GenesisAlloc{
			alice: {Balance: big.NewInt(10_000_000_000_000)},
			carol: {Balance: big.NewInt(params.Ether)},
		})
		engine = beacon.New(ethash.NewFaker())
		signer = types.LatestSigner(env.cfg)
		feeCap = big.NewInt(2 * params.InitialBaseFee)
	)
	_, blocks, _ := GenerateChainWithGenesis(env.gspec, engine, 1, func(_ int, b *BlockGen) {
		b.AddTx(signInclusionListTx(t, signer, carolKey, 0, 21000, feeCap, 1))
		// 0.01 ether, enough for alice's transaction below
		b.AddWithdrawal(&types.Withdrawal{Validator: 1, Address: alice, Amount: 10_000_000})
	})
	bc, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, engine, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer bc.Stop()
	if _, err := bc.InsertChain(blocks); err != nil {
		t.Fatal(err)
	}
	block := blocks[0]

	for _, tc := range []struct {
		name string
		tx   *types.Transaction
		want bool
	}{
		// Alice can only pay with the withdrawal, which isn't applied yet.
		{"before withdrawals", signInclusionListTx(t, signer, aliceKey, 0, 21000, feeCap, 1_000_000_000_000_000), true},
		// The block's last transaction already used carol's nonce 0.
		{"after last transaction", signInclusionListTx(t, signer, carolKey, 1, 21000, feeCap, 1), false},
	} {
		got, err := bc.CheckInclusionList(block, block.GasUsed(), block.GasUsed(), encodeInclusionList(t, tc.tx))
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got != tc.want {
			t.Errorf("%s: satisfied = %v, want %v", tc.name, got, tc.want)
		}
	}
}

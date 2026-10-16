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
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// qkcTestEngine is the beacon engine without header verification, which
// rejects the QuarkChain extra data as too long.
type qkcTestEngine struct {
	consensus.Engine
}

func (qkcTestEngine) VerifyHeader(consensus.ChainHeaderReader, *types.Header) error {
	return nil
}

func (qkcTestEngine) VerifyHeaders(_ consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
	abort, results := make(chan struct{}), make(chan error, len(headers))
	for range headers {
		results <- nil
	}
	return abort, results
}

// testXShardInbox serves fixed deposits, and records its query.
type testXShardInbox struct {
	deposits []*types.DepositTx

	parent        common.Hash
	prevRootBlock common.Hash
}

func (in *testXShardInbox) Deposits(parent *types.Header, prevRootBlock common.Hash) (XShardDeposits, error) {
	in.parent, in.prevRootBlock = parent.Hash(), prevRootBlock
	return &testXShardDeposits{deposits: in.deposits}, nil
}

// testXShardDeposits walks fixed deposits, with the cursor testCursor.
type testXShardDeposits struct {
	deposits []*types.DepositTx
	walked   int
	ended    bool
}

func (d *testXShardDeposits) Next() (*types.DepositTx, error) {
	if d.walked == len(d.deposits) {
		d.ended = true
		return nil, nil
	}
	d.walked++
	return d.deposits[d.walked-1], nil
}

func (d *testXShardDeposits) Cursor() types.XShardCursor {
	return testCursor(d.walked, d.ended)
}

// testCursor returns the cursor of testXShardDeposits once the given number of
// deposits are walked, or at the end.
func testCursor(walked int, ended bool) types.XShardCursor {
	if ended {
		return types.XShardCursor{RootBlockHeight: 6}
	}
	return types.XShardCursor{RootBlockHeight: 5, MinorBlockIndex: 1, XShardDepositIndex: uint64(walked)}
}

// TestValidateXShardDeposits imports blocks of a shard, with a gas limit of
// 100000 leaving 50000 gas to deposits, that is five of them, each block
// following or breaking a rule of the cross-shard deposits.
func TestValidateXShardDeposits(t *testing.T) {
	var (
		key, _   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		sender   = crypto.PubkeyToAddress(key.PublicKey)
		to       = common.HexToAddress("0xe0a")
		prevRoot = common.Hash{0x77}
		gasLimit = uint64(100_000)
	)
	deposit := func(index int) *types.DepositTx {
		return &types.DepositTx{
			Version:         types.DepositTxVersion,
			SourceBlockHash: common.Hash{0x50},
			SourceLogIndex:  uint64(index),
			From:            common.HexToAddress("0xa1"),
			To:              &to,
			Value:           uint256.NewInt(uint64(index) + 1),
			GasPrice:        uint256.NewInt(params.GWei),
			Gas:             params.XShardSendGas,
			IsMove:          true,
		}
	}
	deposits := func(n int) []*types.DepositTx {
		list := make([]*types.DepositTx, n)
		for i := range list {
			list[i] = deposit(i)
		}
		return list
	}
	changed := deposits(2)
	changed[1].Value = uint256.NewInt(100)

	for _, tt := range []struct {
		name      string
		available int                // deposits of the inbox
		deposits  []*types.DepositTx // deposits of the block
		txs       int                // transfers after the deposits
		cursor    types.XShardCursor
		edit      func(extra *types.QKCExtra)
		noInbox   bool
		want      error
	}{
		{name: "as many as fit", available: 6, deposits: deposits(5), txs: 2, cursor: testCursor(5, false)},
		{name: "all when they fit", available: 2, deposits: deposits(2), cursor: testCursor(2, true)},
		// The 63000 gas of three transfers exceed the 50000 the deposits
		// leave, but not the gas limit.
		{name: "deposit gas left to transactions", available: 2, deposits: deposits(2), txs: 3, cursor: testCursor(2, true)},
		{name: "fewer than fit", available: 6, deposits: deposits(4), cursor: testCursor(4, false), want: ErrXShardGas},
		{name: "fewer than left", available: 2, deposits: deposits(1), cursor: testCursor(1, false), want: ErrXShardGas},
		{name: "more than fit", available: 6, deposits: deposits(6), cursor: testCursor(6, true), want: ErrXShardGas},
		{name: "deposit changed", available: 2, deposits: changed, cursor: testCursor(2, true), want: ErrXShardDepositMismatch},
		{name: "deposit added", available: 2, deposits: deposits(3), cursor: testCursor(3, true), want: ErrXShardDepositMismatch},
		{name: "cursor not at the end", available: 2, deposits: deposits(2), cursor: testCursor(2, false), want: ErrXShardCursorMismatch},
		{name: "cursor changed", available: 6, deposits: deposits(5), cursor: testCursor(4, false), want: ErrXShardCursorMismatch},
		{name: "gas used changed", available: 2, deposits: deposits(2), cursor: testCursor(2, true),
			edit: func(e *types.QKCExtra) { e.XShardGasUsed = params.XShardSendGas }, want: ErrXShardGas},
		{name: "gas limit changed", available: 2, deposits: deposits(2), cursor: testCursor(2, true),
			edit: func(e *types.QKCExtra) { e.XShardGasLimit = gasLimit/2 - 1 }, want: ErrXShardGas},
		{name: "unchecked without inbox", available: 2, deposits: deposits(1), noInbox: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			config := *params.MergedTestChainConfig
			config.QuarkChain = &params.QuarkChainConfig{ShardID: 1, ShardIDs: []uint32{0, 1}}
			gspec := &Genesis{
				Config:    &config,
				GasLimit:  gasLimit,
				ExtraData: (&types.QKCExtra{}).Encode(),
				Alloc:     withSystemContracts(types.GenesisAlloc{sender: {Balance: big.NewInt(params.Ether)}}),
			}
			engine := qkcTestEngine{beacon.New(ethash.NewFaker())}
			extra := types.QKCExtra{
				PrevRootBlock:  prevRoot,
				Cursor:         tt.cursor,
				XShardGasUsed:  uint64(len(tt.deposits)) * params.XShardSendGas,
				XShardGasLimit: gasLimit / 2,
			}
			if tt.edit != nil {
				tt.edit(&extra)
			}
			signer := types.LatestSigner(gspec.Config)
			_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
				b.SetExtra(extra.Encode())
				for _, d := range tt.deposits {
					b.AddTx(types.NewTx(d))
				}
				for nonce := range tt.txs {
					b.AddTx(types.MustSignNewTx(key, signer, &types.DynamicFeeTx{
						ChainID:   gspec.Config.ChainID,
						Nonce:     uint64(nonce),
						To:        &to,
						Value:     big.NewInt(1),
						Gas:       params.TxGas,
						GasFeeCap: newGwei(5),
						GasTipCap: big.NewInt(2),
					}))
				}
			})
			chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
			if err != nil {
				t.Fatalf("failed to create tester chain: %v", err)
			}
			defer chain.Stop()
			inbox := &testXShardInbox{deposits: deposits(tt.available)}
			if !tt.noInbox {
				chain.SetXShardInbox(inbox)
			}
			_, err = chain.InsertChain(blocks)
			if tt.want == nil && err != nil {
				t.Fatalf("failed to insert a valid block: %v", err)
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("insert error %v, want %v", err, tt.want)
			}
			// The inbox walks from the parent up to the previous root block.
			if tt.noInbox || extra.XShardGasLimit != gasLimit/2 {
				return
			}
			if inbox.parent != chain.Genesis().Hash() || inbox.prevRootBlock != prevRoot {
				t.Errorf("inbox queried from %x up to %x, want %x and %x", inbox.parent, inbox.prevRootBlock, chain.Genesis().Hash(), prevRoot)
			}
		})
	}
}

// Copyright 2026-2027 The quarkchain Authors
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
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// xShardTestMove returns a move of the test, as sent by a source shard.
func xShardTestMove(logIndex uint64, from, to common.Address, value, gasPrice uint64) *types.DepositTx {
	return &types.DepositTx{
		Version:         types.DepositTxVersion,
		SourceBlockHash: common.Hash{0x50},
		SourceLogIndex:  logIndex,
		From:            from,
		To:              &to,
		Value:           uint256.NewInt(value),
		GasPrice:        uint256.NewInt(gasPrice),
		Gas:             params.XShardSendGas,
		IsMove:          true,
	}
}

// TestDepositExecution runs cross-shard deposits at the start of a block of a
// shard, followed by a transaction of the sender of one of them, and checks
// that each deposit credits its value to its recipient without running any
// code, and pays its gas to the coinbase.
func TestDepositExecution(t *testing.T) {
	t.Run("pre-amsterdam", func(t *testing.T) { testDepositExecution(t, false) })
	t.Run("amsterdam", func(t *testing.T) { testDepositExecution(t, true) })
}

func testDepositExecution(t *testing.T, amsterdam bool) {
	var (
		config   = *params.MergedTestChainConfig
		engine   = beacon.New(ethash.NewFaker())
		key, _   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		sender   = crypto.PubkeyToAddress(key.PublicKey)
		coinbase = common.HexToAddress("0xc0ffee")
		eoa      = common.HexToAddress("0xe0a")
		storer   = common.HexToAddress("0x5703e") // stores the call value at slot 0
		reverter = common.HexToAddress("0x7e7e7")
	)
	if amsterdam {
		config.AmsterdamTime = new(uint64)
	}
	ethConfig := config
	config.QuarkChain = &params.QuarkChainConfig{ShardID: 1, ShardIDs: []uint32{0, 1}}
	gspec := &Genesis{
		Config: &config,
		Alloc: withSystemContracts(types.GenesisAlloc{
			sender:   {Balance: big.NewInt(params.Ether)},
			storer:   {Code: program.New().Op(vm.CALLVALUE).Push(0).Op(vm.SSTORE).Bytes()},
			reverter: {Code: program.New().Push(0).Push(0).Op(vm.REVERT).Bytes()},
		}),
	}
	deposits := []*types.DepositTx{
		// A move to an EOA.
		xShardTestMove(0, common.HexToAddress("0xa1"), eoa, 1000, 7*params.GWei),
		// A move to a contract, at a gas price below the base fee.
		xShardTestMove(1, common.HexToAddress("0xa2"), storer, 500, 1),
		// A move to a contract that reverts any call.
		xShardTestMove(2, common.HexToAddress("0xa3"), reverter, 700, 3*params.GWei),
		// A move from the sender of the transaction below.
		xShardTestMove(3, sender, eoa, 1, params.GWei),
	}
	signer := types.LatestSigner(gspec.Config)
	_, blocks, receipts := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		b.SetCoinbase(coinbase)
		for _, d := range deposits {
			b.AddTx(types.NewTx(d))
		}
		// The deposit from the sender leaves its nonce alone.
		b.AddTx(types.MustSignNewTx(key, signer, &types.DynamicFeeTx{
			ChainID:   gspec.Config.ChainID,
			Nonce:     0,
			To:        &eoa,
			Value:     big.NewInt(1),
			Gas:       params.TxGas,
			GasFeeCap: newGwei(5),
			GasTipCap: big.NewInt(2),
		}))
	})
	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	defer chain.Stop()
	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	}
	statedb, _ := chain.State()

	var (
		block       = blocks[0]
		coinbaseFee = new(big.Int)
		received    = make(map[common.Address]*big.Int)
	)
	for i, d := range deposits {
		r := receipts[0][i]
		if r.Type != types.DepositTxType || r.Status != types.ReceiptStatusSuccessful || len(r.Logs) != 0 {
			t.Errorf("deposit %d: receipt of type %#x with status %d and %d logs, want a successful deposit without logs", i, r.Type, r.Status, len(r.Logs))
		}
		if want := uint64(i+1) * params.XShardSendGas; r.GasUsed != params.XShardSendGas || r.CumulativeGasUsed != want {
			t.Errorf("deposit %d: gas used %d, cumulative %d, want %d and %d", i, r.GasUsed, r.CumulativeGasUsed, params.XShardSendGas, want)
		}
		gasPrice := d.GasPrice.ToBig()
		if r.EffectiveGasPrice.Cmp(gasPrice) != 0 {
			t.Errorf("deposit %d: effective gas price %v, want %v", i, r.EffectiveGasPrice, gasPrice)
		}
		coinbaseFee.Add(coinbaseFee, new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(params.XShardSendGas)))
		if received[*d.To] == nil {
			received[*d.To] = new(big.Int)
		}
		received[*d.To].Add(received[*d.To], d.Value.ToBig())

		// The sender is not touched, the source shard paid for the deposit.
		if d.From != sender && statedb.Exist(d.From) {
			t.Errorf("deposit %d: sender exists with balance %v", i, statedb.GetBalance(d.From))
		}
	}
	// The coinbase is paid all the gas of the deposits, at their gas price, and
	// the tip of the transaction.
	tx := receipts[0][len(deposits)]
	tip := new(big.Int).Sub(tx.EffectiveGasPrice, block.BaseFee())
	coinbaseFee.Add(coinbaseFee, tip.Mul(tip, new(big.Int).SetUint64(tx.GasUsed)))
	if have := statedb.GetBalance(coinbase).ToBig(); have.Cmp(coinbaseFee) != 0 {
		t.Errorf("coinbase balance %v, want %v", have, coinbaseFee)
	}
	if tx.Status != types.ReceiptStatusSuccessful || statedb.GetNonce(sender) != 1 {
		t.Errorf("transaction after the deposit: status %d, sender nonce %d", tx.Status, statedb.GetNonce(sender))
	}
	wantSender := new(big.Int).Sub(big.NewInt(params.Ether), big.NewInt(1))
	wantSender.Sub(wantSender, new(big.Int).Mul(tx.EffectiveGasPrice, new(big.Int).SetUint64(tx.GasUsed)))
	if have := statedb.GetBalance(sender).ToBig(); have.Cmp(wantSender) != 0 {
		t.Errorf("sender balance %v, want %v", have, wantSender)
	}
	// The recipients got the values, the contracts without running their code.
	received[eoa].Add(received[eoa], big.NewInt(1))
	for addr, want := range received {
		if have := statedb.GetBalance(addr).ToBig(); have.Cmp(want) != 0 {
			t.Errorf("recipient %v: balance %v, want %v", addr, have, want)
		}
	}
	if have := statedb.GetState(storer, common.Hash{}); have != (common.Hash{}) {
		t.Errorf("contract code ran, stored %v", have.Big())
	}
	if block.GasUsed() != receipts[0][len(receipts[0])-1].CumulativeGasUsed {
		t.Errorf("block gas used %d, want %d", block.GasUsed(), receipts[0][len(receipts[0])-1].CumulativeGasUsed)
	}

	// A chain that is not a QuarkChain shard rejects the deposits.
	ethChain, err := NewBlockChain(rawdb.NewMemoryDatabase(), &Genesis{Config: &ethConfig, Alloc: gspec.Alloc}, engine, nil)
	if err != nil {
		t.Fatalf("failed to create Ethereum chain: %v", err)
	}
	defer ethChain.Stop()
	if _, err := ethChain.InsertChain(blocks); !errors.Is(err, ErrTxTypeNotSupported) {
		t.Errorf("Ethereum chain inserted deposits, error %v", err)
	}
}

// TestDepositNotMove checks that a deposit other than a move, which no source
// shard sends, is a consensus error that leaves the state alone.
func TestDepositNotMove(t *testing.T) {
	var (
		config   = *params.MergedTestChainConfig
		coinbase = common.HexToAddress("0xc0ffee")
		to       = common.HexToAddress("0xe0a")
	)
	config.QuarkChain = &params.QuarkChainConfig{ShardID: 1, ShardIDs: []uint32{0, 1}}
	header := &types.Header{Number: big.NewInt(1), Difficulty: new(big.Int), BaseFee: big.NewInt(params.GWei), GasLimit: 30_000_000}

	for _, tt := range []struct {
		name string
		edit func(d *types.DepositTx)
		want error
	}{
		{"move", func(d *types.DepositTx) {}, nil},
		{"not a move", func(d *types.DepositTx) { d.IsMove = false }, ErrDepositNotMove},
		{"call", func(d *types.DepositTx) { d.IsMove, d.Gas, d.Data = false, 50000, []byte{1} }, ErrDepositNotMove},
		{"move with data", func(d *types.DepositTx) { d.Data = []byte{1} }, ErrDepositNotMove},
		{"move with more gas", func(d *types.DepositTx) { d.Gas++ }, ErrDepositNotMove},
		{"move without a recipient", func(d *types.DepositTx) { d.To = nil }, ErrDepositNotMove},
	} {
		statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		evm := vm.NewEVM(NewEVMBlockContext(header, nil, &coinbase), statedb, &config, vm.Config{})
		deposit := xShardTestMove(0, common.HexToAddress("0xa1"), to, 1000, params.GWei)
		tt.edit(deposit)
		msg, err := TransactionToMessage(types.NewTx(deposit), types.LatestSigner(&config), header.BaseFee)
		if err != nil {
			t.Fatalf("%s: failed to convert: %v", tt.name, err)
		}
		gp := NewGasPool(header.GasLimit)
		res, err := ApplyMessage(evm, msg, gp)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: error %v, want %v", tt.name, err, tt.want)
		}
		if tt.want == nil {
			if res.Failed() || res.UsedGas != params.XShardSendGas || gp.CumulativeUsed() != params.XShardSendGas {
				t.Errorf("%s: failed %v, gas used %d, block gas used %d", tt.name, res.Failed(), res.UsedGas, gp.CumulativeUsed())
			}
			continue
		}
		if statedb.Exist(to) || statedb.Exist(coinbase) || gp.CumulativeUsed() != 0 {
			t.Errorf("%s: state or block gas changed by an invalid deposit", tt.name)
		}
	}
}

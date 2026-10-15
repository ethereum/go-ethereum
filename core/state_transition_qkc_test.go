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
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// xShardTestDeposit is a deposit of the test, with its expected outcome.
type xShardTestDeposit struct {
	*types.DepositTx
	ok      bool   // whether the call succeeds
	gasUsed uint64 // the gas it uses, 0 if not known in advance
}

// TestDepositExecution runs cross-shard deposits at the start of a block of a
// shard, followed by a transaction of the sender of one of them, and checks how
// the value and gas of each deposit are settled.
func TestDepositExecution(t *testing.T) {
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
	config.QuarkChain = &params.QuarkChainConfig{ShardID: 1, ShardIDs: []uint32{0, 1}}
	gspec := &Genesis{
		Config: &config,
		Alloc: withSystemContracts(types.GenesisAlloc{
			sender:   {Balance: big.NewInt(params.Ether)},
			storer:   {Code: program.New().Op(vm.CALLVALUE).Push(0).Op(vm.SSTORE).Bytes()},
			reverter: {Code: program.New().Push(0).Push(0).Op(vm.REVERT).Bytes()},
		}),
	}
	deposit := func(logIndex uint64, from, to common.Address, value, gasPrice, gas uint64, data []byte) *types.DepositTx {
		return &types.DepositTx{
			Version:         types.DepositTxVersion,
			SourceBlockHash: common.Hash{0x50},
			SourceLogIndex:  logIndex,
			From:            from,
			To:              to,
			Value:           uint256.NewInt(value),
			GasPrice:        uint256.NewInt(gasPrice),
			Gas:             gas,
			Data:            data,
		}
	}
	deposits := []xShardTestDeposit{
		// A transfer, using only the intrinsic gas.
		{deposit(0, common.HexToAddress("0xa1"), eoa, 1000, 7*params.GWei, 30000, nil), true, params.XShardSendGas},
		// A contract call, at a gas price below the base fee.
		{deposit(1, common.HexToAddress("0xa2"), storer, 500, 1, 100_000, nil), true, 0},
		// A reverted call, leaving its value to the sender.
		{deposit(2, common.HexToAddress("0xa3"), reverter, 700, 3*params.GWei, 50000, []byte{0x01}), false, 0},
		// A deposit failing to pay for its calldata, using all of its gas.
		{deposit(3, common.HexToAddress("0xa4"), eoa, 900, 2*params.GWei, 10000, bytes.Repeat([]byte{0xff}, 100)), false, 10000},
		// A deposit to the sender of the transaction below, with exactly the intrinsic gas.
		{deposit(4, sender, eoa, 1, params.GWei, params.XShardSendGas, nil), true, params.XShardSendGas},
	}
	signer := types.LatestSigner(gspec.Config)
	_, blocks, receipts := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		b.SetCoinbase(coinbase)
		for _, d := range deposits {
			b.AddTx(types.NewTx(d.DepositTx))
		}
		// The deposit to the sender leaves its nonce alone.
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
		cumulative  uint64
		eoaReceived = new(big.Int)
	)
	for i, d := range deposits {
		r := receipts[0][i]
		cumulative += r.GasUsed
		if r.Type != types.DepositTxType || (r.Status == types.ReceiptStatusSuccessful) != d.ok || r.CumulativeGasUsed != cumulative {
			t.Errorf("deposit %d: receipt of type %#x with status %d and cumulative gas %d, want success %v and %d",
				i, r.Type, r.Status, r.CumulativeGasUsed, d.ok, cumulative)
		}
		if d.gasUsed != 0 && r.GasUsed != d.gasUsed {
			t.Errorf("deposit %d: gas used %d, want %d", i, r.GasUsed, d.gasUsed)
		}
		gasPrice := d.GasPrice.ToBig()
		if r.EffectiveGasPrice.Cmp(gasPrice) != 0 {
			t.Errorf("deposit %d: effective gas price %v, want %v", i, r.EffectiveGasPrice, gasPrice)
		}
		// The sender keeps the gas left, and the value if the call fails.
		want := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(d.Gas-r.GasUsed))
		if !d.ok {
			want.Add(want, d.Value.ToBig())
		} else if d.To == eoa {
			eoaReceived.Add(eoaReceived, d.Value.ToBig())
		}
		if d.From == sender {
			continue // checked with the transaction below
		}
		if have := statedb.GetBalance(d.From).ToBig(); have.Cmp(want) != 0 {
			t.Errorf("deposit %d: sender balance %v, want %v", i, have, want)
		}
		if nonce := statedb.GetNonce(d.From); nonce != 0 {
			t.Errorf("deposit %d: sender nonce %d, want 0", i, nonce)
		}
		coinbaseFee.Add(coinbaseFee, new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(r.GasUsed)))
	}
	// The coinbase is paid all the gas used by deposits, at their gas price,
	// and the tip of the transaction.
	senderDeposit, tx := receipts[0][len(deposits)-1], receipts[0][len(deposits)]
	coinbaseFee.Add(coinbaseFee, new(big.Int).Mul(big.NewInt(params.GWei), new(big.Int).SetUint64(senderDeposit.GasUsed)))
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
	// The recipients got the values of the successful calls only.
	if have := statedb.GetBalance(eoa).ToBig(); have.Cmp(eoaReceived.Add(eoaReceived, big.NewInt(1))) != 0 {
		t.Errorf("transfer recipient balance %v, want %v", have, eoaReceived)
	}
	if have := statedb.GetState(storer, common.Hash{}); have != common.BigToHash(big.NewInt(500)) {
		t.Errorf("stored call value %v, want 500", have.Big())
	}
	if have := statedb.GetBalance(reverter); !have.IsZero() {
		t.Errorf("reverted recipient balance %v, want 0", have)
	}
	if block.GasUsed() != receipts[0][len(receipts[0])-1].CumulativeGasUsed {
		t.Errorf("block gas used %d, want %d", block.GasUsed(), receipts[0][len(receipts[0])-1].CumulativeGasUsed)
	}

	// A chain that is not a QuarkChain shard rejects the deposits.
	ethConfig := *params.MergedTestChainConfig
	ethChain, err := NewBlockChain(rawdb.NewMemoryDatabase(), &Genesis{Config: &ethConfig, Alloc: gspec.Alloc}, engine, nil)
	if err != nil {
		t.Fatalf("failed to create Ethereum chain: %v", err)
	}
	defer ethChain.Stop()
	if _, err := ethChain.InsertChain(blocks); !errors.Is(err, ErrTxTypeNotSupported) {
		t.Errorf("Ethereum chain inserted deposits, error %v", err)
	}
}

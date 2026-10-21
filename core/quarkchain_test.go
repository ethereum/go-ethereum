// Copyright 2026 The quarkchain Authors
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
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// TestXShardOutbox sends cross-shard messages from an EOA in a block, and
// checks that the gas price of the message is the effective one of the
// transaction and that exactly the message's cost is burnt.
func TestXShardOutbox(t *testing.T) {
	var (
		config = *params.MergedTestChainConfig
		engine = beacon.New(ethash.NewFaker())
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		sender = crypto.PubkeyToAddress(key.PublicKey)
		funds  = big.NewInt(params.Ether)
		outbox = params.XShardOutboxAddress

		value    = big.NewInt(12345)
		gasLimit = params.XShardSendGas
		excess   = big.NewInt(1)
	)
	config.QuarkChain = &params.QuarkChainConfig{ShardID: 0, ShardIDs: []uint32{0, 1}}
	sendABI, err := abi.JSON(strings.NewReader(`[{"type": "function", "name": "send", "stateMutability": "payable", "outputs": [], "inputs": [
		{"name": "dstShard", "type": "uint32"}, {"name": "to", "type": "address"}, {"name": "value", "type": "uint256"},
		{"name": "gasLimit", "type": "uint64"}, {"name": "isMove", "type": "bool"}, {"name": "data", "type": "bytes"}]}]`))
	if err != nil {
		t.Fatal(err)
	}
	gspec := &Genesis{
		Config: &config,
		Alloc:  withSystemContracts(types.GenesisAlloc{sender: {Balance: funds}}),
	}
	signer := types.LatestSigner(gspec.Config)

	// The first message is valid and overpays, the second one goes to an
	// unknown shard and fails.
	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		for nonce, dst := range []uint32{1, 2} {
			input, err := sendABI.Pack("send", dst, common.HexToAddress("0xdead"), value, gasLimit, true, []byte{})
			if err != nil {
				t.Fatal(err)
			}
			// The message cost is only known at inclusion, so pay for the fee cap.
			callValue := new(big.Int).Mul(new(big.Int).SetUint64(gasLimit), newGwei(5))
			callValue.Add(callValue, value)
			b.AddTx(types.MustSignNewTx(key, signer, &types.DynamicFeeTx{
				ChainID:   gspec.Config.ChainID,
				Nonce:     uint64(nonce),
				To:        &outbox,
				Value:     callValue.Add(callValue, excess),
				Gas:       100_000,
				GasFeeCap: newGwei(5),
				GasTipCap: big.NewInt(2),
				Data:      input,
			}))
		}
	})
	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	defer chain.Stop()
	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	}

	receipts := chain.GetReceiptsByHash(blocks[0].Hash())
	if len(receipts) != 2 {
		t.Fatalf("got %d receipts, want 2", len(receipts))
	}
	sent, failed := receipts[0], receipts[1]
	if sent.Status != types.ReceiptStatusSuccessful || len(sent.Logs) != 1 {
		t.Fatalf("valid message: status %d with %d logs, want success with 1 log", sent.Status, len(sent.Logs))
	}
	log := sent.Logs[0]
	if log.Address != outbox || log.Topics[0] != vm.XShardSentMessageTopic {
		t.Fatalf("log from %v with topic %v, want SentMessage from the outbox", log.Address, log.Topics[0])
	}
	// The data starts with the value and the gas price of the message.
	gasPrice := sent.EffectiveGasPrice
	if have := new(big.Int).SetBytes(log.Data[32:64]); have.Cmp(gasPrice) != 0 {
		t.Errorf("message gas price %v, want effective gas price %v", have, gasPrice)
	}
	if failed.Status != types.ReceiptStatusFailed || len(failed.Logs) != 0 || failed.GasUsed != 100_000 {
		t.Errorf("invalid message: status %d with %d logs using %d gas, want failure without logs using all gas",
			failed.Status, len(failed.Logs), failed.GasUsed)
	}

	// The sender pays the gas of both transactions and the cost of the valid
	// message, the excess comes back.
	statedb, _ := chain.State()
	want := new(big.Int).Set(funds)
	want.Sub(want, new(big.Int).Mul(new(big.Int).SetUint64(sent.GasUsed), gasPrice))
	want.Sub(want, new(big.Int).Mul(new(big.Int).SetUint64(failed.GasUsed), failed.EffectiveGasPrice))
	want.Sub(want, value)
	want.Sub(want, new(big.Int).Mul(new(big.Int).SetUint64(gasLimit), gasPrice))
	if have := statedb.GetBalance(sender).ToBig(); have.Cmp(want) != 0 {
		t.Errorf("sender balance %v, want %v", have, want)
	}
	if statedb.Exist(outbox) {
		t.Errorf("outbox account exists with balance %v", statedb.GetBalance(outbox))
	}
}

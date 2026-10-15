// Copyright 2026-2027 The QuarkChain Authors
// This file is part of the QuarkChain library.
//
// The QuarkChain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The QuarkChain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the QuarkChain library. If not, see <http://www.gnu.org/licenses/>.

package shard

import (
	"bytes"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// xShardSend is a move sent to the outbox.
type xShardSend struct {
	dstShard uint32
	to       common.Address
	value    int64
}

// TestXShardDeposits sends cross-shard messages in a block of shard 0, and
// converts them into the deposits of shards 1 and 2.
func TestXShardDeposits(t *testing.T) {
	var (
		config = *params.MergedTestChainConfig
		engine = beacon.New(ethash.NewFaker())
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		sender = crypto.PubkeyToAddress(key.PublicKey)
		outbox = params.XShardOutboxAddress

		// impostor emits a SentMessage topic from another address.
		impostor = common.HexToAddress("0x1111")
	)
	config.QuarkChain = &params.QuarkChainConfig{ShardID: 0, ShardIDs: []uint32{0, 1, 2}}
	sendABI, err := abi.JSON(strings.NewReader(`[{"type": "function", "name": "send", "stateMutability": "payable", "outputs": [], "inputs": [
		{"name": "dstShard", "type": "uint32"}, {"name": "to", "type": "address"}, {"name": "value", "type": "uint256"},
		{"name": "gasLimit", "type": "uint64"}, {"name": "isMove", "type": "bool"}, {"name": "data", "type": "bytes"}]}]`))
	if err != nil {
		t.Fatal(err)
	}
	alloc := core.SystemContractAllocs()
	alloc[sender] = types.Account{Balance: big.NewInt(params.Ether)}
	alloc[impostor] = types.Account{Code: program.New().Push(vm.XShardSentMessageTopic).Push(0).Push(0).Op(vm.LOG1).Bytes()}
	gspec := &core.Genesis{Config: &config, Alloc: alloc}
	signer := types.LatestSigner(gspec.Config)

	// Block 1 sends messages to shards 1 and 2, between a SentMessage topic of
	// the impostor and a send failing on an unknown shard. Block 2 sends none.
	sends := []*xShardSend{
		{dstShard: 1, to: common.HexToAddress("0xa1"), value: 100},
		nil, // the impostor
		{dstShard: 9, to: common.HexToAddress("0xa9"), value: 900},
		{dstShard: 2, to: common.HexToAddress("0xa2"), value: 200},
		{dstShard: 1, to: common.HexToAddress("0xa3"), value: 300},
	}
	var nonce uint64
	_, blocks, receipts := core.GenerateChainWithGenesis(gspec, engine, 2, func(i int, b *core.BlockGen) {
		if i == 1 {
			return
		}
		for _, send := range sends {
			tx := &types.DynamicFeeTx{
				ChainID:   gspec.Config.ChainID,
				Nonce:     nonce,
				To:        &impostor,
				Gas:       200_000,
				GasFeeCap: big.NewInt(5 * params.GWei),
				GasTipCap: big.NewInt(2),
			}
			if send != nil {
				input, err := sendABI.Pack("send", send.dstShard, send.to, big.NewInt(send.value), params.XShardSendGas, true, []byte{})
				if err != nil {
					t.Fatal(err)
				}
				// Pay for the gas at the fee cap, the excess comes back.
				tx.To, tx.Data = &outbox, input
				tx.Value = new(big.Int).Mul(new(big.Int).SetUint64(params.XShardSendGas), tx.GasFeeCap)
				tx.Value.Add(tx.Value, big.NewInt(send.value))
			}
			b.AddTx(types.MustSignNewTx(key, signer, tx))
			nonce++
		}
	})
	chain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	defer chain.Stop()
	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	}
	if receipts[0][2].Status != types.ReceiptStatusFailed {
		t.Fatal("send to an unknown shard succeeded")
	}

	// The logs of block 1 are: the send to shard 1, the impostor, the send to
	// shard 2 and the second send to shard 1, the failed send having none.
	hash := blocks[0].Hash()
	deposit := func(send *xShardSend, logIndex uint64) *types.DepositTx {
		return &types.DepositTx{
			Version:         types.DepositTxVersion,
			SourceBlockHash: hash,
			SourceLogIndex:  logIndex,
			From:            sender,
			To:              send.to,
			Value:           uint256.NewInt(uint64(send.value)),
			GasPrice:        uint256.MustFromBig(receipts[0][0].EffectiveGasPrice),
			Gas:             params.XShardSendGas,
			IsMove:          true,
			Data:            []byte{},
		}
	}
	want := map[uint32][]*types.DepositTx{
		1: {deposit(sends[0], 0), deposit(sends[4], 3)},
		2: {deposit(sends[3], 2)},
	}
	have, err := XShardDeposits(chain, hash)
	if err != nil {
		t.Fatalf("failed to convert block 1: %v", err)
	}
	if !reflect.DeepEqual(have, want) {
		t.Errorf("block 1 deposits:\nhave %+v\nwant %+v", have, want)
	}
	// Block 2 sends every other shard an empty list.
	hash = blocks[1].Hash()
	have, err = XShardDeposits(chain, hash)
	if err != nil {
		t.Fatalf("failed to convert block 2: %v", err)
	}
	if want := map[uint32][]*types.DepositTx{1: {}, 2: {}}; !reflect.DeepEqual(have, want) {
		t.Errorf("block 2 deposits %v, want %v", have, want)
	}
	// A block not inserted yet has no receipts to read.
	if _, err := XShardDeposits(chain, common.Hash{0xff}); err == nil {
		t.Error("converted an unknown block")
	}
}

// sentMessageLog returns a SentMessage log of the outbox sending a move with
// 3 bytes of data to the given shard, encoded as the outbox does.
func sentMessageLog(dstShard byte) *types.Log {
	data := make([]byte, 7*32)
	data[31] = 0x0a                   // value
	data[63] = 0x0b                   // gasPrice
	data[95] = 0x0c                   // gasLimit
	data[127] = 1                     // isMove
	data[159] = 5 * 32                // data offset
	data[191] = 3                     // data length
	copy(data[192:], []byte{1, 2, 3}) // data, padded
	return &types.Log{
		Address: params.XShardOutboxAddress,
		Topics:  []common.Hash{vm.XShardSentMessageTopic, {31: dstShard}, {31: 0x0d}, {31: 0x0e}},
		Data:    data,
		Index:   5,
	}
}

// TestXShardDepositsInvalid checks that a SentMessage log of the outbox that
// cannot be converted fails the conversion, as it means a broken outbox.
func TestXShardDepositsInvalid(t *testing.T) {
	var (
		config    = &params.QuarkChainConfig{ShardID: 0, ShardIDs: []uint32{0, 1}}
		blockHash = common.Hash{0xff}
	)
	have, err := depositsFromLogs(config, blockHash, []*types.Log{sentMessageLog(1)})
	if err != nil {
		t.Fatalf("failed to convert a valid event: %v", err)
	}
	want := map[uint32][]*types.DepositTx{1: {{
		Version:         types.DepositTxVersion,
		SourceBlockHash: blockHash,
		SourceLogIndex:  5,
		From:            common.Address{19: 0x0d},
		To:              common.Address{19: 0x0e},
		Value:           uint256.NewInt(0x0a),
		GasPrice:        uint256.NewInt(0x0b),
		Gas:             0x0c,
		IsMove:          true,
		Data:            []byte{1, 2, 3},
	}}}
	if !reflect.DeepEqual(have, want) {
		t.Fatalf("converted %+v, want %+v", have, want)
	}
	for _, tt := range []struct {
		name string
		edit func(log *types.Log)
	}{
		{"own shard", func(log *types.Log) { log.Topics[1][31] = 0 }},
		{"unknown shard", func(log *types.Log) { log.Topics[1][31] = 2 }},
		{"missing topic", func(log *types.Log) { log.Topics = log.Topics[:3] }},
		{"dirty dstShard", func(log *types.Log) { log.Topics[1][0] = 1 }},
		{"dirty from", func(log *types.Log) { log.Topics[2][0] = 1 }},
		{"dirty to", func(log *types.Log) { log.Topics[3][0] = 1 }},
		{"short data", func(log *types.Log) { log.Data = log.Data[:5*32] }},
		{"dirty gasLimit", func(log *types.Log) { log.Data[2*32] = 1 }},
		{"dirty isMove", func(log *types.Log) { log.Data[3*32] = 1 }},
		{"non-bool isMove", func(log *types.Log) { log.Data[4*32-1] = 2 }},
		{"non-canonical offset", func(log *types.Log) { log.Data[5*32-1] = 0xc0 }},
		{"trailing bytes", func(log *types.Log) { log.Data = append(log.Data, make([]byte, 32)...) }},
		{"missing padding", func(log *types.Log) { log.Data = log.Data[:len(log.Data)-1] }},
		{"huge length", func(log *types.Log) { copy(log.Data[5*32+24:6*32], bytes.Repeat([]byte{0xff}, 8)) }},
	} {
		log := sentMessageLog(1)
		tt.edit(log)
		if _, err := depositsFromLogs(config, blockHash, []*types.Log{log}); err == nil {
			t.Errorf("%s: converted an invalid event", tt.name)
		}
	}
}

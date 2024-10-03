package miner

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/assert"
)

func newTestBlock() *types.Block {
	tx1 := types.NewTransaction(1, common.BytesToAddress([]byte{0x11}), big.NewInt(111), 1111, big.NewInt(11111), []byte{0x11, 0x11, 0x11})
	txs := []*types.Transaction{tx1}

	block := types.NewBlock(&types.Header{Number: big.NewInt(314)}, &types.Body{Transactions: txs}, nil, trie.NewStackTrie(nil))
	return block
}

func TestSetFullBlock_AvoidPanic(t *testing.T) {
	var (
		db        = rawdb.NewMemoryDatabase()
		recipient = common.HexToAddress("0xdeadbeef")
	)
	w, b := newTestWorker(t, params.TestChainConfig, ethash.NewFaker(), db, 0)

	timestamp := uint64(time.Now().Unix())
	args := &BuildPayloadArgs{
		Parent:       b.chain.CurrentBlock().Hash(),
		Timestamp:    timestamp,
		Random:       common.Hash{},
		FeeRecipient: recipient,
	}
	payload, err := w.buildPayload(args, false)
	if err != nil {
		t.Fatalf("Failed to build payload %v", err)
	}

	fees := big.NewInt(1)

	payload.done <- struct{}{}
	close(payload.stop)

	block := newTestBlock()
	// expect not to panic sending to payload.stop
	// now that done is closed
	payload.SetFullBlock(block, fees)
}

func TestAfterSetFullBlock_Panic_DoneChannelNotSent(t *testing.T) {
	var (
		db        = rawdb.NewMemoryDatabase()
		recipient = common.HexToAddress("0xdeadbeef")
	)
	w, b := newTestWorker(t, params.TestChainConfig, ethash.NewFaker(), db, 0)

	timestamp := uint64(time.Now().Unix())
	args := &BuildPayloadArgs{
		Parent:       b.chain.CurrentBlock().Hash(),
		Timestamp:    timestamp,
		Random:       common.Hash{},
		FeeRecipient: recipient,
	}
	payload, err := w.buildPayload(args, false)
	if err != nil {
		t.Fatalf("Failed to build payload %v", err)
	}

	// dont send on done channel, but close stop channel.
	// should panic when sent on.
	close(payload.stop)

	assert.Panics(t, func() {
		payload.afterSetFullBlock()
	})
}

func TestAfterSetFullBlockAvoidPanicDoneChannelSent(t *testing.T) {
	var (
		db        = rawdb.NewMemoryDatabase()
		recipient = common.HexToAddress("0xdeadbeef")
	)
	w, b := newTestWorker(t, params.TestChainConfig, ethash.NewFaker(), db, 0)

	timestamp := uint64(time.Now().Unix())
	args := &BuildPayloadArgs{
		Parent:       b.chain.CurrentBlock().Hash(),
		Timestamp:    timestamp,
		Random:       common.Hash{},
		FeeRecipient: recipient,
	}
	payload, err := w.buildPayload(args, false)
	if err != nil {
		t.Fatalf("Failed to build payload %v", err)
	}

	payload.done <- struct{}{}
	close(payload.stop)

	assert.NotPanics(t, func() {
		payload.afterSetFullBlock()
	})
}

func TestSetFullBlock(t *testing.T) {
	var (
		db        = rawdb.NewMemoryDatabase()
		recipient = common.HexToAddress("0xdeadbeef")
	)
	w, b := newTestWorker(t, params.TestChainConfig, ethash.NewFaker(), db, 0)

	timestamp := uint64(time.Now().Unix())
	args := &BuildPayloadArgs{
		Parent:       b.chain.CurrentBlock().Hash(),
		Timestamp:    timestamp,
		Random:       common.Hash{},
		FeeRecipient: recipient,
	}
	payload, err := w.buildPayload(args, false)
	if err != nil {
		t.Fatalf("Failed to build payload %v", err)
	}

	fees := big.NewInt(1)

	block := newTestBlock()
	payload.SetFullBlock(block, fees)

	assert.Equal(t, block, payload.full)
	assert.Equal(t, fees, payload.fullFees)
}

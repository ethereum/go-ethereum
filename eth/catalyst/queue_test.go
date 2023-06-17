package catalyst

import (
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPayloadQueue(t *testing.T) {
	genesis, blocks := generateMergeChain(10, false)
	n, ethservice := startEthService(t, genesis, blocks)
	defer n.Close()
	api := NewConsensusAPI(ethservice)
	args1 := &miner.BuildPayloadArgs{
		Parent:       api.eth.BlockChain().CurrentBlock().Hash(),
		Timestamp:    1686973894,
		FeeRecipient: common.HexToAddress("0x123"),
		Random:       common.HexToHash("0x123"),
		Withdrawals:  nil,
	}

	payload1, err := api.eth.Miner().BuildPayload(args1)
	assert.Nil(t, err)

	args2 := &miner.BuildPayloadArgs{
		Parent:       api.eth.BlockChain().CurrentBlock().Hash(),
		Timestamp:    1686973894,
		FeeRecipient: common.HexToAddress("0x1234"),
		Random:       common.HexToHash("0x1234"),
		Withdrawals:  nil,
	}
	payload2, err := api.eth.Miner().BuildPayload(args2)
	assert.Nil(t, err)

	id1 := engine.PayloadID([8]byte{1})
	id2 := engine.PayloadID([8]byte{2})
	id3 := engine.PayloadID([8]byte{3})

	queue := newPayloadQueue()

	queue.put(id1, payload1)
	queue.put(id2, payload2)

	retrievedPayload1 := queue.get(id1)
	retrievedPayload2 := queue.get(id2)
	nonExistentPayload := queue.get(id3)

	assert.Equal(t, payload1.Resolve(), retrievedPayload1)
	assert.Equal(t, payload2.Resolve(), retrievedPayload2)
	assert.Nil(t, nonExistentPayload)

	assert.True(t, queue.has(id1))
	assert.True(t, queue.has(id2))
	assert.False(t, queue.has(id3))
}

func TestHeaderQueue(t *testing.T) {
	queue := newHeaderQueue()

	hash1 := common.HexToHash("0x123")
	hash2 := common.HexToHash("0x456")

	header1 := &types.Header{TxHash: hash1}
	header2 := &types.Header{TxHash: hash2}

	queue.put(hash1, header1)
	queue.put(hash2, header2)

	retrievedHeader1 := queue.get(hash1)
	retrievedHeader2 := queue.get(hash2)
	nonExistentHeader := queue.get(common.HexToHash("0x789"))

	assert.Equal(t, header1, retrievedHeader1)
	assert.Equal(t, header2, retrievedHeader2)
	assert.Nil(t, nonExistentHeader)
}

func TestCircularQueue(t *testing.T) {
	queue := newCircularQueue(3)

	item1 := "item1"
	item2 := "item2"
	item3 := "item3"
	item4 := "item4"

	queue.enqueue(item1)
	queue.enqueue(item2)
	queue.enqueue(item3)
	queue.enqueue(item4)

	assert.Equal(t, item2, queue.get(0))
	assert.Equal(t, item3, queue.get(1))
	assert.Equal(t, item4, queue.get(2))
	assert.Nil(t, queue.get(3))
}

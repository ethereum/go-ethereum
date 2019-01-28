// Copyright 2019 The go-ethereum Authors
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

package mocks

import (
	"bytes"
	"math/big"
	"sort"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/statediff"
	"github.com/ethereum/go-ethereum/statediff/testhelpers"
	sdtypes "github.com/ethereum/go-ethereum/statediff/types"
)

var (
	emptyStorage   = make([]sdtypes.StorageNode, 0)
	block0, block1 *types.Block
	minerLeafKey   = testhelpers.AddressToLeafKey(common.HexToAddress("0x0"))
	account1, _    = rlp.EncodeToBytes(state.Account{
		Nonce:    uint64(0),
		Balance:  big.NewInt(10000),
		CodeHash: common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
		Root:     common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
	})
	account1LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3926db69aaced518e9b9f0f434a473e7174109c943548bb8f23be41ca76d9ad2"),
		account1,
	})
	minerAccount, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    uint64(0),
		Balance:  big.NewInt(2000000000000000000),
		CodeHash: common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
		Root:     common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
	})
	minerAccountLeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3380c7b7ae81a58eb98d9c78de4a1fd7fd9535fc953ed2be602daaa41767312a"),
		minerAccount,
	})
	bankAccount, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    uint64(1),
		Balance:  big.NewInt(testhelpers.TestBankFunds.Int64() - 10000),
		CodeHash: common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
		Root:     common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
	})
	bankAccountLeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("30bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"),
		bankAccount,
	})
	mockTotalDifficulty = big.NewInt(1337)
	params              = statediff.Params{
		IntermediateStateNodes: false,
		IncludeTD:              true,
		IncludeBlock:           true,
		IncludeReceipts:        true,
	}
)

func TestAPI(t *testing.T) {
	testSubscriptionAPI(t)
	testHTTPAPI(t)
}

func testSubscriptionAPI(t *testing.T) {
	blocks, chain := testhelpers.MakeChain(1, testhelpers.Genesis, testhelpers.TestChainGen)
	defer chain.Stop()
	block0 = testhelpers.Genesis
	block1 = blocks[0]
	expectedBlockRlp, _ := rlp.EncodeToBytes(block1)
	mockReceipt := &types.Receipt{
		BlockNumber: block1.Number(),
		BlockHash:   block1.Hash(),
	}
	expectedReceiptBytes, _ := rlp.EncodeToBytes(types.Receipts{mockReceipt})
	expectedStateDiff := statediff.StateObject{
		BlockNumber: block1.Number(),
		BlockHash:   block1.Hash(),
		Nodes: []sdtypes.StateNode{
			{
				Path:         []byte{'\x05'},
				NodeType:     sdtypes.Leaf,
				LeafKey:      minerLeafKey,
				NodeValue:    minerAccountLeafNode,
				StorageNodes: emptyStorage,
			},
			{
				Path:         []byte{'\x0e'},
				NodeType:     sdtypes.Leaf,
				LeafKey:      testhelpers.Account1LeafKey,
				NodeValue:    account1LeafNode,
				StorageNodes: emptyStorage,
			},
			{
				Path:         []byte{'\x00'},
				NodeType:     sdtypes.Leaf,
				LeafKey:      testhelpers.BankLeafKey,
				NodeValue:    bankAccountLeafNode,
				StorageNodes: emptyStorage,
			},
		},
	}
	expectedStateDiffBytes, _ := rlp.EncodeToBytes(expectedStateDiff)
	blockChan := make(chan *types.Block)
	parentBlockChain := make(chan *types.Block)
	serviceQuitChan := make(chan bool)
	mockBlockChain := &BlockChain{}
	mockBlockChain.SetReceiptsForHash(block1.Hash(), types.Receipts{mockReceipt})
	mockBlockChain.SetTdByHash(block1.Hash(), mockTotalDifficulty)
	mockService := MockStateDiffService{
		Mutex:             sync.Mutex{},
		Builder:           statediff.NewBuilder(chain.StateCache()),
		BlockChan:         blockChan,
		BlockChain:        mockBlockChain,
		ParentBlockChan:   parentBlockChain,
		QuitChan:          serviceQuitChan,
		Subscriptions:     make(map[common.Hash]map[rpc.ID]statediff.Subscription),
		SubscriptionTypes: make(map[common.Hash]statediff.Params),
	}
	mockService.Start()
	id := rpc.NewID()
	payloadChan := make(chan statediff.Payload)
	quitChan := make(chan bool)
	mockService.Subscribe(id, payloadChan, quitChan, params)
	blockChan <- block1
	parentBlockChain <- block0

	sort.Slice(expectedStateDiffBytes, func(i, j int) bool { return expectedStateDiffBytes[i] < expectedStateDiffBytes[j] })
	select {
	case payload := <-payloadChan:
		if !bytes.Equal(payload.BlockRlp, expectedBlockRlp) {
			t.Errorf("payload does not have expected block\r\nactual block rlp: %v\r\nexpected block rlp: %v", payload.BlockRlp, expectedBlockRlp)
		}
		sort.Slice(payload.StateObjectRlp, func(i, j int) bool { return payload.StateObjectRlp[i] < payload.StateObjectRlp[j] })
		if !bytes.Equal(payload.StateObjectRlp, expectedStateDiffBytes) {
			t.Errorf("payload does not have expected state diff\r\nactual state diff rlp: %v\r\nexpected state diff rlp: %v", payload.StateObjectRlp, expectedStateDiffBytes)
		}
		if !bytes.Equal(expectedReceiptBytes, payload.ReceiptsRlp) {
			t.Errorf("payload does not have expected receipts\r\nactual receipt rlp: %v\r\nexpected receipt rlp: %v", payload.ReceiptsRlp, expectedReceiptBytes)
		}
		if !bytes.Equal(payload.TotalDifficulty.Bytes(), mockTotalDifficulty.Bytes()) {
			t.Errorf("payload does not have expected total difficulty\r\nactual td: %d\r\nexpected td: %d", payload.TotalDifficulty.Int64(), mockTotalDifficulty.Int64())
		}
	case <-quitChan:
		t.Errorf("channel quit before delivering payload")
	}
}

func testHTTPAPI(t *testing.T) {
	blocks, chain := testhelpers.MakeChain(1, testhelpers.Genesis, testhelpers.TestChainGen)
	defer chain.Stop()
	block0 = testhelpers.Genesis
	block1 = blocks[0]
	expectedBlockRlp, _ := rlp.EncodeToBytes(block1)
	mockReceipt := &types.Receipt{
		BlockNumber: block1.Number(),
		BlockHash:   block1.Hash(),
	}
	expectedReceiptBytes, _ := rlp.EncodeToBytes(types.Receipts{mockReceipt})
	expectedStateDiff := statediff.StateObject{
		BlockNumber: block1.Number(),
		BlockHash:   block1.Hash(),
		Nodes: []sdtypes.StateNode{
			{
				Path:         []byte{'\x05'},
				NodeType:     sdtypes.Leaf,
				LeafKey:      minerLeafKey,
				NodeValue:    minerAccountLeafNode,
				StorageNodes: emptyStorage,
			},
			{
				Path:         []byte{'\x0e'},
				NodeType:     sdtypes.Leaf,
				LeafKey:      testhelpers.Account1LeafKey,
				NodeValue:    account1LeafNode,
				StorageNodes: emptyStorage,
			},
			{
				Path:         []byte{'\x00'},
				NodeType:     sdtypes.Leaf,
				LeafKey:      testhelpers.BankLeafKey,
				NodeValue:    bankAccountLeafNode,
				StorageNodes: emptyStorage,
			},
		},
	}
	expectedStateDiffBytes, _ := rlp.EncodeToBytes(expectedStateDiff)
	mockBlockChain := &BlockChain{}
	mockBlockChain.SetBlocksForHashes(map[common.Hash]*types.Block{
		block0.Hash(): block0,
		block1.Hash(): block1,
	})
	mockBlockChain.SetBlockForNumber(block1, block1.Number().Uint64())
	mockBlockChain.SetReceiptsForHash(block1.Hash(), types.Receipts{mockReceipt})
	mockBlockChain.SetTdByHash(block1.Hash(), big.NewInt(1337))
	mockService := MockStateDiffService{
		Mutex:      sync.Mutex{},
		Builder:    statediff.NewBuilder(chain.StateCache()),
		BlockChain: mockBlockChain,
	}
	payload, err := mockService.StateDiffAt(block1.Number().Uint64(), params)
	if err != nil {
		t.Error(err)
	}
	sort.Slice(payload.StateObjectRlp, func(i, j int) bool { return payload.StateObjectRlp[i] < payload.StateObjectRlp[j] })
	sort.Slice(expectedStateDiffBytes, func(i, j int) bool { return expectedStateDiffBytes[i] < expectedStateDiffBytes[j] })
	if !bytes.Equal(payload.BlockRlp, expectedBlockRlp) {
		t.Errorf("payload does not have expected block\r\nactual block rlp: %v\r\nexpected block rlp: %v", payload.BlockRlp, expectedBlockRlp)
	}
	if !bytes.Equal(payload.StateObjectRlp, expectedStateDiffBytes) {
		t.Errorf("payload does not have expected state diff\r\nactual state diff rlp: %v\r\nexpected state diff rlp: %v", payload.StateObjectRlp, expectedStateDiffBytes)
	}
	if !bytes.Equal(expectedReceiptBytes, payload.ReceiptsRlp) {
		t.Errorf("payload does not have expected receipts\r\nactual receipt rlp: %v\r\nexpected receipt rlp: %v", payload.ReceiptsRlp, expectedReceiptBytes)
	}
	if !bytes.Equal(payload.TotalDifficulty.Bytes(), mockTotalDifficulty.Bytes()) {
		t.Errorf("paylaod does not have the expected total difficulty\r\nactual td: %d\r\nexpected td: %d", payload.TotalDifficulty.Int64(), mockTotalDifficulty.Int64())
	}
}

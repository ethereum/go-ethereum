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

package statediff_test

import (
	"bytes"
	"math/big"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/statediff"
	"github.com/ethereum/go-ethereum/statediff/testhelpers"
	sdtypes "github.com/ethereum/go-ethereum/statediff/types"
)

// TODO: add test that filters on address
var (
	contractLeafKey                                        []byte
	emptyDiffs                                             = make([]sdtypes.StateNode, 0)
	emptyStorage                                           = make([]sdtypes.StorageNode, 0)
	block0, block1, block2, block3, block4, block5, block6 *types.Block
	builder                                                statediff.Builder
	miningReward                                           = int64(2000000000000000000)
	minerAddress                                           = common.HexToAddress("0x0")
	minerLeafKey                                           = testhelpers.AddressToLeafKey(minerAddress)

	balanceChange10000    = int64(10000)
	balanceChange1000     = int64(1000)
	block1BankBalance     = int64(99990000)
	block1Account1Balance = int64(10000)
	block2Account2Balance = int64(1000)

	slot0 = common.HexToHash("0")
	slot1 = common.HexToHash("1")
	slot2 = common.HexToHash("2")
	slot3 = common.HexToHash("3")

	slot0StorageKey = crypto.Keccak256Hash(slot0[:])
	slot1StorageKey = crypto.Keccak256Hash(slot1[:])
	slot2StorageKey = crypto.Keccak256Hash(slot2[:])
	slot3StorageKey = crypto.Keccak256Hash(slot3[:])

	slot0StorageValue = common.Hex2Bytes("94703c4b2bd70c169f5717101caee543299fc946c7") // prefixed AccountAddr1
	slot1StorageValue = common.Hex2Bytes("01")
	slot2StorageValue = common.Hex2Bytes("09")
	slot3StorageValue = common.Hex2Bytes("03")

	slot0StorageLeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("390decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563"),
		slot0StorageValue,
	})
	slot1StorageLeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("310e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6"),
		slot1StorageValue,
	})
	slot2StorageLeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("305787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace"),
		slot2StorageValue,
	})
	slot3StorageLeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("32575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b"),
		slot3StorageValue,
	})
	slot0StorageLeafRootNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("20290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563"),
		slot0StorageValue,
	})

	contractAccountAtBlock2, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    1,
		Balance:  big.NewInt(0),
		CodeHash: common.HexToHash("0xaaea5efba4fd7b45d7ec03918ac5d8b31aa93b48986af0e6b591f0f087c80127").Bytes(),
		Root:     crypto.Keccak256Hash(block2StorageBranchRootNode),
	})
	contractAccountAtBlock2LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3114658a74d9cc9f7acf2c5cd696c3494d7c344d78bfec3add0d91ec4e8d1c45"),
		contractAccountAtBlock2,
	})
	contractAccountAtBlock3, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    1,
		Balance:  big.NewInt(0),
		CodeHash: common.HexToHash("0xaaea5efba4fd7b45d7ec03918ac5d8b31aa93b48986af0e6b591f0f087c80127").Bytes(),
		Root:     crypto.Keccak256Hash(block3StorageBranchRootNode),
	})
	contractAccountAtBlock3LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3114658a74d9cc9f7acf2c5cd696c3494d7c344d78bfec3add0d91ec4e8d1c45"),
		contractAccountAtBlock3,
	})
	contractAccountAtBlock4, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    1,
		Balance:  big.NewInt(0),
		CodeHash: common.HexToHash("0xaaea5efba4fd7b45d7ec03918ac5d8b31aa93b48986af0e6b591f0f087c80127").Bytes(),
		Root:     crypto.Keccak256Hash(block4StorageBranchRootNode),
	})
	contractAccountAtBlock4LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3114658a74d9cc9f7acf2c5cd696c3494d7c344d78bfec3add0d91ec4e8d1c45"),
		contractAccountAtBlock4,
	})
	contractAccountAtBlock5, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    1,
		Balance:  big.NewInt(0),
		CodeHash: common.HexToHash("0xaaea5efba4fd7b45d7ec03918ac5d8b31aa93b48986af0e6b591f0f087c80127").Bytes(),
		Root:     crypto.Keccak256Hash(slot0StorageLeafRootNode),
	})
	contractAccountAtBlock5LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3114658a74d9cc9f7acf2c5cd696c3494d7c344d78bfec3add0d91ec4e8d1c45"),
		contractAccountAtBlock5,
	})

	minerAccountAtBlock1, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    0,
		Balance:  big.NewInt(miningReward),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	minerAccountAtBlock1LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3380c7b7ae81a58eb98d9c78de4a1fd7fd9535fc953ed2be602daaa41767312a"),
		minerAccountAtBlock1,
	})
	minerAccountAtBlock2, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    0,
		Balance:  big.NewInt(miningReward + miningReward),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	minerAccountAtBlock2LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3380c7b7ae81a58eb98d9c78de4a1fd7fd9535fc953ed2be602daaa41767312a"),
		minerAccountAtBlock2,
	})

	account1AtBlock1, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    0,
		Balance:  big.NewInt(balanceChange10000),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	account1AtBlock1LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3926db69aaced518e9b9f0f434a473e7174109c943548bb8f23be41ca76d9ad2"),
		account1AtBlock1,
	})
	account1AtBlock2, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    2,
		Balance:  big.NewInt(block1Account1Balance - balanceChange1000 + balanceChange1000),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	account1AtBlock2LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3926db69aaced518e9b9f0f434a473e7174109c943548bb8f23be41ca76d9ad2"),
		account1AtBlock2,
	})
	account1AtBlock5, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    2,
		Balance:  big.NewInt(block1Account1Balance - balanceChange1000 + balanceChange1000 + miningReward),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	account1AtBlock5LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3926db69aaced518e9b9f0f434a473e7174109c943548bb8f23be41ca76d9ad2"),
		account1AtBlock5,
	})
	account1AtBlock6, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    3,
		Balance:  big.NewInt(block1Account1Balance - balanceChange1000 + balanceChange1000 + miningReward),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	account1AtBlock6LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3926db69aaced518e9b9f0f434a473e7174109c943548bb8f23be41ca76d9ad2"),
		account1AtBlock6,
	})

	account2AtBlock2, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    0,
		Balance:  big.NewInt(balanceChange1000),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	account2AtBlock2LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3957f3e2f04a0764c3a0491b175f69926da61efbcc8f61fa1455fd2d2b4cdd45"),
		account2AtBlock2,
	})
	account2AtBlock3, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    0,
		Balance:  big.NewInt(block2Account2Balance + miningReward),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	account2AtBlock3LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3957f3e2f04a0764c3a0491b175f69926da61efbcc8f61fa1455fd2d2b4cdd45"),
		account2AtBlock3,
	})
	account2AtBlock4, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    0,
		Balance:  big.NewInt(block2Account2Balance + miningReward*2),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	account2AtBlock4LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3957f3e2f04a0764c3a0491b175f69926da61efbcc8f61fa1455fd2d2b4cdd45"),
		account2AtBlock4,
	})
	account2AtBlock6, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    0,
		Balance:  big.NewInt(block2Account2Balance + miningReward*3),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	account2AtBlock6LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3957f3e2f04a0764c3a0491b175f69926da61efbcc8f61fa1455fd2d2b4cdd45"),
		account2AtBlock6,
	})

	bankAccountAtBlock0, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    0,
		Balance:  big.NewInt(testhelpers.TestBankFunds.Int64()),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	bankAccountAtBlock0LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("2000bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"),
		bankAccountAtBlock0,
	})
	bankAccountAtBlock1, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    1,
		Balance:  big.NewInt(testhelpers.TestBankFunds.Int64() - balanceChange10000),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	bankAccountAtBlock1LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("30bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"),
		bankAccountAtBlock1,
	})
	bankAccountAtBlock2, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    2,
		Balance:  big.NewInt(block1BankBalance - balanceChange1000),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	bankAccountAtBlock2LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("30bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"),
		bankAccountAtBlock2,
	})
	bankAccountAtBlock3, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    3,
		Balance:  big.NewInt(99989000),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	bankAccountAtBlock3LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("30bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"),
		bankAccountAtBlock3,
	})
	bankAccountAtBlock4, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    6,
		Balance:  big.NewInt(99989000),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	bankAccountAtBlock4LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("30bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"),
		bankAccountAtBlock4,
	})
	bankAccountAtBlock5, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    7,
		Balance:  big.NewInt(99989000),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	bankAccountAtBlock5LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("30bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"),
		bankAccountAtBlock5,
	})

	block1BranchRootNode, _ = rlp.EncodeToBytes([]interface{}{
		crypto.Keccak256(bankAccountAtBlock1LeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		crypto.Keccak256(minerAccountAtBlock1LeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		crypto.Keccak256(account1AtBlock1LeafNode),
		[]byte{},
		[]byte{},
	})
	block2BranchRootNode, _ = rlp.EncodeToBytes([]interface{}{
		crypto.Keccak256(bankAccountAtBlock2LeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		crypto.Keccak256(minerAccountAtBlock2LeafNode),
		crypto.Keccak256(contractAccountAtBlock2LeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		crypto.Keccak256(account2AtBlock2LeafNode),
		[]byte{},
		crypto.Keccak256(account1AtBlock2LeafNode),
		[]byte{},
		[]byte{},
	})
	block3BranchRootNode, _ = rlp.EncodeToBytes([]interface{}{
		crypto.Keccak256(bankAccountAtBlock3LeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		crypto.Keccak256(minerAccountAtBlock2LeafNode),
		crypto.Keccak256(contractAccountAtBlock3LeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		crypto.Keccak256(account2AtBlock3LeafNode),
		[]byte{},
		crypto.Keccak256(account1AtBlock2LeafNode),
		[]byte{},
		[]byte{},
	})
	block4BranchRootNode, _ = rlp.EncodeToBytes([]interface{}{
		crypto.Keccak256(bankAccountAtBlock4LeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		crypto.Keccak256(minerAccountAtBlock2LeafNode),
		crypto.Keccak256(contractAccountAtBlock4LeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		crypto.Keccak256(account2AtBlock4LeafNode),
		[]byte{},
		crypto.Keccak256(account1AtBlock2LeafNode),
		[]byte{},
		[]byte{},
	})
	block5BranchRootNode, _ = rlp.EncodeToBytes([]interface{}{
		crypto.Keccak256(bankAccountAtBlock5LeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		crypto.Keccak256(minerAccountAtBlock2LeafNode),
		crypto.Keccak256(contractAccountAtBlock5LeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		crypto.Keccak256(account2AtBlock4LeafNode),
		[]byte{},
		crypto.Keccak256(account1AtBlock5LeafNode),
		[]byte{},
		[]byte{},
	})
	block6BranchRootNode, _ = rlp.EncodeToBytes([]interface{}{
		crypto.Keccak256(bankAccountAtBlock5LeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		crypto.Keccak256(minerAccountAtBlock2LeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		crypto.Keccak256(account2AtBlock6LeafNode),
		[]byte{},
		crypto.Keccak256(account1AtBlock6LeafNode),
		[]byte{},
		[]byte{},
	})

	block2StorageBranchRootNode, _ = rlp.EncodeToBytes([]interface{}{
		[]byte{},
		[]byte{},
		crypto.Keccak256(slot0StorageLeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		crypto.Keccak256(slot1StorageLeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
	})
	block3StorageBranchRootNode, _ = rlp.EncodeToBytes([]interface{}{
		[]byte{},
		[]byte{},
		crypto.Keccak256(slot0StorageLeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		crypto.Keccak256(slot1StorageLeafNode),
		crypto.Keccak256(slot3StorageLeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
	})
	block4StorageBranchRootNode, _ = rlp.EncodeToBytes([]interface{}{
		[]byte{},
		[]byte{},
		crypto.Keccak256(slot0StorageLeafNode),
		[]byte{},
		crypto.Keccak256(slot2StorageLeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
	})
)

func TestBuilder(t *testing.T) {
	blocks, chain := testhelpers.MakeChain(3, testhelpers.Genesis, testhelpers.TestChainGen)
	contractLeafKey = testhelpers.AddressToLeafKey(testhelpers.ContractAddr)
	defer chain.Stop()
	block0 = testhelpers.Genesis
	block1 = blocks[0]
	block2 = blocks[1]
	block3 = blocks[2]
	params := statediff.Params{}
	builder = statediff.NewBuilder(chain.StateCache())

	var tests = []struct {
		name              string
		startingArguments statediff.Args
		expected          *statediff.StateObject
	}{
		{
			"testEmptyDiff",
			statediff.Args{
				OldStateRoot: block0.Root(),
				NewStateRoot: block0.Root(),
				BlockNumber:  block0.Number(),
				BlockHash:    block0.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block0.Number(),
				BlockHash:   block0.Hash(),
				Nodes:       emptyDiffs,
			},
		},
		{
			"testBlock0",
			//10000 transferred from testBankAddress to account1Addr
			statediff.Args{
				OldStateRoot: testhelpers.NullHash,
				NewStateRoot: block0.Root(),
				BlockNumber:  block0.Number(),
				BlockHash:    block0.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block0.Number(),
				BlockHash:   block0.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock0LeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
		{
			"testBlock1",
			//10000 transferred from testBankAddress to account1Addr
			statediff.Args{
				OldStateRoot: block0.Root(),
				NewStateRoot: block1.Root(),
				BlockNumber:  block1.Number(),
				BlockHash:    block1.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block1.Number(),
				BlockHash:   block1.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{'\x00'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock1LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x05'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      minerLeafKey,
						NodeValue:    minerAccountAtBlock1LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x0e'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account1LeafKey,
						NodeValue:    account1AtBlock1LeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
		{
			"testBlock2",
			// 1000 transferred from testBankAddress to account1Addr
			// 1000 transferred from account1Addr to account2Addr
			// account1addr creates a new contract
			statediff.Args{
				OldStateRoot: block1.Root(),
				NewStateRoot: block2.Root(),
				BlockNumber:  block2.Number(),
				BlockHash:    block2.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block2.Number(),
				BlockHash:   block2.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{'\x00'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock2LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x05'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      minerLeafKey,
						NodeValue:    minerAccountAtBlock2LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x0e'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account1LeafKey,
						NodeValue:    account1AtBlock2LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:      []byte{'\x06'},
						NodeType:  sdtypes.Leaf,
						LeafKey:   contractLeafKey,
						NodeValue: contractAccountAtBlock2LeafNode,
						StorageNodes: []sdtypes.StorageNode{
							{
								Path:      []byte{'\x02'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot0StorageKey.Bytes(),
								NodeValue: slot0StorageLeafNode,
							},
							{
								Path:      []byte{'\x0b'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot1StorageKey.Bytes(),
								NodeValue: slot1StorageLeafNode,
							},
						},
					},
					{
						Path:         []byte{'\x0c'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account2LeafKey,
						NodeValue:    account2AtBlock2LeafNode,
						StorageNodes: emptyStorage,
					},
				},
				CodeAndCodeHashes: []sdtypes.CodeAndCodeHash{
					{
						Hash: testhelpers.CodeHash,
						Code: testhelpers.ByteCodeAfterDeployment,
					},
				},
			},
		},
		{
			"testBlock3",
			//the contract's storage is changed
			//and the block is mined by account 2
			statediff.Args{
				OldStateRoot: block2.Root(),
				NewStateRoot: block3.Root(),
				BlockNumber:  block3.Number(),
				BlockHash:    block3.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block3.Number(),
				BlockHash:   block3.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{'\x00'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock3LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:      []byte{'\x06'},
						NodeType:  sdtypes.Leaf,
						LeafKey:   contractLeafKey,
						NodeValue: contractAccountAtBlock3LeafNode,
						StorageNodes: []sdtypes.StorageNode{
							{
								Path:      []byte{'\x0c'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot3StorageKey.Bytes(),
								NodeValue: slot3StorageLeafNode,
							},
						},
					},
					{
						Path:         []byte{'\x0c'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account2LeafKey,
						NodeValue:    account2AtBlock3LeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
	}

	for _, test := range tests {
		diff, err := builder.BuildStateDiffObject(test.startingArguments, params)
		if err != nil {
			t.Error(err)
		}
		receivedStateDiffRlp, err := rlp.EncodeToBytes(diff)
		if err != nil {
			t.Error(err)
		}
		expectedStateDiffRlp, err := rlp.EncodeToBytes(test.expected)
		if err != nil {
			t.Error(err)
		}
		sort.Slice(receivedStateDiffRlp, func(i, j int) bool { return receivedStateDiffRlp[i] < receivedStateDiffRlp[j] })
		sort.Slice(expectedStateDiffRlp, func(i, j int) bool { return expectedStateDiffRlp[i] < expectedStateDiffRlp[j] })
		if !bytes.Equal(receivedStateDiffRlp, expectedStateDiffRlp) {
			t.Logf("Test failed: %s", test.name)
			t.Errorf("actual state diff: %+v\nexpected state diff: %+v", diff, test.expected)
		}
	}
}

func TestBuilderWithIntermediateNodes(t *testing.T) {
	blocks, chain := testhelpers.MakeChain(3, testhelpers.Genesis, testhelpers.TestChainGen)
	contractLeafKey = testhelpers.AddressToLeafKey(testhelpers.ContractAddr)
	defer chain.Stop()
	block0 = testhelpers.Genesis
	block1 = blocks[0]
	block2 = blocks[1]
	block3 = blocks[2]
	blocks = append([]*types.Block{block0}, blocks...)
	params := statediff.Params{
		IntermediateStateNodes:   true,
		IntermediateStorageNodes: true,
	}
	builder = statediff.NewBuilder(chain.StateCache())

	var tests = []struct {
		name              string
		startingArguments statediff.Args
		expected          *statediff.StateObject
	}{
		{
			"testEmptyDiff",
			statediff.Args{
				OldStateRoot: block0.Root(),
				NewStateRoot: block0.Root(),
				BlockNumber:  block0.Number(),
				BlockHash:    block0.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block0.Number(),
				BlockHash:   block0.Hash(),
				Nodes:       emptyDiffs,
			},
		},
		{
			"testBlock0",
			//10000 transferred from testBankAddress to account1Addr
			statediff.Args{
				OldStateRoot: testhelpers.NullHash,
				NewStateRoot: block0.Root(),
				BlockNumber:  block0.Number(),
				BlockHash:    block0.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block0.Number(),
				BlockHash:   block0.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock0LeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
		{
			"testBlock1",
			//10000 transferred from testBankAddress to account1Addr
			statediff.Args{
				OldStateRoot: block0.Root(),
				NewStateRoot: block1.Root(),
				BlockNumber:  block1.Number(),
				BlockHash:    block1.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block1.Number(),
				BlockHash:   block1.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Branch,
						NodeValue:    block1BranchRootNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x00'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock1LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x05'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      minerLeafKey,
						NodeValue:    minerAccountAtBlock1LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x0e'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account1LeafKey,
						NodeValue:    account1AtBlock1LeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
		{
			"testBlock2",
			// 1000 transferred from testBankAddress to account1Addr
			// 1000 transferred from account1Addr to account2Addr
			// account1addr creates a new contract
			statediff.Args{
				OldStateRoot: block1.Root(),
				NewStateRoot: block2.Root(),
				BlockNumber:  block2.Number(),
				BlockHash:    block2.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block2.Number(),
				BlockHash:   block2.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Branch,
						NodeValue:    block2BranchRootNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x00'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock2LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x05'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      minerLeafKey,
						NodeValue:    minerAccountAtBlock2LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x0e'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account1LeafKey,
						NodeValue:    account1AtBlock2LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:      []byte{'\x06'},
						NodeType:  sdtypes.Leaf,
						LeafKey:   contractLeafKey,
						NodeValue: contractAccountAtBlock2LeafNode,
						StorageNodes: []sdtypes.StorageNode{
							{
								Path:      []byte{},
								NodeType:  sdtypes.Branch,
								NodeValue: block2StorageBranchRootNode,
							},
							{
								Path:      []byte{'\x02'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot0StorageKey.Bytes(),
								NodeValue: slot0StorageLeafNode,
							},
							{
								Path:      []byte{'\x0b'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot1StorageKey.Bytes(),
								NodeValue: slot1StorageLeafNode,
							},
						},
					},
					{
						Path:         []byte{'\x0c'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account2LeafKey,
						NodeValue:    account2AtBlock2LeafNode,
						StorageNodes: emptyStorage,
					},
				},
				CodeAndCodeHashes: []sdtypes.CodeAndCodeHash{
					{
						Hash: testhelpers.CodeHash,
						Code: testhelpers.ByteCodeAfterDeployment,
					},
				},
			},
		},
		{
			"testBlock3",
			//the contract's storage is changed
			//and the block is mined by account 2
			statediff.Args{
				OldStateRoot: block2.Root(),
				NewStateRoot: block3.Root(),
				BlockNumber:  block3.Number(),
				BlockHash:    block3.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block3.Number(),
				BlockHash:   block3.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Branch,
						NodeValue:    block3BranchRootNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x00'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock3LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:      []byte{'\x06'},
						NodeType:  sdtypes.Leaf,
						LeafKey:   contractLeafKey,
						NodeValue: contractAccountAtBlock3LeafNode,
						StorageNodes: []sdtypes.StorageNode{
							{
								Path:      []byte{},
								NodeType:  sdtypes.Branch,
								NodeValue: block3StorageBranchRootNode,
							},
							{
								Path:      []byte{'\x0c'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot3StorageKey.Bytes(),
								NodeValue: slot3StorageLeafNode,
							},
						},
					},
					{
						Path:         []byte{'\x0c'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account2LeafKey,
						NodeValue:    account2AtBlock3LeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
	}

	for i, test := range tests {
		diff, err := builder.BuildStateDiffObject(test.startingArguments, params)
		if err != nil {
			t.Error(err)
		}
		receivedStateDiffRlp, err := rlp.EncodeToBytes(diff)
		if err != nil {
			t.Error(err)
		}
		expectedStateDiffRlp, err := rlp.EncodeToBytes(test.expected)
		if err != nil {
			t.Error(err)
		}
		sort.Slice(receivedStateDiffRlp, func(i, j int) bool { return receivedStateDiffRlp[i] < receivedStateDiffRlp[j] })
		sort.Slice(expectedStateDiffRlp, func(i, j int) bool { return expectedStateDiffRlp[i] < expectedStateDiffRlp[j] })
		if !bytes.Equal(receivedStateDiffRlp, expectedStateDiffRlp) {
			t.Logf("Test failed: %s", test.name)
			t.Errorf("actual state diff: %+v\r\n\r\n\r\nexpected state diff: %+v", diff, test.expected)
		}
		// Let's also confirm that our root state nodes form the state root hash in the headers
		if i > 0 {
			block := blocks[i-1]
			expectedStateRoot := block.Root()
			for _, node := range test.expected.Nodes {
				if bytes.Equal(node.Path, []byte{}) {
					stateRoot := crypto.Keccak256Hash(node.NodeValue)
					if !bytes.Equal(expectedStateRoot.Bytes(), stateRoot.Bytes()) {
						t.Logf("Test failed: %s", test.name)
						t.Errorf("actual stateroot: %x\r\nexpected stateroot: %x", stateRoot.Bytes(), expectedStateRoot.Bytes())
					}
				}
			}
		}
	}
}

func TestBuilderWithWatchedAddressList(t *testing.T) {
	blocks, chain := testhelpers.MakeChain(3, testhelpers.Genesis, testhelpers.TestChainGen)
	contractLeafKey = testhelpers.AddressToLeafKey(testhelpers.ContractAddr)
	defer chain.Stop()
	block0 = testhelpers.Genesis
	block1 = blocks[0]
	block2 = blocks[1]
	block3 = blocks[2]
	params := statediff.Params{
		WatchedAddresses: []common.Address{testhelpers.Account1Addr, testhelpers.ContractAddr},
	}
	builder = statediff.NewBuilder(chain.StateCache())

	var tests = []struct {
		name              string
		startingArguments statediff.Args
		expected          *statediff.StateObject
	}{
		{
			"testEmptyDiff",
			statediff.Args{
				OldStateRoot: block0.Root(),
				NewStateRoot: block0.Root(),
				BlockNumber:  block0.Number(),
				BlockHash:    block0.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block0.Number(),
				BlockHash:   block0.Hash(),
				Nodes:       emptyDiffs,
			},
		},
		{
			"testBlock0",
			//10000 transferred from testBankAddress to account1Addr
			statediff.Args{
				OldStateRoot: testhelpers.NullHash,
				NewStateRoot: block0.Root(),
				BlockNumber:  block0.Number(),
				BlockHash:    block0.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block0.Number(),
				BlockHash:   block0.Hash(),
				Nodes:       emptyDiffs,
			},
		},
		{
			"testBlock1",
			//10000 transferred from testBankAddress to account1Addr
			statediff.Args{
				OldStateRoot: block0.Root(),
				NewStateRoot: block1.Root(),
				BlockNumber:  block1.Number(),
				BlockHash:    block1.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block1.Number(),
				BlockHash:   block1.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{'\x0e'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account1LeafKey,
						NodeValue:    account1AtBlock1LeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
		{
			"testBlock2",
			//1000 transferred from testBankAddress to account1Addr
			//1000 transferred from account1Addr to account2Addr
			statediff.Args{
				OldStateRoot: block1.Root(),
				NewStateRoot: block2.Root(),
				BlockNumber:  block2.Number(),
				BlockHash:    block2.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block2.Number(),
				BlockHash:   block2.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:      []byte{'\x06'},
						NodeType:  sdtypes.Leaf,
						LeafKey:   contractLeafKey,
						NodeValue: contractAccountAtBlock2LeafNode,
						StorageNodes: []sdtypes.StorageNode{
							{
								Path:      []byte{'\x02'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot0StorageKey.Bytes(),
								NodeValue: slot0StorageLeafNode,
							},
							{
								Path:      []byte{'\x0b'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot1StorageKey.Bytes(),
								NodeValue: slot1StorageLeafNode,
							},
						},
					},
					{
						Path:         []byte{'\x0e'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account1LeafKey,
						NodeValue:    account1AtBlock2LeafNode,
						StorageNodes: emptyStorage,
					},
				},
				CodeAndCodeHashes: []sdtypes.CodeAndCodeHash{
					{
						Hash: testhelpers.CodeHash,
						Code: testhelpers.ByteCodeAfterDeployment,
					},
				},
			},
		},
		{
			"testBlock3",
			//the contract's storage is changed
			//and the block is mined by account 2
			statediff.Args{
				OldStateRoot: block2.Root(),
				NewStateRoot: block3.Root(),
				BlockNumber:  block3.Number(),
				BlockHash:    block3.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block3.Number(),
				BlockHash:   block3.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:      []byte{'\x06'},
						NodeType:  sdtypes.Leaf,
						LeafKey:   contractLeafKey,
						NodeValue: contractAccountAtBlock3LeafNode,
						StorageNodes: []sdtypes.StorageNode{
							{
								Path:      []byte{'\x0c'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot3StorageKey.Bytes(),
								NodeValue: slot3StorageLeafNode,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		diff, err := builder.BuildStateDiffObject(test.startingArguments, params)
		if err != nil {
			t.Error(err)
		}
		receivedStateDiffRlp, err := rlp.EncodeToBytes(diff)
		if err != nil {
			t.Error(err)
		}
		expectedStateDiffRlp, err := rlp.EncodeToBytes(test.expected)
		if err != nil {
			t.Error(err)
		}
		sort.Slice(receivedStateDiffRlp, func(i, j int) bool { return receivedStateDiffRlp[i] < receivedStateDiffRlp[j] })
		sort.Slice(expectedStateDiffRlp, func(i, j int) bool { return expectedStateDiffRlp[i] < expectedStateDiffRlp[j] })
		if !bytes.Equal(receivedStateDiffRlp, expectedStateDiffRlp) {
			t.Logf("Test failed: %s", test.name)
			t.Errorf("actual state diff: %+v\nexpected state diff: %+v", diff, test.expected)
		}
	}
}

func TestBuilderWithWatchedAddressAndStorageKeyList(t *testing.T) {
	blocks, chain := testhelpers.MakeChain(3, testhelpers.Genesis, testhelpers.TestChainGen)
	contractLeafKey = testhelpers.AddressToLeafKey(testhelpers.ContractAddr)
	defer chain.Stop()
	block0 = testhelpers.Genesis
	block1 = blocks[0]
	block2 = blocks[1]
	block3 = blocks[2]
	params := statediff.Params{
		WatchedAddresses:    []common.Address{testhelpers.Account1Addr, testhelpers.ContractAddr},
		WatchedStorageSlots: []common.Hash{slot1StorageKey},
	}
	builder = statediff.NewBuilder(chain.StateCache())

	var tests = []struct {
		name              string
		startingArguments statediff.Args
		expected          *statediff.StateObject
	}{
		{
			"testEmptyDiff",
			statediff.Args{
				OldStateRoot: block0.Root(),
				NewStateRoot: block0.Root(),
				BlockNumber:  block0.Number(),
				BlockHash:    block0.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block0.Number(),
				BlockHash:   block0.Hash(),
				Nodes:       emptyDiffs,
			},
		},
		{
			"testBlock0",
			//10000 transferred from testBankAddress to account1Addr
			statediff.Args{
				OldStateRoot: testhelpers.NullHash,
				NewStateRoot: block0.Root(),
				BlockNumber:  block0.Number(),
				BlockHash:    block0.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block0.Number(),
				BlockHash:   block0.Hash(),
				Nodes:       emptyDiffs,
			},
		},
		{
			"testBlock1",
			//10000 transferred from testBankAddress to account1Addr
			statediff.Args{
				OldStateRoot: block0.Root(),
				NewStateRoot: block1.Root(),
				BlockNumber:  block1.Number(),
				BlockHash:    block1.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block1.Number(),
				BlockHash:   block1.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{'\x0e'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account1LeafKey,
						NodeValue:    account1AtBlock1LeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
		{
			"testBlock2",
			//1000 transferred from testBankAddress to account1Addr
			//1000 transferred from account1Addr to account2Addr
			statediff.Args{
				OldStateRoot: block1.Root(),
				NewStateRoot: block2.Root(),
				BlockNumber:  block2.Number(),
				BlockHash:    block2.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block2.Number(),
				BlockHash:   block2.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:      []byte{'\x06'},
						NodeType:  sdtypes.Leaf,
						LeafKey:   contractLeafKey,
						NodeValue: contractAccountAtBlock2LeafNode,
						StorageNodes: []sdtypes.StorageNode{
							{
								Path:      []byte{'\x0b'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot1StorageKey.Bytes(),
								NodeValue: slot1StorageLeafNode,
							},
						},
					},
					{
						Path:         []byte{'\x0e'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account1LeafKey,
						NodeValue:    account1AtBlock2LeafNode,
						StorageNodes: emptyStorage,
					},
				},
				CodeAndCodeHashes: []sdtypes.CodeAndCodeHash{
					{
						Hash: testhelpers.CodeHash,
						Code: testhelpers.ByteCodeAfterDeployment,
					},
				},
			},
		},
		{
			"testBlock3",
			//the contract's storage is changed
			//and the block is mined by account 2
			statediff.Args{
				OldStateRoot: block2.Root(),
				NewStateRoot: block3.Root(),
				BlockNumber:  block3.Number(),
				BlockHash:    block3.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block3.Number(),
				BlockHash:   block3.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{'\x06'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      contractLeafKey,
						NodeValue:    contractAccountAtBlock3LeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
	}

	for _, test := range tests {
		diff, err := builder.BuildStateDiffObject(test.startingArguments, params)
		if err != nil {
			t.Error(err)
		}
		receivedStateDiffRlp, err := rlp.EncodeToBytes(diff)
		if err != nil {
			t.Error(err)
		}
		expectedStateDiffRlp, err := rlp.EncodeToBytes(test.expected)
		if err != nil {
			t.Error(err)
		}
		sort.Slice(receivedStateDiffRlp, func(i, j int) bool { return receivedStateDiffRlp[i] < receivedStateDiffRlp[j] })
		sort.Slice(expectedStateDiffRlp, func(i, j int) bool { return expectedStateDiffRlp[i] < expectedStateDiffRlp[j] })
		if !bytes.Equal(receivedStateDiffRlp, expectedStateDiffRlp) {
			t.Logf("Test failed: %s", test.name)
			t.Errorf("actual state diff: %+v\nexpected state diff: %+v", diff, test.expected)
		}
	}
}

func TestBuilderWithRemovedAccountAndStorage(t *testing.T) {
	blocks, chain := testhelpers.MakeChain(6, testhelpers.Genesis, testhelpers.TestChainGen)
	contractLeafKey = testhelpers.AddressToLeafKey(testhelpers.ContractAddr)
	defer chain.Stop()
	block3 = blocks[2]
	block4 = blocks[3]
	block5 = blocks[4]
	block6 = blocks[5]
	params := statediff.Params{
		IntermediateStateNodes:   true,
		IntermediateStorageNodes: true,
	}
	builder = statediff.NewBuilder(chain.StateCache())

	var tests = []struct {
		name              string
		startingArguments statediff.Args
		expected          *statediff.StateObject
	}{
		// blocks 0-3 are the same as in TestBuilderWithIntermediateNodes
		{
			"testBlock4",
			statediff.Args{
				OldStateRoot: block3.Root(),
				NewStateRoot: block4.Root(),
				BlockNumber:  block4.Number(),
				BlockHash:    block4.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block4.Number(),
				BlockHash:   block4.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Branch,
						NodeValue:    block4BranchRootNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x00'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock4LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:      []byte{'\x06'},
						NodeType:  sdtypes.Leaf,
						LeafKey:   contractLeafKey,
						NodeValue: contractAccountAtBlock4LeafNode,
						StorageNodes: []sdtypes.StorageNode{
							{
								Path:      []byte{},
								NodeType:  sdtypes.Branch,
								NodeValue: block4StorageBranchRootNode,
							},
							{
								Path:      []byte{'\x04'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot2StorageKey.Bytes(),
								NodeValue: slot2StorageLeafNode,
							},
							{
								Path:      []byte{'\x0b'},
								NodeType:  sdtypes.Removed,
								NodeValue: []byte{},
							},
							{
								Path:      []byte{'\x0c'},
								NodeType:  sdtypes.Removed,
								NodeValue: []byte{},
							},
						},
					},
					{
						Path:         []byte{'\x0c'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account2LeafKey,
						NodeValue:    account2AtBlock4LeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
		{
			"testBlock5",
			statediff.Args{
				OldStateRoot: block4.Root(),
				NewStateRoot: block5.Root(),
				BlockNumber:  block5.Number(),
				BlockHash:    block5.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block5.Number(),
				BlockHash:   block5.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Branch,
						NodeValue:    block5BranchRootNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x00'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock5LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:      []byte{'\x06'},
						NodeType:  sdtypes.Leaf,
						LeafKey:   contractLeafKey,
						NodeValue: contractAccountAtBlock5LeafNode,
						StorageNodes: []sdtypes.StorageNode{
							{
								Path:      []byte{},
								NodeType:  sdtypes.Leaf,
								NodeValue: slot0StorageLeafRootNode,
								LeafKey:   slot0StorageKey.Bytes(),
							},
							{
								Path:      []byte{'\x02'},
								NodeType:  sdtypes.Removed,
								NodeValue: []byte{},
							},
							{
								Path:      []byte{'\x04'},
								NodeType:  sdtypes.Removed,
								NodeValue: []byte{},
							},
						},
					},
					{
						Path:         []byte{'\x0e'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account1LeafKey,
						NodeValue:    account1AtBlock5LeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
		{
			"testBlock6",
			statediff.Args{
				OldStateRoot: block5.Root(),
				NewStateRoot: block6.Root(),
				BlockNumber:  block6.Number(),
				BlockHash:    block6.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block6.Number(),
				BlockHash:   block6.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Branch,
						NodeValue:    block6BranchRootNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:      []byte{'\x06'},
						NodeType:  sdtypes.Removed,
						NodeValue: []byte{},
					},
					{
						Path:         []byte{'\x0c'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account2LeafKey,
						NodeValue:    account2AtBlock6LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x0e'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account1LeafKey,
						NodeValue:    account1AtBlock6LeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
	}

	for _, test := range tests {
		diff, err := builder.BuildStateDiffObject(test.startingArguments, params)
		if err != nil {
			t.Error(err)
		}
		receivedStateDiffRlp, err := rlp.EncodeToBytes(diff)
		if err != nil {
			t.Error(err)
		}
		expectedStateDiffRlp, err := rlp.EncodeToBytes(test.expected)
		if err != nil {
			t.Error(err)
		}
		sort.Slice(receivedStateDiffRlp, func(i, j int) bool { return receivedStateDiffRlp[i] < receivedStateDiffRlp[j] })
		sort.Slice(expectedStateDiffRlp, func(i, j int) bool { return expectedStateDiffRlp[i] < expectedStateDiffRlp[j] })
		if !bytes.Equal(receivedStateDiffRlp, expectedStateDiffRlp) {
			t.Logf("Test failed: %s", test.name)
			t.Errorf("actual state diff: %+v\r\n\r\n\r\nexpected state diff: %+v", diff, test.expected)
		}
	}
}

func TestBuilderWithRemovedAccountAndStorageWithoutIntermediateNodes(t *testing.T) {
	blocks, chain := testhelpers.MakeChain(6, testhelpers.Genesis, testhelpers.TestChainGen)
	contractLeafKey = testhelpers.AddressToLeafKey(testhelpers.ContractAddr)
	defer chain.Stop()
	block3 = blocks[2]
	block4 = blocks[3]
	block5 = blocks[4]
	block6 = blocks[5]
	params := statediff.Params{
		IntermediateStateNodes:   false,
		IntermediateStorageNodes: false,
	}
	builder = statediff.NewBuilder(chain.StateCache())

	var tests = []struct {
		name              string
		startingArguments statediff.Args
		expected          *statediff.StateObject
	}{
		// blocks 0-3 are the same as in TestBuilderWithIntermediateNodes
		{
			"testBlock4",
			statediff.Args{
				OldStateRoot: block3.Root(),
				NewStateRoot: block4.Root(),
				BlockNumber:  block4.Number(),
				BlockHash:    block4.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block4.Number(),
				BlockHash:   block4.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{'\x00'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock4LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:      []byte{'\x06'},
						NodeType:  sdtypes.Leaf,
						LeafKey:   contractLeafKey,
						NodeValue: contractAccountAtBlock4LeafNode,
						StorageNodes: []sdtypes.StorageNode{
							{
								Path:      []byte{'\x04'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot2StorageKey.Bytes(),
								NodeValue: slot2StorageLeafNode,
							},
							{
								Path:      []byte{'\x0b'},
								NodeType:  sdtypes.Removed,
								NodeValue: []byte{},
							},
							{
								Path:      []byte{'\x0c'},
								NodeType:  sdtypes.Removed,
								NodeValue: []byte{},
							},
						},
					},
					{
						Path:         []byte{'\x0c'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account2LeafKey,
						NodeValue:    account2AtBlock4LeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
		{
			"testBlock5",
			statediff.Args{
				OldStateRoot: block4.Root(),
				NewStateRoot: block5.Root(),
				BlockNumber:  block5.Number(),
				BlockHash:    block5.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block5.Number(),
				BlockHash:   block5.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{'\x00'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock5LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:      []byte{'\x06'},
						NodeType:  sdtypes.Leaf,
						LeafKey:   contractLeafKey,
						NodeValue: contractAccountAtBlock5LeafNode,
						StorageNodes: []sdtypes.StorageNode{
							{
								Path:      []byte{},
								NodeType:  sdtypes.Leaf,
								NodeValue: slot0StorageLeafRootNode,
								LeafKey:   slot0StorageKey.Bytes(),
							},
							{
								Path:      []byte{'\x02'},
								NodeType:  sdtypes.Removed,
								NodeValue: []byte{},
							},
							{
								Path:      []byte{'\x04'},
								NodeType:  sdtypes.Removed,
								NodeValue: []byte{},
							},
						},
					},
					{
						Path:         []byte{'\x0e'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account1LeafKey,
						NodeValue:    account1AtBlock5LeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
		{
			"testBlock6",
			statediff.Args{
				OldStateRoot: block5.Root(),
				NewStateRoot: block6.Root(),
				BlockNumber:  block6.Number(),
				BlockHash:    block6.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block6.Number(),
				BlockHash:   block6.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:      []byte{'\x06'},
						NodeType:  sdtypes.Removed,
						NodeValue: []byte{},
					},
					{
						Path:         []byte{'\x0c'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account2LeafKey,
						NodeValue:    account2AtBlock6LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x0e'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account1LeafKey,
						NodeValue:    account1AtBlock6LeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
	}

	for _, test := range tests {
		diff, err := builder.BuildStateDiffObject(test.startingArguments, params)
		if err != nil {
			t.Error(err)
		}
		receivedStateDiffRlp, err := rlp.EncodeToBytes(diff)
		if err != nil {
			t.Error(err)
		}
		expectedStateDiffRlp, err := rlp.EncodeToBytes(test.expected)
		if err != nil {
			t.Error(err)
		}
		sort.Slice(receivedStateDiffRlp, func(i, j int) bool { return receivedStateDiffRlp[i] < receivedStateDiffRlp[j] })
		sort.Slice(expectedStateDiffRlp, func(i, j int) bool { return expectedStateDiffRlp[i] < expectedStateDiffRlp[j] })
		if !bytes.Equal(receivedStateDiffRlp, expectedStateDiffRlp) {
			t.Logf("Test failed: %s", test.name)
			t.Errorf("actual state diff: %+v\r\n\r\n\r\nexpected state diff: %+v", diff, test.expected)
		}
	}
}

var (
	slot00StorageValue = common.Hex2Bytes("9471562b71999873db5b286df957af199ec94617f7") // prefixed TestBankAddress

	slot00StorageLeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("390decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563"),
		slot00StorageValue,
	})

	contractAccountAtBlock01, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    1,
		Balance:  big.NewInt(0),
		CodeHash: common.HexToHash("0xaaea5efba4fd7b45d7ec03918ac5d8b31aa93b48986af0e6b591f0f087c80127").Bytes(),
		Root:     crypto.Keccak256Hash(block01StorageBranchRootNode),
	})
	contractAccountAtBlock01LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3cb2583748c26e89ef19c2a8529b05a270f735553b4d44b6f2a1894987a71c8b"),
		contractAccountAtBlock01,
	})

	bankAccountAtBlock01, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    1,
		Balance:  big.NewInt(testhelpers.TestBankFunds.Int64() + miningReward),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	bankAccountAtBlock01LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("30bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"),
		bankAccountAtBlock01,
	})
	bankAccountAtBlock02, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    2,
		Balance:  big.NewInt(testhelpers.TestBankFunds.Int64() + miningReward*2),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	bankAccountAtBlock02LeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("2000bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"),
		bankAccountAtBlock02,
	})

	block01BranchRootNode, _ = rlp.EncodeToBytes([]interface{}{
		crypto.Keccak256Hash(bankAccountAtBlock01LeafNode),
		crypto.Keccak256Hash(contractAccountAtBlock01LeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
	})

	block01StorageBranchRootNode, _ = rlp.EncodeToBytes([]interface{}{
		[]byte{},
		[]byte{},
		crypto.Keccak256(slot00StorageLeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		crypto.Keccak256(slot1StorageLeafNode),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
	})
)

func TestBuilderWithMovedAccount(t *testing.T) {
	blocks, chain := testhelpers.MakeChain(2, testhelpers.Genesis, testhelpers.TestSelfDestructChainGen)
	contractLeafKey = testhelpers.AddressToLeafKey(testhelpers.ContractAddr)
	defer chain.Stop()
	block0 = testhelpers.Genesis
	block1 = blocks[0]
	block2 = blocks[1]
	params := statediff.Params{
		IntermediateStateNodes:   true,
		IntermediateStorageNodes: true,
	}
	builder = statediff.NewBuilder(chain.StateCache())

	var tests = []struct {
		name              string
		startingArguments statediff.Args
		expected          *statediff.StateObject
	}{
		{
			"testBlock1",
			statediff.Args{
				OldStateRoot: block0.Root(),
				NewStateRoot: block1.Root(),
				BlockNumber:  block1.Number(),
				BlockHash:    block1.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block1.Number(),
				BlockHash:   block1.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Branch,
						NodeValue:    block01BranchRootNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x00'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock01LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:      []byte{'\x01'},
						NodeType:  sdtypes.Leaf,
						LeafKey:   contractLeafKey,
						NodeValue: contractAccountAtBlock01LeafNode,
						StorageNodes: []sdtypes.StorageNode{
							{
								Path:      []byte{},
								NodeType:  sdtypes.Branch,
								NodeValue: block01StorageBranchRootNode,
							},
							{
								Path:      []byte{'\x02'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot0StorageKey.Bytes(),
								NodeValue: slot00StorageLeafNode,
							},
							{
								Path:      []byte{'\x0b'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot1StorageKey.Bytes(),
								NodeValue: slot1StorageLeafNode,
							},
						},
					},
				},
				CodeAndCodeHashes: []sdtypes.CodeAndCodeHash{
					{
						Hash: testhelpers.CodeHash,
						Code: testhelpers.ByteCodeAfterDeployment,
					},
				},
			},
		},
		{
			"testBlock2",
			statediff.Args{
				OldStateRoot: block1.Root(),
				NewStateRoot: block2.Root(),
				BlockNumber:  block2.Number(),
				BlockHash:    block2.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block2.Number(),
				BlockHash:   block2.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock02LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:      []byte{'\x01'},
						NodeType:  sdtypes.Removed,
						NodeValue: []byte{},
					},
					{
						Path:      []byte{'\x00'},
						NodeType:  sdtypes.Removed,
						NodeValue: []byte{},
					},
				},
			},
		},
	}

	for _, test := range tests {
		diff, err := builder.BuildStateDiffObject(test.startingArguments, params)
		if err != nil {
			t.Error(err)
		}
		receivedStateDiffRlp, err := rlp.EncodeToBytes(diff)
		if err != nil {
			t.Error(err)
		}
		expectedStateDiffRlp, err := rlp.EncodeToBytes(test.expected)
		if err != nil {
			t.Error(err)
		}
		sort.Slice(receivedStateDiffRlp, func(i, j int) bool { return receivedStateDiffRlp[i] < receivedStateDiffRlp[j] })
		sort.Slice(expectedStateDiffRlp, func(i, j int) bool { return expectedStateDiffRlp[i] < expectedStateDiffRlp[j] })
		if !bytes.Equal(receivedStateDiffRlp, expectedStateDiffRlp) {
			t.Logf("Test failed: %s", test.name)
			t.Errorf("actual state diff: %+v\r\n\r\n\r\nexpected state diff: %+v", diff, test.expected)
		}
	}
}

func TestBuilderWithMovedAccountOnlyLeafs(t *testing.T) {
	blocks, chain := testhelpers.MakeChain(2, testhelpers.Genesis, testhelpers.TestSelfDestructChainGen)
	contractLeafKey = testhelpers.AddressToLeafKey(testhelpers.ContractAddr)
	defer chain.Stop()
	block0 = testhelpers.Genesis
	block1 = blocks[0]
	block2 = blocks[1]
	params := statediff.Params{
		IntermediateStateNodes:   false,
		IntermediateStorageNodes: false,
	}
	builder = statediff.NewBuilder(chain.StateCache())

	var tests = []struct {
		name              string
		startingArguments statediff.Args
		expected          *statediff.StateObject
	}{
		{
			"testBlock1",
			statediff.Args{
				OldStateRoot: block0.Root(),
				NewStateRoot: block1.Root(),
				BlockNumber:  block1.Number(),
				BlockHash:    block1.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block1.Number(),
				BlockHash:   block1.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{'\x00'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock01LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:      []byte{'\x01'},
						NodeType:  sdtypes.Leaf,
						LeafKey:   contractLeafKey,
						NodeValue: contractAccountAtBlock01LeafNode,
						StorageNodes: []sdtypes.StorageNode{
							{
								Path:      []byte{'\x02'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot0StorageKey.Bytes(),
								NodeValue: slot00StorageLeafNode,
							},
							{
								Path:      []byte{'\x0b'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot1StorageKey.Bytes(),
								NodeValue: slot1StorageLeafNode,
							},
						},
					},
				},
				CodeAndCodeHashes: []sdtypes.CodeAndCodeHash{
					{
						Hash: testhelpers.CodeHash,
						Code: testhelpers.ByteCodeAfterDeployment,
					},
				},
			},
		},
		{
			"testBlock2",
			statediff.Args{
				OldStateRoot: block1.Root(),
				NewStateRoot: block2.Root(),
				BlockNumber:  block2.Number(),
				BlockHash:    block2.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block2.Number(),
				BlockHash:   block2.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock02LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:      []byte{'\x01'},
						NodeType:  sdtypes.Removed,
						NodeValue: []byte{},
					},
					{
						Path:      []byte{'\x00'},
						NodeType:  sdtypes.Removed,
						NodeValue: []byte{},
					},
				},
			},
		},
	}

	for _, test := range tests {
		diff, err := builder.BuildStateDiffObject(test.startingArguments, params)
		if err != nil {
			t.Error(err)
		}
		receivedStateDiffRlp, err := rlp.EncodeToBytes(diff)
		if err != nil {
			t.Error(err)
		}
		expectedStateDiffRlp, err := rlp.EncodeToBytes(test.expected)
		if err != nil {
			t.Error(err)
		}
		sort.Slice(receivedStateDiffRlp, func(i, j int) bool { return receivedStateDiffRlp[i] < receivedStateDiffRlp[j] })
		sort.Slice(expectedStateDiffRlp, func(i, j int) bool { return expectedStateDiffRlp[i] < expectedStateDiffRlp[j] })
		if !bytes.Equal(receivedStateDiffRlp, expectedStateDiffRlp) {
			t.Logf("Test failed: %s", test.name)
			t.Errorf("actual state diff: %+v\r\n\r\n\r\nexpected state diff: %+v", diff, test.expected)
		}
	}
}

func TestBuildStateTrie(t *testing.T) {
	blocks, chain := testhelpers.MakeChain(3, testhelpers.Genesis, testhelpers.TestChainGen)
	contractLeafKey = testhelpers.AddressToLeafKey(testhelpers.ContractAddr)
	defer chain.Stop()
	block1 = blocks[0]
	block2 = blocks[1]
	block3 = blocks[2]
	builder = statediff.NewBuilder(chain.StateCache())

	var tests = []struct {
		name     string
		block    *types.Block
		expected *statediff.StateObject
	}{
		{
			"testBlock1",
			block1,
			&statediff.StateObject{
				BlockNumber: block1.Number(),
				BlockHash:   block1.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Branch,
						NodeValue:    block1BranchRootNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x00'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock1LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x05'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      minerLeafKey,
						NodeValue:    minerAccountAtBlock1LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x0e'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account1LeafKey,
						NodeValue:    account1AtBlock1LeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
		{
			"testBlock2",
			block2,
			&statediff.StateObject{
				BlockNumber: block2.Number(),
				BlockHash:   block2.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Branch,
						NodeValue:    block2BranchRootNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x00'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock2LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x05'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      minerLeafKey,
						NodeValue:    minerAccountAtBlock2LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x0e'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account1LeafKey,
						NodeValue:    account1AtBlock2LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:      []byte{'\x06'},
						NodeType:  sdtypes.Leaf,
						LeafKey:   contractLeafKey,
						NodeValue: contractAccountAtBlock2LeafNode,
						StorageNodes: []sdtypes.StorageNode{
							{
								Path:      []byte{},
								NodeType:  sdtypes.Branch,
								NodeValue: block2StorageBranchRootNode,
							},
							{
								Path:      []byte{'\x02'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot0StorageKey.Bytes(),
								NodeValue: slot0StorageLeafNode,
							},
							{
								Path:      []byte{'\x0b'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot1StorageKey.Bytes(),
								NodeValue: slot1StorageLeafNode,
							},
						},
					},
					{
						Path:         []byte{'\x0c'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account2LeafKey,
						NodeValue:    account2AtBlock2LeafNode,
						StorageNodes: emptyStorage,
					},
				},
				CodeAndCodeHashes: []sdtypes.CodeAndCodeHash{
					{
						Hash: testhelpers.CodeHash,
						Code: testhelpers.ByteCodeAfterDeployment,
					},
				},
			},
		},
		{
			"testBlock3",
			block3,
			&statediff.StateObject{
				BlockNumber: block3.Number(),
				BlockHash:   block3.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Branch,
						NodeValue:    block3BranchRootNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x00'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.BankLeafKey,
						NodeValue:    bankAccountAtBlock3LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x05'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      minerLeafKey,
						NodeValue:    minerAccountAtBlock2LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:         []byte{'\x0e'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account1LeafKey,
						NodeValue:    account1AtBlock2LeafNode,
						StorageNodes: emptyStorage,
					},
					{
						Path:      []byte{'\x06'},
						NodeType:  sdtypes.Leaf,
						LeafKey:   contractLeafKey,
						NodeValue: contractAccountAtBlock3LeafNode,
						StorageNodes: []sdtypes.StorageNode{
							{
								Path:      []byte{},
								NodeType:  sdtypes.Branch,
								NodeValue: block3StorageBranchRootNode,
							},
							{
								Path:      []byte{'\x02'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot0StorageKey.Bytes(),
								NodeValue: slot0StorageLeafNode,
							},
							{
								Path:      []byte{'\x0b'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot1StorageKey.Bytes(),
								NodeValue: slot1StorageLeafNode,
							},
							{
								Path:      []byte{'\x0c'},
								NodeType:  sdtypes.Leaf,
								LeafKey:   slot3StorageKey.Bytes(),
								NodeValue: slot3StorageLeafNode,
							},
						},
					},
					{
						Path:         []byte{'\x0c'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      testhelpers.Account2LeafKey,
						NodeValue:    account2AtBlock3LeafNode,
						StorageNodes: emptyStorage,
					},
				},
				CodeAndCodeHashes: []sdtypes.CodeAndCodeHash{
					{
						Hash: testhelpers.CodeHash,
						Code: testhelpers.ByteCodeAfterDeployment,
					},
				},
			},
		},
	}

	for _, test := range tests {
		diff, err := builder.BuildStateTrieObject(test.block)
		if err != nil {
			t.Error(err)
		}
		receivedStateTrieRlp, err := rlp.EncodeToBytes(diff)
		if err != nil {
			t.Error(err)
		}
		expectedStateTrieRlp, err := rlp.EncodeToBytes(test.expected)
		if err != nil {
			t.Error(err)
		}
		sort.Slice(receivedStateTrieRlp, func(i, j int) bool { return receivedStateTrieRlp[i] < receivedStateTrieRlp[j] })
		sort.Slice(expectedStateTrieRlp, func(i, j int) bool { return expectedStateTrieRlp[i] < expectedStateTrieRlp[j] })
		if !bytes.Equal(receivedStateTrieRlp, expectedStateTrieRlp) {
			t.Logf("Test failed: %s", test.name)
			t.Errorf("actual state trie: %+v\r\n\r\n\r\nexpected state trie: %+v", diff, test.expected)
		}
	}
}

/*
pragma solidity ^0.5.10;

contract test {
    address payable owner;

    modifier onlyOwner {
        require(
            msg.sender == owner,
            "Only owner can call this function."
        );
        _;
    }

    uint256[100] data;

	constructor() public {
	    owner = msg.sender;
		data = [1];
	}

    function Put(uint256 addr, uint256 value) public {
        data[addr] = value;
    }

    function close() public onlyOwner { //onlyOwner is custom modifier
        selfdestruct(owner);  // `owner` is the owners address
    }
}
*/

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
)

var block0, block1 *types.Block
var burnLeafKey = testhelpers.AddressToLeafKey(common.HexToAddress("0x0"))
var emptyAccountDiffEventualMap = make([]statediff.AccountDiff, 0)
var account1, _ = rlp.EncodeToBytes(state.Account{
	Nonce:    uint64(0),
	Balance:  big.NewInt(10000),
	CodeHash: common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
	Root:     common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
})
var burnAccount1, _ = rlp.EncodeToBytes(state.Account{
	Nonce:    uint64(0),
	Balance:  big.NewInt(2000000000000000000),
	CodeHash: common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
	Root:     common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
})
var bankAccount1, _ = rlp.EncodeToBytes(state.Account{
	Nonce:    uint64(1),
	Balance:  big.NewInt(testhelpers.TestBankFunds.Int64() - 10000),
	CodeHash: common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
	Root:     common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
})

func TestAPI(t *testing.T) {
	_, blockMap, chain := testhelpers.MakeChain(3, testhelpers.Genesis)
	defer chain.Stop()
	block0Hash := common.HexToHash("0xd1721cfd0b29c36fd7a68f25c128e86413fb666a6e1d68e89b875bd299262661")
	block1Hash := common.HexToHash("0xbbe88de60ba33a3f18c0caa37d827bfb70252e19e40a07cd34041696c35ecb1a")
	block0 = blockMap[block0Hash]
	block1 = blockMap[block1Hash]
	blockChan := make(chan *types.Block)
	parentBlockChain := make(chan *types.Block)
	serviceQuitChan := make(chan bool)
	config := statediff.Config{
		PathsAndProofs: true,
		AllNodes:       false,
	}
	mockService := MockStateDiffService{
		Mutex:           sync.Mutex{},
		Builder:         statediff.NewBuilder(testhelpers.Testdb, chain, config),
		BlockChan:       blockChan,
		ParentBlockChan: parentBlockChain,
		QuitChan:        serviceQuitChan,
		Subscriptions:   make(map[rpc.ID]statediff.Subscription),
		streamBlock:     true,
	}
	mockService.Start(nil)
	id := rpc.NewID()
	payloadChan := make(chan statediff.Payload)
	quitChan := make(chan bool)
	mockService.Subscribe(id, payloadChan, quitChan)
	blockChan <- block1
	parentBlockChain <- block0
	expectedBlockRlp, _ := rlp.EncodeToBytes(block1)
	expectedStateDiff := statediff.StateDiff{
		BlockNumber: block1.Number(),
		BlockHash:   block1.Hash(),
		CreatedAccounts: []statediff.AccountDiff{
			{
				Leaf:  true,
				Key:   burnLeafKey.Bytes(),
				Value: burnAccount1,
				Proof: [][]byte{{248, 113, 160, 87, 118, 82, 182, 37, 183, 123, 219, 91, 247, 123, 196, 63, 49, 37, 202, 215, 70, 77, 103, 157, 21, 117, 86, 82, 119, 211, 97, 27, 128, 83, 231, 128, 128, 128, 128, 160, 254, 136, 159, 16, 229, 219, 143, 44, 43, 243, 85, 146, 129, 82, 161, 127, 110, 59, 185, 154, 146, 65, 172, 109, 132, 199, 126, 98, 100, 80, 156, 121, 128, 128, 128, 128, 128, 128, 128, 128, 160, 17, 219, 12, 218, 52, 168, 150, 218, 190, 182, 131, 155, 176, 106, 56, 244, 149, 20, 207, 164, 134, 67, 89, 132, 235, 1, 59, 125, 249, 238, 133, 197, 128, 128},
					{248, 113, 160, 51, 128, 199, 183, 174, 129, 165, 142, 185, 141, 156, 120, 222, 74, 31, 215, 253, 149, 53, 252, 149, 62, 210, 190, 96, 45, 170, 164, 23, 103, 49, 42, 184, 78, 248, 76, 128, 136, 27, 193, 109, 103, 78, 200, 0, 0, 160, 86, 232, 31, 23, 27, 204, 85, 166, 255, 131, 69, 230, 146, 192, 248, 110, 91, 72, 224, 27, 153, 108, 173, 192, 1, 98, 47, 181, 227, 99, 180, 33, 160, 197, 210, 70, 1, 134, 247, 35, 60, 146, 126, 125, 178, 220, 199, 3, 192, 229, 0, 182, 83, 202, 130, 39, 59, 123, 250, 216, 4, 93, 133, 164, 112}},
				Path:    []byte{5, 3, 8, 0, 12, 7, 11, 7, 10, 14, 8, 1, 10, 5, 8, 14, 11, 9, 8, 13, 9, 12, 7, 8, 13, 14, 4, 10, 1, 15, 13, 7, 15, 13, 9, 5, 3, 5, 15, 12, 9, 5, 3, 14, 13, 2, 11, 14, 6, 0, 2, 13, 10, 10, 10, 4, 1, 7, 6, 7, 3, 1, 2, 10, 16},
				Storage: []statediff.StorageDiff{},
			},
			{
				Leaf:  true,
				Key:   testhelpers.Account1LeafKey.Bytes(),
				Value: account1,
				Proof: [][]byte{{248, 113, 160, 87, 118, 82, 182, 37, 183, 123, 219, 91, 247, 123, 196, 63, 49, 37, 202, 215, 70, 77, 103, 157, 21, 117, 86, 82, 119, 211, 97, 27, 128, 83, 231, 128, 128, 128, 128, 160, 254, 136, 159, 16, 229, 219, 143, 44, 43, 243, 85, 146, 129, 82, 161, 127, 110, 59, 185, 154, 146, 65, 172, 109, 132, 199, 126, 98, 100, 80, 156, 121, 128, 128, 128, 128, 128, 128, 128, 128, 160, 17, 219, 12, 218, 52, 168, 150, 218, 190, 182, 131, 155, 176, 106, 56, 244, 149, 20, 207, 164, 134, 67, 89, 132, 235, 1, 59, 125, 249, 238, 133, 197, 128, 128},
					{248, 107, 160, 57, 38, 219, 105, 170, 206, 213, 24, 233, 185, 240, 244, 52, 164, 115, 231, 23, 65, 9, 201, 67, 84, 139, 184, 242, 59, 228, 28, 167, 109, 154, 210, 184, 72, 248, 70, 128, 130, 39, 16, 160, 86, 232, 31, 23, 27, 204, 85, 166, 255, 131, 69, 230, 146, 192, 248, 110, 91, 72, 224, 27, 153, 108, 173, 192, 1, 98, 47, 181, 227, 99, 180, 33, 160, 197, 210, 70, 1, 134, 247, 35, 60, 146, 126, 125, 178, 220, 199, 3, 192, 229, 0, 182, 83, 202, 130, 39, 59, 123, 250, 216, 4, 93, 133, 164, 112}},
				Path:    []byte{14, 9, 2, 6, 13, 11, 6, 9, 10, 10, 12, 14, 13, 5, 1, 8, 14, 9, 11, 9, 15, 0, 15, 4, 3, 4, 10, 4, 7, 3, 14, 7, 1, 7, 4, 1, 0, 9, 12, 9, 4, 3, 5, 4, 8, 11, 11, 8, 15, 2, 3, 11, 14, 4, 1, 12, 10, 7, 6, 13, 9, 10, 13, 2, 16},
				Storage: []statediff.StorageDiff{},
			},
		},
		DeletedAccounts: emptyAccountDiffEventualMap,
		UpdatedAccounts: []statediff.AccountDiff{
			{
				Leaf:  true,
				Key:   testhelpers.BankLeafKey.Bytes(),
				Value: bankAccount1,
				Proof: [][]byte{{248, 113, 160, 87, 118, 82, 182, 37, 183, 123, 219, 91, 247, 123, 196, 63, 49, 37, 202, 215, 70, 77, 103, 157, 21, 117, 86, 82, 119, 211, 97, 27, 128, 83, 231, 128, 128, 128, 128, 160, 254, 136, 159, 16, 229, 219, 143, 44, 43, 243, 85, 146, 129, 82, 161, 127, 110, 59, 185, 154, 146, 65, 172, 109, 132, 199, 126, 98, 100, 80, 156, 121, 128, 128, 128, 128, 128, 128, 128, 128, 160, 17, 219, 12, 218, 52, 168, 150, 218, 190, 182, 131, 155, 176, 106, 56, 244, 149, 20, 207, 164, 134, 67, 89, 132, 235, 1, 59, 125, 249, 238, 133, 197, 128, 128},
					{248, 109, 160, 48, 191, 73, 244, 64, 161, 205, 5, 39, 228, 208, 110, 39, 101, 101, 76, 15, 86, 69, 34, 87, 81, 109, 121, 58, 155, 141, 96, 77, 207, 223, 42, 184, 74, 248, 72, 1, 132, 5, 245, 185, 240, 160, 86, 232, 31, 23, 27, 204, 85, 166, 255, 131, 69, 230, 146, 192, 248, 110, 91, 72, 224, 27, 153, 108, 173, 192, 1, 98, 47, 181, 227, 99, 180, 33, 160, 197, 210, 70, 1, 134, 247, 35, 60, 146, 126, 125, 178, 220, 199, 3, 192, 229, 0, 182, 83, 202, 130, 39, 59, 123, 250, 216, 4, 93, 133, 164, 112}},
				Path:    []byte{0, 0, 11, 15, 4, 9, 15, 4, 4, 0, 10, 1, 12, 13, 0, 5, 2, 7, 14, 4, 13, 0, 6, 14, 2, 7, 6, 5, 6, 5, 4, 12, 0, 15, 5, 6, 4, 5, 2, 2, 5, 7, 5, 1, 6, 13, 7, 9, 3, 10, 9, 11, 8, 13, 6, 0, 4, 13, 12, 15, 13, 15, 2, 10, 16},
				Storage: []statediff.StorageDiff{},
			},
		},
	}
	expectedStateDiffBytes, err := rlp.EncodeToBytes(expectedStateDiff)
	if err != nil {
		t.Error(err)
	}
	sort.Slice(expectedStateDiffBytes, func(i, j int) bool { return expectedStateDiffBytes[i] < expectedStateDiffBytes[j] })

	select {
	case payload := <-payloadChan:
		if !bytes.Equal(payload.BlockRlp, expectedBlockRlp) {
			t.Errorf("payload does not have expected block\r\actual block rlp: %v\r\nexpected block rlp: %v", payload.BlockRlp, expectedBlockRlp)
		}
		sort.Slice(payload.StateDiffRlp, func(i, j int) bool { return payload.StateDiffRlp[i] < payload.StateDiffRlp[j] })
		if !bytes.Equal(payload.StateDiffRlp, expectedStateDiffBytes) {
			t.Errorf("payload does not have expected state diff\r\actual state diff rlp: %v\r\nexpected state diff rlp: %v", payload.StateDiffRlp, expectedStateDiffBytes)
		}
		if payload.Err != nil {
			t.Errorf("payload should not contain an error, but does: %v", payload.Err)
		}
	case <-quitChan:
		t.Errorf("channel quit before delivering payload")
	}
}

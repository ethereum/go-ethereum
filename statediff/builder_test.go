// Copyright 2015 The go-ethereum Authors
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
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/statediff"
	"github.com/ethereum/go-ethereum/statediff/testhelpers"
)

var (
	contractLeafKey                                common.Hash
	emptyAccountDiffEventualMap                    = make(statediff.AccountDiffsMap)
	emptyAccountDiffIncrementalMap                 = make(statediff.AccountDiffsMap)
	block0Hash, block1Hash, block2Hash, block3Hash common.Hash
	block0, block1, block2, block3                 *types.Block
	builder                                        statediff.Builder
	miningReward                                   = int64(2000000000000000000)
	burnAddress                                    = common.HexToAddress("0x0")
	burnLeafKey                                    = testhelpers.AddressToLeafKey(burnAddress)
)

func TestBuilder(t *testing.T) {
	_, blockMap, chain := testhelpers.MakeChain(3, testhelpers.Genesis)
	contractLeafKey = testhelpers.AddressToLeafKey(testhelpers.ContractAddr)
	defer chain.Stop()
	block0Hash = common.HexToHash("0xd1721cfd0b29c36fd7a68f25c128e86413fb666a6e1d68e89b875bd299262661")
	block1Hash = common.HexToHash("0xbbe88de60ba33a3f18c0caa37d827bfb70252e19e40a07cd34041696c35ecb1a")
	block2Hash = common.HexToHash("0x34ad0fd9bb2911986b75d518c822641079dea823bc6952343ebf05da1062b6f5")
	block3Hash = common.HexToHash("0x9872058136c560a6ebed0c0522b8d3016fc21f4fb0fb6585ddd8fd4c54f9909a")

	block0 = blockMap[block0Hash]
	block1 = blockMap[block1Hash]
	block2 = blockMap[block2Hash]
	block3 = blockMap[block3Hash]
	builder = statediff.NewBuilder(testhelpers.Testdb, chain)

	type arguments struct {
		oldStateRoot common.Hash
		newStateRoot common.Hash
		blockNumber  int64
		blockHash    common.Hash
	}

	var (
		balanceChange10000      = int64(10000)
		balanceChange1000       = int64(1000)
		block1BankBalance       = int64(99990000)
		block1Account1Balance   = int64(10000)
		block2Account2Balance   = int64(1000)
		nonce0                  = uint64(0)
		nonce1                  = uint64(1)
		nonce2                  = uint64(2)
		nonce3                  = uint64(3)
		originalContractRoot    = "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
		contractContractRoot    = "0x821e2556a290c86405f8160a2d662042a431ba456b9db265c79bb837c04be5f0"
		newContractRoot         = "0x71e0d14b2b93e5c7f9748e69e1fe5f17498a1c3ac3cec29f96af13d7f8a4e070"
		originalStorageLocation = common.HexToHash("0")
		originalStorageKey      = crypto.Keccak256Hash(originalStorageLocation[:]).Bytes()
		updatedStorageLocation  = common.HexToHash("2")
		updatedStorageKey       = crypto.Keccak256Hash(updatedStorageLocation[:]).Bytes()
		originalStorageValue    = common.Hex2Bytes("01")
		updatedStorageValue     = common.Hex2Bytes("03")
		account1, _             = rlp.EncodeToBytes(state.Account{
			Nonce:    nonce0,
			Balance:  big.NewInt(balanceChange10000),
			CodeHash: common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
			Root:     common.HexToHash(originalContractRoot),
		})
		burnAccount1, _ = rlp.EncodeToBytes(state.Account{
			Nonce:    nonce0,
			Balance:  big.NewInt(miningReward),
			CodeHash: common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
			Root:     common.HexToHash(originalContractRoot),
		})
		bankAccount1, _ = rlp.EncodeToBytes(state.Account{
			Nonce:    nonce1,
			Balance:  big.NewInt(testhelpers.TestBankFunds.Int64() - balanceChange10000),
			CodeHash: common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
			Root:     common.HexToHash(originalContractRoot),
		})
		account2, _ = rlp.EncodeToBytes(state.Account{
			Nonce:    nonce0,
			Balance:  big.NewInt(balanceChange1000),
			CodeHash: common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
			Root:     common.HexToHash(originalContractRoot),
		})
		contractAccount, _ = rlp.EncodeToBytes(state.Account{
			Nonce:    nonce1,
			Balance:  big.NewInt(0),
			CodeHash: common.HexToHash("0x753f98a8d4328b15636e46f66f2cb4bc860100aa17967cc145fcd17d1d4710ea").Bytes(),
			Root:     common.HexToHash(contractContractRoot),
		})
		bankAccount2, _ = rlp.EncodeToBytes(state.Account{
			Nonce:    nonce2,
			Balance:  big.NewInt(block1BankBalance - balanceChange1000),
			CodeHash: common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
			Root:     common.HexToHash(originalContractRoot),
		})
		account3, _ = rlp.EncodeToBytes(state.Account{
			Nonce:    nonce2,
			Balance:  big.NewInt(block1Account1Balance - balanceChange1000 + balanceChange1000),
			CodeHash: common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
			Root:     common.HexToHash(originalContractRoot),
		})
		burnAccount2, _ = rlp.EncodeToBytes(state.Account{
			Nonce:    nonce0,
			Balance:  big.NewInt(miningReward + miningReward),
			CodeHash: common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
			Root:     common.HexToHash(originalContractRoot),
		})
		account4, _ = rlp.EncodeToBytes(state.Account{
			Nonce:    nonce0,
			Balance:  big.NewInt(block2Account2Balance + miningReward),
			CodeHash: common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
			Root:     common.HexToHash(originalContractRoot),
		})
		contractAccount2, _ = rlp.EncodeToBytes(state.Account{
			Nonce:    nonce1,
			Balance:  big.NewInt(0),
			CodeHash: common.HexToHash("0x753f98a8d4328b15636e46f66f2cb4bc860100aa17967cc145fcd17d1d4710ea").Bytes(),
			Root:     common.HexToHash(newContractRoot),
		})
		bankAccount3, _ = rlp.EncodeToBytes(state.Account{
			Nonce:    nonce3,
			Balance:  big.NewInt(99989000),
			CodeHash: common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
			Root:     common.HexToHash(originalContractRoot),
		})
	)

	var tests = []struct {
		name              string
		startingArguments arguments
		expected          *statediff.StateDiff
	}{
		{
			"testEmptyDiff",
			arguments{
				oldStateRoot: block0.Root(),
				newStateRoot: block0.Root(),
				blockNumber:  block0.Number().Int64(),
				blockHash:    block0Hash,
			},
			&statediff.StateDiff{
				BlockNumber:     block0.Number().Int64(),
				BlockHash:       block0Hash,
				CreatedAccounts: emptyAccountDiffEventualMap,
				DeletedAccounts: emptyAccountDiffEventualMap,
				UpdatedAccounts: emptyAccountDiffIncrementalMap,
			},
		},
		{
			"testBlock1",
			//10000 transferred from testBankAddress to account1Addr
			arguments{
				oldStateRoot: block0.Root(),
				newStateRoot: block1.Root(),
				blockNumber:  block1.Number().Int64(),
				blockHash:    block1Hash,
			},
			&statediff.StateDiff{
				BlockNumber: block1.Number().Int64(),
				BlockHash:   block1.Hash(),
				CreatedAccounts: statediff.AccountDiffsMap{
					testhelpers.Account1LeafKey: {
						Key:   testhelpers.Account1LeafKey.Bytes(),
						Value: account1,
						Proof: [][]byte{{248, 113, 160, 87, 118, 82, 182, 37, 183, 123, 219, 91, 247, 123, 196, 63, 49, 37, 202, 215, 70, 77, 103, 157, 21, 117, 86, 82, 119, 211, 97, 27, 128, 83, 231, 128, 128, 128, 128, 160, 254, 136, 159, 16, 229, 219, 143, 44, 43, 243, 85, 146, 129, 82, 161, 127, 110, 59, 185, 154, 146, 65, 172, 109, 132, 199, 126, 98, 100, 80, 156, 121, 128, 128, 128, 128, 128, 128, 128, 128, 160, 17, 219, 12, 218, 52, 168, 150, 218, 190, 182, 131, 155, 176, 106, 56, 244, 149, 20, 207, 164, 134, 67, 89, 132, 235, 1, 59, 125, 249, 238, 133, 197, 128, 128},
							{248, 107, 160, 57, 38, 219, 105, 170, 206, 213, 24, 233, 185, 240, 244, 52, 164, 115, 231, 23, 65, 9, 201, 67, 84, 139, 184, 242, 59, 228, 28, 167, 109, 154, 210, 184, 72, 248, 70, 128, 130, 39, 16, 160, 86, 232, 31, 23, 27, 204, 85, 166, 255, 131, 69, 230, 146, 192, 248, 110, 91, 72, 224, 27, 153, 108, 173, 192, 1, 98, 47, 181, 227, 99, 180, 33, 160, 197, 210, 70, 1, 134, 247, 35, 60, 146, 126, 125, 178, 220, 199, 3, 192, 229, 0, 182, 83, 202, 130, 39, 59, 123, 250, 216, 4, 93, 133, 164, 112}},
						Path:    []byte{14, 9, 2, 6, 13, 11, 6, 9, 10, 10, 12, 14, 13, 5, 1, 8, 14, 9, 11, 9, 15, 0, 15, 4, 3, 4, 10, 4, 7, 3, 14, 7, 1, 7, 4, 1, 0, 9, 12, 9, 4, 3, 5, 4, 8, 11, 11, 8, 15, 2, 3, 11, 14, 4, 1, 12, 10, 7, 6, 13, 9, 10, 13, 2, 16},
						Storage: []statediff.StorageDiff{},
					},
					burnLeafKey: {
						Key:   burnLeafKey.Bytes(),
						Value: burnAccount1,
						Proof: [][]byte{{248, 113, 160, 87, 118, 82, 182, 37, 183, 123, 219, 91, 247, 123, 196, 63, 49, 37, 202, 215, 70, 77, 103, 157, 21, 117, 86, 82, 119, 211, 97, 27, 128, 83, 231, 128, 128, 128, 128, 160, 254, 136, 159, 16, 229, 219, 143, 44, 43, 243, 85, 146, 129, 82, 161, 127, 110, 59, 185, 154, 146, 65, 172, 109, 132, 199, 126, 98, 100, 80, 156, 121, 128, 128, 128, 128, 128, 128, 128, 128, 160, 17, 219, 12, 218, 52, 168, 150, 218, 190, 182, 131, 155, 176, 106, 56, 244, 149, 20, 207, 164, 134, 67, 89, 132, 235, 1, 59, 125, 249, 238, 133, 197, 128, 128},
							{248, 113, 160, 51, 128, 199, 183, 174, 129, 165, 142, 185, 141, 156, 120, 222, 74, 31, 215, 253, 149, 53, 252, 149, 62, 210, 190, 96, 45, 170, 164, 23, 103, 49, 42, 184, 78, 248, 76, 128, 136, 27, 193, 109, 103, 78, 200, 0, 0, 160, 86, 232, 31, 23, 27, 204, 85, 166, 255, 131, 69, 230, 146, 192, 248, 110, 91, 72, 224, 27, 153, 108, 173, 192, 1, 98, 47, 181, 227, 99, 180, 33, 160, 197, 210, 70, 1, 134, 247, 35, 60, 146, 126, 125, 178, 220, 199, 3, 192, 229, 0, 182, 83, 202, 130, 39, 59, 123, 250, 216, 4, 93, 133, 164, 112}},
						Path:    []byte{5, 3, 8, 0, 12, 7, 11, 7, 10, 14, 8, 1, 10, 5, 8, 14, 11, 9, 8, 13, 9, 12, 7, 8, 13, 14, 4, 10, 1, 15, 13, 7, 15, 13, 9, 5, 3, 5, 15, 12, 9, 5, 3, 14, 13, 2, 11, 14, 6, 0, 2, 13, 10, 10, 10, 4, 1, 7, 6, 7, 3, 1, 2, 10, 16},
						Storage: []statediff.StorageDiff{},
					},
				},
				DeletedAccounts: emptyAccountDiffEventualMap,
				UpdatedAccounts: statediff.AccountDiffsMap{
					testhelpers.BankLeafKey: {
						Key:   testhelpers.BankLeafKey.Bytes(),
						Value: bankAccount1,
						Proof: [][]byte{{248, 113, 160, 87, 118, 82, 182, 37, 183, 123, 219, 91, 247, 123, 196, 63, 49, 37, 202, 215, 70, 77, 103, 157, 21, 117, 86, 82, 119, 211, 97, 27, 128, 83, 231, 128, 128, 128, 128, 160, 254, 136, 159, 16, 229, 219, 143, 44, 43, 243, 85, 146, 129, 82, 161, 127, 110, 59, 185, 154, 146, 65, 172, 109, 132, 199, 126, 98, 100, 80, 156, 121, 128, 128, 128, 128, 128, 128, 128, 128, 160, 17, 219, 12, 218, 52, 168, 150, 218, 190, 182, 131, 155, 176, 106, 56, 244, 149, 20, 207, 164, 134, 67, 89, 132, 235, 1, 59, 125, 249, 238, 133, 197, 128, 128},
							{248, 109, 160, 48, 191, 73, 244, 64, 161, 205, 5, 39, 228, 208, 110, 39, 101, 101, 76, 15, 86, 69, 34, 87, 81, 109, 121, 58, 155, 141, 96, 77, 207, 223, 42, 184, 74, 248, 72, 1, 132, 5, 245, 185, 240, 160, 86, 232, 31, 23, 27, 204, 85, 166, 255, 131, 69, 230, 146, 192, 248, 110, 91, 72, 224, 27, 153, 108, 173, 192, 1, 98, 47, 181, 227, 99, 180, 33, 160, 197, 210, 70, 1, 134, 247, 35, 60, 146, 126, 125, 178, 220, 199, 3, 192, 229, 0, 182, 83, 202, 130, 39, 59, 123, 250, 216, 4, 93, 133, 164, 112}},
						Path:    []byte{0, 0, 11, 15, 4, 9, 15, 4, 4, 0, 10, 1, 12, 13, 0, 5, 2, 7, 14, 4, 13, 0, 6, 14, 2, 7, 6, 5, 6, 5, 4, 12, 0, 15, 5, 6, 4, 5, 2, 2, 5, 7, 5, 1, 6, 13, 7, 9, 3, 10, 9, 11, 8, 13, 6, 0, 4, 13, 12, 15, 13, 15, 2, 10, 16},
						Storage: []statediff.StorageDiff{},
					},
				},
			},
		},
		{
			"testBlock2",
			//1000 transferred from testBankAddress to account1Addr
			//1000 transferred from account1Addr to account2Addr
			arguments{
				oldStateRoot: block1.Root(),
				newStateRoot: block2.Root(),
				blockNumber:  block2.Number().Int64(),
				blockHash:    block2Hash,
			},
			&statediff.StateDiff{
				BlockNumber: block2.Number().Int64(),
				BlockHash:   block2.Hash(),
				CreatedAccounts: statediff.AccountDiffsMap{
					testhelpers.Account2LeafKey: {
						Key:   testhelpers.Account2LeafKey.Bytes(),
						Value: account2,
						Proof: [][]byte{{248, 177, 160, 177, 155, 238, 178, 242, 47, 83, 2, 49, 141, 155, 92, 149, 175, 245, 120, 233, 177, 101, 67, 46, 200, 23, 250, 41, 74, 135, 94, 61, 133, 51, 162, 128, 128, 128, 128, 160, 179, 86, 53, 29, 96, 188, 152, 148, 207, 31, 29, 108, 182, 140, 129, 95, 1, 49, 213, 15, 29, 168, 60, 64, 35, 160, 158, 200, 85, 207, 255, 145, 160, 114, 57, 32, 11, 115, 232, 140, 238, 165, 222, 121, 226, 208, 2, 192, 216, 67, 198, 179, 31, 181, 27, 208, 243, 99, 202, 48, 148, 207, 107, 106, 177, 128, 128, 128, 128, 128, 160, 10, 173, 165, 125, 110, 240, 77, 112, 149, 100, 135, 237, 25, 228, 116, 7, 195, 9, 210, 166, 208, 148, 101, 23, 244, 238, 84, 84, 211, 249, 138, 137, 128, 160, 255, 115, 147, 190, 57, 135, 174, 188, 86, 51, 227, 70, 22, 253, 237, 49, 24, 19, 149, 199, 142, 195, 186, 244, 70, 51, 138, 0, 146, 148, 117, 60, 128, 128},
							{248, 107, 160, 57, 87, 243, 226, 240, 74, 7, 100, 195, 160, 73, 27, 23, 95, 105, 146, 109, 166, 30, 251, 204, 143, 97, 250, 20, 85, 253, 45, 43, 76, 221, 69, 184, 72, 248, 70, 128, 130, 3, 232, 160, 86, 232, 31, 23, 27, 204, 85, 166, 255, 131, 69, 230, 146, 192, 248, 110, 91, 72, 224, 27, 153, 108, 173, 192, 1, 98, 47, 181, 227, 99, 180, 33, 160, 197, 210, 70, 1, 134, 247, 35, 60, 146, 126, 125, 178, 220, 199, 3, 192, 229, 0, 182, 83, 202, 130, 39, 59, 123, 250, 216, 4, 93, 133, 164, 112}},
						Path:    []byte{12, 9, 5, 7, 15, 3, 14, 2, 15, 0, 4, 10, 0, 7, 6, 4, 12, 3, 10, 0, 4, 9, 1, 11, 1, 7, 5, 15, 6, 9, 9, 2, 6, 13, 10, 6, 1, 14, 15, 11, 12, 12, 8, 15, 6, 1, 15, 10, 1, 4, 5, 5, 15, 13, 2, 13, 2, 11, 4, 12, 13, 13, 4, 5, 16},
						Storage: []statediff.StorageDiff{},
					},
					contractLeafKey: {
						Key:   contractLeafKey.Bytes(),
						Value: contractAccount,
						Proof: [][]byte{{248, 177, 160, 177, 155, 238, 178, 242, 47, 83, 2, 49, 141, 155, 92, 149, 175, 245, 120, 233, 177, 101, 67, 46, 200, 23, 250, 41, 74, 135, 94, 61, 133, 51, 162, 128, 128, 128, 128, 160, 179, 86, 53, 29, 96, 188, 152, 148, 207, 31, 29, 108, 182, 140, 129, 95, 1, 49, 213, 15, 29, 168, 60, 64, 35, 160, 158, 200, 85, 207, 255, 145, 160, 114, 57, 32, 11, 115, 232, 140, 238, 165, 222, 121, 226, 208, 2, 192, 216, 67, 198, 179, 31, 181, 27, 208, 243, 99, 202, 48, 148, 207, 107, 106, 177, 128, 128, 128, 128, 128, 160, 10, 173, 165, 125, 110, 240, 77, 112, 149, 100, 135, 237, 25, 228, 116, 7, 195, 9, 210, 166, 208, 148, 101, 23, 244, 238, 84, 84, 211, 249, 138, 137, 128, 160, 255, 115, 147, 190, 57, 135, 174, 188, 86, 51, 227, 70, 22, 253, 237, 49, 24, 19, 149, 199, 142, 195, 186, 244, 70, 51, 138, 0, 146, 148, 117, 60, 128, 128},
							{248, 105, 160, 49, 20, 101, 138, 116, 217, 204, 159, 122, 207, 44, 92, 214, 150, 195, 73, 77, 124, 52, 77, 120, 191, 236, 58, 221, 13, 145, 236, 78, 141, 28, 69, 184, 70, 248, 68, 1, 128, 160, 130, 30, 37, 86, 162, 144, 200, 100, 5, 248, 22, 10, 45, 102, 32, 66, 164, 49, 186, 69, 107, 157, 178, 101, 199, 155, 184, 55, 192, 75, 229, 240, 160, 117, 63, 152, 168, 212, 50, 139, 21, 99, 110, 70, 246, 111, 44, 180, 188, 134, 1, 0, 170, 23, 150, 124, 193, 69, 252, 209, 125, 29, 71, 16, 234}},
						Path: []byte{6, 1, 1, 4, 6, 5, 8, 10, 7, 4, 13, 9, 12, 12, 9, 15, 7, 10, 12, 15, 2, 12, 5, 12, 13, 6, 9, 6, 12, 3, 4, 9, 4, 13, 7, 12, 3, 4, 4, 13, 7, 8, 11, 15, 14, 12, 3, 10, 13, 13, 0, 13, 9, 1, 14, 12, 4, 14, 8, 13, 1, 12, 4, 5, 16},
						Storage: []statediff.StorageDiff{
							{
								Key:   originalStorageKey,
								Value: originalStorageValue,
								Proof: [][]byte{{227, 161, 32, 41, 13, 236, 217, 84, 139, 98, 168, 214, 3, 69, 169, 136, 56, 111, 200, 75, 166, 188, 149, 72, 64, 8, 246, 54, 47, 147, 22, 14, 243, 229, 99, 1}},
								Path:  []byte{2, 9, 0, 13, 14, 12, 13, 9, 5, 4, 8, 11, 6, 2, 10, 8, 13, 6, 0, 3, 4, 5, 10, 9, 8, 8, 3, 8, 6, 15, 12, 8, 4, 11, 10, 6, 11, 12, 9, 5, 4, 8, 4, 0, 0, 8, 15, 6, 3, 6, 2, 15, 9, 3, 1, 6, 0, 14, 15, 3, 14, 5, 6, 3, 16},
							},
						},
					},
				},
				DeletedAccounts: emptyAccountDiffEventualMap,
				UpdatedAccounts: statediff.AccountDiffsMap{
					testhelpers.BankLeafKey: {
						Key:   testhelpers.BankLeafKey.Bytes(),
						Value: bankAccount2,
						Proof: [][]byte{{248, 177, 160, 177, 155, 238, 178, 242, 47, 83, 2, 49, 141, 155, 92, 149, 175, 245, 120, 233, 177, 101, 67, 46, 200, 23, 250, 41, 74, 135, 94, 61, 133, 51, 162, 128, 128, 128, 128, 160, 179, 86, 53, 29, 96, 188, 152, 148, 207, 31, 29, 108, 182, 140, 129, 95, 1, 49, 213, 15, 29, 168, 60, 64, 35, 160, 158, 200, 85, 207, 255, 145, 160, 114, 57, 32, 11, 115, 232, 140, 238, 165, 222, 121, 226, 208, 2, 192, 216, 67, 198, 179, 31, 181, 27, 208, 243, 99, 202, 48, 148, 207, 107, 106, 177, 128, 128, 128, 128, 128, 160, 10, 173, 165, 125, 110, 240, 77, 112, 149, 100, 135, 237, 25, 228, 116, 7, 195, 9, 210, 166, 208, 148, 101, 23, 244, 238, 84, 84, 211, 249, 138, 137, 128, 160, 255, 115, 147, 190, 57, 135, 174, 188, 86, 51, 227, 70, 22, 253, 237, 49, 24, 19, 149, 199, 142, 195, 186, 244, 70, 51, 138, 0, 146, 148, 117, 60, 128, 128},
							{248, 109, 160, 48, 191, 73, 244, 64, 161, 205, 5, 39, 228, 208, 110, 39, 101, 101, 76, 15, 86, 69, 34, 87, 81, 109, 121, 58, 155, 141, 96, 77, 207, 223, 42, 184, 74, 248, 72, 2, 132, 5, 245, 182, 8, 160, 86, 232, 31, 23, 27, 204, 85, 166, 255, 131, 69, 230, 146, 192, 248, 110, 91, 72, 224, 27, 153, 108, 173, 192, 1, 98, 47, 181, 227, 99, 180, 33, 160, 197, 210, 70, 1, 134, 247, 35, 60, 146, 126, 125, 178, 220, 199, 3, 192, 229, 0, 182, 83, 202, 130, 39, 59, 123, 250, 216, 4, 93, 133, 164, 112}},
						Path:    []byte{0, 0, 11, 15, 4, 9, 15, 4, 4, 0, 10, 1, 12, 13, 0, 5, 2, 7, 14, 4, 13, 0, 6, 14, 2, 7, 6, 5, 6, 5, 4, 12, 0, 15, 5, 6, 4, 5, 2, 2, 5, 7, 5, 1, 6, 13, 7, 9, 3, 10, 9, 11, 8, 13, 6, 0, 4, 13, 12, 15, 13, 15, 2, 10, 16},
						Storage: []statediff.StorageDiff{},
					},
					testhelpers.Account1LeafKey: {
						Key:   testhelpers.Account1LeafKey.Bytes(),
						Value: account3,
						Proof: [][]byte{{248, 177, 160, 177, 155, 238, 178, 242, 47, 83, 2, 49, 141, 155, 92, 149, 175, 245, 120, 233, 177, 101, 67, 46, 200, 23, 250, 41, 74, 135, 94, 61, 133, 51, 162, 128, 128, 128, 128, 160, 179, 86, 53, 29, 96, 188, 152, 148, 207, 31, 29, 108, 182, 140, 129, 95, 1, 49, 213, 15, 29, 168, 60, 64, 35, 160, 158, 200, 85, 207, 255, 145, 160, 114, 57, 32, 11, 115, 232, 140, 238, 165, 222, 121, 226, 208, 2, 192, 216, 67, 198, 179, 31, 181, 27, 208, 243, 99, 202, 48, 148, 207, 107, 106, 177, 128, 128, 128, 128, 128, 160, 10, 173, 165, 125, 110, 240, 77, 112, 149, 100, 135, 237, 25, 228, 116, 7, 195, 9, 210, 166, 208, 148, 101, 23, 244, 238, 84, 84, 211, 249, 138, 137, 128, 160, 255, 115, 147, 190, 57, 135, 174, 188, 86, 51, 227, 70, 22, 253, 237, 49, 24, 19, 149, 199, 142, 195, 186, 244, 70, 51, 138, 0, 146, 148, 117, 60, 128, 128},
							{248, 107, 160, 57, 38, 219, 105, 170, 206, 213, 24, 233, 185, 240, 244, 52, 164, 115, 231, 23, 65, 9, 201, 67, 84, 139, 184, 242, 59, 228, 28, 167, 109, 154, 210, 184, 72, 248, 70, 2, 130, 39, 16, 160, 86, 232, 31, 23, 27, 204, 85, 166, 255, 131, 69, 230, 146, 192, 248, 110, 91, 72, 224, 27, 153, 108, 173, 192, 1, 98, 47, 181, 227, 99, 180, 33, 160, 197, 210, 70, 1, 134, 247, 35, 60, 146, 126, 125, 178, 220, 199, 3, 192, 229, 0, 182, 83, 202, 130, 39, 59, 123, 250, 216, 4, 93, 133, 164, 112}},
						Path:    []byte{14, 9, 2, 6, 13, 11, 6, 9, 10, 10, 12, 14, 13, 5, 1, 8, 14, 9, 11, 9, 15, 0, 15, 4, 3, 4, 10, 4, 7, 3, 14, 7, 1, 7, 4, 1, 0, 9, 12, 9, 4, 3, 5, 4, 8, 11, 11, 8, 15, 2, 3, 11, 14, 4, 1, 12, 10, 7, 6, 13, 9, 10, 13, 2, 16},
						Storage: []statediff.StorageDiff{},
					},
					burnLeafKey: {
						Key:   burnLeafKey.Bytes(),
						Value: burnAccount2,
						Proof: [][]byte{{248, 177, 160, 177, 155, 238, 178, 242, 47, 83, 2, 49, 141, 155, 92, 149, 175, 245, 120, 233, 177, 101, 67, 46, 200, 23, 250, 41, 74, 135, 94, 61, 133, 51, 162, 128, 128, 128, 128, 160, 179, 86, 53, 29, 96, 188, 152, 148, 207, 31, 29, 108, 182, 140, 129, 95, 1, 49, 213, 15, 29, 168, 60, 64, 35, 160, 158, 200, 85, 207, 255, 145, 160, 114, 57, 32, 11, 115, 232, 140, 238, 165, 222, 121, 226, 208, 2, 192, 216, 67, 198, 179, 31, 181, 27, 208, 243, 99, 202, 48, 148, 207, 107, 106, 177, 128, 128, 128, 128, 128, 160, 10, 173, 165, 125, 110, 240, 77, 112, 149, 100, 135, 237, 25, 228, 116, 7, 195, 9, 210, 166, 208, 148, 101, 23, 244, 238, 84, 84, 211, 249, 138, 137, 128, 160, 255, 115, 147, 190, 57, 135, 174, 188, 86, 51, 227, 70, 22, 253, 237, 49, 24, 19, 149, 199, 142, 195, 186, 244, 70, 51, 138, 0, 146, 148, 117, 60, 128, 128},
							{248, 113, 160, 51, 128, 199, 183, 174, 129, 165, 142, 185, 141, 156, 120, 222, 74, 31, 215, 253, 149, 53, 252, 149, 62, 210, 190, 96, 45, 170, 164, 23, 103, 49, 42, 184, 78, 248, 76, 128, 136, 55, 130, 218, 206, 157, 144, 0, 0, 160, 86, 232, 31, 23, 27, 204, 85, 166, 255, 131, 69, 230, 146, 192, 248, 110, 91, 72, 224, 27, 153, 108, 173, 192, 1, 98, 47, 181, 227, 99, 180, 33, 160, 197, 210, 70, 1, 134, 247, 35, 60, 146, 126, 125, 178, 220, 199, 3, 192, 229, 0, 182, 83, 202, 130, 39, 59, 123, 250, 216, 4, 93, 133, 164, 112}},
						Path:    []byte{5, 3, 8, 0, 12, 7, 11, 7, 10, 14, 8, 1, 10, 5, 8, 14, 11, 9, 8, 13, 9, 12, 7, 8, 13, 14, 4, 10, 1, 15, 13, 7, 15, 13, 9, 5, 3, 5, 15, 12, 9, 5, 3, 14, 13, 2, 11, 14, 6, 0, 2, 13, 10, 10, 10, 4, 1, 7, 6, 7, 3, 1, 2, 10, 16},
						Storage: []statediff.StorageDiff{},
					},
				},
			},
		},
		{
			"testBlock3",
			//the contract's storage is changed
			//and the block is mined by account 2
			arguments{
				oldStateRoot: block2.Root(),
				newStateRoot: block3.Root(),
				blockNumber:  block3.Number().Int64(),
				blockHash:    block3.Hash(),
			},
			&statediff.StateDiff{
				BlockNumber:     block3.Number().Int64(),
				BlockHash:       block3.Hash(),
				CreatedAccounts: statediff.AccountDiffsMap{},
				DeletedAccounts: emptyAccountDiffEventualMap,
				UpdatedAccounts: statediff.AccountDiffsMap{
					testhelpers.Account2LeafKey: {
						Key:   testhelpers.Account2LeafKey.Bytes(),
						Value: account4,
						Proof: [][]byte{{248, 177, 160, 101, 223, 138, 81, 34, 40, 229, 170, 198, 188, 136, 99, 7, 55, 33, 112, 160, 111, 181, 131, 167, 201, 131, 24, 201, 211, 177, 30, 159, 229, 246, 6, 128, 128, 128, 128, 160, 179, 86, 53, 29, 96, 188, 152, 148, 207, 31, 29, 108, 182, 140, 129, 95, 1, 49, 213, 15, 29, 168, 60, 64, 35, 160, 158, 200, 85, 207, 255, 145, 160, 32, 135, 108, 213, 150, 150, 110, 44, 170, 65, 75, 154, 74, 249, 94, 65, 74, 107, 100, 115, 39, 5, 3, 26, 22, 238, 138, 114, 254, 21, 6, 171, 128, 128, 128, 128, 128, 160, 4, 228, 121, 222, 255, 218, 60, 247, 15, 0, 34, 198, 28, 229, 180, 129, 109, 157, 68, 181, 248, 229, 200, 123, 29, 81, 145, 114, 90, 209, 205, 210, 128, 160, 255, 115, 147, 190, 57, 135, 174, 188, 86, 51, 227, 70, 22, 253, 237, 49, 24, 19, 149, 199, 142, 195, 186, 244, 70, 51, 138, 0, 146, 148, 117, 60, 128, 128},
							{248, 113, 160, 57, 87, 243, 226, 240, 74, 7, 100, 195, 160, 73, 27, 23, 95, 105, 146, 109, 166, 30, 251, 204, 143, 97, 250, 20, 85, 253, 45, 43, 76, 221, 69, 184, 78, 248, 76, 128, 136, 27, 193, 109, 103, 78, 200, 3, 232, 160, 86, 232, 31, 23, 27, 204, 85, 166, 255, 131, 69, 230, 146, 192, 248, 110, 91, 72, 224, 27, 153, 108, 173, 192, 1, 98, 47, 181, 227, 99, 180, 33, 160, 197, 210, 70, 1, 134, 247, 35, 60, 146, 126, 125, 178, 220, 199, 3, 192, 229, 0, 182, 83, 202, 130, 39, 59, 123, 250, 216, 4, 93, 133, 164, 112}},
						Path:    []byte{12, 9, 5, 7, 15, 3, 14, 2, 15, 0, 4, 10, 0, 7, 6, 4, 12, 3, 10, 0, 4, 9, 1, 11, 1, 7, 5, 15, 6, 9, 9, 2, 6, 13, 10, 6, 1, 14, 15, 11, 12, 12, 8, 15, 6, 1, 15, 10, 1, 4, 5, 5, 15, 13, 2, 13, 2, 11, 4, 12, 13, 13, 4, 5, 16},
						Storage: []statediff.StorageDiff{},
					},
					contractLeafKey: {
						Key:   contractLeafKey.Bytes(),
						Value: contractAccount2,
						Proof: [][]byte{{248, 177, 160, 101, 223, 138, 81, 34, 40, 229, 170, 198, 188, 136, 99, 7, 55, 33, 112, 160, 111, 181, 131, 167, 201, 131, 24, 201, 211, 177, 30, 159, 229, 246, 6, 128, 128, 128, 128, 160, 179, 86, 53, 29, 96, 188, 152, 148, 207, 31, 29, 108, 182, 140, 129, 95, 1, 49, 213, 15, 29, 168, 60, 64, 35, 160, 158, 200, 85, 207, 255, 145, 160, 32, 135, 108, 213, 150, 150, 110, 44, 170, 65, 75, 154, 74, 249, 94, 65, 74, 107, 100, 115, 39, 5, 3, 26, 22, 238, 138, 114, 254, 21, 6, 171, 128, 128, 128, 128, 128, 160, 4, 228, 121, 222, 255, 218, 60, 247, 15, 0, 34, 198, 28, 229, 180, 129, 109, 157, 68, 181, 248, 229, 200, 123, 29, 81, 145, 114, 90, 209, 205, 210, 128, 160, 255, 115, 147, 190, 57, 135, 174, 188, 86, 51, 227, 70, 22, 253, 237, 49, 24, 19, 149, 199, 142, 195, 186, 244, 70, 51, 138, 0, 146, 148, 117, 60, 128, 128},
							{248, 105, 160, 49, 20, 101, 138, 116, 217, 204, 159, 122, 207, 44, 92, 214, 150, 195, 73, 77, 124, 52, 77, 120, 191, 236, 58, 221, 13, 145, 236, 78, 141, 28, 69, 184, 70, 248, 68, 1, 128, 160, 113, 224, 209, 75, 43, 147, 229, 199, 249, 116, 142, 105, 225, 254, 95, 23, 73, 138, 28, 58, 195, 206, 194, 159, 150, 175, 19, 215, 248, 164, 224, 112, 160, 117, 63, 152, 168, 212, 50, 139, 21, 99, 110, 70, 246, 111, 44, 180, 188, 134, 1, 0, 170, 23, 150, 124, 193, 69, 252, 209, 125, 29, 71, 16, 234}},
						Path: []byte{6, 1, 1, 4, 6, 5, 8, 10, 7, 4, 13, 9, 12, 12, 9, 15, 7, 10, 12, 15, 2, 12, 5, 12, 13, 6, 9, 6, 12, 3, 4, 9, 4, 13, 7, 12, 3, 4, 4, 13, 7, 8, 11, 15, 14, 12, 3, 10, 13, 13, 0, 13, 9, 1, 14, 12, 4, 14, 8, 13, 1, 12, 4, 5, 16},
						Storage: []statediff.StorageDiff{
							{
								Key:   updatedStorageKey,
								Value: updatedStorageValue,
								Proof: [][]byte{{248, 81, 128, 128, 160, 79, 197, 241, 58, 178, 249, 186, 12, 45, 168, 139, 1, 81, 171, 14, 124, 244, 216, 93, 8, 204, 164, 92, 205, 146, 60, 106, 183, 99, 35, 235, 40, 128, 160, 205, 69, 114, 89, 105, 97, 21, 35, 94, 100, 199, 130, 35, 52, 214, 33, 41, 226, 241, 96, 68, 37, 167, 218, 100, 148, 243, 95, 196, 91, 229, 24, 128, 128, 128, 128, 128, 128, 128, 128, 128, 128, 128, 128},
									{226, 160, 48, 87, 135, 250, 18, 168, 35, 224, 242, 183, 99, 28, 196, 27, 59, 168, 130, 139, 51, 33, 202, 129, 17, 17, 250, 117, 205, 58, 163, 187, 90, 206, 3}},
								Path: []byte{4, 0, 5, 7, 8, 7, 15, 10, 1, 2, 10, 8, 2, 3, 14, 0, 15, 2, 11, 7, 6, 3, 1, 12, 12, 4, 1, 11, 3, 11, 10, 8, 8, 2, 8, 11, 3, 3, 2, 1, 12, 10, 8, 1, 1, 1, 1, 1, 15, 10, 7, 5, 12, 13, 3, 10, 10, 3, 11, 11, 5, 10, 12, 14, 16},
							},
						},
					},
					testhelpers.BankLeafKey: {
						Key:   testhelpers.BankLeafKey.Bytes(),
						Value: bankAccount3,
						Proof: [][]byte{{248, 177, 160, 101, 223, 138, 81, 34, 40, 229, 170, 198, 188, 136, 99, 7, 55, 33, 112, 160, 111, 181, 131, 167, 201, 131, 24, 201, 211, 177, 30, 159, 229, 246, 6, 128, 128, 128, 128, 160, 179, 86, 53, 29, 96, 188, 152, 148, 207, 31, 29, 108, 182, 140, 129, 95, 1, 49, 213, 15, 29, 168, 60, 64, 35, 160, 158, 200, 85, 207, 255, 145, 160, 32, 135, 108, 213, 150, 150, 110, 44, 170, 65, 75, 154, 74, 249, 94, 65, 74, 107, 100, 115, 39, 5, 3, 26, 22, 238, 138, 114, 254, 21, 6, 171, 128, 128, 128, 128, 128, 160, 4, 228, 121, 222, 255, 218, 60, 247, 15, 0, 34, 198, 28, 229, 180, 129, 109, 157, 68, 181, 248, 229, 200, 123, 29, 81, 145, 114, 90, 209, 205, 210, 128, 160, 255, 115, 147, 190, 57, 135, 174, 188, 86, 51, 227, 70, 22, 253, 237, 49, 24, 19, 149, 199, 142, 195, 186, 244, 70, 51, 138, 0, 146, 148, 117, 60, 128, 128},
							{248, 109, 160, 48, 191, 73, 244, 64, 161, 205, 5, 39, 228, 208, 110, 39, 101, 101, 76, 15, 86, 69, 34, 87, 81, 109, 121, 58, 155, 141, 96, 77, 207, 223, 42, 184, 74, 248, 72, 3, 132, 5, 245, 182, 8, 160, 86, 232, 31, 23, 27, 204, 85, 166, 255, 131, 69, 230, 146, 192, 248, 110, 91, 72, 224, 27, 153, 108, 173, 192, 1, 98, 47, 181, 227, 99, 180, 33, 160, 197, 210, 70, 1, 134, 247, 35, 60, 146, 126, 125, 178, 220, 199, 3, 192, 229, 0, 182, 83, 202, 130, 39, 59, 123, 250, 216, 4, 93, 133, 164, 112}},
						Path:    []byte{0, 0, 11, 15, 4, 9, 15, 4, 4, 0, 10, 1, 12, 13, 0, 5, 2, 7, 14, 4, 13, 0, 6, 14, 2, 7, 6, 5, 6, 5, 4, 12, 0, 15, 5, 6, 4, 5, 2, 2, 5, 7, 5, 1, 6, 13, 7, 9, 3, 10, 9, 11, 8, 13, 6, 0, 4, 13, 12, 15, 13, 15, 2, 10, 16},
						Storage: []statediff.StorageDiff{},
					},
				},
			},
		},
	}

	for _, test := range tests {
		arguments := test.startingArguments
		diff, err := builder.BuildStateDiff(arguments.oldStateRoot, arguments.newStateRoot, arguments.blockNumber, arguments.blockHash)
		if err != nil {
			t.Error(err)
		}
		fields := []string{"BlockNumber", "BlockHash", "DeletedAccounts", "UpdatedAccounts", "CreatedAccounts"}

		for _, field := range fields {
			reflectionOfDiff := reflect.ValueOf(diff)
			diffValue := reflect.Indirect(reflectionOfDiff).FieldByName(field)

			reflectionOfExpected := reflect.ValueOf(test.expected)
			expectedValue := reflect.Indirect(reflectionOfExpected).FieldByName(field)

			diffValueInterface := diffValue.Interface()
			expectedValueInterface := expectedValue.Interface()

			if !equals(diffValueInterface, expectedValueInterface) {
				t.Logf("Test failed: %s", test.name)
				t.Errorf("field: %+v\nactual: %+v\nexpected: %+v", field, diffValueInterface, expectedValueInterface)
			}
		}
	}
}

func equals(actual, expected interface{}) (success bool) {
	if actualByteSlice, ok := actual.([]byte); ok {
		if expectedByteSlice, ok := expected.([]byte); ok {
			return bytes.Equal(actualByteSlice, expectedByteSlice)
		}
	}

	return reflect.DeepEqual(actual, expected)
}

/*
contract test {

    uint256[100] data;

	constructor() public {
		data = [1];
	}

    function Put(uint256 addr, uint256 value) {
        data[addr] = value;
    }

    function Get(uint256 addr) constant returns (uint256 value) {
        return data[addr];
    }
}
*/

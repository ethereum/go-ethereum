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

package testhelpers

import (
	"math/big"
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/statediff"
)

// AddressToLeafKey hashes an returns an address
func AddressToLeafKey(address common.Address) common.Hash {
	return common.BytesToHash(crypto.Keccak256(address[:]))
}

// Test variables
var (
	BlockNumber     = big.NewInt(rand.Int63())
	BlockHash       = "0xfa40fbe2d98d98b3363a778d52f2bcd29d6790b9b3f3cab2b167fd12d3550f73"
	CodeHash        = common.Hex2Bytes("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")
	NewNonceValue   = rand.Uint64()
	NewBalanceValue = rand.Int63()
	ContractRoot    = common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	StoragePath     = common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes()
	StorageKey      = common.HexToHash("0000000000000000000000000000000000000000000000000000000000000001").Bytes()
	StorageValue    = common.Hex2Bytes("0x03")
	storage         = []statediff.StorageDiff{{
		Key:   StorageKey,
		Value: StorageValue,
		Path:  StoragePath,
		Proof: [][]byte{},
	}}
	emptyStorage           = make([]statediff.StorageDiff, 0)
	address                = common.HexToAddress("0xaE9BEa628c4Ce503DcFD7E305CaB4e29E7476592")
	ContractLeafKey        = AddressToLeafKey(address)
	anotherAddress         = common.HexToAddress("0xaE9BEa628c4Ce503DcFD7E305CaB4e29E7476593")
	AnotherContractLeafKey = AddressToLeafKey(anotherAddress)
	testAccount            = state.Account{
		Nonce:    NewNonceValue,
		Balance:  big.NewInt(NewBalanceValue),
		Root:     ContractRoot,
		CodeHash: CodeHash,
	}
	valueBytes, _       = rlp.EncodeToBytes(testAccount)
	CreatedAccountDiffs = []statediff.AccountDiff{
		{
			Key:     ContractLeafKey.Bytes(),
			Value:   valueBytes,
			Storage: storage,
		},
		{
			Key:     AnotherContractLeafKey.Bytes(),
			Value:   valueBytes,
			Storage: emptyStorage,
		},
	}

	UpdatedAccountDiffs = []statediff.AccountDiff{{
		Key:     ContractLeafKey.Bytes(),
		Value:   valueBytes,
		Storage: storage,
	}}

	DeletedAccountDiffs = []statediff.AccountDiff{{
		Key:     ContractLeafKey.Bytes(),
		Value:   valueBytes,
		Storage: storage,
	}}

	TestStateDiff = statediff.StateDiff{
		BlockNumber:     BlockNumber,
		BlockHash:       common.HexToHash(BlockHash),
		CreatedAccounts: CreatedAccountDiffs,
		DeletedAccounts: DeletedAccountDiffs,
		UpdatedAccounts: UpdatedAccountDiffs,
	}
	Testdb = rawdb.NewMemoryDatabase()

	TestBankKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	TestBankAddress = crypto.PubkeyToAddress(TestBankKey.PublicKey) //0x71562b71999873DB5b286dF957af199Ec94617F7
	BankLeafKey     = AddressToLeafKey(TestBankAddress)
	TestBankFunds   = big.NewInt(100000000)
	Genesis         = core.GenesisBlockForTesting(Testdb, TestBankAddress, TestBankFunds)

	Account1Key, _  = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	Account2Key, _  = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	Account1Addr    = crypto.PubkeyToAddress(Account1Key.PublicKey) //0x703c4b2bD70c169f5717101CaeE543299Fc946C7
	Account2Addr    = crypto.PubkeyToAddress(Account2Key.PublicKey) //0x0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e
	Account1LeafKey = AddressToLeafKey(Account1Addr)
	Account2LeafKey = AddressToLeafKey(Account2Addr)
	ContractCode    = common.Hex2Bytes("608060405234801561001057600080fd5b50602060405190810160405280600160ff16815250600090600161003592919061003b565b506100a5565b826064810192821561006f579160200282015b8281111561006e578251829060ff1690559160200191906001019061004e565b5b50905061007c9190610080565b5090565b6100a291905b8082111561009e576000816000905550600101610086565b5090565b90565b610124806100b46000396000f3fe6080604052348015600f57600080fd5b5060043610604f576000357c01000000000000000000000000000000000000000000000000000000009004806360cd2685146054578063c16431b9146093575b600080fd5b607d60048036036020811015606857600080fd5b810190808035906020019092919050505060c8565b6040518082815260200191505060405180910390f35b60c66004803603604081101560a757600080fd5b81019080803590602001909291908035906020019092919050505060e0565b005b6000808260648110151560d757fe5b01549050919050565b8060008360648110151560ef57fe5b0181905550505056fea165627a7a7230582064e918c3140a117bf3aa65865a9b9e83fae21ad1720506e7933b2a9f54bb40260029")
	ContractAddr    common.Address
)

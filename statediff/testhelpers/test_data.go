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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// AddressToLeafKey hashes an returns an address
func AddressToLeafKey(address common.Address) []byte {
	return crypto.Keccak256(address[:])
}

// AddressToEncodedPath hashes an address and appends the even-number leaf flag to it
func AddressToEncodedPath(address common.Address) []byte {
	addrHash := crypto.Keccak256(address[:])
	decodedPath := append(EvenLeafFlag, addrHash...)
	return decodedPath
}

// Test variables
var (
	EvenLeafFlag = []byte{byte(2) << 4}
	BlockNumber  = big.NewInt(rand.Int63())
	BlockHash    = "0xfa40fbe2d98d98b3363a778d52f2bcd29d6790b9b3f3cab2b167fd12d3550f73"
	NullCodeHash = crypto.Keccak256Hash([]byte{})
	StoragePath  = common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes()
	StorageKey   = common.HexToHash("0000000000000000000000000000000000000000000000000000000000000001").Bytes()
	StorageValue = common.Hex2Bytes("0x03")
	NullHash     = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")

	Testdb          = rawdb.NewMemoryDatabase()
	TestBankKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	TestBankAddress = crypto.PubkeyToAddress(TestBankKey.PublicKey) //0x71562b71999873DB5b286dF957af199Ec94617F7
	BankLeafKey     = AddressToLeafKey(TestBankAddress)
	TestBankFunds   = big.NewInt(100000000)
	Genesis         = core.GenesisBlockForTesting(Testdb, TestBankAddress, TestBankFunds)

	Account1Key, _          = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	Account2Key, _          = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	Account1Addr            = crypto.PubkeyToAddress(Account1Key.PublicKey) //0x703c4b2bD70c169f5717101CaeE543299Fc946C7
	Account2Addr            = crypto.PubkeyToAddress(Account2Key.PublicKey) //0x0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e
	Account1LeafKey         = AddressToLeafKey(Account1Addr)
	Account2LeafKey         = AddressToLeafKey(Account2Addr)
	ContractCode            = common.Hex2Bytes("608060405234801561001057600080fd5b50336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506040518060200160405280600160ff16815250600190600161007492919061007a565b506100e4565b82606481019282156100ae579160200282015b828111156100ad578251829060ff1690559160200191906001019061008d565b5b5090506100bb91906100bf565b5090565b6100e191905b808211156100dd5760008160009055506001016100c5565b5090565b90565b6101ca806100f36000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c806343d726d61461003b578063c16431b914610045575b600080fd5b61004361007d565b005b61007b6004803603604081101561005b57600080fd5b81019080803590602001909291908035906020019092919050505061015c565b005b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610122576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260228152602001806101746022913960400191505060405180910390fd5b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16ff5b806001836064811061016a57fe5b0181905550505056fe4f6e6c79206f776e65722063616e2063616c6c20746869732066756e6374696f6e2ea265627a7a72305820e3747183708fb6bff3f6f7a80fb57dcc1c19f83f9cb25457a3ed5c0424bde66864736f6c634300050a0032")
	ByteCodeAfterDeployment = common.Hex2Bytes("608060405234801561001057600080fd5b50600436106100365760003560e01c806343d726d61461003b578063c16431b914610045575b600080fd5b61004361007d565b005b61007b6004803603604081101561005b57600080fd5b81019080803590602001909291908035906020019092919050505061015c565b005b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610122576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260228152602001806101746022913960400191505060405180910390fd5b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16ff5b806001836064811061016a57fe5b0181905550505056fe4f6e6c79206f776e65722063616e2063616c6c20746869732066756e6374696f6e2ea265627a7a72305820e3747183708fb6bff3f6f7a80fb57dcc1c19f83f9cb25457a3ed5c0424bde66864736f6c634300050a0032")
	CodeHash                = common.HexToHash("0xaaea5efba4fd7b45d7ec03918ac5d8b31aa93b48986af0e6b591f0f087c80127")
	ContractAddr            common.Address

	EmptyRootNode, _  = rlp.EncodeToBytes([]byte{})
	EmptyContractRoot = crypto.Keccak256Hash(EmptyRootNode)
)

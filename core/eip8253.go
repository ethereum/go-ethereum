// Copyright 2026 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// EIP-8253 sets the nonce of a fixed list of accounts to one at the start of
// the Bogota fork block. These accounts have empty code, a zero nonce and
// non-empty storage, which only contract creations before EIP-161 could
// produce. With a nonce of one, a CREATE onto any of them fails the EIP-684
// collision check instead of inheriting the old storage. Chains that ran
// EIP-161 from genesis have no such accounts, so only mainnet has a list.

// zeroNonceStorageAccounts lists the accounts bumped by EIP-8253, keyed by chain ID.
var zeroNonceStorageAccounts = map[uint64][]common.Address{
	params.MainnetChainConfig.ChainID.Uint64(): {
		common.HexToAddress("0x02820E4bEE488C40f7455fDCa53125565148708F"),
		common.HexToAddress("0x14725085d004f1b10Ee07234A4ab28c5Ad2a7b9E"),
		common.HexToAddress("0x19272418753B90D9a3E3Efc8430b1612c55fcB3A"),
		common.HexToAddress("0x2c081Ed1949D7Dd9447F9d96e509befE576D4461"),
		common.HexToAddress("0x3311c08066580cb906a7287b6786E504C2EBD09f"),
		common.HexToAddress("0x361d7a60b43587c7f6bbA4f9fD9642747F65210A"),
		common.HexToAddress("0x40490C9c468622d5c89646D6F3097F8Eaf80c411"),
		common.HexToAddress("0x4d149EB99BDEEFC1f858f8fd22289C6beAE99f2c"),
		common.HexToAddress("0x5071cb62aA170b7f66b26cae8004d90E6078Bb1E"),
		common.HexToAddress("0x50b1497068bAE652Df3562EB8Ea7677ff84477FA"),
		common.HexToAddress("0x5983C6aC846DcF85fbBC4303F43eb91C379F79ae"),
		common.HexToAddress("0x59EC0410867828E3b8c23Dd8A29d9796ef523b17"),
		common.HexToAddress("0x5cC182faBFb81A056B6080d4200BC5150673D06f"),
		common.HexToAddress("0x6f156dbf8Ed30e53F7C9Df73144E69f65cBB7E94"),
		common.HexToAddress("0x7D6ae067De8d44Ae1A08750e7D626D61A623C44A"),
		common.HexToAddress("0x8398fF6c618e9515468c1c4b198d53666CBe8462"),
		common.HexToAddress("0xA21B22389bfC1cd6Bc7BA19A4Fc96aDC3D0FE074"),
		common.HexToAddress("0xaDD92e0650457C5Db0c4c08cbf7cA580175d33d2"),
		common.HexToAddress("0xAE3703584494Ade958AD27EC2d289b7a67c19E90"),
		common.HexToAddress("0xb619f45637C39Ca49A41ac64c11637A0A194455E"),
		common.HexToAddress("0xD8253352f6044cFE55bcC0748C3FA37b7dF81F98"),
		common.HexToAddress("0xDB7C577B93Baeb56dAB50aF4D6f86F99A06B96a2"),
		common.HexToAddress("0xdE425ad4B8d2d9E0E12F65CBcD6D55F447B44083"),
		common.HexToAddress("0xe62dc49C92fA799033644d2A9aFD7e3BAbE5A80a"),
		common.HexToAddress("0xF468BcBC4a0BFDB06336E773382C5202E674db71"),
		common.HexToAddress("0xF4a835ec1364809003dE3925685F24cD360bdffe"),
		common.HexToAddress("0xFc4465F84B29a1F8794Dc753F41BeF1F4b025ED2"),
		common.HexToAddress("0xfeE7707fa4b8C0A923A0E40399Db3e7Ce26069C6"),
	},
}

// ProcessZeroNonceStorageAccounts applies the EIP-8253 nonce bump at block access index zero.
func ProcessZeroNonceStorageAccounts(evm *vm.EVM, blockAccessList *bal.ConstructionBlockAccessList) {
	accounts := zeroNonceStorageAccounts[evm.ChainConfig().ChainID.Uint64()]
	if len(accounts) == 0 {
		return
	}
	// Record the changes at the same index as the pre-execution system calls.
	evm.StateDB.Prepare(evm.GetRules(), common.Address{}, common.Address{}, nil, nil, nil)
	evm.StateDB.SetTxContext(common.Hash{}, 0, 0)

	// The bump doesn't look at the account. A listed account that is missing
	// from the state is created with a nonce of one.
	for _, addr := range accounts {
		evm.StateDB.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
	}
	accessList := evm.StateDB.Finalise(evm.GetRules())
	if evm.GetRules().IsAmsterdam {
		blockAccessList.Merge(accessList)
	}
}

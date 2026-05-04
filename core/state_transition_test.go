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
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

func TestFloorDataGas(t *testing.T) {
	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	key1 := common.HexToHash("0xaa")
	key2 := common.HexToHash("0xbb")

	tests := []struct {
		name       string
		amsterdam  bool
		data       []byte
		accessList types.AccessList
		want       uint64
	}{
		{
			name: "pre-amsterdam/empty",
			want: params.TxGas,
		},
		{
			name: "pre-amsterdam/zero-bytes-only",
			data: bytes.Repeat([]byte{0x00}, 100),
			// 100 zero tokens * 10 cost = 1000
			want: params.TxGas + 100*params.TxCostFloorPerToken,
		},
		{
			name: "pre-amsterdam/non-zero-bytes-only",
			data: bytes.Repeat([]byte{0xff}, 100),
			// 100 nz * 4 tokens * 10 cost = 4000
			want: params.TxGas + 100*params.TxTokenPerNonZeroByte*params.TxCostFloorPerToken,
		},
		{
			name: "pre-amsterdam/mixed",
			data: append(bytes.Repeat([]byte{0x00}, 50), bytes.Repeat([]byte{0xff}, 50)...),
			// 50 zero + 50*4 nz = 250 tokens * 10 = 2500
			want: params.TxGas + (50+50*params.TxTokenPerNonZeroByte)*params.TxCostFloorPerToken,
		},
		{
			name: "pre-amsterdam/access-list-ignored",
			data: bytes.Repeat([]byte{0xff}, 10),
			accessList: types.AccessList{
				{Address: addr1, StorageKeys: []common.Hash{key1, key2}},
			},
			// pre-amsterdam: floor calculation does not include access list
			want: params.TxGas + 10*params.TxTokenPerNonZeroByte*params.TxCostFloorPerToken,
		},
		{
			name:      "amsterdam/empty",
			amsterdam: true,
			want:      params.TxGas,
		},
		{
			name:      "amsterdam/data-only",
			amsterdam: true,
			data:      bytes.Repeat([]byte{0x00}, 1024),
			// post-amsterdam: every byte = 4 tokens regardless of value
			want: params.TxGas + 1024*params.TxTokenPerNonZeroByte*params.TxCostFloorPerToken7976,
		},
		{
			name:      "amsterdam/data-non-zero",
			amsterdam: true,
			data:      bytes.Repeat([]byte{0xff}, 1024),
			// same as zero data post-amsterdam
			want: params.TxGas + 1024*params.TxTokenPerNonZeroByte*params.TxCostFloorPerToken7976,
		},
		{
			name:      "amsterdam/access-list-addresses-only",
			amsterdam: true,
			accessList: types.AccessList{
				{Address: addr1},
				{Address: addr2},
			},
			// 2 * 20 bytes * 4 tokens/byte * 16 cost/token
			want: params.TxGas + 2*common.AddressLength*params.TxTokenPerNonZeroByte*params.TxCostFloorPerToken7976,
		},
		{
			name:      "amsterdam/access-list-with-storage-keys",
			amsterdam: true,
			accessList: types.AccessList{
				{Address: addr1, StorageKeys: []common.Hash{key1, key2}},
			},
			// 1 addr * 20 * 4 + 2 keys * 32 * 4 = 80 + 256 = 336 tokens * 16
			want: params.TxGas + (1*common.AddressLength+2*common.HashLength)*params.TxTokenPerNonZeroByte*params.TxCostFloorPerToken7976,
		},
		{
			name:      "amsterdam/mixed",
			amsterdam: true,
			data:      bytes.Repeat([]byte{0xff}, 100),
			accessList: types.AccessList{
				{Address: addr1, StorageKeys: []common.Hash{key1}},
				{Address: addr2, StorageKeys: []common.Hash{key1, key2}},
			},
			// data: 100*4 = 400; addrs: 2*20*4 = 160; keys: 3*32*4 = 384; total = 944 * 16
			want: params.TxGas + (100*params.TxTokenPerNonZeroByte+2*common.AddressLength*params.TxTokenPerNonZeroByte+3*common.HashLength*params.TxTokenPerNonZeroByte)*params.TxCostFloorPerToken7976,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := params.Rules{IsAmsterdam: tt.amsterdam}
			got, err := FloorDataGas(rules, tt.data, tt.accessList)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("gas mismatch: got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestIntrinsicGas(t *testing.T) {
	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	key1 := common.HexToHash("0xaa")
	key2 := common.HexToHash("0xbb")

	const (
		amsterdamAddressCost    = uint64(common.AddressLength) * params.TxCostFloorPerToken7976 * params.TxTokenPerNonZeroByte // 1280
		amsterdamStorageKeyCost = uint64(common.HashLength) * params.TxCostFloorPerToken7976 * params.TxTokenPerNonZeroByte    // 2048
	)

	tests := []struct {
		name        string
		data        []byte
		accessList  types.AccessList
		authList    []types.SetCodeAuthorization
		creation    bool
		isHomestead bool
		isEIP2028   bool
		isEIP3860   bool
		isAmsterdam bool
		want        vm.GasCosts
	}{
		{
			name: "frontier/empty-call",
			want: vm.GasCosts{RegularGas: params.TxGas},
		},
		{
			name:        "frontier/contract-creation-pre-homestead",
			creation:    true,
			isHomestead: false,
			// pre-homestead, contract creation still uses TxGas
			want: vm.GasCosts{RegularGas: params.TxGas},
		},
		{
			name:        "homestead/contract-creation",
			creation:    true,
			isHomestead: true,
			want:        vm.GasCosts{RegularGas: params.TxGasContractCreation},
		},
		{
			name: "frontier/non-zero-data",
			data: bytes.Repeat([]byte{0xff}, 100),
			// 100 nz bytes * 68 (frontier)
			want: vm.GasCosts{RegularGas: params.TxGas + 100*params.TxDataNonZeroGasFrontier},
		},
		{
			name:      "istanbul/non-zero-data",
			data:      bytes.Repeat([]byte{0xff}, 100),
			isEIP2028: true,
			// 100 nz bytes * 16 (post-EIP2028)
			want: vm.GasCosts{RegularGas: params.TxGas + 100*params.TxDataNonZeroGasEIP2028},
		},
		{
			name:      "istanbul/zero-data",
			data:      bytes.Repeat([]byte{0x00}, 100),
			isEIP2028: true,
			// 100 zero bytes * 4
			want: vm.GasCosts{RegularGas: params.TxGas + 100*params.TxDataZeroGas},
		},
		{
			name:      "istanbul/mixed-data",
			data:      append(bytes.Repeat([]byte{0x00}, 50), bytes.Repeat([]byte{0xff}, 50)...),
			isEIP2028: true,
			want:      vm.GasCosts{RegularGas: params.TxGas + 50*params.TxDataZeroGas + 50*params.TxDataNonZeroGasEIP2028},
		},
		{
			name:        "shanghai/init-code-word-gas",
			data:        bytes.Repeat([]byte{0x00}, 64), // 2 words
			creation:    true,
			isHomestead: true,
			isEIP2028:   true,
			isEIP3860:   true,
			// TxGasContractCreation + 64 zero bytes * 4 + 2 words * 2
			want: vm.GasCosts{RegularGas: params.TxGasContractCreation + 64*params.TxDataZeroGas + 2*params.InitCodeWordGas},
		},
		{
			name:        "shanghai/init-code-non-multiple-of-32",
			data:        bytes.Repeat([]byte{0x00}, 33), // 2 words (rounded up)
			creation:    true,
			isHomestead: true,
			isEIP2028:   true,
			isEIP3860:   true,
			want:        vm.GasCosts{RegularGas: params.TxGasContractCreation + 33*params.TxDataZeroGas + 2*params.InitCodeWordGas},
		},
		{
			name: "berlin/access-list",
			accessList: types.AccessList{
				{Address: addr1, StorageKeys: []common.Hash{key1, key2}},
				{Address: addr2, StorageKeys: []common.Hash{key1}},
			},
			isEIP2028: true,
			// 2 addrs * 2400 + 3 keys * 1900
			want: vm.GasCosts{RegularGas: params.TxGas + 2*params.TxAccessListAddressGas + 3*params.TxAccessListStorageKeyGas},
		},
		{
			name: "amsterdam/access-list-extra-cost",
			accessList: types.AccessList{
				{Address: addr1, StorageKeys: []common.Hash{key1, key2}},
				{Address: addr2, StorageKeys: []common.Hash{key1}},
			},
			isEIP2028:   true,
			isAmsterdam: true,
			// base access-list charge + EIP-7981 extra
			want: vm.GasCosts{RegularGas: params.TxGas +
				2*params.TxAccessListAddressGas + 3*params.TxAccessListStorageKeyGas +
				2*amsterdamAddressCost + 3*amsterdamStorageKeyCost},
		},
		{
			name: "prague/auth-list",
			authList: []types.SetCodeAuthorization{
				{Address: addr1},
				{Address: addr2},
				{Address: addr1},
			},
			isEIP2028: true,
			// 3 auths * 25000 (pre-Amsterdam: CallNewAccountGas per auth tuple)
			want: vm.GasCosts{RegularGas: params.TxGas + 3*params.CallNewAccountGas},
		},
		{
			name:        "amsterdam/contract-creation-empty",
			creation:    true,
			isHomestead: true,
			isEIP2028:   true,
			isAmsterdam: true,
			// EIP-8037: creation regular gas is TxGas + CreateGasAmsterdam (not TxGasContractCreation),
			// and account-creation cost is moved to state gas.
			want: vm.GasCosts{
				RegularGas: params.TxGas + params.CreateGasAmsterdam,
				StateGas:   params.AccountCreationSize * params.CostPerStateByte,
			},
		},
		{
			name:        "amsterdam/contract-creation-init-code",
			data:        bytes.Repeat([]byte{0x00}, 64), // 2 words of init code
			creation:    true,
			isHomestead: true,
			isEIP2028:   true,
			isEIP3860:   true, // Shanghai gates init-code word gas
			isAmsterdam: true,
			want: vm.GasCosts{
				RegularGas: params.TxGas + params.CreateGasAmsterdam +
					64*params.TxDataZeroGas + 2*params.InitCodeWordGas,
				StateGas: params.AccountCreationSize * params.CostPerStateByte,
			},
		},
		{
			name: "amsterdam/contract-creation-with-access-list",
			data: bytes.Repeat([]byte{0xff}, 32), // 1 word of non-zero init code
			accessList: types.AccessList{
				{Address: addr1, StorageKeys: []common.Hash{key1}},
			},
			creation:    true,
			isHomestead: true,
			isEIP2028:   true,
			isEIP3860:   true,
			isAmsterdam: true,
			want: vm.GasCosts{
				RegularGas: params.TxGas + params.CreateGasAmsterdam +
					32*params.TxDataNonZeroGasEIP2028 + 1*params.InitCodeWordGas +
					1*params.TxAccessListAddressGas + 1*params.TxAccessListStorageKeyGas +
					1*amsterdamAddressCost + 1*amsterdamStorageKeyCost,
				StateGas: params.AccountCreationSize * params.CostPerStateByte,
			},
		},
		{
			name: "amsterdam/combined",
			data: bytes.Repeat([]byte{0xff}, 100),
			accessList: types.AccessList{
				{Address: addr1, StorageKeys: []common.Hash{key1}},
			},
			authList: []types.SetCodeAuthorization{
				{Address: addr2},
			},
			isEIP2028:   true,
			isAmsterdam: true,
			// EIP-8037 splits the auth-tuple charge into regular + state gas:
			//   regular: TxAuthTupleRegularGas (7500) per auth
			//   state:   (AuthorizationCreationSize + AccountCreationSize) * CostPerStateByte per auth
			want: vm.GasCosts{
				RegularGas: params.TxGas +
					100*params.TxDataNonZeroGasEIP2028 +
					1*params.TxAccessListAddressGas + 1*params.TxAccessListStorageKeyGas +
					1*amsterdamAddressCost + 1*amsterdamStorageKeyCost +
					1*params.TxAuthTupleRegularGas,
				StateGas: 1 * (params.AuthorizationCreationSize + params.AccountCreationSize) * params.CostPerStateByte,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := params.Rules{
				IsHomestead: tt.isHomestead,
				IsIstanbul:  tt.isEIP2028,
				IsShanghai:  tt.isEIP3860,
				IsAmsterdam: tt.isAmsterdam,
			}
			got, err := IntrinsicGas(tt.data, tt.accessList, tt.authList,
				tt.creation, rules, params.CostPerStateByte)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("gas mismatch: got %+v, want %+v", got, tt.want)
			}
		})
	}
}

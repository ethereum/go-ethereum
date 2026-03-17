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

package vm

import (
	"fmt"
	"maps"
	"slices"

	"github.com/ethereum/go-ethereum/common"
)

func Example_mainnetEIP7610Accounts() {
	list := slices.SortedFunc(maps.Keys(mainnetEIP7610Accounts), common.Address.Cmp)
	for _, addr := range list {
		fmt.Println(addr.Hex())
	}
	// Output:
	// 0x02820E4bEE488C40f7455fDCa53125565148708F
	// 0x14725085d004f1b10Ee07234A4ab28c5Ad2a7b9E
	// 0x19272418753B90D9a3E3Efc8430b1612c55fcB3A
	// 0x2c081Ed1949D7Dd9447F9d96e509befE576D4461
	// 0x3311c08066580cb906a7287b6786E504C2EBD09f
	// 0x361d7a60b43587c7f6bbA4f9fD9642747F65210A
	// 0x40490C9c468622d5c89646D6F3097F8Eaf80c411
	// 0x4d149EB99BDEEFC1f858f8fd22289C6beAE99f2c
	// 0x5071cb62aA170b7f66b26cae8004d90E6078Bb1E
	// 0x50b1497068bAE652Df3562EB8Ea7677ff84477FA
	// 0x5983C6aC846DcF85fbBC4303F43eb91C379F79ae
	// 0x59EC0410867828E3b8c23Dd8A29d9796ef523b17
	// 0x5cC182faBFb81A056B6080d4200BC5150673D06f
	// 0x6f156dbf8Ed30e53F7C9Df73144E69f65cBB7E94
	// 0x7D6ae067De8d44Ae1A08750e7D626D61A623C44A
	// 0x8398fF6c618e9515468c1c4b198d53666CBe8462
	// 0xA21B22389bfC1cd6Bc7BA19A4Fc96aDC3D0FE074
	// 0xaDD92e0650457C5Db0c4c08cbf7cA580175d33d2
	// 0xAE3703584494Ade958AD27EC2d289b7a67c19E90
	// 0xb619f45637C39Ca49A41ac64c11637A0A194455E
	// 0xD8253352f6044cFE55bcC0748C3FA37b7dF81F98
	// 0xDB7C577B93Baeb56dAB50aF4D6f86F99A06B96a2
	// 0xdE425ad4B8d2d9E0E12F65CBcD6D55F447B44083
	// 0xe62dc49C92fA799033644d2A9aFD7e3BAbE5A80a
	// 0xF468BcBC4a0BFDB06336E773382C5202E674db71
	// 0xF4a835ec1364809003dE3925685F24cD360bdffe
	// 0xFc4465F84B29a1F8794Dc753F41BeF1F4b025ED2
	// 0xfeE7707fa4b8C0A923A0E40399Db3e7Ce26069C6
}

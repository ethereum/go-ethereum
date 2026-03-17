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
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

func decodeEIP7610AccountSet(str string) map[common.Address]struct{} {
	if str == "" {
		return make(map[common.Address]struct{})
	}
	b, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	if len(b)%common.AddressLength != 0 {
		panic(fmt.Sprintf("invalid length, %d", len(b)))
	}
	addresses := make(map[common.Address]struct{}, len(b)/common.AddressLength)
	for i := 0; i < len(b)/common.AddressLength; i++ {
		addresses[common.BytesToAddress(b[i*common.AddressLength:(i+1)*common.AddressLength])] = struct{}{}
	}
	return addresses
}

const mainnetEIP7610Data = "02820e4bee488c40f7455fdca53125565148708f14725085d004f1b10ee07234a4ab28c5ad2a7b9e19272418753b90d9a3e3efc8430b1612c55fcb3a2c081ed1949d7dd9447f9d96e509befe576d44613311c08066580cb906a7287b6786e504c2ebd09f361d7a60b43587c7f6bba4f9fd9642747f65210a40490c9c468622d5c89646d6f3097f8eaf80c4114d149eb99bdeefc1f858f8fd22289c6beae99f2c5071cb62aa170b7f66b26cae8004d90e6078bb1e50b1497068bae652df3562eb8ea7677ff84477fa5983c6ac846dcf85fbbc4303f43eb91c379f79ae59ec0410867828e3b8c23dd8a29d9796ef523b175cc182fabfb81a056b6080d4200bc5150673d06f6f156dbf8ed30e53f7c9df73144e69f65cbb7e947d6ae067de8d44ae1a08750e7d626d61a623c44a8398ff6c618e9515468c1c4b198d53666cbe8462a21b22389bfc1cd6bc7ba19a4fc96adc3d0fe074add92e0650457c5db0c4c08cbf7ca580175d33d2ae3703584494ade958ad27ec2d289b7a67c19e90b619f45637c39ca49a41ac64c11637a0a194455ed8253352f6044cfe55bcc0748c3fa37b7df81f98db7c577b93baeb56dab50af4d6f86f99a06b96a2de425ad4b8d2d9e0e12f65cbcd6d55f447b44083e62dc49c92fa799033644d2a9afd7e3babe5a80af468bcbc4a0bfdb06336e773382c5202e674db71f4a835ec1364809003de3925685f24cd360bdffefc4465f84b29a1f8794dc753f41bef1f4b025ed2fee7707fa4b8c0a923a0e40399db3e7ce26069c6"
const sepoliaEIP7610Data = ""
const holeskyEIP7610Data = ""
const hoodiEIP7610Data = ""

var (
	mainnetEIP7610Accounts = decodeEIP7610AccountSet(mainnetEIP7610Data)
	sepoliaEIP7610Accounts = decodeEIP7610AccountSet(sepoliaEIP7610Data)
	holeskyEIP7610Accounts = decodeEIP7610AccountSet(hoodiEIP7610Data)
	hoodiEIP7610Accounts   = decodeEIP7610AccountSet(holeskyEIP7610Data)
)

// isEIP7610RejectedAccount reports whether the account identified by the
// address is eligible for contract deployment rejection due to having
// non-empty storage.
//
// Note that, historically, there has been no case where a contract deployment
// targets an already existing account in Ethereum. This situation would only
// occur in the event of an address collision, which is extremely unlikely.
//
// This check is skipped for blocks prior to EIP-158, serving as a safeguard
// against potential address collisions in the future.
func isEIP7610RejectedAccount(chainID *big.Int, addr common.Address, isEIP158 bool) bool {
	// Short circuit for blocks prior to EIP-158.
	if !isEIP158 {
		return false
	}
	var accountSet map[common.Address]struct{}
	switch chainID {
	case params.MainnetChainConfig.ChainID:
		accountSet = mainnetEIP7610Accounts
	case params.SepoliaChainConfig.ChainID:
		accountSet = sepoliaEIP7610Accounts
	case params.HoleskyChainConfig.ChainID:
		accountSet = holeskyEIP7610Accounts
	case params.HoodiChainConfig.ChainID:
		accountSet = hoodiEIP7610Accounts
	default:
		// The network is unknown, so the account set must be provided by the
		// network operators themselves. Notably, only a small number of
		// networks enabled EIP-158 after genesis; for all others, this set
		// will always be empty.
		return false
	}
	_, exist := accountSet[addr]
	return exist
}

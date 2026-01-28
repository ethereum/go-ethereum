// Copyright (c) 2018 XDPoSChain
// Copyright 2024 The go-ethereum Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Package blocksigner provides the XDC Network block signer contract interface.
// The block signer contract is deployed at address 0x0000000000000000000000000000000000000089
// and records block signatures for reward distribution.
package blocksigner

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// ContractAddress is the block signer contract address on XDC Network
var ContractAddress = common.HexToAddress("0x0000000000000000000000000000000000000089")

// BlockSignerInfo contains information about a block signature
type BlockSignerInfo struct {
	BlockNumber *big.Int
	BlockHash   common.Hash
	Signer      common.Address
}

// SignMethodID is the method ID for sign(uint256,bytes32) function
// keccak256("sign(uint256,bytes32)")[:4]
var SignMethodID = []byte{0x44, 0x00, 0x8f, 0x05}

// GetSignersMethodID is the method ID for getSigners(uint256) function
// keccak256("getSigners(uint256)")[:4]
var GetSignersMethodID = []byte{0xe7, 0xec, 0x6a, 0xef}

// EncodeSign encodes the sign(uint256,bytes32) call
func EncodeSign(blockNumber *big.Int, blockHash common.Hash) []byte {
	data := make([]byte, 4+64)
	copy(data[:4], SignMethodID)
	
	// Encode block number (uint256)
	blockNumBytes := blockNumber.Bytes()
	copy(data[4+32-len(blockNumBytes):4+32], blockNumBytes)
	
	// Encode block hash (bytes32)
	copy(data[4+32:], blockHash.Bytes())
	
	return data
}

// EncodeGetSigners encodes the getSigners(uint256) call
func EncodeGetSigners(blockNumber *big.Int) []byte {
	data := make([]byte, 4+32)
	copy(data[:4], GetSignersMethodID)
	
	// Encode block number (uint256)
	blockNumBytes := blockNumber.Bytes()
	copy(data[4+32-len(blockNumBytes):4+32], blockNumBytes)
	
	return data
}

// DecodeSigners decodes a list of signer addresses from contract return data
func DecodeSigners(data []byte) ([]common.Address, error) {
	if len(data) < 64 {
		return nil, nil
	}
	// Skip offset (32 bytes) and get length
	length := new(big.Int).SetBytes(data[32:64]).Uint64()
	signers := make([]common.Address, 0, length)
	
	offset := 64
	for i := uint64(0); i < length && offset+32 <= len(data); i++ {
		var addr common.Address
		copy(addr[:], data[offset+12:offset+32])
		signers = append(signers, addr)
		offset += 32
	}
	return signers, nil
}

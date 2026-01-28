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

// Package validator provides the XDC Network validator contract interface.
// The validator contract is deployed at address 0x0000000000000000000000000000000000000088
// and manages the masternode validator set.
package validator

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Validator contract address on XDC Network
var ContractAddress = common.HexToAddress("0x0000000000000000000000000000000000000088")

// MinMasternodeDeposit is the minimum deposit required to become a masternode (10M XDC)
var MinMasternodeDeposit = new(big.Int).Mul(big.NewInt(10000000), big.NewInt(1e18))

// MinVoterCap is the minimum amount to vote for a masternode (25000 XDC)
var MinVoterCap = new(big.Int).Mul(big.NewInt(25000), big.NewInt(1e18))

// MaxMasternodes is the maximum number of masternodes
const MaxMasternodes = 150

// CandidateWithdrawDelay is the delay for candidates to withdraw (30 days in blocks)
const CandidateWithdrawDelay = 1296000

// VoterWithdrawDelay is the delay for voters to withdraw (10 days in blocks)
const VoterWithdrawDelay = 432000

// ValidatorInfo contains information about a masternode validator
type ValidatorInfo struct {
	Address common.Address
	Cap     *big.Int
	Owner   common.Address
}

// GetCandidates is the method ID for getCandidates() function
// keccak256("getCandidates()")[:4]
var GetCandidatesMethodID = []byte{0x06, 0xa4, 0x9e, 0x84}

// GetCandidateCap is the method ID for getCandidateCap(address) function
// keccak256("getCandidateCap(address)")[:4]
var GetCandidateCapMethodID = []byte{0x58, 0xe7, 0x52, 0x5f}

// GetCandidateOwner is the method ID for getCandidateOwner(address) function
// keccak256("getCandidateOwner(address)")[:4]
var GetCandidateOwnerMethodID = []byte{0xb6, 0x42, 0xfa, 0xcd}

// IsCandidate is the method ID for isCandidate(address) function
// keccak256("isCandidate(address)")[:4]
var IsCandidateMethodID = []byte{0xd5, 0x1b, 0x9e, 0x93}

// GetVoterCap is the method ID for getVoterCap(address,address) function
// keccak256("getVoterCap(address,address)")[:4]
var GetVoterCapMethodID = []byte{0x30, 0x2b, 0x68, 0x72}

// GetVoters is the method ID for getVoters(address) function
// keccak256("getVoters(address)")[:4]
var GetVotersMethodID = []byte{0x2d, 0x15, 0xcc, 0x04}

// EncodeGetCandidates encodes the getCandidates() call
func EncodeGetCandidates() []byte {
	return GetCandidatesMethodID
}

// EncodeGetCandidateCap encodes the getCandidateCap(address) call
func EncodeGetCandidateCap(candidate common.Address) []byte {
	data := make([]byte, 4+32)
	copy(data[:4], GetCandidateCapMethodID)
	copy(data[4+12:], candidate.Bytes())
	return data
}

// EncodeIsCandidate encodes the isCandidate(address) call
func EncodeIsCandidate(candidate common.Address) []byte {
	data := make([]byte, 4+32)
	copy(data[:4], IsCandidateMethodID)
	copy(data[4+12:], candidate.Bytes())
	return data
}

// DecodeAddresses decodes a list of addresses from contract return data
func DecodeAddresses(data []byte) ([]common.Address, error) {
	if len(data) < 64 {
		return nil, nil
	}
	// Skip offset (32 bytes) and get length
	length := new(big.Int).SetBytes(data[32:64]).Uint64()
	addresses := make([]common.Address, 0, length)
	
	offset := 64
	for i := uint64(0); i < length && offset+32 <= len(data); i++ {
		var addr common.Address
		copy(addr[:], data[offset+12:offset+32])
		addresses = append(addresses, addr)
		offset += 32
	}
	return addresses, nil
}

// DecodeBigInt decodes a big.Int from contract return data
func DecodeBigInt(data []byte) *big.Int {
	if len(data) < 32 {
		return big.NewInt(0)
	}
	return new(big.Int).SetBytes(data[:32])
}

// DecodeBool decodes a boolean from contract return data
func DecodeBool(data []byte) bool {
	if len(data) < 32 {
		return false
	}
	return data[31] != 0
}

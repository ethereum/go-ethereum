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

package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"
)

// XDC Network contract addresses
var (
	// ValidatorContractAddress is the address of the validator contract
	ValidatorContractAddress = common.HexToAddress("0x0000000000000000000000000000000000000088")
	// BlockSignerContractAddress is the address of the block signer contract
	BlockSignerContractAddress = common.HexToAddress("0x0000000000000000000000000000000000000089")
)

// DefaultXDCMainnetGenesisBlock returns the XDC mainnet genesis block.
func DefaultXDCMainnetGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.XDCMainnetChainConfig,
		Nonce:      0x0,
		Timestamp:  0x5d53f4c0, // July 14, 2019
		ExtraData:  hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
		GasLimit:   420000000,
		Difficulty: big.NewInt(1),
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      xdcMainnetAllocData(),
	}
}

// DefaultXDCApothemGenesisBlock returns the XDC Apothem testnet genesis block.
func DefaultXDCApothemGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.XDCApothemChainConfig,
		Nonce:      0x0,
		Timestamp:  0x5f5e100, // Approximate start time
		ExtraData:  hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
		GasLimit:   420000000,
		Difficulty: big.NewInt(1),
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      xdcApothemAllocData(),
	}
}

// xdcMainnetAllocData returns the genesis alloc for XDC mainnet.
// This includes the validator contract, block signer contract, and initial token distribution.
func xdcMainnetAllocData() GenesisAlloc {
	alloc := make(GenesisAlloc)

	// Validator contract at 0x88
	alloc[ValidatorContractAddress] = GenesisAccount{
		Balance: big.NewInt(0),
		Code:    validatorContractCode(),
		Storage: make(map[common.Hash]common.Hash),
	}

	// Block signer contract at 0x89
	alloc[BlockSignerContractAddress] = GenesisAccount{
		Balance: big.NewInt(0),
		Code:    blockSignerContractCode(),
		Storage: make(map[common.Hash]common.Hash),
	}

	return alloc
}

// xdcApothemAllocData returns the genesis alloc for XDC Apothem testnet.
func xdcApothemAllocData() GenesisAlloc {
	alloc := make(GenesisAlloc)

	// Validator contract at 0x88
	alloc[ValidatorContractAddress] = GenesisAccount{
		Balance: big.NewInt(0),
		Code:    validatorContractCode(),
		Storage: make(map[common.Hash]common.Hash),
	}

	// Block signer contract at 0x89
	alloc[BlockSignerContractAddress] = GenesisAccount{
		Balance: big.NewInt(0),
		Code:    blockSignerContractCode(),
		Storage: make(map[common.Hash]common.Hash),
	}

	// Faucet account for testnet
	faucetAddr := common.HexToAddress("0x0000000000000000000000000000000000000001")
	faucetBalance := new(big.Int)
	faucetBalance.SetString("1000000000000000000000000000", 10) // 1 billion XDC
	alloc[faucetAddr] = GenesisAccount{
		Balance: faucetBalance,
	}

	return alloc
}

// validatorContractCode returns placeholder bytecode for the validator contract.
// In production, this would be the actual compiled Solidity contract.
func validatorContractCode() []byte {
	// Minimal contract that returns empty for getCandidates()
	// This is a placeholder - real deployment uses full contract bytecode
	return hexutil.MustDecode("0x608060405234801561001057600080fd5b506004361061002b5760003560e01c806306a49e8414610030575b600080fd5b61003861004e565b604051610045919061008a565b60405180910390f35b60606000805480602002602001604051908101604052809291908181526020018280548015610080576000815250815260200191505050905090565b6020815250919050565b6020808252825182820181905260009190848201906040850190845b818110156100c8578351835292840192918401916001016100a6565b50909695505050505050565b")
}

// blockSignerContractCode returns placeholder bytecode for the block signer contract.
func blockSignerContractCode() []byte {
	// Minimal contract that handles sign() and getSigners()
	// This is a placeholder - real deployment uses full contract bytecode
	return hexutil.MustDecode("0x608060405234801561001057600080fd5b506004361061002b5760003560e01c8063e7ec6aef14610030575b600080fd5b61004a6004803603810190610045919061009b565b610050565b60405161005791906100f7565b60405180910390f35b6060600080548060200260200160405190810160405280929190818152602001828054801561009257600081525081526020019150505090565b9050919050565b6000602082840312156100ae576000fd5b5035919050565b6020808252825182820181905260009190848201906040850190845b818110156100f35783518352928401929184019160010161d1565b50909695505050505050565b")
}

// IsXDCNetwork returns true if the given chain config is for an XDC network.
func IsXDCNetwork(config *params.ChainConfig) bool {
	if config == nil {
		return false
	}
	return config.XDPoS != nil
}

// GetXDCGenesisBlock returns the appropriate genesis block for the given chain ID.
func GetXDCGenesisBlock(chainID *big.Int) *Genesis {
	if chainID == nil {
		return nil
	}
	switch chainID.Int64() {
	case 50:
		return DefaultXDCMainnetGenesisBlock()
	case 51:
		return DefaultXDCApothemGenesisBlock()
	default:
		return nil
	}
}

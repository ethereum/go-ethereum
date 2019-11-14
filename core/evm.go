// Copyright 2016 The go-ethereum Authors
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
	"math/big"
	"fmt"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

// ChainContext supports retrieving headers and consensus parameters from the
// current blockchain to be used during transaction processing.
type ChainContext interface {
	// Engine retrieves the chain's consensus engine.
	Engine() consensus.Engine

	// GetHeader returns the hash corresponding to their hash.
	GetHeader(common.Hash, uint64) *types.Header
}

// NewEVMContext creates a new context for use in the EVM.
func NewEVMContext(msg Message, header *types.Header, chain ChainContext, author *common.Address) vm.Context {
	// If we don't have an explicit author (i.e. not mining), extract from the header
	var beneficiary common.Address
	if author == nil {
		beneficiary, _ = chain.Engine().Author(header) // Ignore error, we're past header validation
	} else {
		beneficiary = *author
	}
	return vm.Context{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     GetHashFn(header, chain),
		Origin:      msg.From(),
		Coinbase:    beneficiary,
		BlockNumber: new(big.Int).Set(header.Number),
		Time:        new(big.Int).SetUint64(header.Time),
		Difficulty:  new(big.Int).Set(header.Difficulty),
		GasLimit:    header.GasLimit,
		GasPrice:    new(big.Int).Set(msg.GasPrice()),
	}
}

// GetHashFn returns a GetHashFunc which retrieves header hashes by number
func GetHashFn(ref *types.Header, chain ChainContext) func(n uint64) common.Hash {
	var cache map[uint64]common.Hash

	return func(n uint64) common.Hash {
		// If there's no hash cache yet, make one
		if cache == nil {
			cache = map[uint64]common.Hash{
				ref.Number.Uint64() - 1: ref.ParentHash,
			}
		}
		// Try to fulfill the request from the cache
		if hash, ok := cache[n]; ok {
			return hash
		}
		// Not cached, iterate the blocks and cache the hashes
		for header := chain.GetHeader(ref.ParentHash, ref.Number.Uint64()-1); header != nil; header = chain.GetHeader(header.ParentHash, header.Number.Uint64()-1) {
			cache[header.Number.Uint64()-1] = header.ParentHash
			if n == header.Number.Uint64()-1 {
				return header.ParentHash
			}
		}
		return common.Hash{}
	}
}

// CanTransfer checks whether there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(db vm.StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
// func Transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) { // LydianElectrum requires knowledge of evm, not just evm.StateDB
func Transfer(evm *vm.EVM, sender, recipient common.Address, amount *big.Int) {
	// db.SubBalance(sender, amount) // LydianElectrum
	// evm.StateDB.SubBalance(sender, amount) // LydianElectrum: an ERC-20 transfer overrides this operation
	// db.AddBalance(recipient, amount) // LydianElectrum
	// evm.StateDB.AddBalance(recipient, amount) // LydianElectrum: an ERC-20 transfer overrides this operation

	// LydianElectrum: CryptoEuro is an ERC-20 SC deployed on bootstrapping. Being the first one deployed, its address becomes predictable
	address := common.HexToAddress("0x88e726de6cbadc47159c6ccd4f7868ae7a037730") // LydianElectrum: CriptoEuro contract hardcoded address
	contract := vm.AccountRef(address)
	// caller := vm.AccountRef(sender)
		
	// methodHash := "a9059cbb" // transfer method
	// methodHash := "9063e860" // transferOrigin method
	methodHash := "222f5be0" // transferInternal method
	// addressTo := "ca35b7d915458ef540ade6068dfe2f44e8fa733c" // con o sin left padding de leading zeros
	addressTo := recipient.String()[2:] // removing leading 0x
	addressSender := sender.String()[2:] // removing leading 0x

	// padding del addressTo
	for len(addressTo) < 64 { addressTo = "0" + addressTo }

	// padding del addressSender
	for len(addressSender) < 64 { addressSender = "0" + addressSender }

	// convertimos la cantidad a hexadecimal
	amountStr := fmt.Sprintf("%x", amount)

	// padding
	for len(amountStr) < 64 {
		amountStr = "0" + amountStr
	}

	// juntamos todo en una cadena hexadecimal (SIN el 0x delante)
	// inputDataHex := methodHash + addressTo + amountStr
	inputDataHex := methodHash + addressSender + addressTo + amountStr
	
	fmt.Println("inputDataHex: ", inputDataHex)

	// lo convertimos de hexadecimal a []byte que es lo que necesitamos
	inputData, err := hex.DecodeString(inputDataHex)

	gas := uint64(3000000)
	value := new(big.Int)
	
	fmt.Println("sender: ", sender)
	fmt.Println("senderString: ", sender.String())
	fmt.Println("addressSender: ", addressSender)
	fmt.Println("addressTo: ", addressTo)
	fmt.Println("amountStr: ", amountStr)
	fmt.Println("address: ", address)

	// LydianElectrum: this is the ERC-20 transfer that overrides native coin operations
	ret, returnGas, err := evm.CallCode(contract, address, inputData, gas, value)

	fmt.Println("ret: ", ret)
	fmt.Println("returnGas: ", returnGas)
	fmt.Println("err: ", err)
}

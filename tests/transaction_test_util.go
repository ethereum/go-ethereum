// Copyright 2015 The go-ethereum Authors
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

package tests

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// TransactionTest checks RLP decoding and sender derivation of transactions.
type TransactionTest struct {
	RLP            hexutil.Bytes `json:"rlp"`
	Byzantium      ttFork
	Constantinople ttFork
	EIP150         ttFork
	EIP158         ttFork
	Frontier       ttFork
	Homestead      ttFork
}

type ttJSON struct {
	RLP hexutil.Bytes `json:"rlp"`
}

type ttFork struct {
	Sender common.UnprefixedAddress `json:"sender"`
	Hash   common.UnprefixedHash    `json:"hash"`
}

func (tt *TransactionTest) Run(config *params.ChainConfig) error {

	validateTx := func(rlpData hexutil.Bytes, signer types.Signer, block *big.Int) (*common.Address, error) {
		tx := new(types.Transaction)
		if err := rlp.DecodeBytes(rlpData, tx); err != nil {
			return nil, err
		}
		sender, err := types.Sender(signer, tx)
		if err != nil {
			return nil, err
		}
		// Intrinsic gas
		requiredGas, err := core.IntrinsicGas(tx.Data(), tx.To() == nil, config.IsHomestead(block))
		if err != nil {
			return nil, err
		}
		if requiredGas > tx.Gas() {
			return nil, fmt.Errorf("insufficient gas ( %d < %d )", tx.Gas(), requiredGas)
		}
		//if signer.Hash(tx) != common.Hash(fork.Hash) {
		//	return fmt.Errorf("Tx hash mismatch, got %v want %v", signer.Hash(tx), fork.Hash)
		//}
		return &sender, nil
	}

	checkFork := func(block *big.Int, fork ttFork, rlpData hexutil.Bytes) error {
		signer := types.MakeSigner(config, block)
		sender, err := validateTx(rlpData, signer, block)

		// This testcase has an invalid tx
		if fork.Sender == (common.UnprefixedAddress{}) {
			if err == nil {
				return fmt.Errorf("Expected error, got none (address %v)", sender.String())
			}

			return nil
		}
		// Should resolve the right address
		if err != nil {
			return fmt.Errorf("Got error, expected none: %v", err)
		}
		if *sender != common.Address(fork.Sender) {
			return fmt.Errorf("Sender mismatch: got %x, want %x", sender, fork.Sender)
		}
		return nil
	}
	if err := checkFork(new(big.Int), tt.Frontier, tt.RLP); err != nil {
		return fmt.Errorf("Frontier: %v", err)
	}
	if err := checkFork(config.HomesteadBlock, tt.Homestead, tt.RLP); err != nil {
		return fmt.Errorf("Homestead: %v", err)
	}
	if err := checkFork(config.EIP150Block, tt.EIP150, tt.RLP); err != nil {
		return fmt.Errorf("EIP150: %v", err)
	}
	if err := checkFork(config.EIP158Block, tt.EIP158, tt.RLP); err != nil {
		return fmt.Errorf("EIP158: %v", err)
	}
	if err := checkFork(config.ByzantiumBlock, tt.Byzantium, tt.RLP); err != nil {
		return fmt.Errorf("Byzantium: %v", err)
	}
	if err := checkFork(config.ConstantinopleBlock, tt.Constantinople, tt.RLP); err != nil {
		return fmt.Errorf("Constantinople: %v", err)
	}
	return nil
}

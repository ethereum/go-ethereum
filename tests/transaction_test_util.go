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
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// TransactionTest checks RLP decoding and sender derivation of transactions.
type TransactionTest struct {
	Txbytes hexutil.Bytes `json:"txbytes"`
	Result  map[string]*ttFork
}

type ttFork struct {
	Sender       *common.UnprefixedAddress `json:"sender"`
	Hash         *common.UnprefixedHash    `json:"hash"`
	Exception    *string                   `json:"exception"`
	IntrinsicGas math.HexOrDecimal64       `json:"intrinsicGas"`
}

func (tt *TransactionTest) validate() error {
	if tt.Txbytes == nil {
		return fmt.Errorf("missing txbytes")
	}
	for name, fork := range tt.Result {
		if err := tt.validateFork(fork); err != nil {
			return fmt.Errorf("invalid %s: %v", name, err)
		}
	}
	return nil
}

func (tt *TransactionTest) validateFork(fork *ttFork) error {
	if fork == nil {
		return nil
	}
	if fork.Hash == nil && fork.Exception == nil {
		return fmt.Errorf("missing hash and exception")
	}
	if fork.Hash != nil && fork.Sender == nil {
		return fmt.Errorf("missing sender")
	}
	return nil
}

func (tt *TransactionTest) Run() error {
	if err := tt.validate(); err != nil {
		return err
	}
	validateTx := func(rlpData hexutil.Bytes, signer types.Signer, rules *params.Rules) (sender common.Address, hash common.Hash, requiredGas uint64, err error) {
		tx := new(types.Transaction)
		if err = tx.UnmarshalBinary(rlpData); err != nil {
			return
		}
		sender, err = types.Sender(signer, tx)
		if err != nil {
			return
		}
		// Intrinsic gas
		requiredGas, err = core.IntrinsicGas(tx.Data(), tx.AccessList(), tx.SetCodeAuthorizations(), tx.To() == nil, rules.IsHomestead, rules.IsIstanbul, rules.IsShanghai)
		if err != nil {
			return
		}
		if requiredGas > tx.Gas() {
			return sender, hash, 0, fmt.Errorf("insufficient gas ( %d < %d )", tx.Gas(), requiredGas)
		}

		if rules.IsPrague {
			var floorDataGas uint64
			floorDataGas, err = core.FloorDataGas(tx.Data())
			if err != nil {
				return
			}
			if tx.Gas() < floorDataGas {
				return sender, hash, 0, fmt.Errorf("%w: have %d, want %d", core.ErrFloorDataGas, tx.Gas(), floorDataGas)
			}
		}
		hash = tx.Hash()
		return sender, hash, requiredGas, nil
	}
	for _, testcase := range []struct {
		name    string
		isMerge bool
	}{
		{"Frontier", false},
		{"Homestead", false},
		{"EIP150", false},
		{"EIP158", false},
		{"Byzantium", false},
		{"Constantinople", false},
		{"Istanbul", false},
		{"Berlin", false},
		{"London", false},
		{"Paris", true},
		{"Shanghai", true},
		{"Cancun", true},
		{"Prague", true},
	} {
		expected := tt.Result[testcase.name]
		if expected == nil {
			continue
		}
		config, ok := Forks[testcase.name]
		if !ok || config == nil {
			return UnsupportedForkError{Name: testcase.name}
		}
		var (
			rules  = config.Rules(new(big.Int), testcase.isMerge, 0)
			signer = types.MakeSigner(config, new(big.Int), 0)
		)
		sender, hash, gas, err := validateTx(tt.Txbytes, signer, &rules)
		if err != nil {
			if expected.Hash != nil {
				return fmt.Errorf("unexpected error fork %s: %v", testcase.name, err)
			}
			continue
		}
		if expected.Exception != nil {
			return fmt.Errorf("expected error %v, got none (%v), fork %s", *expected.Exception, err, testcase.name)
		}
		if common.Hash(*expected.Hash) != hash {
			return fmt.Errorf("hash mismatch: got %x, want %x", hash, common.Hash(*expected.Hash))
		}
		if common.Address(*expected.Sender) != sender {
			return fmt.Errorf("sender mismatch: got %x, want %x", sender, expected.Sender)
		}
		if uint64(expected.IntrinsicGas) != gas {
			return fmt.Errorf("intrinsic gas mismatch: got %d, want %d", gas, uint64(expected.IntrinsicGas))
		}
	}
	return nil
}

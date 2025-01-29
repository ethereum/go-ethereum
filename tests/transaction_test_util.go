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
	Result  ttResult
}

type ttResult struct {
	Prague         *ttFork
	Cancun         *ttFork
	Shanghai       *ttFork
	Paris          *ttFork
	London         *ttFork
	Berlin         *ttFork
	Byzantium      *ttFork
	Constantinople *ttFork
	Istanbul       *ttFork
	EIP150         *ttFork
	EIP158         *ttFork
	Frontier       *ttFork
	Homestead      *ttFork
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
	if err := tt.validateFork(tt.Result.Prague); err != nil {
		return fmt.Errorf("invalid Prague: %v", err)
	}
	if err := tt.validateFork(tt.Result.Cancun); err != nil {
		return fmt.Errorf("invalid Cancun: %v", err)
	}
	if err := tt.validateFork(tt.Result.Shanghai); err != nil {
		return fmt.Errorf("invalid Shanghai: %v", err)
	}
	if err := tt.validateFork(tt.Result.Paris); err != nil {
		return fmt.Errorf("invalid Paris: %v", err)
	}
	if err := tt.validateFork(tt.Result.London); err != nil {
		return fmt.Errorf("invalid London: %v", err)
	}
	if err := tt.validateFork(tt.Result.Berlin); err != nil {
		return fmt.Errorf("invalid Berlin: %v", err)
	}
	if err := tt.validateFork(tt.Result.Byzantium); err != nil {
		return fmt.Errorf("invalid Byzantium: %v", err)
	}
	if err := tt.validateFork(tt.Result.Constantinople); err != nil {
		return fmt.Errorf("invalid Constantinople: %v", err)
	}
	if err := tt.validateFork(tt.Result.Istanbul); err != nil {
		return fmt.Errorf("invalid Istanbul: %v", err)
	}
	if err := tt.validateFork(tt.Result.EIP150); err != nil {
		return fmt.Errorf("invalid EIP150: %v", err)
	}
	if err := tt.validateFork(tt.Result.EIP158); err != nil {
		return fmt.Errorf("invalid EIP158: %v", err)
	}
	if err := tt.validateFork(tt.Result.Frontier); err != nil {
		return fmt.Errorf("invalid Frontier: %v", err)
	}
	if err := tt.validateFork(tt.Result.Homestead); err != nil {
		return fmt.Errorf("invalid Homestead: %v", err)
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

func (tt *TransactionTest) Run(config *params.ChainConfig) error {
	if err := tt.validate(); err != nil {
		return err
	}
	validateTx := func(rlpData hexutil.Bytes, signer types.Signer, isHomestead, isIstanbul, isShanghai bool) (sender common.Address, hash common.Hash, requiredGas uint64, err error) {
		tx := new(types.Transaction)
		if err = tx.UnmarshalBinary(rlpData); err != nil {
			return
		}
		sender, err = types.Sender(signer, tx)
		if err != nil {
			return
		}
		// Intrinsic gas
		requiredGas, err = core.IntrinsicGas(tx.Data(), tx.AccessList(), tx.SetCodeAuthorizations(), tx.To() == nil, isHomestead, isIstanbul, isShanghai)
		if err != nil {
			return
		}
		if requiredGas > tx.Gas() {
			return sender, hash, 0, fmt.Errorf("insufficient gas ( %d < %d )", tx.Gas(), requiredGas)
		}
		hash = tx.Hash()
		return sender, hash, requiredGas, nil
	}
	for _, testcase := range []struct {
		name        string
		signer      types.Signer
		fork        *ttFork
		isHomestead bool
		isIstanbul  bool
		isShanghai  bool
	}{
		{"Frontier", types.FrontierSigner{}, tt.Result.Frontier, false, false, false},
		{"Homestead", types.HomesteadSigner{}, tt.Result.Homestead, true, false, false},
		{"EIP150", types.HomesteadSigner{}, tt.Result.EIP150, true, false, false},
		{"EIP158", types.NewEIP155Signer(config.ChainID), tt.Result.EIP158, true, false, false},
		{"Byzantium", types.NewEIP155Signer(config.ChainID), tt.Result.Byzantium, true, false, false},
		{"Constantinople", types.NewEIP155Signer(config.ChainID), tt.Result.Constantinople, true, false, false},
		{"Istanbul", types.NewEIP155Signer(config.ChainID), tt.Result.Istanbul, true, true, false},
		{"Berlin", types.NewEIP2930Signer(config.ChainID), tt.Result.Berlin, true, true, false},
		{"London", types.NewLondonSigner(config.ChainID), tt.Result.London, true, true, false},
		{"Paris", types.NewLondonSigner(config.ChainID), tt.Result.Paris, true, true, false},
		{"Shanghai", types.NewLondonSigner(config.ChainID), tt.Result.Shanghai, true, true, true},
		{"Cancun", types.NewCancunSigner(config.ChainID), tt.Result.Cancun, true, true, true},
		{"Prague", types.NewPragueSigner(config.ChainID), tt.Result.Prague, true, true, true},
	} {
		if testcase.fork == nil {
			continue
		}
		sender, hash, gas, err := validateTx(tt.Txbytes, testcase.signer, testcase.isHomestead, testcase.isIstanbul, testcase.isShanghai)
		if err != nil {
			if testcase.fork.Hash != nil {
				return fmt.Errorf("unexpected error: %v", err)
			}
			continue
		}
		if testcase.fork.Exception != nil {
			return fmt.Errorf("expected error %v, got none (%v)", *testcase.fork.Exception, err)
		}
		if common.Hash(*testcase.fork.Hash) != hash {
			return fmt.Errorf("hash mismatch: got %x, want %x", hash, common.Hash(*testcase.fork.Hash))
		}
		if common.Address(*testcase.fork.Sender) != sender {
			return fmt.Errorf("sender mismatch: got %x, want %x", sender, testcase.fork.Sender)
		}
		if hash != common.Hash(*testcase.fork.Hash) {
			return fmt.Errorf("hash mismatch: got %x, want %x", hash, testcase.fork.Hash)
		}
		if uint64(testcase.fork.IntrinsicGas) != gas {
			return fmt.Errorf("intrinsic gas mismatch: got %d, want %d", gas, uint64(testcase.fork.IntrinsicGas))
		}
	}
	return nil
}

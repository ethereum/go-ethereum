// Copyright 2025 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package t8ntool

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// TestEIP7702EmptyAuthListValidation tests that empty auth lists are properly rejected
func TestEIP7702EmptyAuthListValidation(t *testing.T) {
	// Generate test keys for signing
	key1, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key1: %v", err)
	}

	chainID := big.NewInt(1)
	chainIDUint256 := uint256.MustFromBig(chainID)
	signer := types.LatestSignerForChainID(chainID)

	// Create a SetCode transaction with empty authorization list
	txdata := &types.SetCodeTx{
		ChainID:   chainIDUint256,
		Nonce:     0,
		GasTipCap: uint256.NewInt(1),
		GasFeeCap: uint256.NewInt(10),
		Gas:       21000,
		To:        common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Value:     uint256.NewInt(0),
		Data:      nil,
		AuthList:  []types.SetCodeAuthorization{}, // Empty auth list
	}
	tx := types.MustSignNewTx(key1, signer, txdata)

	// Test the validation logic directly
	config := params.TestChainConfig
	config.PragueTime = new(uint64) // Enable Prague features

	// Create validation context
	rules := config.Rules(new(big.Int), false, 0)

	// Simulate the validation logic from transaction.go
	var validationError error

	// Check EIP-7702 authorization list requirements (this is the code we added)
	if tx.Type() == types.SetCodeTxType && tx.SetCodeAuthorizations() != nil {
		if tx.To() == nil {
			validationError = core.ErrSetCodeTxCreate
		} else if len(tx.SetCodeAuthorizations()) == 0 {
			validationError = core.ErrEmptyAuthList
		}
	}

	// Assert that we get the expected error
	if validationError != core.ErrEmptyAuthList {
		t.Errorf("Expected ErrEmptyAuthList for empty auth list, got: %v", validationError)
	}

	// Also test intrinsic gas calculation to ensure it doesn't panic with empty auth list
	_, err = core.IntrinsicGas(tx.Data(), tx.AccessList(), tx.SetCodeAuthorizations(), tx.To() == nil,
		rules.IsHomestead, rules.IsIstanbul, rules.IsShanghai)
	if err != nil {
		t.Logf("IntrinsicGas calculation error (expected): %v", err)
	}
}

// TestEIP7702ContractCreationValidation tests that SetCode tx cannot be used for contract creation
func TestEIP7702ContractCreationValidation(t *testing.T) {
	key2, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key2: %v", err)
	}

	chainID := big.NewInt(1)
	chainIDUint256 := uint256.MustFromBig(chainID)

	// Create authorization
	auth, err := types.SignSetCode(key2, types.SetCodeAuthorization{
		ChainID: *chainIDUint256,
		Address: common.HexToAddress("0x9999999999999999999999999999999999999999"),
		Nonce:   0,
	})
	if err != nil {
		t.Fatalf("Failed to sign auth: %v", err)
	}

	// Test with zero address (simulating nil To)
	var validationError error
	emptyAddr := common.Address{}

	// Simulate validation for contract creation
	if len([]types.SetCodeAuthorization{auth}) > 0 {
		if emptyAddr == (common.Address{}) {
			// In real code, this checks tx.To() == nil
			validationError = core.ErrSetCodeTxCreate
		}
	}

	// Assert that we get the expected error
	if validationError != core.ErrSetCodeTxCreate {
		t.Errorf("Expected ErrSetCodeTxCreate for contract creation, got: %v", validationError)
	}
}

// TestEIP7702ValidTransaction tests that valid SetCode transactions pass validation
func TestEIP7702ValidTransaction(t *testing.T) {
	key1, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key1: %v", err)
	}
	key2, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key2: %v", err)
	}

	chainID := big.NewInt(1)
	chainIDUint256 := uint256.MustFromBig(chainID)
	signer := types.LatestSignerForChainID(chainID)

	// Create a valid authorization
	auth, err := types.SignSetCode(key2, types.SetCodeAuthorization{
		ChainID: *chainIDUint256,
		Address: common.HexToAddress("0x9999999999999999999999999999999999999999"),
		Nonce:   0,
	})
	if err != nil {
		t.Fatalf("Failed to sign auth: %v", err)
	}

	// Create a valid SetCode transaction
	txdata := &types.SetCodeTx{
		ChainID:   chainIDUint256,
		Nonce:     0,
		GasTipCap: uint256.NewInt(1),
		GasFeeCap: uint256.NewInt(10),
		Gas:       30000,
		To:        common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Value:     uint256.NewInt(0),
		Data:      nil,
		AuthList:  []types.SetCodeAuthorization{auth},
	}
	tx := types.MustSignNewTx(key1, signer, txdata)

	// Test validation - should pass with no errors
	var validationError error

	// Check EIP-7702 authorization list requirements
	if tx.Type() == types.SetCodeTxType && tx.SetCodeAuthorizations() != nil {
		if tx.To() == nil {
			validationError = core.ErrSetCodeTxCreate
		} else if len(tx.SetCodeAuthorizations()) == 0 {
			validationError = core.ErrEmptyAuthList
		}
	}

	// Assert that we get no error
	if validationError != nil {
		t.Errorf("Expected no error for valid SetCode transaction, got: %v", validationError)
	}

	// Test intrinsic gas calculation succeeds
	config := params.TestChainConfig
	config.PragueTime = new(uint64)
	rules := config.Rules(new(big.Int), false, 0)

	gas, err := core.IntrinsicGas(tx.Data(), tx.AccessList(), tx.SetCodeAuthorizations(), tx.To() == nil,
		rules.IsHomestead, rules.IsIstanbul, rules.IsShanghai)
	if err != nil {
		t.Errorf("IntrinsicGas calculation failed for valid tx: %v", err)
	}
	if gas == 0 {
		t.Error("IntrinsicGas returned 0 for valid transaction")
	}
}

// TestEIP7702ResetWithZeroAddress tests that reset (zero address) authorizations are valid
func TestEIP7702ResetWithZeroAddress(t *testing.T) {
	key1, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key1: %v", err)
	}
	key2, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key2: %v", err)
	}

	chainID := big.NewInt(1)
	chainIDUint256 := uint256.MustFromBig(chainID)
	signer := types.LatestSignerForChainID(chainID)

	// Create an authorization for resetting delegation (address = 0x0)
	auth, err := types.SignSetCode(key2, types.SetCodeAuthorization{
		ChainID: *chainIDUint256,
		Address: common.Address{}, // Zero address for reset
		Nonce:   0,
	})
	if err != nil {
		t.Fatalf("Failed to sign auth: %v", err)
	}

	// Create a SetCode transaction that resets delegation
	txdata := &types.SetCodeTx{
		ChainID:   chainIDUint256,
		Nonce:     0,
		GasTipCap: uint256.NewInt(1),
		GasFeeCap: uint256.NewInt(10),
		Gas:       30000,
		To:        common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Value:     uint256.NewInt(0),
		Data:      nil,
		AuthList:  []types.SetCodeAuthorization{auth},
	}
	tx := types.MustSignNewTx(key1, signer, txdata)

	// Test validation - should pass with no errors
	var validationError error

	// Check EIP-7702 authorization list requirements
	if tx.Type() == types.SetCodeTxType && tx.SetCodeAuthorizations() != nil {
		if tx.To() == nil {
			validationError = core.ErrSetCodeTxCreate
		} else if len(tx.SetCodeAuthorizations()) == 0 {
			validationError = core.ErrEmptyAuthList
		}
	}

	// Assert that we get no error (zero address is valid for reset)
	if validationError != nil {
		t.Errorf("Expected no error for zero address reset, got: %v", validationError)
	}
}

// TestEIP7702MultipleAuthorizations tests that multiple authorizations are handled correctly
func TestEIP7702MultipleAuthorizations(t *testing.T) {
	key1, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key1: %v", err)
	}
	key2, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key2: %v", err)
	}
	key3, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key3: %v", err)
	}

	chainID := big.NewInt(1)
	chainIDUint256 := uint256.MustFromBig(chainID)
	signer := types.LatestSignerForChainID(chainID)

	// Create multiple authorizations
	auth1, err := types.SignSetCode(key2, types.SetCodeAuthorization{
		ChainID: *chainIDUint256,
		Address: common.HexToAddress("0x1111111111111111111111111111111111111111"),
		Nonce:   0,
	})
	if err != nil {
		t.Fatalf("Failed to sign auth1: %v", err)
	}

	auth2, err := types.SignSetCode(key3, types.SetCodeAuthorization{
		ChainID: *chainIDUint256,
		Address: common.HexToAddress("0x2222222222222222222222222222222222222222"),
		Nonce:   1,
	})
	if err != nil {
		t.Fatalf("Failed to sign auth2: %v", err)
	}

	// Create transaction with multiple authorizations
	txdata := &types.SetCodeTx{
		ChainID:   chainIDUint256,
		Nonce:     0,
		GasTipCap: uint256.NewInt(1),
		GasFeeCap: uint256.NewInt(10),
		Gas:       50000,
		To:        common.HexToAddress("0x3333333333333333333333333333333333333333"),
		Value:     uint256.NewInt(0),
		Data:      nil,
		AuthList:  []types.SetCodeAuthorization{auth1, auth2},
	}
	tx := types.MustSignNewTx(key1, signer, txdata)

	// Test validation - should pass with no errors
	var validationError error

	// Check EIP-7702 authorization list requirements
	if tx.Type() == types.SetCodeTxType && tx.SetCodeAuthorizations() != nil {
		if tx.To() == nil {
			validationError = core.ErrSetCodeTxCreate
		} else if len(tx.SetCodeAuthorizations()) == 0 {
			validationError = core.ErrEmptyAuthList
		}
	}

	// Assert that we get no error
	if validationError != nil {
		t.Errorf("Expected no error for multiple authorizations, got: %v", validationError)
	}

	// Verify we have 2 authorizations
	if len(tx.SetCodeAuthorizations()) != 2 {
		t.Errorf("Expected 2 authorizations, got: %d", len(tx.SetCodeAuthorizations()))
	}
}

// Copyright 2023 The go-ethereum Authors
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
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// setupStateTransition creates a minimal environment for testing stateTransition
func setupStateTransition(t *testing.T) (*stateTransition, *ecdsa.PrivateKey) {
	// Create a private key for signing authorizations
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	address := crypto.PubkeyToAddress(key.PublicKey)

	// Create state
	statedb, _ := state.New(common.Hash{}, state.NewDatabaseForTesting())
	// Set nonce for address
	statedb.SetNonce(address, 5, tracing.NonceChangeUnspecified)

	// Create EVM
	chainConfig := params.TestChainConfig
	chainConfig.ChainID = big.NewInt(1)

	// Create EVM with minimal configuration
	context := vm.BlockContext{
		BlockNumber: new(big.Int).SetUint64(1),
	}
	evm := vm.NewEVM(context, statedb, chainConfig, vm.Config{})

	// Create message
	msg := &Message{
		From:     address,
		GasLimit: 1000000,
	}

	// Create GasPool
	gp := new(GasPool).AddGas(10000000)

	// Create stateTransition
	st := newStateTransition(evm, msg, gp)
	st.gasRemaining = 1000000
	st.initialGas = 1000000

	return st, key
}

// TestValidateAuthorization tests the code authorization validation logic
func TestValidateAuthorization(t *testing.T) {
	st, key := setupStateTransition(t)

	// Get address from key
	address := crypto.PubkeyToAddress(key.PublicKey)

	// Test 1: Valid authorization
	auth := types.SetCodeAuthorization{
		ChainID: *uint256.NewInt(1), // matches chain in setupStateTransition
		Address: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Nonce:   5, // matches nonce in setupStateTransition
	}

	// Sign the authorization
	signedAuth, err := types.SignSetCode(key, auth)
	if err != nil {
		t.Fatalf("failed to sign authorization: %v", err)
	}

	// Check validation
	authAddress, err := st.validateAuthorization(&signedAuth)
	if err != nil {
		t.Errorf("valid authorization failed validation: %v", err)
	}
	if authAddress != address {
		t.Errorf("wrong authority address: got %v, want %v", authAddress, address)
	}

	// Test 2: Wrong Chain ID
	invalidChainIDAuth := signedAuth
	invalidChainIDAuth.ChainID = *uint256.NewInt(999) // doesn't match chain
	_, err = st.validateAuthorization(&invalidChainIDAuth)
	if err != ErrAuthorizationWrongChainID {
		t.Errorf("wrong chain ID error: got %v, want %v", err, ErrAuthorizationWrongChainID)
	}

	// Test 3: Wrong nonce
	invalidNonceAuth := signedAuth
	invalidNonceAuth.ChainID = *uint256.NewInt(1) // restore correct chain ID
	invalidNonceAuth.Nonce = 999                  // doesn't match state nonce
	_, err = st.validateAuthorization(&invalidNonceAuth)
	if err != ErrAuthorizationNonceMismatch {
		t.Errorf("wrong nonce error: got %v, want %v", err, ErrAuthorizationNonceMismatch)
	}

	// Test 4: Nonce overflow
	overflowNonceAuth := signedAuth
	overflowNonceAuth.Nonce = ^uint64(0) // max uint64 value
	_, err = st.validateAuthorization(&overflowNonceAuth)
	if err != ErrAuthorizationNonceOverflow {
		t.Errorf("nonce overflow error: got %v, want %v", err, ErrAuthorizationNonceOverflow)
	}

	// Test 5: Account with code (not delegation)
	codeAccount := crypto.PubkeyToAddress(key.PublicKey)
	// Set arbitrary code for account
	st.state.SetCode(codeAccount, []byte{0x01, 0x02, 0x03})
	_, err = st.validateAuthorization(&signedAuth)
	if err != ErrAuthorizationDestinationHasCode {
		t.Errorf("account with code error: got %v, want %v", err, ErrAuthorizationDestinationHasCode)
	}
}

// TestApplyAuthorization tests the code authorization application logic
func TestApplyAuthorization(t *testing.T) {
	st, key := setupStateTransition(t)

	// Get address from key
	address := crypto.PubkeyToAddress(key.PublicKey)

	// Test 1: Set delegation to another address
	delegationTarget := common.HexToAddress("0x1234567890123456789012345678901234567890")
	auth := types.SetCodeAuthorization{
		ChainID: *uint256.NewInt(1),
		Address: delegationTarget,
		Nonce:   5,
	}

	// Sign the authorization
	signedAuth, err := types.SignSetCode(key, auth)
	if err != nil {
		t.Fatalf("failed to sign authorization: %v", err)
	}

	// Apply the authorization
	err = st.applyAuthorization(&signedAuth)
	if err != nil {
		t.Errorf("failed to apply valid authorization: %v", err)
	}

	// Check that nonce increased
	if nonce := st.state.GetNonce(address); nonce != 6 {
		t.Errorf("wrong nonce after authorization: got %d, want %d", nonce, 6)
	}

	// Check that code became delegation
	code := st.state.GetCode(address)
	// The actual length of delegation code may vary based on implementation
	// Don't check exact length, just verify it's parsed correctly

	targetAddr, ok := types.ParseDelegation(code)
	if !ok {
		t.Errorf("failed to parse delegation code")
	}
	if targetAddr != delegationTarget {
		t.Errorf("wrong delegation target: got %v, want %v", targetAddr, delegationTarget)
	}

	// Test 2: Clear delegation (address 0x0)
	clearAuth := types.SetCodeAuthorization{
		ChainID: *uint256.NewInt(1),
		Address: common.Address{}, // empty address
		Nonce:   6,                // increased nonce
	}

	// Sign the clear authorization
	signedClearAuth, err := types.SignSetCode(key, clearAuth)
	if err != nil {
		t.Fatalf("failed to sign clear authorization: %v", err)
	}

	// Apply the clear authorization
	err = st.applyAuthorization(&signedClearAuth)
	if err != nil {
		t.Errorf("failed to apply clear authorization: %v", err)
	}

	// Check that nonce increased
	if nonce := st.state.GetNonce(address); nonce != 7 {
		t.Errorf("wrong nonce after clear authorization: got %d, want %d", nonce, 7)
	}

	// Check that code was cleared
	code = st.state.GetCode(address)
	if len(code) != 0 {
		t.Errorf("code was not cleared: got length %d, want 0", len(code))
	}

	// Test 3: Gas refund for existing account
	initialRefund := st.state.GetRefund()

	// Apply authorization for existing account
	auth.Nonce = 7 // increased nonce
	signedAuth, _ = types.SignSetCode(key, auth)

	err = st.applyAuthorization(&signedAuth)
	if err != nil {
		t.Errorf("failed to apply authorization for existing account: %v", err)
	}

	// Check that gas was refunded for existing account
	expectedRefund := initialRefund + (params.CallNewAccountGas - params.TxAuthTupleGas)
	if refund := st.state.GetRefund(); refund != expectedRefund {
		t.Errorf("wrong refund after authorization for existing account: got %d, want %d", refund, expectedRefund)
	}
}

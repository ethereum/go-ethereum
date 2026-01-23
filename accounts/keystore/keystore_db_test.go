// Copyright 2024 The go-ethereum Authors
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

package keystore

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestDBKeyStore_NewAccount(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	// Create a new account
	a, err := ks.NewAccount("password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	if a.Address == (common.Address{}) {
		t.Fatal("Account address is zero")
	}

	// Verify the account exists
	if !ks.HasAddress(a.Address) {
		t.Fatal("Account not found in keystore")
	}

	// Verify the account is in the list
	accounts := ks.Accounts()
	if len(accounts) != 1 {
		t.Fatalf("Expected 1 account, got %d", len(accounts))
	}
	if accounts[0].Address != a.Address {
		t.Fatalf("Account address mismatch")
	}
}

func TestDBKeyStore_MultipleAccounts(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	// Create multiple accounts
	numAccounts := 10
	addresses := make(map[common.Address]bool)

	for i := 0; i < numAccounts; i++ {
		a, err := ks.NewAccount("password123")
		if err != nil {
			t.Fatalf("Failed to create account %d: %v", i, err)
		}
		addresses[a.Address] = true
	}

	// Verify all accounts exist
	accounts := ks.Accounts()
	if len(accounts) != numAccounts {
		t.Fatalf("Expected %d accounts, got %d", numAccounts, len(accounts))
	}

	for _, acc := range accounts {
		if !addresses[acc.Address] {
			t.Fatalf("Unexpected account in list: %s", acc.Address.Hex())
		}
	}

	// Check count
	if ks.Count() != uint64(numAccounts) {
		t.Fatalf("Expected count %d, got %d", numAccounts, ks.Count())
	}
}

func TestDBKeyStore_Delete(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	// Create an account
	a, err := ks.NewAccount("password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Delete with wrong password should fail
	if err := ks.Delete(a, "wrongpassword"); err == nil {
		t.Fatal("Delete with wrong password should fail")
	}

	// Delete with correct password should succeed
	if err := ks.Delete(a, "password123"); err != nil {
		t.Fatalf("Failed to delete account: %v", err)
	}

	// Verify the account is gone
	if ks.HasAddress(a.Address) {
		t.Fatal("Account should not exist after deletion")
	}

	accounts := ks.Accounts()
	if len(accounts) != 0 {
		t.Fatalf("Expected 0 accounts, got %d", len(accounts))
	}
}

func TestDBKeyStore_UnlockAndSign(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	// Create an account
	a, err := ks.NewAccount("password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Try to sign without unlocking - should fail
	hash := crypto.Keccak256([]byte("test message"))
	_, err = ks.SignHash(a, hash)
	if err != ErrLocked {
		t.Fatal("SignHash should fail when account is locked")
	}

	// Unlock the account
	if err := ks.Unlock(a, "password123"); err != nil {
		t.Fatalf("Failed to unlock account: %v", err)
	}

	// Now signing should work
	sig, err := ks.SignHash(a, hash)
	if err != nil {
		t.Fatalf("Failed to sign hash: %v", err)
	}

	if len(sig) != 65 {
		t.Fatalf("Unexpected signature length: %d", len(sig))
	}

	// Lock the account
	if err := ks.Lock(a.Address); err != nil {
		t.Fatalf("Failed to lock account: %v", err)
	}

	// Signing should fail again
	_, err = ks.SignHash(a, hash)
	if err != ErrLocked {
		t.Fatal("SignHash should fail when account is locked")
	}
}

func TestDBKeyStore_SignWithPassphrase(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	// Create an account
	a, err := ks.NewAccount("password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	hash := crypto.Keccak256([]byte("test message"))

	// Sign with wrong passphrase should fail
	_, err = ks.SignHashWithPassphrase(a, "wrongpassword", hash)
	if err == nil {
		t.Fatal("SignHashWithPassphrase with wrong password should fail")
	}

	// Sign with correct passphrase should succeed
	sig, err := ks.SignHashWithPassphrase(a, "password123", hash)
	if err != nil {
		t.Fatalf("Failed to sign hash: %v", err)
	}

	if len(sig) != 65 {
		t.Fatalf("Unexpected signature length: %d", len(sig))
	}
}

func TestDBKeyStore_TimedUnlock(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	// Create an account
	a, err := ks.NewAccount("password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Unlock for a short duration
	if err := ks.TimedUnlock(a, "password123", 100*time.Millisecond); err != nil {
		t.Fatalf("Failed to timed unlock account: %v", err)
	}

	// Signing should work immediately
	hash := crypto.Keccak256([]byte("test message"))
	_, err = ks.SignHash(a, hash)
	if err != nil {
		t.Fatalf("Failed to sign hash: %v", err)
	}

	// Wait for unlock to expire
	time.Sleep(200 * time.Millisecond)

	// Signing should fail after timeout
	_, err = ks.SignHash(a, hash)
	if err != ErrLocked {
		t.Fatal("SignHash should fail after timed unlock expires")
	}
}

func TestDBKeyStore_SignTx(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	// Create an account
	a, err := ks.NewAccount("password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Create a transaction
	to := common.HexToAddress("0x1234567890123456789012345678901234567890")
	tx := types.NewTransaction(0, to, big.NewInt(1000), 21000, big.NewInt(1), nil)
	chainID := big.NewInt(1)

	// Sign without unlocking should fail
	_, err = ks.SignTx(a, tx, chainID)
	if err != ErrLocked {
		t.Fatal("SignTx should fail when account is locked")
	}

	// Unlock and sign
	if err := ks.Unlock(a, "password123"); err != nil {
		t.Fatalf("Failed to unlock account: %v", err)
	}

	signedTx, err := ks.SignTx(a, tx, chainID)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Verify the signature
	signer := types.LatestSignerForChainID(chainID)
	from, err := types.Sender(signer, signedTx)
	if err != nil {
		t.Fatalf("Failed to recover sender: %v", err)
	}

	if from != a.Address {
		t.Fatalf("Sender mismatch: got %s, want %s", from.Hex(), a.Address.Hex())
	}
}

func TestDBKeyStore_ImportExport(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	// Create an account
	a, err := ks.NewAccount("password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Export the account
	keyJSON, err := ks.Export(a, "password123", "newpassword456")
	if err != nil {
		t.Fatalf("Failed to export account: %v", err)
	}

	// Create a new keystore
	dir2 := t.TempDir()
	ks2, err := NewDBKeyStore(dir2, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to create second DB keystore: %v", err)
	}
	defer ks2.Close()

	// Import the account
	importedAcc, err := ks2.Import(keyJSON, "newpassword456", "finalpassword789")
	if err != nil {
		t.Fatalf("Failed to import account: %v", err)
	}

	if importedAcc.Address != a.Address {
		t.Fatalf("Imported account address mismatch")
	}

	// Verify the imported account works
	if err := ks2.Unlock(importedAcc, "finalpassword789"); err != nil {
		t.Fatalf("Failed to unlock imported account: %v", err)
	}
}

func TestDBKeyStore_ImportECDSA(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	// Generate a private key
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	expectedAddr := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Import the key
	a, err := ks.ImportECDSA(privateKey, "password123")
	if err != nil {
		t.Fatalf("Failed to import ECDSA key: %v", err)
	}

	if a.Address != expectedAddr {
		t.Fatalf("Imported address mismatch: got %s, want %s", a.Address.Hex(), expectedAddr.Hex())
	}

	// Verify the account works
	if err := ks.Unlock(a, "password123"); err != nil {
		t.Fatalf("Failed to unlock imported account: %v", err)
	}

	hash := crypto.Keccak256([]byte("test"))
	sig, err := ks.SignHash(a, hash)
	if err != nil {
		t.Fatalf("Failed to sign with imported key: %v", err)
	}

	// Verify the signature
	pubKey, err := crypto.SigToPub(hash, sig)
	if err != nil {
		t.Fatalf("Failed to recover public key: %v", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	if recoveredAddr != expectedAddr {
		t.Fatalf("Recovered address mismatch")
	}
}

func TestDBKeyStore_Update(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	// Create an account
	a, err := ks.NewAccount("password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Update password with wrong old password should fail
	if err := ks.Update(a, "wrongpassword", "newpassword456"); err == nil {
		t.Fatal("Update with wrong password should fail")
	}

	// Update password with correct old password
	if err := ks.Update(a, "password123", "newpassword456"); err != nil {
		t.Fatalf("Failed to update password: %v", err)
	}

	// Old password should no longer work
	if err := ks.Unlock(a, "password123"); err == nil {
		t.Fatal("Old password should not work after update")
	}

	// New password should work
	if err := ks.Unlock(a, "newpassword456"); err != nil {
		t.Fatalf("New password should work: %v", err)
	}
}

func TestDBKeyStore_DuplicateImport(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	// Generate and import a key
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_, err = ks.ImportECDSA(privateKey, "password123")
	if err != nil {
		t.Fatalf("Failed to import key: %v", err)
	}

	// Try to import the same key again - should fail
	_, err = ks.ImportECDSA(privateKey, "password123")
	if err != ErrAccountAlreadyExists {
		t.Fatalf("Expected ErrAccountAlreadyExists, got: %v", err)
	}
}

func TestDBKeyStore_Wallets(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	// Create accounts
	a1, err := ks.NewAccount("password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	a2, err := ks.NewAccount("password456")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Get wallets
	wallets := ks.Wallets()
	if len(wallets) != 2 {
		t.Fatalf("Expected 2 wallets, got %d", len(wallets))
	}

	// Find wallet for each account
	foundA1, foundA2 := false, false
	for _, w := range wallets {
		accs := w.Accounts()
		if len(accs) != 1 {
			t.Fatal("Each wallet should have exactly 1 account")
		}
		if accs[0].Address == a1.Address {
			foundA1 = true
		}
		if accs[0].Address == a2.Address {
			foundA2 = true
		}
	}

	if !foundA1 || !foundA2 {
		t.Fatal("Not all accounts found in wallets")
	}
}

func TestDBKeyStore_WalletSignTx(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	// Create an account
	a, err := ks.NewAccount("password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Get wallet
	wallets := ks.Wallets()
	if len(wallets) != 1 {
		t.Fatalf("Expected 1 wallet, got %d", len(wallets))
	}
	wallet := wallets[0]

	// Create a transaction
	to := common.HexToAddress("0x1234567890123456789012345678901234567890")
	tx := types.NewTransaction(0, to, big.NewInt(1000), 21000, big.NewInt(1), nil)
	chainID := big.NewInt(1)

	// Sign transaction with passphrase through wallet
	signedTx, err := wallet.SignTxWithPassphrase(a, "password123", tx, chainID)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Verify the signature
	signer := types.LatestSignerForChainID(chainID)
	from, err := types.Sender(signer, signedTx)
	if err != nil {
		t.Fatalf("Failed to recover sender: %v", err)
	}

	if from != a.Address {
		t.Fatalf("Sender mismatch: got %s, want %s", from.Hex(), a.Address.Hex())
	}
}

func TestDBKeyStore_Persistence(t *testing.T) {
	dir := t.TempDir()

	// Create keystore and add an account
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to create DB keystore: %v", err)
	}

	a, err := ks.NewAccount("password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	addr := a.Address
	ks.Close()

	// Reopen keystore and verify account persists
	ks2, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to reopen DB keystore: %v", err)
	}
	defer ks2.Close()

	if !ks2.HasAddress(addr) {
		t.Fatal("Account should persist after reopening keystore")
	}

	accounts := ks2.Accounts()
	if len(accounts) != 1 {
		t.Fatalf("Expected 1 account, got %d", len(accounts))
	}

	if accounts[0].Address != addr {
		t.Fatal("Account address mismatch after reopen")
	}

	// Verify the account still works
	if err := ks2.Unlock(accounts[0], "password123"); err != nil {
		t.Fatalf("Failed to unlock account after reopen: %v", err)
	}
}

func BenchmarkDBKeyStore_NewAccount(b *testing.B) {
	dir := b.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		b.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ks.NewAccount("password123")
		if err != nil {
			b.Fatalf("Failed to create account: %v", err)
		}
	}
}

func BenchmarkDBKeyStore_HasAddress(b *testing.B) {
	dir := b.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		b.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	// Pre-populate with accounts
	var addrs []common.Address
	for i := 0; i < 1000; i++ {
		a, err := ks.NewAccount("password123")
		if err != nil {
			b.Fatalf("Failed to create account: %v", err)
		}
		addrs = append(addrs, a.Address)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		addr := addrs[i%len(addrs)]
		ks.HasAddress(addr)
	}
}

func BenchmarkDBKeyStore_Accounts(b *testing.B) {
	dir := b.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		b.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	// Pre-populate with accounts
	for i := 0; i < 1000; i++ {
		_, err := ks.NewAccount("password123")
		if err != nil {
			b.Fatalf("Failed to create account: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		accounts := ks.Accounts()
		if len(accounts) != 1000 {
			b.Fatalf("Expected 1000 accounts, got %d", len(accounts))
		}
	}
}

// Helper function to check if account implements wallet interface
func TestDBKeystoreWallet_Interface(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewDBKeyStore(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("Failed to create DB keystore: %v", err)
	}
	defer ks.Close()

	_, err = ks.NewAccount("password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	wallets := ks.Wallets()
	if len(wallets) != 1 {
		t.Fatalf("Expected 1 wallet, got %d", len(wallets))
	}

	// Check the wallet implements accounts.Wallet
	var _ accounts.Wallet = wallets[0]
}

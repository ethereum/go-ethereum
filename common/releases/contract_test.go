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

package releases

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

// setupReleaseTest creates a blockchain simulator and deploys a version oracle
// contract for testing.
func setupReleaseTest(t *testing.T, prefund ...*ecdsa.PrivateKey) (*ecdsa.PrivateKey, *ReleaseOracle, *backends.SimulatedBackend) {
	// Generate a new random account and a funded simulator
	key, _ := crypto.GenerateKey()
	auth := bind.NewKeyedTransactor(key)

	accounts := []core.GenesisAccount{{Address: auth.From, Balance: big.NewInt(10000000000)}}
	for _, key := range prefund {
		accounts = append(accounts, core.GenesisAccount{Address: crypto.PubkeyToAddress(key.PublicKey), Balance: big.NewInt(10000000000)})
	}
	sim := backends.NewSimulatedBackend(accounts...)

	// Deploy a version oracle contract, commit and return
	_, _, oracle, err := DeployReleaseOracle(auth, sim)
	if err != nil {
		t.Fatalf("Failed to deploy version contract: %v", err)
	}
	sim.Commit()

	return key, oracle, sim
}

// Tests that the version contract can be deployed and the creator is assigned
// the sole authorized signer.
func TestContractCreation(t *testing.T) {
	key, oracle, _ := setupReleaseTest(t)

	owner := crypto.PubkeyToAddress(key.PublicKey)
	signers, err := oracle.Signers(nil)
	if err != nil {
		t.Fatalf("Failed to retrieve list of signers: %v", err)
	}
	if len(signers) != 1 || signers[0] != owner {
		t.Fatalf("Initial signer mismatch: have %v, want %v", signers, owner)
	}
}

// Tests that subsequent signers can be promoted, each requiring half plus one
// votes for it to pass through.
func TestSignerPromotion(t *testing.T) {
	// Prefund a few accounts to authorize with and create the oracle
	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
	}
	key, oracle, sim := setupReleaseTest(t, keys...)

	// Gradually promote the keys, until all are authorized
	keys = append([]*ecdsa.PrivateKey{key}, keys...)
	for i := 1; i < len(keys); i++ {
		// Check that no votes are accepted from the not yet authed user
		if _, err := oracle.Promote(bind.NewKeyedTransactor(keys[i]), common.Address{}); err != nil {
			t.Fatalf("Iter #%d: failed invalid promotion attempt: %v", i, err)
		}
		sim.Commit()

		pend, err := oracle.AuthProposals(nil)
		if err != nil {
			t.Fatalf("Iter #%d: failed to retrieve active proposals: %v", i, err)
		}
		if len(pend) != 0 {
			t.Fatalf("Iter #%d: proposal count mismatch: have %d, want 0", i, len(pend))
		}
		// Promote with half - 1 voters and check that the user's not yet authorized
		for j := 0; j < i/2; j++ {
			if _, err = oracle.Promote(bind.NewKeyedTransactor(keys[j]), crypto.PubkeyToAddress(keys[i].PublicKey)); err != nil {
				t.Fatalf("Iter #%d: failed valid promotion attempt: %v", i, err)
			}
		}
		sim.Commit()

		signers, err := oracle.Signers(nil)
		if err != nil {
			t.Fatalf("Iter #%d: failed to retrieve list of signers: %v", i, err)
		}
		if len(signers) != i {
			t.Fatalf("Iter #%d: signer count mismatch: have %v, want %v", i, len(signers), i)
		}
		// Promote with the last one needed to pass the promotion
		if _, err = oracle.Promote(bind.NewKeyedTransactor(keys[i/2]), crypto.PubkeyToAddress(keys[i].PublicKey)); err != nil {
			t.Fatalf("Iter #%d: failed valid promotion completion attempt: %v", i, err)
		}
		sim.Commit()

		signers, err = oracle.Signers(nil)
		if err != nil {
			t.Fatalf("Iter #%d: failed to retrieve list of signers: %v", i, err)
		}
		if len(signers) != i+1 {
			t.Fatalf("Iter #%d: signer count mismatch: have %v, want %v", i, len(signers), i+1)
		}
	}
}

// Tests that subsequent signers can be demoted, each requiring half plus one
// votes for it to pass through.
func TestSignerDemotion(t *testing.T) {
	// Prefund a few accounts to authorize with and create the oracle
	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
	}
	key, oracle, sim := setupReleaseTest(t, keys...)

	// Authorize all the keys as valid signers and verify cardinality
	keys = append([]*ecdsa.PrivateKey{key}, keys...)
	for i := 1; i < len(keys); i++ {
		for j := 0; j <= i/2; j++ {
			if _, err := oracle.Promote(bind.NewKeyedTransactor(keys[j]), crypto.PubkeyToAddress(keys[i].PublicKey)); err != nil {
				t.Fatalf("Iter #%d: failed valid promotion attempt: %v", i, err)
			}
		}
		sim.Commit()
	}
	signers, err := oracle.Signers(nil)
	if err != nil {
		t.Fatalf("Failed to retrieve list of signers: %v", err)
	}
	if len(signers) != len(keys) {
		t.Fatalf("Signer count mismatch: have %v, want %v", len(signers), len(keys))
	}
	// Gradually demote users until we run out of signers
	for i := len(keys) - 1; i >= 0; i-- {
		// Demote with half - 1 voters and check that the user's not yet dropped
		for j := 0; j < (i+1)/2; j++ {
			if _, err = oracle.Demote(bind.NewKeyedTransactor(keys[j]), crypto.PubkeyToAddress(keys[i].PublicKey)); err != nil {
				t.Fatalf("Iter #%d: failed valid demotion attempt: %v", len(keys)-i, err)
			}
		}
		sim.Commit()

		signers, err := oracle.Signers(nil)
		if err != nil {
			t.Fatalf("Iter #%d: failed to retrieve list of signers: %v", len(keys)-i, err)
		}
		if len(signers) != i+1 {
			t.Fatalf("Iter #%d: signer count mismatch: have %v, want %v", len(keys)-i, len(signers), i+1)
		}
		// Demote with the last one needed to pass the demotion
		if _, err = oracle.Demote(bind.NewKeyedTransactor(keys[(i+1)/2]), crypto.PubkeyToAddress(keys[i].PublicKey)); err != nil {
			t.Fatalf("Iter #%d: failed valid demotion completion attempt: %v", i, err)
		}
		sim.Commit()

		signers, err = oracle.Signers(nil)
		if err != nil {
			t.Fatalf("Iter #%d: failed to retrieve list of signers: %v", len(keys)-i, err)
		}
		if len(signers) != i {
			t.Fatalf("Iter #%d: signer count mismatch: have %v, want %v", len(keys)-i, len(signers), i)
		}
		// Check that no votes are accepted from the already demoted users
		if _, err = oracle.Promote(bind.NewKeyedTransactor(keys[i]), common.Address{}); err != nil {
			t.Fatalf("Iter #%d: failed invalid promotion attempt: %v", i, err)
		}
		sim.Commit()

		pend, err := oracle.AuthProposals(nil)
		if err != nil {
			t.Fatalf("Iter #%d: failed to retrieve active proposals: %v", i, err)
		}
		if len(pend) != 0 {
			t.Fatalf("Iter #%d: proposal count mismatch: have %d, want 0", i, len(pend))
		}
	}
}

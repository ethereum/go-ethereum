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

package release

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ubiq/go-ubiq/accounts/abi/bind"
	"github.com/ubiq/go-ubiq/accounts/abi/bind/backends"
	"github.com/ubiq/go-ubiq/common"
	"github.com/ubiq/go-ubiq/core"
	"github.com/ubiq/go-ubiq/crypto"
)

// setupReleaseTest creates a blockchain simulator and deploys a version oracle
// contract for testing.
func setupReleaseTest(t *testing.T, prefund ...*ecdsa.PrivateKey) (*ecdsa.PrivateKey, *ReleaseOracle, *backends.SimulatedBackend) {
	// Generate a new random account and a funded simulator
	key, _ := crypto.GenerateKey()
	auth := bind.NewKeyedTransactor(key)

	alloc := core.GenesisAlloc{auth.From: {Balance: big.NewInt(10000000000)}}
	for _, key := range prefund {
		alloc[crypto.PubkeyToAddress(key.PublicKey)] = core.GenesisAccount{Balance: big.NewInt(10000000000)}
	}
	sim := backends.NewSimulatedBackend(alloc)

	// Deploy a version oracle contract, commit and return
	_, _, oracle, err := DeployReleaseOracle(auth, sim, []common.Address{auth.From})
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
		// Check that no votes are accepted from the not yet authorized user
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

// Tests that new versions can be released, honouring both voting rights as well
// as the minimum required vote count.
func TestVersionRelease(t *testing.T) {
	// Prefund a few accounts to authorize with and create the oracle
	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
	}
	key, oracle, sim := setupReleaseTest(t, keys...)

	// Track the "current release"
	var (
		verMajor  = uint32(0)
		verMinor  = uint32(0)
		verPatch  = uint32(0)
		verCommit = [20]byte{}
	)
	// Gradually push releases, always requiring more signers than previously
	keys = append([]*ecdsa.PrivateKey{key}, keys...)
	for i := 1; i < len(keys); i++ {
		// Check that no votes are accepted from the not yet authorized user
		if _, err := oracle.Release(bind.NewKeyedTransactor(keys[i]), 0, 0, 0, [20]byte{0}); err != nil {
			t.Fatalf("Iter #%d: failed invalid release attempt: %v", i, err)
		}
		sim.Commit()

		prop, err := oracle.ProposedVersion(nil)
		if err != nil {
			t.Fatalf("Iter #%d: failed to retrieve active proposal: %v", i, err)
		}
		if len(prop.Pass) != 0 {
			t.Fatalf("Iter #%d: proposal vote count mismatch: have %d, want 0", i, len(prop.Pass))
		}
		// Authorize the user to make releases
		for j := 0; j <= i/2; j++ {
			if _, err = oracle.Promote(bind.NewKeyedTransactor(keys[j]), crypto.PubkeyToAddress(keys[i].PublicKey)); err != nil {
				t.Fatalf("Iter #%d: failed valid promotion attempt: %v", i, err)
			}
		}
		sim.Commit()

		// Propose release with half voters and check that the release does not yet go through
		for j := 0; j < (i+1)/2; j++ {
			if _, err = oracle.Release(bind.NewKeyedTransactor(keys[j]), uint32(i), uint32(i+1), uint32(i+2), [20]byte{byte(i + 3)}); err != nil {
				t.Fatalf("Iter #%d: failed valid release attempt: %v", i, err)
			}
		}
		sim.Commit()

		ver, err := oracle.CurrentVersion(nil)
		if err != nil {
			t.Fatalf("Iter #%d: failed to retrieve current version: %v", i, err)
		}
		if ver.Major != verMajor || ver.Minor != verMinor || ver.Patch != verPatch || ver.Commit != verCommit {
			t.Fatalf("Iter #%d: version mismatch: have %d.%d.%d-%x, want %d.%d.%d-%x", i, ver.Major, ver.Minor, ver.Patch, ver.Commit, verMajor, verMinor, verPatch, verCommit)
		}

		// Pass the release and check that it became the next version
		verMajor, verMinor, verPatch, verCommit = uint32(i), uint32(i+1), uint32(i+2), [20]byte{byte(i + 3)}
		if _, err = oracle.Release(bind.NewKeyedTransactor(keys[(i+1)/2]), uint32(i), uint32(i+1), uint32(i+2), [20]byte{byte(i + 3)}); err != nil {
			t.Fatalf("Iter #%d: failed valid release completion attempt: %v", i, err)
		}
		sim.Commit()

		ver, err = oracle.CurrentVersion(nil)
		if err != nil {
			t.Fatalf("Iter #%d: failed to retrieve current version: %v", i, err)
		}
		if ver.Major != verMajor || ver.Minor != verMinor || ver.Patch != verPatch || ver.Commit != verCommit {
			t.Fatalf("Iter #%d: version mismatch: have %d.%d.%d-%x, want %d.%d.%d-%x", i, ver.Major, ver.Minor, ver.Patch, ver.Commit, verMajor, verMinor, verPatch, verCommit)
		}
	}
}

// Tests that proposed versions can be nuked out of existence.
func TestVersionNuking(t *testing.T) {
	// Prefund a few accounts to authorize with and create the oracle
	keys := make([]*ecdsa.PrivateKey, 9)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
	}
	key, oracle, sim := setupReleaseTest(t, keys...)

	// Authorize all the keys as valid signers
	keys = append([]*ecdsa.PrivateKey{key}, keys...)
	for i := 1; i < len(keys); i++ {
		for j := 0; j <= i/2; j++ {
			if _, err := oracle.Promote(bind.NewKeyedTransactor(keys[j]), crypto.PubkeyToAddress(keys[i].PublicKey)); err != nil {
				t.Fatalf("Iter #%d: failed valid promotion attempt: %v", i, err)
			}
		}
		sim.Commit()
	}
	// Propose releases with more and more keys, always retaining enough users to nuke the proposals
	for i := 1; i < (len(keys)+1)/2; i++ {
		// Propose release with an initial set of signers
		for j := 0; j < i; j++ {
			if _, err := oracle.Release(bind.NewKeyedTransactor(keys[j]), uint32(i), uint32(i+1), uint32(i+2), [20]byte{byte(i + 3)}); err != nil {
				t.Fatalf("Iter #%d: failed valid proposal attempt: %v", i, err)
			}
		}
		sim.Commit()

		prop, err := oracle.ProposedVersion(nil)
		if err != nil {
			t.Fatalf("Iter #%d: failed to retrieve active proposal: %v", i, err)
		}
		if len(prop.Pass) != i {
			t.Fatalf("Iter #%d: proposal vote count mismatch: have %d, want %d", i, len(prop.Pass), i)
		}
		// Nuke the release with half+1 voters
		for j := i; j <= i+(len(keys)+1)/2; j++ {
			if _, err := oracle.Nuke(bind.NewKeyedTransactor(keys[j])); err != nil {
				t.Fatalf("Iter #%d: failed valid nuke attempt: %v", i, err)
			}
		}
		sim.Commit()

		prop, err = oracle.ProposedVersion(nil)
		if err != nil {
			t.Fatalf("Iter #%d: failed to retrieve active proposal: %v", i, err)
		}
		if len(prop.Pass) != 0 || len(prop.Fail) != 0 {
			t.Fatalf("Iter #%d: proposal vote count mismatch: have %d/%d pass/fail, want 0/0", i, len(prop.Pass), len(prop.Fail))
		}
	}
}

// Tests that demoting a signer will auto-nuke the currently pending release.
func TestVersionAutoNuke(t *testing.T) {
	// Prefund a few accounts to authorize with and create the oracle
	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
	}
	key, oracle, sim := setupReleaseTest(t, keys...)

	// Authorize all the keys as valid signers
	keys = append([]*ecdsa.PrivateKey{key}, keys...)
	for i := 1; i < len(keys); i++ {
		for j := 0; j <= i/2; j++ {
			if _, err := oracle.Promote(bind.NewKeyedTransactor(keys[j]), crypto.PubkeyToAddress(keys[i].PublicKey)); err != nil {
				t.Fatalf("Iter #%d: failed valid promotion attempt: %v", i, err)
			}
		}
		sim.Commit()
	}
	// Make a release proposal and check it's existence
	if _, err := oracle.Release(bind.NewKeyedTransactor(keys[0]), 1, 2, 3, [20]byte{4}); err != nil {
		t.Fatalf("Failed valid proposal attempt: %v", err)
	}
	sim.Commit()

	prop, err := oracle.ProposedVersion(nil)
	if err != nil {
		t.Fatalf("Failed to retrieve active proposal: %v", err)
	}
	if len(prop.Pass) != 1 {
		t.Fatalf("Proposal vote count mismatch: have %d, want 1", len(prop.Pass))
	}
	// Demote a signer and check release proposal deletion
	for i := 0; i <= len(keys)/2; i++ {
		if _, err := oracle.Demote(bind.NewKeyedTransactor(keys[i]), crypto.PubkeyToAddress(keys[len(keys)-1].PublicKey)); err != nil {
			t.Fatalf("Iter #%d: failed valid demotion attempt: %v", i, err)
		}
	}
	sim.Commit()

	prop, err = oracle.ProposedVersion(nil)
	if err != nil {
		t.Fatalf("Failed to retrieve active proposal: %v", err)
	}
	if len(prop.Pass) != 0 {
		t.Fatalf("Proposal vote count mismatch: have %d, want 0", len(prop.Pass))
	}
}

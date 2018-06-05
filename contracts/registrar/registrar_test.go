// Copyright 2018 The go-ethereum Authors
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

package registrar

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/registrar/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/light"
)

var (
	key, _    = crypto.GenerateKey()
	addr      = crypto.PubkeyToAddress(key.PublicKey)
	emptyHash = [32]byte{}

	trustedCheckpoint = light.TrustedCheckpoint{
		SectionIdx:    0,
		SectionHead:   common.HexToHash("14c8639dfc32812ed20839f5a11993cd59b22e5226cb2179640ba5c1f0c08f87"),
		ChtRoot:       common.HexToHash("cf92fd2a79464354e8dae4d589ae92acdf90a3a4f8f7d8a3ec5fb9c114ae81cd"),
		BloomTrieRoot: common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
	}
)

func TestAdminManagement(t *testing.T) {
	var (
		adminCandidate  = common.HexToAddress("0x123")
		adminCandidate2 = common.HexToAddress("0x456")
	)

	// Deploy registrar contract
	transactOpts := bind.NewKeyedTransactor(key)
	contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(1000000000)}})
	_, _, contract, err := contract.DeployContract(transactOpts, contractBackend, nil)
	if err != nil {
		t.Error("deploy registrar contract failed", err)
	}
	contractBackend.Commit()

	// Test AddAdmin function
	contract.AddAdmin(transactOpts, addr) // Contract should ignore the duplicate registration
	contract.AddAdmin(transactOpts, adminCandidate)
	contract.AddAdmin(transactOpts, adminCandidate2)
	contractBackend.Commit()
	adminList, err := contract.GetAllAdmin(nil)
	if err != nil {
		t.Error("fetch admin list failed", err)
	}
	if !reflect.DeepEqual(adminList, []common.Address{addr, adminCandidate, adminCandidate2}) {
		t.Error("expect the returned admin list contain 3 address")
	}

	// Test RemoveAdmin function (remove at the middle)
	contract.RemoveAdmin(transactOpts, adminCandidate)
	contractBackend.Commit()
	adminList, err = contract.GetAllAdmin(nil)
	if err != nil {
		t.Error("fetch admin list failed", err)
	}
	if !reflect.DeepEqual(adminList, []common.Address{addr, adminCandidate2}) {
		t.Error("expect the returned admin list contain 3 address")
	}

	// Test RemoveAdmin function (remove at the head)
	contract.RemoveAdmin(transactOpts, addr)
	contractBackend.Commit()
	adminList, err = contract.GetAllAdmin(nil)
	if err != nil {
		t.Error("fetch admin list failed", err)
	}
	if !reflect.DeepEqual(adminList, []common.Address{adminCandidate2}) {
		t.Error("expect the returned admin list contain 3 address")
	}

	// Test unauthorized operation
	contract.AddAdmin(transactOpts, adminCandidate)
	contractBackend.Commit()
	adminList, err = contract.GetAllAdmin(nil)
	if err != nil {
		t.Error("fetch admin list failed", err)
	}
	if !reflect.DeepEqual(adminList, []common.Address{adminCandidate2}) {
		t.Error("expect the returned admin list contain 3 address")
	}
}

func TestCheckpointRegister(t *testing.T) {
	// Deploy registrar contract
	transactOpts := bind.NewKeyedTransactor(key)
	contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(1000000000)}})
	_, _, contract, err := contract.DeployContract(transactOpts, contractBackend, []common.Address{addr})
	if err != nil {
		t.Error("deploy registrar contract failed", err)
	}
	contractBackend.Commit()

	// Register an unstable checkpoint
	contract.SetCheckpoint(transactOpts, big.NewInt(int64(trustedCheckpoint.SectionIdx)), trustedCheckpoint.SectionHead,
		trustedCheckpoint.ChtRoot, trustedCheckpoint.BloomTrieRoot)
	contractBackend.Commit()
	head, chtRoot, bloomTrieRoot, err := contract.GetCheckpoint(nil, big.NewInt(int64(trustedCheckpoint.SectionIdx)))
	if err != nil {
		t.Error("fetch checkpoint failed", err)
	}
	if head != emptyHash || chtRoot != emptyHash || bloomTrieRoot != emptyHash {
		t.Error("the unstable checkpoint is not allowed to be registered")
	}

	// Register a stable checkpoint
	contractBackend.ShiftBlocks(sectionSize + checkpointConfirmation)
	contract.SetCheckpoint(transactOpts, big.NewInt(int64(trustedCheckpoint.SectionIdx)), trustedCheckpoint.SectionHead,
		trustedCheckpoint.ChtRoot, trustedCheckpoint.BloomTrieRoot)
	contractBackend.Commit()
	head, chtRoot, bloomTrieRoot, err = contract.GetCheckpoint(nil, big.NewInt(int64(trustedCheckpoint.SectionIdx)))
	if err != nil {
		t.Error("fetch checkpoint failed", err)
	}
	if !reflect.DeepEqual(head[:], trustedCheckpoint.SectionHead.Bytes()) || !reflect.DeepEqual(chtRoot[:], trustedCheckpoint.ChtRoot.Bytes()) ||
		!reflect.DeepEqual(bloomTrieRoot[:], trustedCheckpoint.BloomTrieRoot.Bytes()) {
		t.Error("expect the returned checkpoint should be same with the given one")
	}
}

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
	"errors"
	"math/big"
	"reflect"
	"testing"
	"time"

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

// validateOperation executes the operation, watches and delivers all events fired by the backend and ensures the
// correctness by assert function.
func validateOperation(t *testing.T, c *contract.Contract, backend *backends.SimulatedBackend, operation func(),
	assert func(<-chan *contract.ContractNewCheckpointEvent, <-chan *contract.ContractAddAdminEvent, <-chan *contract.ContractRemoveAdminEvent) error, opName string) {
	// Watch all events and deliver them to assert function
	var (
		sink1 = make(chan *contract.ContractNewCheckpointEvent)
		sink2 = make(chan *contract.ContractAddAdminEvent)
		sink3 = make(chan *contract.ContractRemoveAdminEvent)
	)
	sub1, _ := c.WatchNewCheckpointEvent(nil, sink1, nil)
	sub2, _ := c.WatchAddAdminEvent(nil, sink2)
	sub3, _ := c.WatchRemoveAdminEvent(nil, sink3)
	defer func() {
		// Close all subscribers
		sub1.Unsubscribe()
		sub2.Unsubscribe()
		sub3.Unsubscribe()
	}()
	operation()

	// flush pending block
	backend.Commit()
	if err := assert(sink1, sink2, sink3); err != nil {
		t.Errorf("operation {%s} failed, err %s", opName, err)
	}
}

// validateEvents checks that the correct number of contract events
// fired by contract backend.
func validateEvents(target int, sink interface{}) bool {
	chanval := reflect.ValueOf(sink)
	chantyp := chanval.Type()
	if chantyp.Kind() != reflect.Chan || chantyp.ChanDir()&reflect.RecvDir == 0 {
		return false
	}
	count := 0
	timeout := time.After(1 * time.Second)
	cases := []reflect.SelectCase{{Chan: chanval, Dir: reflect.SelectRecv}, {Chan: reflect.ValueOf(timeout), Dir: reflect.SelectRecv}}
	for {
		chose, _, _ := reflect.Select(cases)
		if chose == 1 {
			// Not enough event received
			return false
		}
		count += 1
		if count == target {
			break
		}
	}
	done := time.After(50 * time.Millisecond)
	cases = cases[:1]
	cases = append(cases, reflect.SelectCase{Chan: reflect.ValueOf(done), Dir: reflect.SelectRecv})
	chose, _, _ := reflect.Select(cases)
	// If chose equal 0, it means receiving redundant events.
	return chose == 1
}

// Tests contract administrator managements.
func TestAdminManagement(t *testing.T) {
	var (
		adminCandidate  = common.HexToAddress("0xdead")
		adminCandidate2 = common.HexToAddress("0xbeef")
	)

	// Deploy registrar contract
	transactOpts := bind.NewKeyedTransactor(key)
	contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(1000000000)}})
	_, _, c, err := contract.DeployContract(transactOpts, contractBackend, nil)
	if err != nil {
		t.Error("deploy registrar contract failed", err)
	}
	contractBackend.Commit()

	// Test AddAdmin function
	validateOperation(t, c, contractBackend, func() {
		for _, a := range []common.Address{addr, adminCandidate, adminCandidate2} {
			c.AddAdmin(transactOpts, a, "")
		}
	}, func(sink1 <-chan *contract.ContractNewCheckpointEvent, sink2 <-chan *contract.ContractAddAdminEvent, sink3 <-chan *contract.ContractRemoveAdminEvent) error {
		adminList, err := c.GetAllAdmin(nil)
		if err != nil {
			return errors.New("get admin list failed")
		}
		if !reflect.DeepEqual(adminList, []common.Address{addr, adminCandidate, adminCandidate2}) {
			return errors.New("add admin failed")
		}
		if !validateEvents(2, sink2) {
			return errors.New("receive incorrect number of events")
		}
		return nil
	}, "add admin")

	// Test Remove admin function
	validateOperation(t, c, contractBackend, func() {
		c.RemoveAdmin(transactOpts, adminCandidate, "")
	}, func(events <-chan *contract.ContractNewCheckpointEvent, events2 <-chan *contract.ContractAddAdminEvent, events3 <-chan *contract.ContractRemoveAdminEvent) error {
		adminList, err := c.GetAllAdmin(nil)
		if err != nil {
			return errors.New("get admin list failed")
		}
		if !reflect.DeepEqual(adminList, []common.Address{addr, adminCandidate2}) {
			return errors.New("remove admin failed")
		}
		if !validateEvents(1, events3) {
			return errors.New("receive incorrect number of events")
		}
		return nil
	}, "remove admin at middle")

	// Test RemoveAdmin function (remove at the head)
	validateOperation(t, c, contractBackend, func() {
		c.RemoveAdmin(transactOpts, addr, "")
	}, func(events <-chan *contract.ContractNewCheckpointEvent, events2 <-chan *contract.ContractAddAdminEvent, events3 <-chan *contract.ContractRemoveAdminEvent) error {
		adminList, err := c.GetAllAdmin(nil)
		if err != nil {
			return errors.New("get admin list failed")
		}
		if !reflect.DeepEqual(adminList, []common.Address{adminCandidate2}) {
			return errors.New("remove admin failed")
		}
		if !validateEvents(1, events3) {
			return errors.New("receive incorrect number of events")
		}
		return nil
	}, "remove admin at head")

	// Test unauthorized operation
	validateOperation(t, c, contractBackend, func() {
		c.AddAdmin(transactOpts, adminCandidate, "")
	}, func(events <-chan *contract.ContractNewCheckpointEvent, events2 <-chan *contract.ContractAddAdminEvent, events3 <-chan *contract.ContractRemoveAdminEvent) error {
		adminList, err := c.GetAllAdmin(nil)
		if err != nil {
			return errors.New("get admin list failed")
		}
		if !reflect.DeepEqual(adminList, []common.Address{adminCandidate2}) {
			return errors.New("unauthorized operation should be banned")
		}
		return nil
	}, "unauthorized operation")
}

// Tests checkpoint managements.
func TestCheckpointRegister(t *testing.T) {
	// Deploy registrar contract
	transactOpts := bind.NewKeyedTransactor(key)
	contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(1000000000)}})
	_, _, c, err := contract.DeployContract(transactOpts, contractBackend, []common.Address{addr})
	if err != nil {
		t.Error("deploy registrar contract failed", err)
	}
	contractBackend.Commit()

	// Register unstable checkpoint
	validateOperation(t, c, contractBackend, func() {
		c.SetCheckpoint(transactOpts, big.NewInt(int64(trustedCheckpoint.SectionIdx)), trustedCheckpoint.SectionHead,
			trustedCheckpoint.ChtRoot, trustedCheckpoint.BloomTrieRoot)
	}, func(events <-chan *contract.ContractNewCheckpointEvent, events2 <-chan *contract.ContractAddAdminEvent, events3 <-chan *contract.ContractRemoveAdminEvent) error {
		hash, err := c.GetCheckpoint(nil, big.NewInt(int64(trustedCheckpoint.SectionIdx)))
		if err != nil {
			return errors.New("get checkpoint failed")
		}
		if hash != emptyHash {
			return errors.New("unstable checkpoint should be banned")
		}
		return nil
	}, "register unstable checkpoint")

	// Register a stable checkpoint
	validateOperation(t, c, contractBackend, func() {
		contractBackend.ShiftBlocks(sectionSize + checkpointConfirmation)
		c.SetCheckpoint(transactOpts, big.NewInt(int64(trustedCheckpoint.SectionIdx)), trustedCheckpoint.SectionHead,
			trustedCheckpoint.ChtRoot, trustedCheckpoint.BloomTrieRoot)
	}, func(events <-chan *contract.ContractNewCheckpointEvent, events2 <-chan *contract.ContractAddAdminEvent, events3 <-chan *contract.ContractRemoveAdminEvent) error {
		hash, err := c.GetCheckpoint(nil, big.NewInt(int64(trustedCheckpoint.SectionIdx)))
		if err != nil {
			return errors.New("get checkpoint failed")
		}
		if common.Hash(hash).Hex() != crypto.Keccak256Hash(trustedCheckpoint.SectionHead.Bytes(), trustedCheckpoint.ChtRoot.Bytes(), trustedCheckpoint.BloomTrieRoot.Bytes()).Hex() {
			return errors.New("register stable checkpoint failed")
		}
		if !validateEvents(1, events) {
			return errors.New("receive incorrect number of events")
		}
		return nil
	}, "register stable checkpoint")

	// Modify the latest checkpoint
	validateOperation(t, c, contractBackend, func() {
		trustedCheckpoint.SectionHead = common.HexToHash("dead")
		c.SetCheckpoint(transactOpts, big.NewInt(int64(trustedCheckpoint.SectionIdx)), trustedCheckpoint.SectionHead,
			trustedCheckpoint.ChtRoot, trustedCheckpoint.BloomTrieRoot)
	}, func(events <-chan *contract.ContractNewCheckpointEvent, events2 <-chan *contract.ContractAddAdminEvent, events3 <-chan *contract.ContractRemoveAdminEvent) error {
		hash, err := c.GetCheckpoint(nil, big.NewInt(int64(trustedCheckpoint.SectionIdx)))
		if err != nil {
			return errors.New("get checkpoint failed")
		}
		if common.Hash(hash).Hex() != crypto.Keccak256Hash(trustedCheckpoint.SectionHead.Bytes(), trustedCheckpoint.ChtRoot.Bytes(), trustedCheckpoint.BloomTrieRoot.Bytes()).Hex() {
			return errors.New("register stable checkpoint failed")
		}
		if !validateEvents(1, events) {
			return errors.New("receive incorrect number of events")
		}
		return nil
	}, "modify latest checkpoint")
}

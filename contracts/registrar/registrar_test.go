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
	"crypto/ecdsa"
	"errors"
	"fmt"
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

const signatureLen = 65

var (
	emptyHash = [32]byte{}

	checkpoint0 = light.TrustedCheckpoint{
		SectionIndex: 0,
		SectionHead:  common.HexToHash("0x7fa3c32f996c2bfb41a1a65b3d8ea3e0a33a1674cde43678ad6f4235e764d17d"),
		CHTRoot:      common.HexToHash("0x98fc5d3de23a0fecebad236f6655533c157d26a1aedcd0852a514dc1169e6350"),
		BloomRoot:    common.HexToHash("0x99b5adb52b337fe25e74c1c6d3835b896bd638611b3aebddb2317cce27a3f9fa"),
	}
	checkpoint1 = light.TrustedCheckpoint{
		SectionIndex: 1,
		SectionHead:  common.HexToHash("0x2d4dee68102125e59b0cc61b176bd89f0d12b3b91cfaf52ef8c2c82fb920c2d2"),
		CHTRoot:      common.HexToHash("0x7d428008ece3b4c4ef5439f071930aad0bb75108d381308df73beadcd01ded95"),
		BloomRoot:    common.HexToHash("0x652571f7736de17e7bbb427ac881474da684c6988a88bf51b10cca9a2ee148f4"),
	}
	checkpoint2 = light.TrustedCheckpoint{
		SectionIndex: 2,
		SectionHead:  common.HexToHash("0x61c0de578c0115b1dff8ef39aa600588c7c6ecb8a2f102003d7cf4c4146e9291"),
		CHTRoot:      common.HexToHash("0x407a08a407a2bc3838b74ca3eb206903c9c8a186ccf5ef14af07794efff1970b"),
		BloomRoot:    common.HexToHash("0x058b4161f558ce295a92925efc57f34f9210d5a30088d7475c183e0d3e58f5ac"),
	}
)

var (
	// The block frequency for creating checkpoint(only used in test)
	sectionSize = big.NewInt(512)

	// The number of confirmations needed to generate a checkpoint(only used in test).
	processConfirms = big.NewInt(4)
)

// validateOperation executes the operation, watches and delivers all events fired by the backend and ensures the
// correctness by assert function.
func validateOperation(t *testing.T, c *contract.Contract, backend *backends.SimulatedBackend, operation func(),
	assert func(<-chan *contract.ContractNewCheckpointEvent) error, opName string) {
	// Watch all events and deliver them to assert function
	var (
		sink   = make(chan *contract.ContractNewCheckpointEvent)
		sub, _ = c.WatchNewCheckpointEvent(nil, sink, nil)
	)
	defer func() {
		// Close all subscribers
		sub.Unsubscribe()
	}()
	operation()

	// flush pending block
	backend.Commit()
	if err := assert(sink); err != nil {
		t.Errorf("operation {%s} failed, err %s", opName, err)
	}
}

// validateEvents checks that the correct number of contract events
// fired by contract backend.
func validateEvents(target int, sink interface{}) (bool, []reflect.Value) {
	chanval := reflect.ValueOf(sink)
	chantyp := chanval.Type()
	if chantyp.Kind() != reflect.Chan || chantyp.ChanDir()&reflect.RecvDir == 0 {
		return false, nil
	}
	count := 0
	var recv []reflect.Value
	timeout := time.After(1 * time.Second)
	cases := []reflect.SelectCase{{Chan: chanval, Dir: reflect.SelectRecv}, {Chan: reflect.ValueOf(timeout), Dir: reflect.SelectRecv}}
	for {
		chose, v, _ := reflect.Select(cases)
		if chose == 1 {
			// Not enough event received
			return false, nil
		}
		count += 1
		recv = append(recv, v)
		if count == target {
			break
		}
	}
	done := time.After(50 * time.Millisecond)
	cases = cases[:1]
	cases = append(cases, reflect.SelectCase{Chan: reflect.ValueOf(done), Dir: reflect.SelectRecv})
	chose, _, _ := reflect.Select(cases)
	// If chose equal 0, it means receiving redundant events.
	return chose == 1, recv
}

func signCheckpoint(privateKey *ecdsa.PrivateKey, hash common.Hash) []byte {
	sig, _ := crypto.Sign(hash.Bytes(), privateKey)
	return sig
}

// assertSignature verifies whether the recovered signers are equal with expected.
func assertSignature(hash [32]byte, sigs []byte, expectSigners []common.Address) bool {
	if len(sigs) != signatureLen*len(expectSigners) {
		return false
	}

	signerMap := make(map[common.Address]struct{})
	for i := 0; i < len(expectSigners); i += 1 {
		pubkey, err := crypto.Ecrecover(hash[:], sigs[i*signatureLen:(i+1)*signatureLen])
		if err != nil {
			return false
		}
		var signer common.Address
		copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])
		signerMap[signer] = struct{}{}
	}
	for i := 0; i < len(expectSigners); i += 1 {
		if _, exist := signerMap[expectSigners[i]]; !exist {
			return false
		}
	}
	return true
}

// Tests checkpoint managements.
func TestCheckpointRegister(t *testing.T) {
	// Initialize test accounts
	type Account struct {
		key  *ecdsa.PrivateKey
		addr common.Address
	}
	var accounts []Account
	for i := 0; i < 3; i++ {
		key, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(key.PublicKey)
		accounts = append(accounts, Account{key: key, addr: addr})
	}

	// Deploy registrar contract
	transactOpts := bind.NewKeyedTransactor(accounts[0].key)
	contractBackend := backends.NewSimulatedBackend(nil, core.GenesisAlloc{accounts[0].addr: {Balance: big.NewInt(1000000000)}, accounts[1].addr: {Balance: big.NewInt(1000000000)}, accounts[2].addr: {Balance: big.NewInt(1000000000)}}, 10000000)
	// 3 trusted signers, threshold 2
	_, _, c, err := contract.DeployContract(transactOpts, contractBackend, []common.Address{accounts[0].addr, accounts[1].addr, accounts[2].addr}, sectionSize, processConfirms, big.NewInt(2))
	if err != nil {
		t.Error("deploy registrar contract failed", err)
	}
	contractBackend.Commit()

	// Register unstable checkpoint
	validateOperation(t, c, contractBackend, func() {
		c.SetCheckpoint(transactOpts, big.NewInt(0), checkpoint0.Hash(), signCheckpoint(accounts[0].key, checkpoint0.Hash()))
	}, func(events <-chan *contract.ContractNewCheckpointEvent) error {
		hash, _, err := c.GetCheckpoint(nil, big.NewInt(0))
		if err != nil {
			return errors.New("get checkpoint failed")
		}
		if hash != emptyHash {
			return errors.New("unstable checkpoint should be rejected")
		}
		return nil
	}, "register unstable checkpoint")

	contractBackend.InsertEmptyBlocks(int(sectionSize.Uint64() + processConfirms.Uint64()))

	// Register by unauthorized user
	validateOperation(t, c, contractBackend, func() {
		u, _ := crypto.GenerateKey()
		unauthorized := bind.NewKeyedTransactor(u)
		c.SetCheckpoint(unauthorized, big.NewInt(0), checkpoint0.Hash(), signCheckpoint(u, checkpoint0.Hash()))
	}, func(events <-chan *contract.ContractNewCheckpointEvent) error {
		hash, _, err := c.GetCheckpoint(nil, big.NewInt(0))
		if err != nil {
			return errors.New("get checkpoint failed")
		}
		if hash != emptyHash {
			return errors.New("checkpoint from unauthorized user should be rejected")
		}
		return nil
	}, "register by unauthorized user")

	// Submit a new checkpoint announcement
	validateOperation(t, c, contractBackend, func() {
		c.SetCheckpoint(transactOpts, big.NewInt(0), checkpoint0.Hash(), signCheckpoint(accounts[0].key, checkpoint0.Hash()))
	}, func(events <-chan *contract.ContractNewCheckpointEvent) error {
		return assert(c, 0, []common.Address{accounts[0].addr}, []common.Hash{checkpoint0.Hash()})
	}, "single checkpoint announcement")

	// Submit a duplicate checkpoint announcement
	validateOperation(t, c, contractBackend, func() {
		c.SetCheckpoint(transactOpts, big.NewInt(0), checkpoint0.Hash(), signCheckpoint(accounts[0].key, checkpoint0.Hash()))
	}, func(events <-chan *contract.ContractNewCheckpointEvent) error {
		return assert(c, 0, []common.Address{accounts[0].addr}, []common.Hash{checkpoint0.Hash()})
	}, "duplicate checkpoint announcement")

	// Modification
	validateOperation(t, c, contractBackend, func() {
		c.SetCheckpoint(transactOpts, big.NewInt(0), common.HexToHash("deadbeef"), signCheckpoint(accounts[0].key, common.HexToHash("deadbeef")))
	}, func(events <-chan *contract.ContractNewCheckpointEvent) error {
		return assert(c, 0, []common.Address{accounts[0].addr}, []common.Hash{common.HexToHash("deadbeef")})
	}, "checkpoint modification")

	// Modification
	validateOperation(t, c, contractBackend, func() {
		c.SetCheckpoint(transactOpts, big.NewInt(0), common.HexToHash("deadbeef2"), signCheckpoint(accounts[0].key, common.HexToHash("deadbeef2")))
	}, func(events <-chan *contract.ContractNewCheckpointEvent) error {
		return assert(c, 0, []common.Address{accounts[0].addr}, []common.Hash{common.HexToHash("deadbeef2")})
	}, "checkpoint modification")

	// Another correct checkpoint announcement
	validateOperation(t, c, contractBackend, func() {
		c.SetCheckpoint(bind.NewKeyedTransactor(accounts[1].key), big.NewInt(0), checkpoint0.Hash(), signCheckpoint(accounts[1].key, checkpoint0.Hash()))
	}, func(events <-chan *contract.ContractNewCheckpointEvent) error {
		return assert(c, 0, []common.Address{accounts[0].addr, accounts[1].addr}, []common.Hash{common.HexToHash("deadbeef2"), checkpoint0.Hash()})
	}, "another checkpoint announcement")

	// enough checkpoint announcement
	validateOperation(t, c, contractBackend, func() {
		c.SetCheckpoint(bind.NewKeyedTransactor(accounts[2].key), big.NewInt(0), checkpoint0.Hash(), signCheckpoint(accounts[2].key, checkpoint0.Hash()))
	}, func(events <-chan *contract.ContractNewCheckpointEvent) error {
		if valid, recv := validateEvents(1, events); !valid {
			return errors.New("receive incorrect number of events")
		} else {
			event := recv[0].Interface().(*contract.ContractNewCheckpointEvent)
			if !assertSignature(event.CheckpointHash, event.Signature, []common.Address{accounts[1].addr, accounts[2].addr}) {
				return errors.New("recover signer failed")
			}
		}
		index, hash, height, err := c.GetLatestCheckpoint(nil)
		if err != nil || index.Uint64() != 0 || hash != checkpoint0.Hash() ||
			height.Uint64() != contractBackend.Blockchain().CurrentHeader().Number.Uint64() {
			return errors.New("stable checkpoint mismatch")
		}
		return assert(c, 0, nil, nil)
	}, "enough checkpoint announcement")

	// submit a stale checkpoint announcement
	validateOperation(t, c, contractBackend, func() {
		c.SetCheckpoint(transactOpts, big.NewInt(0), checkpoint0.Hash(), signCheckpoint(accounts[0].key, checkpoint0.Hash()))
	}, func(events <-chan *contract.ContractNewCheckpointEvent) error {
		return assert(c, 0, nil, nil)
	}, "submit stale checkpoint announcement")

	// submit a future checkpoint announcement
	validateOperation(t, c, contractBackend, func() {
		c.SetCheckpoint(transactOpts, big.NewInt(1), checkpoint1.Hash(), signCheckpoint(accounts[0].key, checkpoint1.Hash()))
	}, func(events <-chan *contract.ContractNewCheckpointEvent) error {
		return assert(c, 0, nil, nil)
	}, "submit future checkpoint announcement")

	// submit uncontinuous checkpoint announcement
	distance := 3*sectionSize.Uint64() + processConfirms.Uint64() - contractBackend.Blockchain().CurrentHeader().Number.Uint64()
	contractBackend.InsertEmptyBlocks(int(distance))
	validateOperation(t, c, contractBackend, func() {
		c.SetCheckpoint(transactOpts, big.NewInt(2), checkpoint2.Hash(), signCheckpoint(accounts[0].key, checkpoint2.Hash()))
		c.SetCheckpoint(bind.NewKeyedTransactor(accounts[1].key), big.NewInt(2), checkpoint2.Hash(), signCheckpoint(accounts[1].key, checkpoint2.Hash()))
	}, func(events <-chan *contract.ContractNewCheckpointEvent) error {
		if valid, recv := validateEvents(1, events); !valid {
			return errors.New("receive incorrect number of events")
		} else {
			event := recv[0].Interface().(*contract.ContractNewCheckpointEvent)
			if !assertSignature(event.CheckpointHash, event.Signature, []common.Address{accounts[0].addr, accounts[1].addr}) {
				return errors.New("recover signer failed")
			}
		}
		index, hash, height, err := c.GetLatestCheckpoint(nil)
		if err != nil || index.Uint64() != 2 || hash != checkpoint2.Hash() ||
			height.Uint64() != contractBackend.Blockchain().CurrentHeader().Number.Uint64() {
			return errors.New("stable checkpoint mismatch")
		}
		return assert(c, 0, nil, nil)
	}, "uncontinuous checkpoint announcement")
}

func assert(c *contract.Contract, index uint64, signers []common.Address, hashes []common.Hash) error {
	pi, pSigners, pHashes, err := c.GetPending(nil)
	if err != nil {
		return errors.New("get pending proposal failed")
	}
	if pi.Uint64() != index {
		return fmt.Errorf("pending checkpoint index mismatch, want=%d, got=%d", index, pi.Uint64())
	}
	// Assert signers
	if len(pSigners) != len(signers) {
		return fmt.Errorf("pending checkpoint signers number mismatch, want=%d, got=%d", len(signers), len(pSigners))
	}
	for _, a := range signers {
		found := false
		for _, b := range pSigners {
			if a == b {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("signer %s not found", a.Hex())
		}
	}
	// Assert hashes
	if len(hashes) != len(pHashes) {
		return fmt.Errorf("pending checkpoint hash number mismatch, want=%d, got=%d", len(hashes), len(pHashes))
	}
	for _, a := range hashes {
		found := false
		for _, b := range pHashes {
			if a == b {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("hash %s not found", a.Hex())
		}
	}
	return nil
}

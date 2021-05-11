// Copyright 2019 The go-ethereum Authors
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

package checkpointoracle

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"math/big"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/checkpointoracle/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

var (
	emptyHash = [32]byte{}

	checkpoint0 = params.TrustedCheckpoint{
		SectionIndex: 0,
		SectionHead:  common.HexToHash("0x7fa3c32f996c2bfb41a1a65b3d8ea3e0a33a1674cde43678ad6f4235e764d17d"),
		CHTRoot:      common.HexToHash("0x98fc5d3de23a0fecebad236f6655533c157d26a1aedcd0852a514dc1169e6350"),
		BloomRoot:    common.HexToHash("0x99b5adb52b337fe25e74c1c6d3835b896bd638611b3aebddb2317cce27a3f9fa"),
	}
	checkpoint1 = params.TrustedCheckpoint{
		SectionIndex: 1,
		SectionHead:  common.HexToHash("0x2d4dee68102125e59b0cc61b176bd89f0d12b3b91cfaf52ef8c2c82fb920c2d2"),
		CHTRoot:      common.HexToHash("0x7d428008ece3b4c4ef5439f071930aad0bb75108d381308df73beadcd01ded95"),
		BloomRoot:    common.HexToHash("0x652571f7736de17e7bbb427ac881474da684c6988a88bf51b10cca9a2ee148f4"),
	}
	checkpoint2 = params.TrustedCheckpoint{
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
func validateOperation(t *testing.T, c *contract.CheckpointOracle, backend *backends.SimulatedBackend, operation func(),
	assert func(<-chan *contract.CheckpointOracleNewCheckpointVote) error, opName string) {
	// Watch all events and deliver them to assert function
	var (
		sink   = make(chan *contract.CheckpointOracleNewCheckpointVote)
		sub, _ = c.WatchNewCheckpointVote(nil, sink, nil)
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

func signCheckpoint(addr common.Address, privateKey *ecdsa.PrivateKey, index uint64, hash common.Hash) []byte {
	// EIP 191 style signatures
	//
	// Arguments when calculating hash to validate
	// 1: byte(0x19) - the initial 0x19 byte
	// 2: byte(0) - the version byte (data with intended validator)
	// 3: this - the validator address
	// --  Application specific data
	// 4 : checkpoint section_index(uint64)
	// 5 : checkpoint hash (bytes32)
	//     hash = keccak256(checkpoint_index, section_head, cht_root, bloom_root)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, index)
	data := append([]byte{0x19, 0x00}, append(addr.Bytes(), append(buf, hash.Bytes()...)...)...)
	sig, _ := crypto.Sign(crypto.Keccak256(data), privateKey)
	sig[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	return sig
}

// assertSignature verifies whether the recovered signers are equal with expected.
func assertSignature(addr common.Address, index uint64, hash [32]byte, r, s [32]byte, v uint8, expect common.Address) bool {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, index)
	data := append([]byte{0x19, 0x00}, append(addr.Bytes(), append(buf, hash[:]...)...)...)
	pubkey, err := crypto.Ecrecover(crypto.Keccak256(data), append(r[:], append(s[:], v-27)...))
	if err != nil {
		return false
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])
	return bytes.Equal(signer.Bytes(), expect.Bytes())
}

type Account struct {
	key  *ecdsa.PrivateKey
	addr common.Address
}
type Accounts []Account

func (a Accounts) Len() int           { return len(a) }
func (a Accounts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Accounts) Less(i, j int) bool { return bytes.Compare(a[i].addr.Bytes(), a[j].addr.Bytes()) < 0 }

func TestCheckpointRegister(t *testing.T) {
	// Initialize test accounts
	var accounts Accounts
	for i := 0; i < 3; i++ {
		key, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(key.PublicKey)
		accounts = append(accounts, Account{key: key, addr: addr})
	}
	sort.Sort(accounts)

	// Deploy registrar contract
	contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{accounts[0].addr: {Balance: big.NewInt(1000000000)}, accounts[1].addr: {Balance: big.NewInt(1000000000)}, accounts[2].addr: {Balance: big.NewInt(1000000000)}}, 10000000)
	defer contractBackend.Close()

	transactOpts, _ := bind.NewKeyedTransactorWithChainID(accounts[0].key, big.NewInt(1337))

	// 3 trusted signers, threshold 2
	contractAddr, _, c, err := contract.DeployCheckpointOracle(transactOpts, contractBackend, []common.Address{accounts[0].addr, accounts[1].addr, accounts[2].addr}, sectionSize, processConfirms, big.NewInt(2))
	if err != nil {
		t.Error("Failed to deploy registrar contract", err)
	}
	contractBackend.Commit()

	// getRecent returns block height and hash of the head parent.
	getRecent := func() (*big.Int, common.Hash) {
		parentNumber := new(big.Int).Sub(contractBackend.Blockchain().CurrentHeader().Number, big.NewInt(1))
		parentHash := contractBackend.Blockchain().CurrentHeader().ParentHash
		return parentNumber, parentHash
	}
	// collectSig generates specified number signatures.
	collectSig := func(index uint64, hash common.Hash, n int, unauthorized *ecdsa.PrivateKey) (v []uint8, r [][32]byte, s [][32]byte) {
		for i := 0; i < n; i++ {
			sig := signCheckpoint(contractAddr, accounts[i].key, index, hash)
			if unauthorized != nil {
				sig = signCheckpoint(contractAddr, unauthorized, index, hash)
			}
			r = append(r, common.BytesToHash(sig[:32]))
			s = append(s, common.BytesToHash(sig[32:64]))
			v = append(v, sig[64])
		}
		return v, r, s
	}
	// insertEmptyBlocks inserts a batch of empty blocks to blockchain.
	insertEmptyBlocks := func(number int) {
		for i := 0; i < number; i++ {
			contractBackend.Commit()
		}
	}
	// assert checks whether the current contract status is same with
	// the expected.
	assert := func(index uint64, hash [32]byte, height *big.Int) error {
		lindex, lhash, lheight, err := c.GetLatestCheckpoint(nil)
		if err != nil {
			return err
		}
		if lindex != index {
			return errors.New("latest checkpoint index mismatch")
		}
		if !bytes.Equal(lhash[:], hash[:]) {
			return errors.New("latest checkpoint hash mismatch")
		}
		if lheight.Cmp(height) != 0 {
			return errors.New("latest checkpoint height mismatch")
		}
		return nil
	}

	// Test future checkpoint registration
	validateOperation(t, c, contractBackend, func() {
		number, hash := getRecent()
		v, r, s := collectSig(0, checkpoint0.Hash(), 2, nil)
		c.SetCheckpoint(transactOpts, number, hash, checkpoint0.Hash(), 0, v, r, s)
	}, func(events <-chan *contract.CheckpointOracleNewCheckpointVote) error {
		return assert(0, emptyHash, big.NewInt(0))
	}, "test future checkpoint registration")

	insertEmptyBlocks(int(sectionSize.Uint64() + processConfirms.Uint64()))

	// Test transaction replay protection
	validateOperation(t, c, contractBackend, func() {
		number, _ := getRecent()
		v, r, s := collectSig(0, checkpoint0.Hash(), 2, nil)
		hash := common.HexToHash("deadbeef")
		c.SetCheckpoint(transactOpts, number, hash, checkpoint0.Hash(), 0, v, r, s)
	}, func(events <-chan *contract.CheckpointOracleNewCheckpointVote) error {
		return assert(0, emptyHash, big.NewInt(0))
	}, "test transaction replay protection")

	// Test unauthorized signature checking
	validateOperation(t, c, contractBackend, func() {
		number, hash := getRecent()
		u, _ := crypto.GenerateKey()
		v, r, s := collectSig(0, checkpoint0.Hash(), 2, u)
		c.SetCheckpoint(transactOpts, number, hash, checkpoint0.Hash(), 0, v, r, s)
	}, func(events <-chan *contract.CheckpointOracleNewCheckpointVote) error {
		return assert(0, emptyHash, big.NewInt(0))
	}, "test unauthorized signature checking")

	// Test un-multi-signature checkpoint registration
	validateOperation(t, c, contractBackend, func() {
		number, hash := getRecent()
		v, r, s := collectSig(0, checkpoint0.Hash(), 1, nil)
		c.SetCheckpoint(transactOpts, number, hash, checkpoint0.Hash(), 0, v, r, s)
	}, func(events <-chan *contract.CheckpointOracleNewCheckpointVote) error {
		return assert(0, emptyHash, big.NewInt(0))
	}, "test un-multi-signature checkpoint registration")

	// Test valid checkpoint registration
	validateOperation(t, c, contractBackend, func() {
		number, hash := getRecent()
		v, r, s := collectSig(0, checkpoint0.Hash(), 2, nil)
		c.SetCheckpoint(transactOpts, number, hash, checkpoint0.Hash(), 0, v, r, s)
	}, func(events <-chan *contract.CheckpointOracleNewCheckpointVote) error {
		if valid, recv := validateEvents(2, events); !valid {
			return errors.New("receive incorrect number of events")
		} else {
			for i := 0; i < len(recv); i++ {
				event := recv[i].Interface().(*contract.CheckpointOracleNewCheckpointVote)
				if !assertSignature(contractAddr, event.Index, event.CheckpointHash, event.R, event.S, event.V, accounts[i].addr) {
					return errors.New("recover signer failed")
				}
			}
		}
		number, _ := getRecent()
		return assert(0, checkpoint0.Hash(), number.Add(number, big.NewInt(1)))
	}, "test valid checkpoint registration")

	distance := 3*sectionSize.Uint64() + processConfirms.Uint64() - contractBackend.Blockchain().CurrentHeader().Number.Uint64()
	insertEmptyBlocks(int(distance))

	// Test uncontinuous checkpoint registration
	validateOperation(t, c, contractBackend, func() {
		number, hash := getRecent()
		v, r, s := collectSig(2, checkpoint2.Hash(), 2, nil)
		c.SetCheckpoint(transactOpts, number, hash, checkpoint2.Hash(), 2, v, r, s)
	}, func(events <-chan *contract.CheckpointOracleNewCheckpointVote) error {
		if valid, recv := validateEvents(2, events); !valid {
			return errors.New("receive incorrect number of events")
		} else {
			for i := 0; i < len(recv); i++ {
				event := recv[i].Interface().(*contract.CheckpointOracleNewCheckpointVote)
				if !assertSignature(contractAddr, event.Index, event.CheckpointHash, event.R, event.S, event.V, accounts[i].addr) {
					return errors.New("recover signer failed")
				}
			}
		}
		number, _ := getRecent()
		return assert(2, checkpoint2.Hash(), number.Add(number, big.NewInt(1)))
	}, "test uncontinuous checkpoint registration")

	// Test old checkpoint registration
	validateOperation(t, c, contractBackend, func() {
		number, hash := getRecent()
		v, r, s := collectSig(1, checkpoint1.Hash(), 2, nil)
		c.SetCheckpoint(transactOpts, number, hash, checkpoint1.Hash(), 1, v, r, s)
	}, func(events <-chan *contract.CheckpointOracleNewCheckpointVote) error {
		number, _ := getRecent()
		return assert(2, checkpoint2.Hash(), number)
	}, "test uncontinuous checkpoint registration")

	// Test stale checkpoint registration
	validateOperation(t, c, contractBackend, func() {
		number, hash := getRecent()
		v, r, s := collectSig(2, checkpoint2.Hash(), 2, nil)
		c.SetCheckpoint(transactOpts, number, hash, checkpoint2.Hash(), 2, v, r, s)
	}, func(events <-chan *contract.CheckpointOracleNewCheckpointVote) error {
		number, _ := getRecent()
		return assert(2, checkpoint2.Hash(), number.Sub(number, big.NewInt(1)))
	}, "test stale checkpoint registration")
}

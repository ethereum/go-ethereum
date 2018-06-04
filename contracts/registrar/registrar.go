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

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/registrar/contract"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/internal/ethapi"
)

var (
	MainNetAddr = common.HexToAddress("")
	TestNetAddr = common.HexToAddress("0x3b934494985d17bcb49557671e1bc8ec32cccdd5") // Rinkeby
)

var errEventNotFound = errors.New("contract event not found")

const (
	sectionSize            = 32768 // The frequency for creating a checkpoint
	checkpointConfirmation = 10000 // The number of confirmations needed before a checkpoint becoming stable.
)

// Checkpoint represents a set of post-processed trie roots (CHT and BloomTrie) associated with
// the appropriate section index and head hash.
//
// It is used to start light syncing from this checkpoint and avoid downloading the entire header chain
// while still being able to securely access old headers/logs.
type Checkpoint struct {
	SectionIndex  uint64
	SectionHead   common.Hash // Block Hash for the last block in the section
	ChtRoot       common.Hash // CHT(Canonical Hash Trie) root associated to the section
	BloomTrieRoot common.Hash // Bloom Trie root associated to the section
}

type Registrar struct {
	contract *contract.Contract
}

// NewRegistrar binds checkpoint contract and returns a registrar instance.
func NewRegistrar(contractAddr common.Address, backend ethapi.Backend, lightMode bool) (*Registrar, error) {
	contract, err := contract.NewContract(contractAddr, eth.NewContractBackend(backend, lightMode))
	if err != nil {
		return nil, err
	}

	return &Registrar{
		contract: contract,
	}, nil
}

// WatchNewCheckpointEvent watches new fired NewCheckpointEvent and delivers all matching events by result channel.
func (registrar *Registrar) WatchNewCheckpointEvent(sink chan<- *contract.ContractNewCheckpointEvent) (event.Subscription, error) {
	return registrar.contract.WatchNewCheckpointEvent(nil, sink, nil)
}

// FilterNewCheckpointEvent filters out NewCheckpointEvent for specific section number.
func (registrar *Registrar) FilterNewCheckpointEvent(head uint64, section uint64) (*contract.ContractNewCheckpointEventIterator, error) {
	start := (section + 1) * sectionSize
	end := head - checkpointConfirmation
	if end < start {
		return nil, errEventNotFound
	}
	opt := &bind.FilterOpts{
		Start: start,
		End:   &end,
	}
	return registrar.contract.FilterNewCheckpointEvent(opt, []*big.Int{big.NewInt(int64(section))})
}

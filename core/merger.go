// Copyright 2021 The go-ethereum Authors
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
	"sync"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// transitionStatus describes the status of eth1/2 merging. This switch
// between modes is a one-way action which is triggered by corresponding
// consensus-layer message.
type transitionStatus struct {
	LeftPoW    bool // The flag is set when the first NewHead message received
	EnteredPoS bool // The flag is set when the first FinaliseBlock message received
}

// Merger is an internal help structure used to track the eth1/2 merging status.
// It's a common structure can be used in both full node and light client.
type Merger struct {
	db            ethdb.KeyValueStore
	status        transitionStatus
	leavePoWCalls []func()
	enterPoSCalls []func()
	lock          sync.Mutex
}

func NewMerger(db ethdb.KeyValueStore) *Merger {
	var status transitionStatus
	blob := rawdb.ReadTransitionStatus(db)
	if len(blob) != 0 {
		if err := rlp.DecodeBytes(blob, &status); err != nil {
			log.Crit("Failed to decode the transition status", "err", err)
		}
	}
	return &Merger{
		db:     db,
		status: status,
	}
}

// SubscribeLeavePoW registers callback so that if the chain leaves
// from the PoW stage and enters to 'transition' stage it can be invoked.
func (m *Merger) SubscribeLeavePoW(callback func()) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.leavePoWCalls = append(m.leavePoWCalls, callback)
}

// SubscribeEnterPoS registers callback so that if the chain leaves
// from the 'transition' stage and enters the PoS stage it can be invoked.
func (m *Merger) SubscribeEnterPoS(callback func()) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.enterPoSCalls = append(m.enterPoSCalls, callback)
}

// LeavePoW is called whenever the first NewHead message received
// from the consensus-layer.
func (m *Merger) LeavePoW() {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.status.LeftPoW {
		return
	}
	m.status = transitionStatus{LeftPoW: true}
	blob, err := rlp.EncodeToBytes(m.status)
	if err != nil {
		log.Crit("Failed to encode the transition status", "err", err)
	}
	rawdb.WriteTransitionStatus(m.db, blob)
	for _, call := range m.leavePoWCalls {
		call()
	}
}

// EnterPoS is called whenever the first FinalisedBlock message received
// from the consensus-layer.
func (m *Merger) EnterPoS() {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.status.EnteredPoS {
		return
	}
	m.status = transitionStatus{LeftPoW: true, EnteredPoS: true}
	blob, err := rlp.EncodeToBytes(m.status)
	if err != nil {
		log.Crit("Failed to encode the transition status", "err", err)
	}
	rawdb.WriteTransitionStatus(m.db, blob)
	for _, call := range m.enterPoSCalls {
		call()
	}
}

// LeftPoW reports whether the chain has left the PoW stage.
func (m *Merger) LeftPoW() bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.status.LeftPoW
}

// EnteredPoS reports whether the chain has entered the PoS stage.
func (m *Merger) EnteredPoS() bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.status.EnteredPoS
}

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
)

// Merger is an internal help structure used to tracking
// the eth1/2 merging status. It's a common structure can
// be used in both full node and light client.
type Merger struct {
	db            ethdb.KeyValueStore
	status        *rawdb.TransitionStatus
	leavePoWCalls []func()
	enterPoSCalls []func()
	lock          sync.Mutex
}

func NewMerger(db ethdb.KeyValueStore) *Merger {
	return &Merger{
		db:     db,
		status: rawdb.ReadTransitionStatus(db),
	}
}

// SubscribeLeavePoW registers callback so that if the chain transitions
// from the PoW stage to 'transition' stage it can be invoked.
func (m *Merger) SubscribeLeavePoW(callback func()) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.leavePoWCalls = append(m.leavePoWCalls, callback)
}

// SubscribeEnterPoS registers callback so that if the chain transitions
// from the 'transition' stage to the PoS stage it can be invoked.
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

	if m.status != nil && m.status.LeavedPoW {
		return
	}
	m.status = &rawdb.TransitionStatus{LeavedPoW: true}
	rawdb.WriteTransitionStatus(m.db, m.status)
	for _, call := range m.leavePoWCalls {
		call()
	}
}

// EnterPoS is called whenever the first FinalisedBlock message received
// from the consensus-layer.
func (m *Merger) EnterPoS() {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.status != nil && m.status.EnteredPoS {
		return
	}
	m.status = &rawdb.TransitionStatus{LeavedPoW: true, EnteredPoS: true}
	rawdb.WriteTransitionStatus(m.db, m.status)
	for _, call := range m.enterPoSCalls {
		call()
	}
}

// LeavedPoW reports whether the chain has leaved the PoW stage.
func (m *Merger) LeavedPoW() bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.status == nil {
		return false
	}
	return m.status.LeavedPoW
}

// EnteredPoS reports whether the chain has entered the PoS stage.
func (m *Merger) EnteredPoS() bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.status == nil {
		return false
	}
	return m.status.EnteredPoS
}

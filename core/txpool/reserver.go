// Copyright 2025 The go-ethereum Authors
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

package txpool

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	// reservationsGaugeName is the prefix of a per-subpool address reservation
	// metric.
	//
	// This is mostly a sanity metric to ensure there's no bug that would make
	// some subpool hog all the reservations due to mis-accounting.
	reservationsGaugeName = "txpool/reservations"
)

// ReservationTracker is a struct shared between different subpools. It is used to reserve
// the account and ensure that one address cannot initiate transactions, authorizations,
// and other state-changing behaviors in different pools at the same time.
type ReservationTracker struct {
	accounts map[common.Address]int
	lock     sync.RWMutex
}

// NewReservationTracker initializes the account reservation tracker.
func NewReservationTracker() *ReservationTracker {
	return &ReservationTracker{
		accounts: make(map[common.Address]int),
	}
}

// NewHandle creates a named handle on the ReservationTracker. The handle
// identifies the subpool so ownership of reservations can be determined.
func (r *ReservationTracker) NewHandle(id int) *ReservationHandle {
	return &ReservationHandle{r, id}
}

// Reserver is an interface for creating and releasing owned reservations in the
// ReservationTracker struct, which is shared between subpools.
type Reserver interface {
	// Hold attempts to reserve the specified account address for the given pool.
	// Returns an error if the account is already reserved.
	Hold(addr common.Address) error

	// Release attempts to release the reservation for the specified account.
	// Returns an error if the address is not reserved or is reserved by another pool.
	Release(addr common.Address) error

	// Has returns a flag indicating if the address has been reserved by a pool
	// other than one with the current Reserver handle.
	Has(address common.Address) bool
}

// ReservationHandle is a named handle on ReservationTracker. It is held by subpools to
// make reservations for accounts it is tracking. The id is used to determine
// which pool owns an address and disallows non-owners to hold or release
// addresses it doesn't own.
type ReservationHandle struct {
	tracker *ReservationTracker
	id      int
}

// Hold implements the Reserver interface.
func (h *ReservationHandle) Hold(addr common.Address) error {
	h.tracker.lock.Lock()
	defer h.tracker.lock.Unlock()

	// Double reservations are forbidden even from the same pool to
	// avoid subtle bugs in the long term.
	owner, exists := h.tracker.accounts[addr]
	if exists {
		if owner == h.id {
			log.Error("pool attempted to reserve already-owned address", "address", addr)
			return nil // Ignore fault to give the pool a chance to recover while the bug gets fixed
		}
		return ErrAlreadyReserved
	}
	h.tracker.accounts[addr] = h.id
	if metrics.Enabled() {
		m := fmt.Sprintf("%s/%d", reservationsGaugeName, h.id)
		metrics.GetOrRegisterGauge(m, nil).Inc(1)
	}
	return nil
}

// Release implements the Reserver interface.
func (h *ReservationHandle) Release(addr common.Address) error {
	h.tracker.lock.Lock()
	defer h.tracker.lock.Unlock()

	// Ensure subpools only attempt to unreserve their own owned addresses,
	// otherwise flag as a programming error.
	owner, exists := h.tracker.accounts[addr]
	if !exists {
		log.Error("pool attempted to unreserve non-reserved address", "address", addr)
		return errors.New("address not reserved")
	}
	if owner != h.id {
		log.Error("pool attempted to unreserve non-owned address", "address", addr)
		return errors.New("address not owned")
	}
	delete(h.tracker.accounts, addr)
	if metrics.Enabled() {
		m := fmt.Sprintf("%s/%d", reservationsGaugeName, h.id)
		metrics.GetOrRegisterGauge(m, nil).Dec(1)
	}
	return nil
}

// Has implements the Reserver interface.
func (h *ReservationHandle) Has(address common.Address) bool {
	h.tracker.lock.RLock()
	defer h.tracker.lock.RUnlock()

	id, exists := h.tracker.accounts[address]
	return exists && id != h.id
}

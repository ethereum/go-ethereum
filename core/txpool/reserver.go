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

// Reserver is a struct shared between different subpools. It is used to reserve
// the account and ensure that one address cannot initiate transactions, authorizations,
// and other state-changing behaviors in different pools at the same time.
type Reserver struct {
	accounts map[common.Address]string
	lock     sync.RWMutex
}

// NewReserver initializes the account reserver.
func NewReserver() *Reserver {
	return &Reserver{
		accounts: make(map[common.Address]string),
	}
}

// Hold attempts to reserve the specified account address for the given pool.
// Returns an error if the account is already reserved.
func (r *Reserver) Hold(addr common.Address, id string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	// Double reservations are forbidden even from the same pool to
	// avoid subtle bugs in the long term.
	owner, exists := r.accounts[addr]
	if exists {
		if owner == id {
			log.Error("pool attempted to reserve already-owned address", "address", addr)
			return nil // Ignore fault to give the pool a chance to recover while the bug gets fixed
		}
		return ErrAlreadyReserved
	}
	r.accounts[addr] = id
	if metrics.Enabled() {
		m := fmt.Sprintf("%s/%s", reservationsGaugeName, id)
		metrics.GetOrRegisterGauge(m, nil).Inc(1)
	}
	return nil
}

// Release attempts to release the reservation for the specified account.
// Returns an error if the address is not reserved or is reserved by another pool.
func (r *Reserver) Release(addr common.Address, id string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	// Ensure subpools only attempt to unreserve their own owned addresses,
	// otherwise flag as a programming error.
	owner, exists := r.accounts[addr]
	if !exists {
		log.Error("pool attempted to unreserve non-reserved address", "address", addr)
		return errors.New("address not reserved")
	}
	if id != owner {
		log.Error("pool attempted to unreserve non-owned address", "address", addr)
		return errors.New("address not owned")
	}
	delete(r.accounts, addr)
	if metrics.Enabled() {
		m := fmt.Sprintf("%s/%s", reservationsGaugeName, id)
		metrics.GetOrRegisterGauge(m, nil).Dec(1)
	}
	return nil
}

// Has returns a flag indicating if the address has been reserved or not.
func (r *Reserver) Has(address common.Address) bool {
	r.lock.RLock()
	defer r.lock.RUnlock()

	_, exists := r.accounts[address]
	return exists
}

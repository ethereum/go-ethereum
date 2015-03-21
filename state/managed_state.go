package state

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type account struct {
	stateObject *StateObject
	nstart      uint64
	nonces      []bool
}

type ManagedState struct {
	*StateDB

	mu sync.RWMutex

	accounts map[string]*account
}

func ManageState(statedb *StateDB) *ManagedState {
	return &ManagedState{
		StateDB:  statedb,
		accounts: make(map[string]*account),
	}
}

func (ms *ManagedState) SetState(statedb *StateDB) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.StateDB = statedb
}

func (ms *ManagedState) RemoveNonce(addr common.Address, n uint64) {
	if ms.hasAccount(addr) {
		ms.mu.Lock()
		defer ms.mu.Unlock()

		account := ms.getAccount(addr)
		if n-account.nstart <= uint64(len(account.nonces)) {
			reslice := make([]bool, n-account.nstart)
			copy(reslice, account.nonces[:n-account.nstart])
			account.nonces = reslice
		}
	}
}

func (ms *ManagedState) NewNonce(addr common.Address) uint64 {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	account := ms.getAccount(addr)
	for i, nonce := range account.nonces {
		if !nonce {
			return account.nstart + uint64(i)
		}
	}
	account.nonces = append(account.nonces, true)
	return uint64(len(account.nonces)) + account.nstart
}

func (ms *ManagedState) hasAccount(addr common.Address) bool {
	_, ok := ms.accounts[addr.Str()]
	return ok
}

func (ms *ManagedState) getAccount(addr common.Address) *account {
	straddr := addr.Str()
	if account, ok := ms.accounts[straddr]; !ok {
		so := ms.GetOrNewStateObject(addr)
		ms.accounts[straddr] = newAccount(so)
	} else {
		// Always make sure the state account nonce isn't actually higher
		// than the tracked one.
		so := ms.StateDB.GetStateObject(addr)
		if so != nil && uint64(len(account.nonces))+account.nstart < so.nonce {
			ms.accounts[straddr] = newAccount(so)
		}

	}

	return ms.accounts[straddr]
}

func newAccount(so *StateObject) *account {
	return &account{so, so.nonce - 1, nil}
}

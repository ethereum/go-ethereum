package state

import "sync"

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

func (ms *ManagedState) RemoveNonce(addr []byte, n uint64) {
	if ms.hasAccount(addr) {
		ms.mu.Lock()
		defer ms.mu.Unlock()

		account := ms.getAccount(addr)
		if n-account.nstart < uint64(len(account.nonces)) {
			reslice := make([]bool, n-account.nstart)
			copy(reslice, account.nonces[:n-account.nstart])
			account.nonces = reslice
		}
	}
}

func (ms *ManagedState) NewNonce(addr []byte) uint64 {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	account := ms.getAccount(addr)
	for i, nonce := range account.nonces {
		if !nonce {
			return account.nstart + uint64(i)
		}
	}
	account.nonces = append(account.nonces, false)
	return uint64(len(account.nonces)) + account.nstart
}

func (ms *ManagedState) hasAccount(addr []byte) bool {
	_, ok := ms.accounts[string(addr)]
	return ok
}

func (ms *ManagedState) getAccount(addr []byte) *account {
	if _, ok := ms.accounts[string(addr)]; !ok {
		so := ms.GetOrNewStateObject(addr)
		ms.accounts[string(addr)] = newAccount(so)
	}

	return ms.accounts[string(addr)]
}

func newAccount(so *StateObject) *account {
	return &account{so, so.nonce - 1, nil}
}

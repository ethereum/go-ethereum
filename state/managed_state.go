package state

import "sync"

type ManagedState struct {
	*StateDB

	mu sync.RWMutex

	accounts map[string]*StateObject
}

func ManageState(statedb *StateDB) *ManagedState {
	return &ManagedState{
		StateDB:  statedb,
		accounts: make(map[string]*StateObject),
	}
}

func (ms *ManagedState) IncrementNonce(addr []byte) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.getAccount(addr).nonce++
}

func (ms *ManagedState) DecrementNonce(addr []byte) {
	// Decrementing a nonce does not mean we are interested in the account
	// incrementing only happens if you control the account, therefor
	// incrementing  behaves differently from decrementing
	if ms.hasAccount(addr) {
		ms.mu.Lock()
		defer ms.mu.Unlock()

		ms.getAccount(addr).nonce--
	}
}

func (ms *ManagedState) GetNonce(addr []byte) uint64 {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.getAccount(addr).nonce
}

func (ms *ManagedState) hasAccount(addr []byte) bool {
	_, ok := ms.accounts[string(addr)]
	return ok
}

func (ms *ManagedState) getAccount(addr []byte) *StateObject {
	if _, ok := ms.accounts[string(addr)]; !ok {
		ms.accounts[string(addr)] = ms.GetOrNewStateObject(addr)
	}

	return ms.accounts[string(addr)]
}

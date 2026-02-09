package state

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
	"golang.org/x/sync/errgroup"
	"maps"
	"sync"
)

// TODO: probably unnecessary to cache the resolved state object here as it will already be in the db cache?
// ^ experiment with the performance of keeping this as-is vs just using the db cache.

// prestateResolver asynchronously fetches the prestate state accounts of addresses
// which are reported as modified in EIP-7928 access lists in order to produce the full
// updated state account (including fields that weren't modified in the BAL) for the
// state root update
type prestateResolver struct {
	inProgress map[common.Address]chan struct{}
	resolved   sync.Map

	inProgressStorage map[common.Address]map[common.Hash]chan struct{}
	resolvedStorage   map[common.Address]*sync.Map

	ctx    context.Context
	cancel func()
}

// schedule begins the retrieval of a set of state accounts running on
// a background goroutine.
func (p *prestateResolver) schedule(r Reader, accounts []common.Address, storage map[common.Address][]common.Hash) {
	p.inProgress = make(map[common.Address]chan struct{})
	p.inProgressStorage = make(map[common.Address]map[common.Hash]chan struct{})
	p.resolvedStorage = make(map[common.Address]*sync.Map)
	p.ctx, p.cancel = context.WithCancel(context.Background())

	var workers errgroup.Group
	for _, addr := range accounts {
		p.inProgress[addr] = make(chan struct{})
	}

	for addr, slots := range storage {
		p.inProgressStorage[addr] = make(map[common.Hash]chan struct{})
		for _, slot := range slots {
			p.inProgressStorage[addr][slot] = make(chan struct{})
		}
		p.resolvedStorage[addr] = &sync.Map{}
	}

	for _, addr := range accounts {
		resolveAddr := addr
		workers.Go(func() error {
			select {
			case <-p.ctx.Done():
				return nil
			default:
			}

			acct, err := r.Account(resolveAddr)
			if err != nil {
				return err
			}
			p.resolved.Store(resolveAddr, acct)
			close(p.inProgress[resolveAddr])
			return nil
		})
	}
	for addr, slots := range storage {
		resolveAddr := addr
		for _, s := range slots {
			slot := s
			workers.Go(func() error {
				select {
				case <-p.ctx.Done():
					return nil
				default:
				}

				value, err := r.Storage(resolveAddr, slot)
				if err != nil {
					// TODO: need to surface this error somehow so that execution can quit.
					// right now, it's silently consumed because we don't block using workers.Wait() anywhere...
					return err
				}
				p.resolvedStorage[resolveAddr].Store(slot, value)
				close(p.inProgressStorage[resolveAddr][slot])
				return nil
			})
		}
	}
}

func (p *prestateResolver) stop() {
	p.cancel()
}

func (p *prestateResolver) storage(addr common.Address, key common.Hash) *common.Hash {
	// check that the slot was actually scheduled
	storages, ok := p.inProgressStorage[addr]
	if !ok {
		return nil
	}
	_, ok = storages[key]
	if !ok {
		return nil
	}

	// block if the value of the slot is still being fetched
	select {
	case <-p.inProgressStorage[addr][key]:
	}
	res, exist := p.resolvedStorage[addr].Load(key)
	if !exist {
		// storage was scheduled, attempted to retrieve, but not set.
		// TODO: this is an error case that should be explicitly dealt with (the underlying reader failed to retrieve the storage slot)
		return nil
	}
	hashRes := res.(common.Hash)
	return &hashRes
}

// account returns the state account for the given address, blocking if it is
// still being resolved from disk.
func (p *prestateResolver) account(addr common.Address) *types.StateAccount {
	if _, ok := p.inProgress[addr]; !ok {
		return nil
	}

	select {
	case <-p.inProgress[addr]:
	}
	res, exist := p.resolved.Load(addr)
	if !exist {
		return nil
	}
	return res.(*types.StateAccount)
}

func (r *BALReader) initObjFromDiff(db *StateDB, addr common.Address, a *types.StateAccount, diff *bal.AccountMutations) *stateObject {
	var acct *types.StateAccount
	if a == nil {
		acct = &types.StateAccount{
			Nonce:    0,
			Balance:  uint256.NewInt(0),
			Root:     types.EmptyRootHash,
			CodeHash: types.EmptyCodeHash[:],
		}
	} else {
		acct = a.Copy()
	}
	if diff == nil {
		return newObject(db, addr, acct)
	}

	if diff.Nonce != nil {
		acct.Nonce = *diff.Nonce
	}
	if diff.Balance != nil {
		acct.Balance = new(uint256.Int).Set(diff.Balance)
	}
	obj := newObject(db, addr, acct)
	if diff.Code != nil {
		obj.setCode(crypto.Keccak256Hash(diff.Code), diff.Code)
	}
	if diff.StorageWrites != nil {
		for key, val := range diff.StorageWrites {
			obj.pendingStorage[common.Hash(key)] = common.Hash(val)
		}
	}
	if obj.empty() {
		return nil
	}
	return obj
}

// BALReader provides methods for reading account state from a block access
// list.  State values returned from the Reader methods must not be modified.
type BALReader struct {
	block          *types.Block
	accesses       map[common.Address]*bal.AccountAccess
	prestateReader prestateResolver
}

// NewBALReader constructs a new reader from an access list. db is expected to have been instantiated with a reader.
func NewBALReader(block *types.Block, reader Reader, useAsyncReads bool) *BALReader {
	r := &BALReader{accesses: make(map[common.Address]*bal.AccountAccess), block: block}
	finalIdx := len(block.Transactions()) + 1
	for _, acctDiff := range *block.AccessList() {
		r.accesses[acctDiff.Address] = &acctDiff
	}
	modifiedAccounts := r.ModifiedAccounts()
	storage := make(map[common.Address][]common.Hash)
	for _, addr := range modifiedAccounts {
		diff := r.readAccountDiff(addr, finalIdx)
		var scheduledStorageKeys []common.Hash
		if len(diff.StorageWrites) > 0 {
			writtenKeys := maps.Keys(diff.StorageWrites)
			for key := range writtenKeys {
				scheduledStorageKeys = append(scheduledStorageKeys, key)
			}
		}
		if useAsyncReads {
			scheduledStorageKeys = append(scheduledStorageKeys, r.accountStorageReads(addr)...)
		}
		if len(scheduledStorageKeys) > 0 {
			storage[addr] = scheduledStorageKeys
		}
	}
	r.prestateReader.schedule(reader, r.ModifiedAccounts(), storage)
	return r
}

func (r *BALReader) Storage(addr common.Address, key common.Hash) *common.Hash {
	return r.prestateReader.storage(addr, key)
}

// ModifiedAccounts returns a list of all accounts with mutations in the access list
func (r *BALReader) ModifiedAccounts() (res []common.Address) {
	for addr, access := range r.accesses {
		if len(access.NonceChanges) != 0 || len(access.CodeChanges) != 0 || len(access.StorageChanges) != 0 || len(access.BalanceChanges) != 0 {
			res = append(res, addr)
		}
	}
	return res
}

func logReadsDiff(idx int, address common.Address, computedReads map[common.Hash]struct{}, expectedReads []*bal.EncodedStorage) {
	expectedReadsMap := make(map[common.Hash]struct{})
	for _, er := range expectedReads {
		expectedReadsMap[er.ToHash()] = struct{}{}
	}

	allReads := make(map[common.Hash]struct{})

	for er := range expectedReadsMap {
		allReads[er] = struct{}{}
	}
	for cr := range computedReads {
		allReads[cr] = struct{}{}
	}

	var missingExpected, missingComputed []common.Hash

	for storage := range allReads {
		_, hasComputed := computedReads[storage]
		_, hasExpected := expectedReadsMap[storage]
		if hasComputed && !hasExpected {
			missingExpected = append(missingExpected, storage)
		}
		if !hasComputed && hasExpected {
			missingComputed = append(missingComputed, storage)
		}
	}
	if len(missingExpected) > 0 {
		log.Error("read storage slots which were not reported in the BAL", "index", idx, "address", address, missingExpected)
	}
	if len(missingComputed) > 0 {
		log.Error("did not read storage slots which were reported in the BAL", "index", idx, "address", address, missingComputed)
	}
}

func (r *BALReader) ValidateStateReads(idx int, computedReads bal.StateAccesses) bool {
	// 1. remove any slots from 'allReads' which were written
	// 2. validate that the read set in the BAL matches 'allReads' exactly
	for addr, reads := range computedReads {
		balAcctDiff := r.readAccountDiff(addr, len(r.block.Transactions())+2)
		if balAcctDiff != nil {
			for writeSlot := range balAcctDiff.StorageWrites {
				delete(reads, writeSlot)
			}
		}
		if _, ok := r.accesses[addr]; !ok {
			log.Error(fmt.Sprintf("account %x was accessed during execution but is not present in the access list", addr))
			return false
		}

		expectedReads := r.accesses[addr].StorageReads
		if len(reads) != len(expectedReads) {
			logReadsDiff(idx, addr, reads, expectedReads)
			return false
		}

		for _, slot := range expectedReads {
			if _, ok := reads[slot.ToHash()]; !ok {
				log.Error("expected read is missing from BAL")
				return false
			}
		}
	}

	return true
}

// changesAt returns all state changes occurring at the given index.
func (r *BALReader) changesAt(idx int) *bal.StateDiff {
	res := &bal.StateDiff{make(map[common.Address]*bal.AccountMutations)}
	for addr, _ := range r.accesses {
		accountChanges := r.accountChangesAt(addr, idx)
		if accountChanges != nil {
			res.Mutations[addr] = accountChanges
		}
	}
	return res
}

// accountChangesAt returns the state changes of an account at a given index,
// or nil if there are no changes.
func (r *BALReader) accountChangesAt(addr common.Address, idx int) *bal.AccountMutations {
	acct, exist := r.accesses[addr]
	if !exist {
		return nil
	}

	var res bal.AccountMutations

	// TODO: remove the reverse iteration here to clean the code up

	for i := len(acct.BalanceChanges) - 1; i >= 0; i-- {
		if acct.BalanceChanges[i].TxIdx == uint16(idx) {
			res.Balance = acct.BalanceChanges[i].Balance
		}
		if acct.BalanceChanges[i].TxIdx < uint16(idx) {
			break
		}
	}

	for i := len(acct.CodeChanges) - 1; i >= 0; i-- {
		if acct.CodeChanges[i].TxIdx == uint16(idx) {
			res.Code = acct.CodeChanges[i].Code
			break
		}
		if acct.CodeChanges[i].TxIdx < uint16(idx) {
			break
		}
	}

	for i := len(acct.NonceChanges) - 1; i >= 0; i-- {
		if acct.NonceChanges[i].TxIdx == uint16(idx) {
			res.Nonce = &acct.NonceChanges[i].Nonce
			break
		}
		if acct.NonceChanges[i].TxIdx < uint16(idx) {
			break
		}
	}

	for i := len(acct.StorageChanges) - 1; i >= 0; i-- {
		if res.StorageWrites == nil {
			res.StorageWrites = make(map[common.Hash]common.Hash)
		}
		slotWrites := acct.StorageChanges[i]

		for j := len(slotWrites.Accesses) - 1; j >= 0; j-- {
			if slotWrites.Accesses[j].TxIdx == uint16(idx) {
				res.StorageWrites[slotWrites.Slot.ToHash()] = slotWrites.Accesses[j].ValueAfter.ToHash()
				break
			}
			if slotWrites.Accesses[j].TxIdx < uint16(idx) {
				break
			}
		}
		if len(res.StorageWrites) == 0 {
			res.StorageWrites = nil
		}
	}

	if res.Code == nil && res.Nonce == nil && len(res.StorageWrites) == 0 && res.Balance == nil {
		return nil
	}
	return &res
}

func (r *BALReader) isModified(addr common.Address) bool {
	access, ok := r.accesses[addr]
	if !ok {
		return false
	}
	return len(access.StorageChanges) > 0 || len(access.BalanceChanges) > 0 || len(access.CodeChanges) > 0 || len(access.NonceChanges) > 0
}

func (r *BALReader) readAccount(db *StateDB, addr common.Address, idx int) *stateObject {
	diff := r.readAccountDiff(addr, idx)
	prestate := r.prestateReader.account(addr)
	return r.initObjFromDiff(db, addr, prestate, diff)
}

func (r *BALReader) accountStorageReads(addr common.Address) []common.Hash {
	diff, exist := r.accesses[addr]
	if !exist {
		return []common.Hash{}
	}

	var reads []common.Hash
	for _, key := range diff.StorageReads {
		reads = append(reads, key.ToHash())
	}
	return reads
}

// readAccountDiff returns the accumulated state changes of an account up
// through, and including the given index.
func (r *BALReader) readAccountDiff(addr common.Address, idx int) *bal.AccountMutations {
	diff, exist := r.accesses[addr]
	if !exist {
		return nil
	}

	var res bal.AccountMutations

	for i := 0; i < len(diff.BalanceChanges) && diff.BalanceChanges[i].TxIdx <= uint16(idx); i++ {
		res.Balance = diff.BalanceChanges[i].Balance
	}

	for i := 0; i < len(diff.CodeChanges) && diff.CodeChanges[i].TxIdx <= uint16(idx); i++ {
		res.Code = diff.CodeChanges[i].Code
	}

	for i := 0; i < len(diff.NonceChanges) && diff.NonceChanges[i].TxIdx <= uint16(idx); i++ {
		res.Nonce = &diff.NonceChanges[i].Nonce
	}

	if len(diff.StorageChanges) > 0 {
		res.StorageWrites = make(map[common.Hash]common.Hash)
		for _, slotWrites := range diff.StorageChanges {
			for i := 0; i < len(slotWrites.Accesses) && slotWrites.Accesses[i].TxIdx <= uint16(idx); i++ {
				res.StorageWrites[slotWrites.Slot.ToHash()] = slotWrites.Accesses[i].ValueAfter.ToHash()
			}
		}
	}

	return &res
}

func mutationsLogfmt(prefix string, mutations *bal.AccountMutations) (logs []interface{}) {
	if mutations.Code != nil {
		logs = append(logs, fmt.Sprintf("%s-code", prefix), fmt.Sprintf("%x", mutations.Code))
	}
	if mutations.Balance != nil {
		logs = append(logs, fmt.Sprintf("%s-balance", prefix), mutations.Balance.String())
	}
	if mutations.Nonce != nil {
		logs = append(logs, fmt.Sprintf("%s-nonce", prefix), mutations.Nonce)
	}
	if mutations.StorageWrites != nil {
		for key, val := range mutations.StorageWrites {
			logs = append(logs, fmt.Sprintf("%s-storage-write-key"), key, fmt.Sprintf("%s-storage-write-value"), val)
		}
	}
	return logs
}

func logfmtMutationsDiff(local, remote map[common.Address]*bal.AccountMutations) (logs []interface{}) {
	keys := make(map[common.Address]struct{})

	for addr, _ := range local {
		keys[addr] = struct{}{}
	}
	for addr, _ := range remote {
		keys[addr] = struct{}{}
	}

	for addr := range keys {
		_, hasLocal := local[addr]
		_, hasRemote := remote[addr]

		if hasLocal && !hasRemote {
			logs = append(logs, mutationsLogfmt(fmt.Sprintf("local-%x", addr), local[addr])...)
		}
		if !hasLocal && hasRemote {
			logs = append(logs, mutationsLogfmt(fmt.Sprintf("remote-%x", addr), remote[addr])...)
		}
	}
	return logs
}

// ValidateStateDiff returns an error if the computed state diff is not equal to
// diff reported from the access list at the given index.
func (r *BALReader) ValidateStateDiff(idx int, computedDiff *bal.StateDiff) bool {
	balChanges := r.changesAt(idx)
	for addr, state := range balChanges.Mutations {
		computedAccountDiff, ok := computedDiff.Mutations[addr]
		if !ok {
			// TODO: print out the full fields here
			log.Error("BAL contained account which wasn't present in computed state diff", "address", addr)
			return false
		}

		if !state.Eq(computedAccountDiff) {
			state.LogDiff(addr, computedAccountDiff)
			return false
		}
	}

	if len(balChanges.Mutations) != len(computedDiff.Mutations) {
		log.Error("computed state diff contained accounts that weren't reported in BAL", logfmtMutationsDiff(computedDiff.Mutations, balChanges.Mutations))
		return false
	}

	return true
}

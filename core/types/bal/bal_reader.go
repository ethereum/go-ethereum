package bal

import (
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// AccessListReader enables efficient state diff lookups from a block access
// list during block execution.
type AccessListReader struct {
	accounts map[common.Address]*preparedAccount
}

type preparedAccount struct {
	storage map[common.Hash]preparedSlot
	AccountAccess
}

type preparedSlot struct {
	changes []encodingStorageWrite // borrowed, sorted asc by BlockAccessIndex
}

// NewAccessListReader instantiates an access list reader.
func NewAccessListReader(list BlockAccessList) *AccessListReader {
	accounts := make(map[common.Address]*preparedAccount, len(list))
	for i := range list {
		a := list[i] // index; do not range-copy the AccountAccess
		pa := &preparedAccount{
			AccountAccess: a,
		}
		if len(a.StorageChanges) > 0 {
			pa.storage = make(map[common.Hash]preparedSlot, len(a.StorageChanges))
			for j := range a.StorageChanges {
				sc := &a.StorageChanges[j]
				pa.storage[sc.Slot.Bytes32()] = preparedSlot{changes: sc.SlotChanges}
			}
		}
		accounts[a.Address] = pa
	}
	return &AccessListReader{accounts: accounts}
}

// lastBefore returns the position of the last element in a slice of n elements
// sorted ascending by BlockAccessIndex whose key is strictly less than idx, or
// -1 if no such element exists. keyAt returns the BlockAccessIndex at position k.
func lastBefore(n int, idx uint32, keyAt func(k int) uint32) int {
	// sort.Search returns the smallest position whose key is >= idx; everything
	// before it is strictly less than idx, so the answer is that position - 1.
	return sort.Search(n, func(k int) bool { return keyAt(k) >= idx }) - 1
}

// Balance returns the post-balance in effect immediately before the given block
// access index, or nil if the account's balance was not changed before idx.
// The returned pointer aliases the access list and must not be mutated.
func (p *AccessListReader) Balance(addr common.Address, idx int) *uint256.Int {
	a := p.accounts[addr]
	if a == nil {
		return nil
	}
	k := lastBefore(len(a.BalanceChanges), uint32(idx), func(i int) uint32 { return a.BalanceChanges[i].BlockAccessIndex })
	if k < 0 {
		return nil
	}
	return a.BalanceChanges[k].PostBalance
}

// Nonce returns the post-nonce in effect immediately before the given block
// access index. The boolean is false if the nonce was not changed before idx.
func (p *AccessListReader) Nonce(addr common.Address, idx int) (uint64, bool) {
	a := p.accounts[addr]
	if a == nil {
		return 0, false
	}
	k := lastBefore(len(a.NonceChanges), uint32(idx), func(i int) uint32 { return a.NonceChanges[i].BlockAccessIndex })
	if k < 0 {
		return 0, false
	}
	return a.NonceChanges[k].PostNonce, true
}

// Code returns the contract code in effect immediately before the given block
// access index, or nil if the code was not changed before idx. The returned
// slice aliases the access list and must not be mutated.
func (p *AccessListReader) Code(addr common.Address, idx int) []byte {
	a := p.accounts[addr]
	if a == nil {
		return nil
	}
	k := lastBefore(len(a.CodeChanges), uint32(idx), func(i int) uint32 { return a.CodeChanges[i].BlockAccessIndex })
	if k < 0 {
		return nil
	}
	return a.CodeChanges[k].NewCode
}

// StorageAt returns the post-value of a storage slot immediately before the
// given block access index. The boolean is false if the slot was not written
// before idx.
func (p *AccessListReader) StorageAt(addr common.Address, slot common.Hash, idx int) (common.Hash, bool) {
	a := p.accounts[addr]
	if a == nil {
		return common.Hash{}, false
	}
	s, ok := a.storage[slot]
	if !ok {
		return common.Hash{}, false
	}
	k := lastBefore(len(s.changes), uint32(idx), func(i int) uint32 { return s.changes[i].BlockAccessIndex })
	if k < 0 {
		return common.Hash{}, false
	}
	return s.changes[k].PostValue.Bytes32(), true
}

// AccountMutations returns the aggregate mutation for an account up until (and
// not including) the given block access list index, or nil if the account was
// not mutated before idx.
func (p *AccessListReader) AccountMutations(addr common.Address, idx int) *AccountMutations {
	a := p.accounts[addr]
	if a == nil {
		return nil
	}
	res := &AccountMutations{}
	if bal := p.Balance(addr, idx); bal != nil {
		res.Balance = bal.Clone()
	}
	if code := p.Code(addr, idx); code != nil {
		res.Code = code
	}
	if nonce, ok := p.Nonce(addr, idx); ok {
		res.Nonce = new(uint64)
		*res.Nonce = nonce
	}
	for slot, s := range a.storage {
		k := lastBefore(len(s.changes), uint32(idx), func(i int) uint32 { return s.changes[i].BlockAccessIndex })
		if k < 0 {
			continue
		}
		if res.StorageWrites == nil {
			res.StorageWrites = make(map[common.Hash]common.Hash)
		}
		res.StorageWrites[slot] = s.changes[k].PostValue.Bytes32()
	}
	if res.Code == nil && res.Nonce == nil && len(res.StorageWrites) == 0 && res.Balance == nil {
		return nil
	}
	return res
}

type StorageKeys map[common.Address][]common.Hash

// StorageKeys returns the set of accounts and storage keys mutated in the access
// list. If reads is set, the un-mutated accounts/keys are included in the result.
func (p *AccessListReader) StorageKeys() (keys StorageKeys) {
	keys = make(StorageKeys)
	for addr, a := range p.accounts {
		for _, storageChange := range a.StorageChanges {
			keys[addr] = append(keys[addr], storageChange.Slot.Bytes32())
		}
		for _, storageRead := range a.StorageReads {
			keys[addr] = append(keys[addr], storageRead.Bytes32())
		}
	}
	return
}

// Mutations returns the aggregate state mutations from bal indices [0, idx).
func (p *AccessListReader) Mutations(idx int) *StateMutations {
	res := make(StateMutations)
	for addr := range p.accounts {
		if mut := p.AccountMutations(addr, idx); mut != nil {
			res[addr] = *mut
		}
	}
	return &res
}

// AllDestructions returns all accounts that experienced a destruction, regardless
// of whether they were later resurrected and exist after the block. It excludes
// ephemeral contracts from the result.
func (p *AccessListReader) AllDestructions() (res []common.Address) {
	for addr, a := range p.accounts {
		for _, nonce := range a.NonceChanges {
			if nonce.PostNonce == 0 {
				res = append(res, addr)
				break
			}
		}
	}
	return res
}

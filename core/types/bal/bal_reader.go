package bal

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
)

// AccessListReader exposes utilities to read state mutations and accesses from an access list
type AccessListReader map[common.Address]*AccountAccess

func NewAccessListReader(bal BlockAccessList) (reader AccessListReader) {
	reader = make(AccessListReader)
	for _, accountAccess := range bal {
		reader[accountAccess.Address] = &accountAccess
	}
	return
}

// AccountMutations returns the aggregate mutation for an account up until (and not including) the given block access
// list index.
func (a AccessListReader) AccountMutations(addr common.Address, idx int) (res *AccountMutations) {
	diff, exist := a[addr]
	if !exist {
		return nil
	}

	res = &AccountMutations{}

	for i := 0; i < len(diff.BalanceChanges) && diff.BalanceChanges[i].BlockAccessIndex < uint32(idx); i++ {
		res.Balance = diff.BalanceChanges[i].PostBalance.Clone()
	}

	for i := 0; i < len(diff.CodeChanges) && diff.CodeChanges[i].BlockAccessIndex < uint32(idx); i++ {
		res.Code = bytes.Clone(diff.CodeChanges[i].NewCode)
	}

	for i := 0; i < len(diff.NonceChanges) && diff.NonceChanges[i].BlockAccessIndex < uint32(idx); i++ {
		res.Nonce = new(uint64)
		*res.Nonce = diff.NonceChanges[i].PostNonce
	}

	if len(diff.StorageChanges) > 0 {
		res.StorageWrites = make(map[common.Hash]common.Hash)
		for _, slotWrites := range diff.StorageChanges {
			for i := 0; i < len(slotWrites.SlotChanges) && slotWrites.SlotChanges[i].BlockAccessIndex < uint32(idx); i++ {
				res.StorageWrites[slotWrites.Slot.Bytes32()] = slotWrites.SlotChanges[i].PostValue.Bytes32()
			}
		}
	}

	if res.Code == nil && res.Nonce == nil && len(res.StorageWrites) == 0 && res.Balance == nil {
		return nil
	}
	return res
}

type StorageKeys map[common.Address][]common.Hash

// StorageKeys returns the set of accounts and storage keys mutated in the access list.
// If reads is set, the un-mutated accounts/keys are included in the result.
func (a AccessListReader) StorageKeys(reads bool) (keys StorageKeys) {
	keys = make(StorageKeys)
	for addr, acct := range a {
		for _, storageChange := range acct.StorageChanges {
			keys[addr] = append(keys[addr], storageChange.Slot.Bytes32())
		}
		if !(reads && len(acct.StorageReads) > 0) {
			continue
		}
		for _, storageRead := range acct.StorageReads {
			keys[addr] = append(keys[addr], storageRead.Bytes32())
		}
	}
	return
}

// Storage returns the value of a storage key at the start of executing an index.
// If the slot has no mutations in the access list, it returns nil.
func (a AccessListReader) Storage(addr common.Address, key common.Hash, idx int) (val *common.Hash) {
	storageMuts := a.AccountMutations(addr, idx)
	if storageMuts != nil {
		res, ok := storageMuts.StorageWrites[key]
		if ok {
			return &res
		}
	}
	return nil
}

// Mutations returns the aggregate state mutations from bal indices [0, idx)
func (a AccessListReader) Mutations(idx int) *StateMutations {
	res := make(StateMutations)
	for addr := range a {
		if mut := a.AccountMutations(addr, idx); mut != nil {
			res[addr] = *mut
		}
	}
	return &res
}

// AllDestructions returns all accounts that experienced a destruction, regardless of whether
// they were later resurrected and exist after the block.  It excludes ephemeral contracts from
// the result.
func (a AccessListReader) AllDestructions() (res []common.Address) {
	for addr, access := range a {
		for _, nonce := range access.NonceChanges {
			if nonce.PostNonce == 0 {
				res = append(res, addr)
				break
			}
		}
	}
	return res
}

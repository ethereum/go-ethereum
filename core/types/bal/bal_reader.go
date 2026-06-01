package bal

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
)

// AccessListReader exposes utilities to read state mutations and accesses from an access list
// TODO: expose this an an interface?
type AccessListReader map[common.Address]*AccountAccess

func NewAccessListReader(bal BlockAccessList) (reader AccessListReader) {
	reader = make(AccessListReader)
	for _, accountAccess := range bal {
		reader[accountAccess.Address] = &accountAccess
	}
	return
}

// TODO: these methods should return the mutations accrued before the execution of the given index

// TODO: strip the storage mutations from the returned result
// the returned object should be able to be modified
func (a AccessListReader) accountMutationsAt(addr common.Address, idx int) (res *AccountMutations) {
	acct, exist := a[addr]
	if !exist {
		return nil
	}

	res = &AccountMutations{}
	// TODO: remove the reverse iteration here to clean the code up

	for i := len(acct.BalanceChanges) - 1; i >= 0; i-- {
		if acct.BalanceChanges[i].BlockAccessIndex == uint32(idx) {
			res.Balance = acct.BalanceChanges[i].PostBalance
		}
		if acct.BalanceChanges[i].BlockAccessIndex < uint32(idx) {
			break
		}
	}

	for i := len(acct.CodeChanges) - 1; i >= 0; i-- {
		if acct.CodeChanges[i].BlockAccessIndex == uint32(idx) {
			res.Code = bytes.Clone(acct.CodeChanges[i].NewCode)
			break
		}
		if acct.CodeChanges[i].BlockAccessIndex < uint32(idx) {
			break
		}
	}

	for i := len(acct.NonceChanges) - 1; i >= 0; i-- {
		if acct.NonceChanges[i].BlockAccessIndex == uint32(idx) {
			res.Nonce = new(uint64)
			*res.Nonce = acct.NonceChanges[i].PostNonce
			break
		}
		if acct.NonceChanges[i].BlockAccessIndex < uint32(idx) {
			break
		}
	}

	for i := len(acct.StorageChanges) - 1; i >= 0; i-- {
		if res.StorageWrites == nil {
			res.StorageWrites = make(map[common.Hash]common.Hash)
		}
		slotWrites := acct.StorageChanges[i]

		for j := len(slotWrites.SlotChanges) - 1; j >= 0; j-- {
			if slotWrites.SlotChanges[j].BlockAccessIndex == uint32(idx) {
				res.StorageWrites[slotWrites.Slot.Bytes32()] = slotWrites.SlotChanges[j].PostValue.Bytes32()
				break
			}
			if slotWrites.SlotChanges[j].BlockAccessIndex < uint32(idx) {
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
	return res
}

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

// MutationsAt returns the state mutations from a bal index
func (a AccessListReader) MutationsAt(idx int) *StateMutations {
	res := make(StateMutations)
	for addr := range a {
		if mut := a.accountMutationsAt(addr, idx); mut != nil {
			res[addr] = *mut
		}
	}
	return &res
}

func copyStorageReads(a *AccountAccess) map[common.Hash]struct{} {
	return nil
}

func (a AccessListReader) Accesses() (res StateAccesses) {
	res = make(StateAccesses)
	for addr, acct := range a {
		if len(acct.StorageReads) > 0 {
			res[addr] = copyStorageReads(acct)
		}
	}
	return
}

func (a AccessListReader) Contains(bal *ConstructionBlockAccessList, balIdx uint32) bool {
	for addr, access := range bal.Accounts {
		otherAccess, ok := a[addr]
		if !ok {
			return false
		}
		if len(access.BalanceChanges) > 0 {
			if len(otherAccess.BalanceChanges) >= int(balIdx) {
				if !otherAccess.BalanceChanges[balIdx].PostBalance.Eq(access.BalanceChanges[balIdx]) {
					return false
				}
			} else {
				return false
			}
		}

		if len(access.NonceChanges) > 0 {
			if len(otherAccess.NonceChanges) >= int(balIdx) {
				if otherAccess.NonceChanges[balIdx].PostNonce != access.NonceChanges[balIdx] {
					return false
				}
			} else {
				return false
			}
		}

		if len(access.CodeChange) > 0 {
			if len(otherAccess.CodeChanges) >= int(balIdx) {
				if !bytes.Equal(otherAccess.CodeChanges[balIdx].NewCode, access.CodeChange[balIdx]) {
					return false
				}
			} else {
				return false
			}
		}

		panic("TODO: Finish the function...")
		//if len(access.StorageWrites)
	}
	return true
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

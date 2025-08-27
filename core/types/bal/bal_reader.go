package bal

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
)

// Reader provides methods for reading account state from a block access
// list.  State values returned from the Reader methods must not be modified.
type Reader struct {
	accesses map[common.Address]*AccountAccess
}

// NewReader constructs a new reader from an access list
func NewReader(al *BlockAccessList) Reader {
	r := Reader{make(map[common.Address]*AccountAccess)}
	for _, acctDiff := range al.Accesses {
		r.accesses[acctDiff.Address] = &acctDiff
	}
	return r
}

// Accounts returns a list of all accounts from the access list
func (r *Reader) Accounts() (res []common.Address) {
	for addr, _ := range r.accesses {
		res = append(res, addr)
	}
	return res
}

// Iterate computes the accumulated state changes of each account in the
// access list up through an index.  These are passed to the provided
// callback, which if it returns false, will stop iteration.
func (r *Reader) Iterate(idx int, cb func(addr common.Address, state *AccountState) bool) {
	for addr, _ := range r.accesses {
		acct := r.ReadAccount(addr, idx)
		if acct != nil && !cb(addr, acct) {
			return
		}
	}
}

// changesAt returns all state changes at the given index.
func (r *Reader) changesAt(idx int) *StateDiff {
	res := &StateDiff{make(map[common.Address]*AccountState)}
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
func (r *Reader) accountChangesAt(addr common.Address, idx int) *AccountState {
	acct, exist := r.accesses[addr]
	if !exist {
		return nil
	}

	var res AccountState

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

	for i := len(acct.StorageWrites) - 1; i >= 0; i-- {
		if res.StorageWrites == nil {
			res.StorageWrites = make(map[common.Hash]common.Hash)
		}
		slotWrites := acct.StorageWrites[i]

		for j := len(slotWrites.Accesses) - 1; j >= 0; j-- {
			if slotWrites.Accesses[j].TxIdx == uint16(idx) {
				res.StorageWrites[slotWrites.Slot] = slotWrites.Accesses[j].ValueAfter
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

// ReadAccount returns the accumulated state changes of an account up through idx.
func (r *Reader) ReadAccount(addr common.Address, idx int) *AccountState {
	acct, exist := r.accesses[addr]
	if !exist {
		return nil
	}

	var res AccountState

	for i := 0; i < len(acct.BalanceChanges) && acct.BalanceChanges[i].TxIdx <= uint16(idx); i++ {
		res.Balance = acct.BalanceChanges[i].Balance
	}

	for i := 0; i < len(acct.CodeChanges) && acct.CodeChanges[i].TxIdx <= uint16(idx); i++ {
		res.Code = acct.CodeChanges[i].Code
	}

	for i := 0; i < len(acct.NonceChanges) && acct.NonceChanges[i].TxIdx <= uint16(idx); i++ {
		res.Nonce = &acct.NonceChanges[i].Nonce
	}

	if len(acct.StorageWrites) > 0 {
		res.StorageWrites = make(map[common.Hash]common.Hash)
		for _, slotWrites := range acct.StorageWrites {
			for i := 0; i < len(slotWrites.Accesses) && slotWrites.Accesses[i].TxIdx <= uint16(idx); i++ {
				res.StorageWrites[slotWrites.Slot] = slotWrites.Accesses[i].ValueAfter
			}
		}
	}

	if res.Code == nil && res.Nonce == nil && len(res.StorageWrites) == 0 && res.Balance == nil {
		return nil
	}
	return &res
}

// ValidateStateDiff returns an error if the computed state diff is not equal to
// diff reported from the access list at the given index.
func (r *Reader) ValidateStateDiff(idx int, computedDiff *StateDiff) error {
	balChanges := r.changesAt(idx)
	for addr, state := range balChanges.Mutations {
		computedAccountDiff, ok := computedDiff.Mutations[addr]
		if !ok {
			return fmt.Errorf("BAL change not reported in computed")
		}

		if !state.Eq(computedAccountDiff) {
			return fmt.Errorf("unequal")
		}
	}

	if len(balChanges.Mutations) != len(computedDiff.Mutations) {
		return fmt.Errorf("computed diff contained additional mutations compared to BAL")
	}

	return nil
}

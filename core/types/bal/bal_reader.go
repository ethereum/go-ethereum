package bal

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
)

type Reader struct {
	accesses map[common.Address]*AccountAccess
}

func NewReader(al *BlockAccessList) Reader {
	r := Reader{make(map[common.Address]*AccountAccess)}
	for _, acctDiff := range al.Accesses {
		r.accesses[acctDiff.Address] = &acctDiff
	}
	return r
}

func (r *Reader) Accounts() (res []common.Address) {
	for addr, _ := range r.accesses {
		res = append(res, addr)
	}
	return res
}

func (r *Reader) Iterate(idx int, cb func(addr common.Address, state *AccountState) bool) {
	for addr, _ := range r.accesses {
		acct := r.ReadAccount(addr, idx)
		if !cb(addr, acct) {
			return
		}
	}
}

// ReadAccount returns the post-state of the account at the given bal index.
// Do not modify the returned object.
func (r *Reader) ReadAccount(addr common.Address, idx int) *AccountState {
	acct, exist := r.accesses[addr]
	if !exist {
		return nil
	}

	var res AccountState

	for i := len(acct.BalanceChanges) - 1; i >= 0; i-- {
		if acct.BalanceChanges[i].TxIdx <= uint16(idx) {
			res.Balance = &acct.BalanceChanges[i].Balance
			break
		}
	}

	for i := len(acct.CodeChanges) - 1; i >= 0; i-- {
		if acct.CodeChanges[i].TxIndex <= uint16(idx) {
			res.Code = acct.CodeChanges[i].Code
			break
		}
	}

	for i := len(acct.NonceChanges) - 1; i >= 0; i-- {
		if acct.NonceChanges[i].TxIdx <= uint16(idx) {
			res.Nonce = &acct.NonceChanges[i].Nonce
			break
		}
	}

	for i := len(acct.StorageWrites) - 1; i >= 0; i-- {
		if res.StorageWrites == nil {
			res.StorageWrites = make(map[common.Hash]common.Hash)
		}
		slotWrites := acct.StorageWrites[i]

		for j := len(slotWrites.Accesses) - 1; i >= 0; i-- {
			if slotWrites.Accesses[j].TxIdx <= uint16(idx) {
				if _, exist := res.StorageWrites[slotWrites.Slot]; !exist {
					res.StorageWrites[slotWrites.Slot] = slotWrites.Accesses[j].ValueAfter
					break
				}
			}
		}
	}
	return &res
}

// ValidateStateDiff asserts that both state diffs are equivalent.
func (r *Reader) ValidateStateDiff(idx int, computedDiff *StateDiff) error {
	var err error
	var balDiffCount int
	r.Iterate(idx, func(addr common.Address, state *AccountState) bool {
		computedAccountDiff, ok := computedDiff.Mutations[addr]
		if !ok {
			err = fmt.Errorf("missing from BAL")
			return false
		}

		if !state.Eq(computedAccountDiff) {
			err = fmt.Errorf("unequal")
			return false
		}
		balDiffCount++
		return true
	})
	if err != nil {
		return err
	}

	if balDiffCount != len(computedDiff.Mutations) {
		return fmt.Errorf("computed diff contained additional mutations compared to BAL")
	}

	return nil
}

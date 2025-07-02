package bal

import (
	"bytes"
	"fmt"
	"maps"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// BlockAccessList contains post-block modified state and some state accessed
// in execution (account addresses and storage keys).
type BlockAccessList struct {
	accounts map[common.Address]*accountAccess
}

// NewBlockAccessList instantiates an empty access list.
func NewBlockAccessList() BlockAccessList {
	return BlockAccessList{
		accounts: make(map[common.Address]*accountAccess),
	}
}

// AccountRead records the address of an account that has been read during execution.
func (b *BlockAccessList) AccountRead(addr common.Address) {
	if _, ok := b.accounts[addr]; !ok {
		b.accounts[addr] = newAccountAccess()
	}
}

// StorageRead records a storage key read during execution.
func (b *BlockAccessList) StorageRead(address common.Address, key common.Hash) {
	if _, ok := b.accounts[address]; !ok {
		b.accounts[address] = newAccountAccess()
	}

	if _, ok := b.accounts[address].StorageWrites[key]; ok {
		return
	}

	b.accounts[address].StorageReads[key] = struct{}{}
}

// StorageWrite records the post-transaction value of a mutated storage slot.
// The storage slot is removed from the list of read slots.
func (b *BlockAccessList) StorageWrite(txIdx uint16, address common.Address, key, value common.Hash) {
	if _, ok := b.accounts[address]; !ok {
		b.accounts[address] = newAccountAccess()
	}

	if _, ok := b.accounts[address].StorageWrites[key]; !ok {
		b.accounts[address].StorageWrites[key] = make(slotWrites)
	}
	b.accounts[address].StorageWrites[key][txIdx] = value
	delete(b.accounts[address].StorageReads, key)
}

// CodeChange records the code of a newly-created contract.
func (b *BlockAccessList) CodeChange(address common.Address, txIndex uint16, code []byte) {
	if _, ok := b.accounts[address]; !ok {
		b.accounts[address] = newAccountAccess()
	}

	b.accounts[address].CodeChange = &codeChange{
		TxIndex: txIndex,
		Code:    bytes.Clone(code),
	}
}

// NonceDiff records tx post-state nonce of any contract-like accounts whose nonce was incremented
func (b *BlockAccessList) NonceDiff(address common.Address, txIdx uint16, postNonce uint64) {
	if _, ok := b.accounts[address]; !ok {
		b.accounts[address] = newAccountAccess()
	}

	b.accounts[address].NonceChanges[txIdx] = postNonce
}

// BalanceChange records the post-transaction balance of an account whose
// balance changed.
func (b *BlockAccessList) BalanceChange(txIdx uint16, address common.Address, balance *uint256.Int) {
	if _, ok := b.accounts[address]; !ok {
		b.accounts[address] = newAccountAccess()
	}

	b.accounts[address].BalanceChanges[txIdx] = balance.Clone()
}

// contains the post-transaction balances of an account, keyed by transaction indices
// where it was changed.
type balanceDiff map[uint16]*uint256.Int

// copy returns a deep copy of the object
func (b balanceDiff) copy() balanceDiff {
	res := make(balanceDiff)
	for idx, balance := range b {
		res[idx] = balance.Clone()
	}
	return res
}

// PrettyPrint returns a human-readable representation of the access list
func (b *BlockAccessList) PrettyPrint() string {
	enc := b.toEncodingObj()
	return enc.prettyPrint()
}

// Hash computes the SSZ hash of the access list
func (b *BlockAccessList) Hash() common.Hash {
	hash, err := b.toEncodingObj().HashTreeRoot()
	if err != nil {
		// errors here are related to BAL values exceeding maximum size defined
		// by the spec.  Hard-fail because these cases are not expected to be hit
		// under reasonable conditions.
		panic(err)
	}
	return hash
}

// codeChange contains the code deployed at an address and the transaction
// index where the deployment took place.
type codeChange struct {
	TxIndex uint16
	Code    []byte `json:"code,omitempty"`
}

// post-state values of an account's storage slots modified in a block, keyed
// by slot key
type storageWrites map[common.Hash]slotWrites

func (s storageWrites) copy() storageWrites {
	res := make(storageWrites)
	for slot, writes := range s {
		res[slot] = maps.Clone(writes)
	}
	return res
}

// accountAccess contains post-block account state for mutations as well as
// all storage keys that were read during execution.
type accountAccess struct {
	StorageWrites  storageWrites            `json:"storageWrites,omitempty"`
	StorageReads   map[common.Hash]struct{} `json:"storageReads,omitempty"`
	BalanceChanges balanceDiff              `json:"balanceChanges,omitempty"`
	NonceChanges   accountNonceDiffs        `json:"nonceChanges,omitempty"`

	// only set for contract accounts which were deployed in the block
	CodeChange *codeChange `json:"codeChange,omitempty"`
}

func newAccountAccess() *accountAccess {
	return &accountAccess{
		StorageWrites:  make(map[common.Hash]slotWrites),
		StorageReads:   make(map[common.Hash]struct{}),
		BalanceChanges: make(balanceDiff),
		NonceChanges:   make(accountNonceDiffs),
	}
}

// the post-state nonce values of a contract account keyed by tx index
type accountNonceDiffs map[uint16]uint64

// the post-state values of a storage slot, keyed by tx index
type slotWrites map[uint16]common.Hash

// Copy returns a deep copy of the access list.
func (b *BlockAccessList) Copy() *BlockAccessList {
	res := new(BlockAccessList)
	for addr, aa := range b.accounts {
		var aaCopy accountAccess
		aaCopy.StorageWrites = aa.StorageWrites.copy()
		aaCopy.StorageReads = maps.Clone(aa.StorageReads)
		aaCopy.BalanceChanges = aa.BalanceChanges.copy()
		aaCopy.NonceChanges = maps.Clone(aa.NonceChanges)
		if aa.CodeChange != nil {
			aaCopy.CodeChange = &codeChange{
				TxIndex: aa.CodeChange.TxIndex,
				Code:    bytes.Clone(aa.CodeChange.Code),
			}
		}
		res.accounts[addr] = &aaCopy
	}
	return res
}

func (e *encodingBlockAccessList) prettyPrint() string {
	var res bytes.Buffer
	printWithIndent := func(indent int, text string) {
		fmt.Fprintf(&res, "%s%s\n", strings.Repeat("    ", indent), text)
	}
	for _, accountDiff := range e.Accesses {
		printWithIndent(0, fmt.Sprintf("%x:", accountDiff.Address))

		printWithIndent(1, "storage writes:")
		for _, sWrite := range accountDiff.StorageWrites {
			printWithIndent(2, fmt.Sprintf("%x:", sWrite.Slot))
			for _, access := range sWrite.Accesses {
				printWithIndent(3, fmt.Sprintf("%d: %x", access.TxIdx, access.ValueAfter))
			}
		}

		printWithIndent(1, "storage reads:")
		for _, slot := range accountDiff.StorageReads {
			printWithIndent(2, fmt.Sprintf("%x", slot))
		}

		printWithIndent(1, "balance changes:")
		for _, change := range accountDiff.BalanceChanges {
			balance := new(uint256.Int).SetBytes(change.Balance[:]).String()
			printWithIndent(2, fmt.Sprintf("%d: %s", change.TxIdx, balance))
		}

		printWithIndent(1, "nonce changes:")
		for _, change := range accountDiff.NonceChanges {
			printWithIndent(2, fmt.Sprintf("%d: %d", change.TxIdx, change.Nonce))
		}

		if len(accountDiff.Code) > 0 {
			printWithIndent(1, "code:")
			printWithIndent(2, fmt.Sprintf("%d: %x", accountDiff.Code[0].TxIndex, accountDiff.Code[0].Code))
		}
	}

	return res.String()
}

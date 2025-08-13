// Copyright 2025 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package bal

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

//go:generate go run github.com/ethereum/go-ethereum/rlp/rlpgen -out bal_encoding_rlp_generated.go -type BlockAccessList -decoder

// These are objects used as input for the access list encoding. They mirror
// the spec format.

// BlockAccessList is the encoding format of ConstructionBlockAccessList.
type BlockAccessList struct {
	Accesses []AccountAccess `ssz-max:"300000" json:"accesses"`
}

func (e *BlockAccessList) String() string {
	var res bytes.Buffer
	enc := json.NewEncoder(&res)
	enc.SetIndent("", "    ")
	// TODO: check error
	enc.Encode(e)
	return res.String()
}

// Validate returns an error if the contents of the access list are not ordered
// according to the spec or any code changes are contained which exceed protocol
// max code size.
func (e *BlockAccessList) Validate() error {
	if !slices.IsSortedFunc(e.Accesses, func(a, b AccountAccess) int {
		return bytes.Compare(a.Address[:], b.Address[:])
	}) {
		return errors.New("block access list accounts not in lexicographic order")
	}
	for _, entry := range e.Accesses {
		if err := entry.validate(); err != nil {
			return err
		}
	}
	return nil
}

// Hash computes the keccak256 hash of the access list
func (e *BlockAccessList) Hash() common.Hash {
	var enc bytes.Buffer
	err := e.EncodeRLP(&enc)
	if err != nil {
		// errors here are related to BAL values exceeding maximum size defined
		// by the spec. Hard-fail because these cases are not expected to be hit
		// under reasonable conditions.
		panic(err)
	}
	return crypto.Keccak256Hash(enc.Bytes())
}

// encodeBalance encodes the provided balance into 16-bytes.
func encodeBalance(val *uint256.Int) [16]byte {
	valBytes := val.Bytes()
	if len(valBytes) > 16 {
		panic("can't encode value that is greater than 16 bytes in size")
	}
	var enc [16]byte
	copy(enc[16-len(valBytes):], valBytes[:])
	return enc
}

// TODO: if we don't go with SSZ, this can be a variable-sized byte array
type Balance [16]byte

func (b Balance) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%x", b))
}

func (b Balance) IsZero() bool {
	zeroBytes := [16]byte{}
	return bytes.Equal(b[:], zeroBytes[:])
}

// encodingBalanceChange is the encoding format of BalanceChange.
type encodingBalanceChange struct {
	TxIdx   uint16  `ssz-size:"2" json:"txIndex"`
	Balance Balance `ssz-size:"16" json:"balance"`
}

// encodingAccountNonce is the encoding format of NonceChange.
type encodingAccountNonce struct {
	TxIdx uint16 `ssz-size:"2" json:"txIndex"`
	Nonce uint64 `ssz-size:"8" json:"nonce"`
}

// encodingStorageWrite is the encoding format of StorageWrites.
type encodingStorageWrite struct {
	TxIdx      uint16      `json:"txIndex"`
	ValueAfter common.Hash `ssz-size:"32" json:"valueAfter"`
}

// encodingStorageWrite is the encoding format of SlotWrites.
type encodingSlotWrites struct {
	Slot     common.Hash            `ssz-size:"32" json:"slot"`
	Accesses []encodingStorageWrite `ssz-max:"300000" json:"accesses"`
}

// validate returns an instance of the encoding-representation slot writes in
// working representation.
func (e *encodingSlotWrites) validate() error {
	if slices.IsSortedFunc(e.Accesses, func(a, b encodingStorageWrite) int {
		return cmp.Compare[uint16](a.TxIdx, b.TxIdx)
	}) {
		return nil
	}
	return errors.New("storage write tx indices not in order")
}

// AccountAccess is the encoding format of ConstructionAccountAccess.
type AccountAccess struct {
	Address        common.Address          `ssz-size:"20" json:"address,omitempty"`           // 20-byte Ethereum address
	StorageWrites  []encodingSlotWrites    `ssz-max:"300000" json:"storageWrites,omitempty"`  // Storage changes (slot -> [tx_index -> new_value])
	StorageReads   []common.Hash           `ssz-max:"300000" json:"storageReads,omitempty"`   // Read-only storage keys
	BalanceChanges []encodingBalanceChange `ssz-max:"300000" json:"balanceChanges,omitempty"` // Balance changes ([tx_index -> post_balance])
	NonceChanges   []encodingAccountNonce  `ssz-max:"300000" json:"nonceChanges,omitempty"`   // Nonce changes ([tx_index -> new_nonce])
	CodeChanges    []CodeChange            `ssz-max:"1" json:"code,omitempty"`                // CodeChanges changes ([tx_index -> new_code])
}

// validate converts the account accesses out of encoding format.
// If any of the keys in the encoding object are not ordered according to the
// spec, an error is returned.
func (e *AccountAccess) validate() error {
	// Check the storage write slots are sorted in order
	if !slices.IsSortedFunc(e.StorageWrites, func(a, b encodingSlotWrites) int {
		return bytes.Compare(a.Slot[:], b.Slot[:])
	}) {
		return errors.New("storage writes slots not in lexicographic order")
	}
	for _, write := range e.StorageWrites {
		if err := write.validate(); err != nil {
			return err
		}
	}

	// Check the storage read slots are sorted in order
	if !slices.IsSortedFunc(e.StorageReads, func(a, b common.Hash) int {
		return bytes.Compare(a[:], b[:])
	}) {
		return errors.New("storage read slots not in lexicographic order")
	}

	// Check the balance changes are sorted in order
	if !slices.IsSortedFunc(e.BalanceChanges, func(a, b encodingBalanceChange) int {
		return cmp.Compare[uint16](a.TxIdx, b.TxIdx)
	}) {
		return errors.New("balance changes not in ascending order by tx index")
	}

	// Check the nonce changes are sorted in order
	if !slices.IsSortedFunc(e.NonceChanges, func(a, b encodingAccountNonce) int {
		return cmp.Compare[uint16](a.TxIdx, b.TxIdx)
	}) {
		return errors.New("nonce changes not in ascending order by tx index")
	}

	// Convert code change
	for _, codeChange := range e.CodeChanges {
		if len(codeChange.Code) > params.MaxCodeSize {
			return fmt.Errorf("code change contained oversized code")
		}
	}
	return nil
}

// Copy returns a deep copy of the account access
func (e *AccountAccess) Copy() AccountAccess {
	res := AccountAccess{
		Address:        e.Address,
		StorageReads:   slices.Clone(e.StorageReads),
		BalanceChanges: slices.Clone(e.BalanceChanges),
		NonceChanges:   slices.Clone(e.NonceChanges),
	}
	for _, storageWrite := range e.StorageWrites {
		res.StorageWrites = append(res.StorageWrites, encodingSlotWrites{
			Slot:     storageWrite.Slot,
			Accesses: slices.Clone(storageWrite.Accesses),
		})
	}
	for _, codeChange := range e.CodeChanges {
		res.CodeChanges = append(res.CodeChanges,
			CodeChange{
				codeChange.TxIndex,
				bytes.Clone(codeChange.Code),
			})
	}
	return res
}

// EncodeRLP returns the RLP-encoded access list
func (c *ConstructionBlockAccessList) EncodeRLP(wr io.Writer) error {
	return c.ToEncodingObj().EncodeRLP(wr)
}

var _ rlp.Encoder = &ConstructionBlockAccessList{}

// toEncodingObj creates an instance of the ConstructionAccountAccess of the type that is
// used as input for the encoding.
func (a *ConstructionAccountAccess) toEncodingObj(addr common.Address) AccountAccess {
	res := AccountAccess{
		Address:        addr,
		StorageWrites:  make([]encodingSlotWrites, 0),
		StorageReads:   make([]common.Hash, 0),
		BalanceChanges: make([]encodingBalanceChange, 0),
		NonceChanges:   make([]encodingAccountNonce, 0),
		CodeChanges:    make([]CodeChange, 0),
	}

	// Convert write slots
	writeSlots := slices.Collect(maps.Keys(a.StorageWrites))
	slices.SortFunc(writeSlots, common.Hash.Cmp)
	for _, slot := range writeSlots {
		var obj encodingSlotWrites
		obj.Slot = slot

		slotWrites := a.StorageWrites[slot]
		obj.Accesses = make([]encodingStorageWrite, 0, len(slotWrites))

		indices := slices.Collect(maps.Keys(slotWrites))
		slices.SortFunc(indices, cmp.Compare[uint16])
		for _, index := range indices {
			obj.Accesses = append(obj.Accesses, encodingStorageWrite{
				TxIdx:      index,
				ValueAfter: slotWrites[index],
			})
		}
		res.StorageWrites = append(res.StorageWrites, obj)
	}

	// Convert read slots
	readSlots := slices.Collect(maps.Keys(a.StorageReads))
	slices.SortFunc(readSlots, common.Hash.Cmp)
	for _, slot := range readSlots {
		res.StorageReads = append(res.StorageReads, slot)
	}

	// Convert balance changes
	balanceIndices := slices.Collect(maps.Keys(a.BalanceChanges))
	slices.SortFunc(balanceIndices, cmp.Compare[uint16])
	for _, idx := range balanceIndices {
		res.BalanceChanges = append(res.BalanceChanges, encodingBalanceChange{
			TxIdx:   idx,
			Balance: encodeBalance(a.BalanceChanges[idx]),
		})
	}

	// Convert nonce changes
	nonceIndices := slices.Collect(maps.Keys(a.NonceChanges))
	slices.SortFunc(nonceIndices, cmp.Compare[uint16])
	for _, idx := range nonceIndices {
		res.NonceChanges = append(res.NonceChanges, encodingAccountNonce{
			TxIdx: idx,
			Nonce: a.NonceChanges[idx],
		})
	}

	// Convert code change
	codeChangeIdxs := slices.Collect(maps.Keys(a.CodeChanges))
	slices.SortFunc(codeChangeIdxs, cmp.Compare[uint16])
	for _, idx := range codeChangeIdxs {
		res.CodeChanges = append(res.CodeChanges, CodeChange{
			idx,
			bytes.Clone(a.CodeChanges[idx].Code),
		})
	}
	return res
}

// ToEncodingObj returns an instance of the access list expressed as the type
// which is used as input for the encoding/decoding.
func (c *ConstructionBlockAccessList) ToEncodingObj() *BlockAccessList {
	var addresses []common.Address
	for addr := range c.Accounts {
		addresses = append(addresses, addr)
	}
	slices.SortFunc(addresses, common.Address.Cmp)

	var res BlockAccessList
	for _, addr := range addresses {
		res.Accesses = append(res.Accesses, c.Accounts[addr].toEncodingObj(addr))
	}
	return &res
}

func (e *BlockAccessList) PrettyPrint() string {
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

		if len(accountDiff.CodeChanges) > 0 {
			printWithIndent(1, "code:")
			for _, change := range accountDiff.CodeChanges {
				printWithIndent(2, fmt.Sprintf("%d: %x", change.TxIndex, change.Code))
			}
		}
	}
	return res.String()
}

// Copy returns a deep copy of the access list
func (e *BlockAccessList) Copy() (res BlockAccessList) {
	for _, accountAccess := range e.Accesses {
		res.Accesses = append(res.Accesses, accountAccess.Copy())
	}
	return
}

type ContractCode []byte

func (c *ContractCode) MarshalJSON() ([]byte, error) {
	hexStr := fmt.Sprintf("%x", *c)
	return json.Marshal(hexStr)
}

type AccountState struct {
	Balance       *Balance                    `json:"Balance,omitempty"`
	Nonce         *uint64                     `json:"Nonce,omitempty"`
	Code          ContractCode                `json:"Code,omitempty"`
	StorageWrites map[common.Hash]common.Hash `json:"StorageWrites,omitempty"`
}

// Merge the changes of a future AccountState into the caller, resulting in the
// combined state changes through next.
func (a *AccountState) Merge(next *AccountState) {
	if next.Balance != nil {
		a.Balance = next.Balance
	}
	if next.Nonce != nil {
		a.Nonce = next.Nonce
	}
	if next.Code != nil {
		a.Code = next.Code
	}
	if next.StorageWrites != nil {
		if a.StorageWrites == nil {
			a.StorageWrites = maps.Clone(next.StorageWrites)
		} else {
			for key, val := range next.StorageWrites {
				a.StorageWrites[key] = val
			}
		}
	}
}

func NewEmptyAccountState() *AccountState {
	return &AccountState{
		nil,
		nil,
		nil,
		nil,
	}
}

func (a *AccountState) Eq(other *AccountState) bool {
	if a.Balance != nil || other.Balance != nil {
		if a.Balance == nil || other.Balance == nil {
			return false
		}

		if !bytes.Equal(a.Balance[:], other.Balance[:]) {
			return false
		}
	}

	if (len(a.Code) != 0 || len(other.Code) != 0) && !bytes.Equal(a.Code, other.Code) {
		return false
	}

	if a.Nonce != nil || other.Nonce != nil {
		if a.Nonce == nil || other.Nonce == nil {
			return false
		}

		if *a.Nonce != *other.Nonce {
			return false
		}
	}

	if a.StorageWrites != nil || other.StorageWrites != nil {
		if a.StorageWrites == nil || other.StorageWrites == nil {
			return false
		}

		if !maps.Equal(a.StorageWrites, other.StorageWrites) {
			return false
		}
	}
	return true
}

type StateDiff struct {
	Mutations map[common.Address]*AccountState `json:"Mutations,omitempty"`
}

func (a *AccountState) Copy() *AccountState {
	res := NewEmptyAccountState()
	if a.Nonce != nil {
		res.Nonce = new(uint64)
		*res.Nonce = *a.Nonce
	}
	if a.Code != nil {
		res.Code = bytes.Clone(a.Code)
	}
	if a.Balance != nil {
		res.Balance = new(Balance)
		copy(res.Balance[:], (*a.Balance)[:])
	}
	if a.StorageWrites != nil {
		res.StorageWrites = maps.Clone(a.StorageWrites)
	}
	return res
}

// ValidateStateDiff asserts that both state diffs are equivalent.
func ValidateStateDiff(balDiff, computedDiff *StateDiff) error {
	for addr, computedAccountDiff := range computedDiff.Mutations {
		balAccountDiff, ok := balDiff.Mutations[addr]
		if !ok {
			return fmt.Errorf("missing from BAL")
		}

		if !computedAccountDiff.Eq(balAccountDiff) {
			return fmt.Errorf("mismatch between BAl value and computed value")
		}
	}
	if len(balDiff.Mutations) != len(computedDiff.Mutations) {
		return fmt.Errorf("BAL contained unexpected mutations compared to computed")
	}
	return nil
}

func (s *StateDiff) String() string {
	var res bytes.Buffer
	enc := json.NewEncoder(&res)
	enc.SetIndent("", "    ")
	enc.Encode(s)
	return res.String()
}

// Merge merges the state changes present in next into the caller.  After,
// the state of the caller is the aggregate diff through next.
func (s *StateDiff) Merge(next *StateDiff) {
	for account, diff := range next.Mutations {
		if mut, ok := s.Mutations[account]; ok {
			if diff.Balance != nil {
				mut.Balance = diff.Balance
			}
			if diff.Code != nil {
				mut.Code = diff.Code
			}
			if diff.Nonce != nil {
				mut.Nonce = diff.Nonce
			}
			if len(diff.StorageWrites) > 0 {
				if mut.StorageWrites == nil {
					mut.StorageWrites = maps.Clone(diff.StorageWrites)
				} else {
					for key, val := range diff.StorageWrites {
						mut.StorageWrites[key] = val
					}
				}

			}
		} else {
			s.Mutations[account] = diff.Copy()
		}
	}
}

func (s *StateDiff) Copy() *StateDiff {
	res := &StateDiff{make(map[common.Address]*AccountState)}
	for addr, accountDiff := range s.Mutations {
		cpy := accountDiff.Copy()
		res.Mutations[addr] = cpy
	}
	return res
}

// AccountIterator facilitates the iteration of an account's changes at each txindex in the BAL
type AccountIterator struct {
	address          common.Address
	slotWriteIndices [][]int
	balanceChangeIdx int
	nonceChangeIdx   int
	codeChangeIdx    int

	curTxIdx int
	maxIdx   int
	aa       *AccountAccess
}

func NewAccountIterator(accesses *AccountAccess, txCount int) *AccountIterator {
	slotWriteIndices := make([][]int, len(accesses.StorageWrites))
	for i, slotWrites := range accesses.StorageWrites {
		slotWriteIndices[i] = make([]int, len(slotWrites.Accesses))
	}
	return &AccountIterator{
		address:          accesses.Address,
		slotWriteIndices: slotWriteIndices,
		balanceChangeIdx: 0,
		nonceChangeIdx:   0,
		codeChangeIdx:    0,
		curTxIdx:         0,
		maxIdx:           txCount + 2,
		aa:               accesses,
	}
}

// Increment increments the account iterator by one, returning only the mutated state by the new transaction
func (it *AccountIterator) Increment() (accountState *AccountState, mut bool) {
	if it.curTxIdx > it.maxIdx {
		return nil, false
	}

	layerMut := NewEmptyAccountState()

	for i, accountSlotsIdxs := range it.slotWriteIndices {
		for j, curSlotIdx := range accountSlotsIdxs {
			if curSlotIdx < len(it.aa.StorageWrites[i].Accesses) {
				storageWrite := it.aa.StorageWrites[i].Accesses[curSlotIdx]
				if storageWrite.TxIdx == uint16(it.curTxIdx) {
					if layerMut.StorageWrites == nil {
						layerMut.StorageWrites = make(map[common.Hash]common.Hash)
					}
					layerMut.StorageWrites[it.aa.StorageWrites[i].Slot] = storageWrite.ValueAfter
					accountSlotsIdxs[j]++
				}
			}
		}
	}

	if it.balanceChangeIdx < len(it.aa.BalanceChanges) && it.aa.BalanceChanges[it.balanceChangeIdx].TxIdx == uint16(it.curTxIdx) {
		balance := it.aa.BalanceChanges[it.balanceChangeIdx].Balance
		layerMut.Balance = &balance
		it.balanceChangeIdx++
	}

	if it.codeChangeIdx < len(it.aa.CodeChanges) && it.aa.CodeChanges[it.codeChangeIdx].TxIndex == uint16(it.curTxIdx) {
		newCode := bytes.Clone(it.aa.CodeChanges[it.codeChangeIdx].Code)
		if newCode == nil {
			newCode = make([]byte, 0)
		}
		layerMut.Code = newCode
		it.codeChangeIdx++
	}

	if it.nonceChangeIdx < len(it.aa.NonceChanges) && it.aa.NonceChanges[it.nonceChangeIdx].TxIdx == uint16(it.curTxIdx) {
		layerMut.Nonce = new(uint64)
		*layerMut.Nonce = it.aa.NonceChanges[it.nonceChangeIdx].Nonce
		it.nonceChangeIdx++
	}
	it.curTxIdx++

	isMut := len(layerMut.StorageWrites) > 0 || layerMut.Code != nil || layerMut.Nonce != nil || layerMut.Balance != nil
	return layerMut, isMut
}

// BALIterator facilitates the txindex ordered iteration of an access list
// allowing for an access list to be converted into a set of ordered state diffs
// that correspond to each txindex.
type BALIterator struct {
	bal           *BlockAccessList
	acctIterators map[common.Address]*AccountIterator
	curIdx        uint16
	maxIdx        uint16
}

func NewIterator(b *BlockAccessList, txCount int) *BALIterator {
	accounts := make(map[common.Address]*AccountIterator)
	for _, aa := range b.Accesses {
		accounts[aa.Address] = NewAccountIterator(&aa, txCount)
	}
	return &BALIterator{
		b,
		accounts,
		0,
		uint16(txCount) + 2,
	}
}

// Next iterates one transaction into the BAL, returning the state diff from that tx
func (it *BALIterator) Next() (mutations *StateDiff) {
	if it.curIdx == it.maxIdx {
		return nil
	}
	diff := StateDiff{Mutations: make(map[common.Address]*AccountState)}
	for addr, acctIt := range it.acctIterators {
		acctMut, isMut := acctIt.Increment()
		if isMut {
			diff.Mutations[addr] = acctMut
		}
	}
	it.curIdx++
	return &diff
}

// BuildStateDiffs computes the ordered set of state diffs from an access list.
func BuildStateDiffs(bal *BlockAccessList, txCount int) []*StateDiff {
	stateDiffs := make([]*StateDiff, txCount+2)
	for i := 0; i < len(stateDiffs); i++ {
		stateDiffs[i] = &StateDiff{make(map[common.Address]*AccountState)}
	}

	for _, accountDiff := range bal.Accesses {

		if len(accountDiff.StorageWrites) > 0 {
			for _, storageWrites := range accountDiff.StorageWrites {
				for _, storageWrite := range storageWrites.Accesses {
					if _, ok := stateDiffs[storageWrite.TxIdx].Mutations[accountDiff.Address]; !ok {
						stateDiffs[storageWrite.TxIdx].Mutations[accountDiff.Address] = &AccountState{
							StorageWrites: make(map[common.Hash]common.Hash),
						}
					}
					if _, ok := stateDiffs[storageWrite.TxIdx].Mutations[accountDiff.Address]; !ok {
						stateDiffs[storageWrite.TxIdx].Mutations[accountDiff.Address] = &AccountState{}
					}
					stateDiffs[storageWrite.TxIdx].Mutations[accountDiff.Address].StorageWrites[storageWrites.Slot] = storageWrite.ValueAfter
				}
			}
		}
		if len(accountDiff.BalanceChanges) > 0 {
			for _, balanceChange := range accountDiff.BalanceChanges {
				if _, ok := stateDiffs[balanceChange.TxIdx].Mutations[accountDiff.Address]; !ok {
					stateDiffs[balanceChange.TxIdx].Mutations[accountDiff.Address] = &AccountState{}
				}
				var postBalance Balance
				copy(postBalance[:], balanceChange.Balance[:])
				stateDiffs[balanceChange.TxIdx].Mutations[accountDiff.Address].Balance = &postBalance
			}
		}

		if len(accountDiff.NonceChanges) > 0 {
			for _, nonceChange := range accountDiff.NonceChanges {
				if _, ok := stateDiffs[nonceChange.TxIdx].Mutations[accountDiff.Address]; !ok {
					stateDiffs[nonceChange.TxIdx].Mutations[accountDiff.Address] = &AccountState{}
				}
				if _, ok := stateDiffs[nonceChange.TxIdx].Mutations[accountDiff.Address]; !ok {
					stateDiffs[nonceChange.TxIdx].Mutations[accountDiff.Address] = &AccountState{}
				}

				newNonce := nonceChange.Nonce
				stateDiffs[nonceChange.TxIdx].Mutations[accountDiff.Address].Nonce = &newNonce
			}
		}

		if len(accountDiff.CodeChanges) > 0 {
			for _, codeChange := range accountDiff.CodeChanges {
				if _, ok := stateDiffs[codeChange.TxIndex].Mutations[accountDiff.Address]; !ok {
					stateDiffs[codeChange.TxIndex].Mutations[accountDiff.Address] = &AccountState{}
				}
				// TODO: rename TxIndex -> TxIdx (or vice versa with everything else)
				if _, ok := stateDiffs[codeChange.TxIndex].Mutations[accountDiff.Address]; !ok {
					stateDiffs[codeChange.TxIndex].Mutations[accountDiff.Address] = &AccountState{}
				}

				stateDiffs[codeChange.TxIndex].Mutations[accountDiff.Address].Code = codeChange.Code
			}
		}
	}

	return stateDiffs
}

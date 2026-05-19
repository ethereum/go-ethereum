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

//go:generate go run github.com/ethereum/go-ethereum/rlp/rlpgen -out bal_encoding_rlp_generated.go -type AccountAccess -decoder

// These are objects used as input for the access list encoding. They mirror
// the spec format.

// BlockAccessList is the encoding format of ConstructionBlockAccessList.
type BlockAccessList []AccountAccess

// EncodeRLP implements rlp.Encoder. It encodes the access list as a single
// RLP list of AccountAccess entries.
func (e BlockAccessList) EncodeRLP(w io.Writer) error {
	buf := rlp.NewEncoderBuffer(w)
	l := buf.List()
	for i := range e {
		if err := e[i].EncodeRLP(buf); err != nil {
			return err
		}
	}
	buf.ListEnd(l)
	return buf.Flush()
}

// DecodeRLP implements rlp.Decoder.
func (e *BlockAccessList) DecodeRLP(s *rlp.Stream) error {
	if _, err := s.List(); err != nil {
		return err
	}
	var list BlockAccessList
	for s.MoreDataInList() {
		var a AccountAccess
		if err := a.DecodeRLP(s); err != nil {
			return err
		}
		list = append(list, a)
	}
	if err := s.ListEnd(); err != nil {
		return err
	}
	*e = list
	return nil
}

// Validate returns an error if the contents of the access list are not ordered
// according to the spec or any code changes are contained which exceed protocol
// max code size.
func (e *BlockAccessList) Validate(rules params.Rules) error {
	if !slices.IsSortedFunc(*e, func(a, b AccountAccess) int {
		return bytes.Compare(a.Address[:], b.Address[:])
	}) {
		return errors.New("block access list accounts not in lexicographic order")
	}
	for _, entry := range *e {
		if err := entry.validate(rules); err != nil {
			return err
		}
	}
	return nil
}

// Hash computes the keccak256 hash of the access list
func (e *BlockAccessList) Hash() common.Hash {
	var enc bytes.Buffer
	if err := e.EncodeRLP(&enc); err != nil {
		// Errors here are related to BAL values exceeding maximum size defined
		// by the spec. Return empty hash because these cases are not expected
		// to be hit under reasonable conditions.
		return common.Hash{}
	}
	return crypto.Keccak256Hash(enc.Bytes())
}

// encodingBalanceChange is the encoding format of BalanceChange.
type encodingBalanceChange struct {
	TxIdx   uint32
	Balance *uint256.Int
}

// encodingAccountNonce is the encoding format of NonceChange.
type encodingAccountNonce struct {
	TxIdx uint32
	Nonce uint64
}

// encodingStorageWrite is the encoding format of StorageWrites.
type encodingStorageWrite struct {
	TxIdx      uint32
	ValueAfter *uint256.Int
}

// encodingStorageWrite is the encoding format of SlotWrites.
type encodingSlotWrites struct {
	Slot     *uint256.Int
	Accesses []encodingStorageWrite
}

// validate returns an instance of the encoding-representation slot writes in
// working representation.
func (e *encodingSlotWrites) validate() error {
	if slices.IsSortedFunc(e.Accesses, func(a, b encodingStorageWrite) int {
		return cmp.Compare[uint32](a.TxIdx, b.TxIdx)
	}) {
		return nil
	}
	return errors.New("storage write tx indices not in order")
}

// encodingCodeChange contains the runtime bytecode deployed at an address
// and the transaction index where the deployment took place.
type encodingCodeChange struct {
	TxIndex uint32
	Code    []byte
}

// AccountAccess is the encoding format of ConstructionAccountAccess.
type AccountAccess struct {
	Address        [20]byte                // 20-byte Ethereum address
	StorageWrites  []encodingSlotWrites    // Storage changes (slot -> [tx_index -> new_value])
	StorageReads   []*uint256.Int          // Read-only storage keys
	BalanceChanges []encodingBalanceChange // Balance changes ([tx_index -> post_balance])
	NonceChanges   []encodingAccountNonce  // Nonce changes ([tx_index -> new_nonce])
	CodeChanges    []encodingCodeChange    // Code changes ([tx_index -> new_code])
}

// validate converts the account accesses out of encoding format.
// If any of the keys in the encoding object are not ordered according to the
// spec, an error is returned.
func (e *AccountAccess) validate(rules params.Rules) error {
	// Check the storage write slots are sorted in order
	if !slices.IsSortedFunc(e.StorageWrites, func(a, b encodingSlotWrites) int {
		return a.Slot.Cmp(b.Slot)
	}) {
		return errors.New("storage writes slots not in lexicographic order")
	}
	for _, write := range e.StorageWrites {
		if err := write.validate(); err != nil {
			return err
		}
	}

	// Check the storage read slots are sorted in order
	if !slices.IsSortedFunc(e.StorageReads, func(a, b *uint256.Int) int {
		return a.Cmp(b)
	}) {
		return errors.New("storage read slots not in lexicographic order")
	}

	// Check the balance changes are sorted in order
	if !slices.IsSortedFunc(e.BalanceChanges, func(a, b encodingBalanceChange) int {
		return cmp.Compare[uint32](a.TxIdx, b.TxIdx)
	}) {
		return errors.New("balance changes not in ascending order by tx index")
	}

	// Check the nonce changes are sorted in order
	if !slices.IsSortedFunc(e.NonceChanges, func(a, b encodingAccountNonce) int {
		return cmp.Compare[uint32](a.TxIdx, b.TxIdx)
	}) {
		return errors.New("nonce changes not in ascending order by tx index")
	}

	// Check the code changes are sorted in order
	if !slices.IsSortedFunc(e.CodeChanges, func(a, b encodingCodeChange) int {
		return cmp.Compare[uint32](a.TxIndex, b.TxIndex)
	}) {
		return errors.New("code changes not in ascending order by tx index")
	}
	for _, change := range e.CodeChanges {
		var sizeLimit int
		switch {
		case rules.IsAmsterdam:
			sizeLimit = params.MaxCodeSizeAmsterdam
		default:
			sizeLimit = params.MaxCodeSize
		}
		if len(change.Code) > sizeLimit {
			return errors.New("code change contained oversized code")
		}
	}
	return nil
}

// Copy returns a deep copy of the account access
func (e *AccountAccess) Copy() AccountAccess {
	res := AccountAccess{
		Address:        e.Address,
		StorageReads:   make([]*uint256.Int, 0, len(e.StorageReads)),
		BalanceChanges: make([]encodingBalanceChange, 0, len(e.BalanceChanges)),
		NonceChanges:   slices.Clone(e.NonceChanges),
		StorageWrites:  make([]encodingSlotWrites, 0, len(e.StorageWrites)),
		CodeChanges:    make([]encodingCodeChange, 0, len(e.CodeChanges)),
	}
	for _, slot := range e.StorageReads {
		res.StorageReads = append(res.StorageReads, slot.Clone())
	}
	for _, change := range e.BalanceChanges {
		res.BalanceChanges = append(res.BalanceChanges, encodingBalanceChange{
			TxIdx:   change.TxIdx,
			Balance: change.Balance.Clone(),
		})
	}
	for _, storageWrite := range e.StorageWrites {
		accesses := make([]encodingStorageWrite, 0, len(storageWrite.Accesses))
		for _, w := range storageWrite.Accesses {
			accesses = append(accesses, encodingStorageWrite{
				TxIdx:      w.TxIdx,
				ValueAfter: w.ValueAfter.Clone(),
			})
		}
		res.StorageWrites = append(res.StorageWrites, encodingSlotWrites{
			Slot:     storageWrite.Slot.Clone(),
			Accesses: accesses,
		})
	}
	for _, codeChange := range e.CodeChanges {
		res.CodeChanges = append(res.CodeChanges, encodingCodeChange{
			TxIndex: codeChange.TxIndex,
			Code:    bytes.Clone(codeChange.Code),
		})
	}
	return res
}

// EncodeRLP returns the RLP-encoded access list
func (b *ConstructionBlockAccessList) EncodeRLP(wr io.Writer) error {
	return b.toEncodingObj().EncodeRLP(wr)
}

var _ rlp.Encoder = &ConstructionBlockAccessList{}

// toEncodingObj creates an instance of the ConstructionAccountAccess of the type
// that is used as input for the encoding.
func (a *ConstructionAccountAccess) toEncodingObj(addr common.Address) AccountAccess {
	res := AccountAccess{
		Address:        addr,
		StorageWrites:  make([]encodingSlotWrites, 0, len(a.StorageWrites)),
		StorageReads:   make([]*uint256.Int, 0, len(a.StorageReads)),
		BalanceChanges: make([]encodingBalanceChange, 0, len(a.BalanceChanges)),
		NonceChanges:   make([]encodingAccountNonce, 0, len(a.NonceChanges)),
		CodeChanges:    make([]encodingCodeChange, 0, len(a.CodeChange)),
	}

	// Convert write slots
	writeSlots := slices.Collect(maps.Keys(a.StorageWrites))
	slices.SortFunc(writeSlots, common.Hash.Cmp)
	for _, slot := range writeSlots {
		obj := encodingSlotWrites{
			Slot: new(uint256.Int).SetBytes(slot[:]),
		}
		slotWrites := a.StorageWrites[slot]
		obj.Accesses = make([]encodingStorageWrite, 0, len(slotWrites))

		indices := slices.Collect(maps.Keys(slotWrites))
		slices.SortFunc(indices, cmp.Compare[uint32])
		for _, index := range indices {
			val := slotWrites[index]
			obj.Accesses = append(obj.Accesses, encodingStorageWrite{
				TxIdx:      index,
				ValueAfter: new(uint256.Int).SetBytes(val[:]),
			})
		}
		res.StorageWrites = append(res.StorageWrites, obj)
	}

	// Convert read slots
	readSlots := slices.Collect(maps.Keys(a.StorageReads))
	slices.SortFunc(readSlots, common.Hash.Cmp)
	for _, slot := range readSlots {
		res.StorageReads = append(res.StorageReads, new(uint256.Int).SetBytes(slot[:]))
	}

	// Convert balance changes
	balanceIndices := slices.Collect(maps.Keys(a.BalanceChanges))
	slices.SortFunc(balanceIndices, cmp.Compare[uint32])
	for _, idx := range balanceIndices {
		res.BalanceChanges = append(res.BalanceChanges, encodingBalanceChange{
			TxIdx:   idx,
			Balance: a.BalanceChanges[idx].Clone(),
		})
	}

	// Convert nonce changes
	nonceIndices := slices.Collect(maps.Keys(a.NonceChanges))
	slices.SortFunc(nonceIndices, cmp.Compare[uint32])
	for _, idx := range nonceIndices {
		res.NonceChanges = append(res.NonceChanges, encodingAccountNonce{
			TxIdx: idx,
			Nonce: a.NonceChanges[idx],
		})
	}

	// Convert code change
	codeIndices := slices.Collect(maps.Keys(a.CodeChange))
	slices.SortFunc(codeIndices, cmp.Compare[uint32])
	for _, idx := range codeIndices {
		res.CodeChanges = append(res.CodeChanges, encodingCodeChange{
			TxIndex: idx,

			// TODO(rjl493456442) the contract code is not deep-copied.
			// In theory the deep-copy is unnecessary, the semantics of
			// the function should be probably changed that the returned
			// AccessList is unsafe for modification.
			Code: a.CodeChange[idx],
		})
	}
	return res
}

// toEncodingObj returns an instance of the access list expressed as the type
// which is used as input for the encoding/decoding.
func (b *ConstructionBlockAccessList) toEncodingObj() *BlockAccessList {
	var addresses []common.Address
	for addr := range b.Accounts {
		addresses = append(addresses, addr)
	}
	slices.SortFunc(addresses, common.Address.Cmp)

	res := make(BlockAccessList, 0, len(addresses))
	for _, addr := range addresses {
		res = append(res, b.Accounts[addr].toEncodingObj(addr))
	}
	return &res
}

func (e *BlockAccessList) PrettyPrint() string {
	var res bytes.Buffer
	printWithIndent := func(indent int, text string) {
		fmt.Fprintf(&res, "%s%s\n", strings.Repeat("    ", indent), text)
	}
	for _, accountDiff := range *e {
		printWithIndent(0, fmt.Sprintf("%x:", accountDiff.Address))

		printWithIndent(1, "storage writes:")
		for _, sWrite := range accountDiff.StorageWrites {
			printWithIndent(2, fmt.Sprintf("%s:", sWrite.Slot.Hex()))
			for _, access := range sWrite.Accesses {
				printWithIndent(3, fmt.Sprintf("%d: %s", access.TxIdx, access.ValueAfter.Hex()))
			}
		}

		printWithIndent(1, "storage reads:")
		for _, slot := range accountDiff.StorageReads {
			printWithIndent(2, slot.Hex())
		}

		printWithIndent(1, "balance changes:")
		for _, change := range accountDiff.BalanceChanges {
			printWithIndent(2, fmt.Sprintf("%d: %s", change.TxIdx, change.Balance))
		}

		printWithIndent(1, "nonce changes:")
		for _, change := range accountDiff.NonceChanges {
			printWithIndent(2, fmt.Sprintf("%d: %d", change.TxIdx, change.Nonce))
		}

		printWithIndent(1, "code changes:")
		for _, change := range accountDiff.CodeChanges {
			printWithIndent(2, fmt.Sprintf("%d: %x", change.TxIndex, change.Code))
		}
	}
	return res.String()
}

// Copy returns a deep copy of the access list
func (e *BlockAccessList) Copy() *BlockAccessList {
	cpy := make(BlockAccessList, 0, len(*e))
	for _, accountAccess := range *e {
		cpy = append(cpy, accountAccess.Copy())
	}
	return &cpy
}

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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

//go:generate go run github.com/ethereum/go-ethereum/rlp/rlpgen -out bal_encoding_rlp_generated.go -type AccountAccess -decoder

// These are objects used as input for the access list encoding. They mirror
// the spec format.

// BlockAccessList is the encoding format of AccessListBuilder.
type BlockAccessList []AccountAccess

func (e BlockAccessList) EncodeRLP(_w io.Writer) error {
	w := rlp.NewEncoderBuffer(_w)
	l := w.List()
	for _, access := range e {
		access.EncodeRLP(w)
	}
	w.ListEnd(l)
	return w.Flush()
}

func (e *BlockAccessList) DecodeRLP(dec *rlp.Stream) error {
	if _, err := dec.List(); err != nil {
		return err
	}
	*e = (*e)[:0]
	for dec.MoreDataInList() {
		var access AccountAccess
		if err := access.DecodeRLP(dec); err != nil {
			return err
		}
		*e = append(*e, access)
	}
	dec.ListEnd()
	return nil
}

// StringableRepresentation returns an instance of the block access list
// which can be converted to a human-readable JSON representation.
func (e *BlockAccessList) StringableRepresentation() interface{} {
	res := []AccountAccess{}
	for _, aa := range *e {
		res = append(res, aa)
	}
	return &res
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
func (e BlockAccessList) Validate() error {
	if !slices.IsSortedFunc(e, func(a, b AccountAccess) int {
		return bytes.Compare(a.Address[:], b.Address[:])
	}) {
		return errors.New("block access list accounts not in lexicographic order")
	}
	for _, entry := range e {
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

// encodingBalanceChange is the encoding format of BalanceChange.
type encodingBalanceChange struct {
	TxIdx   uint16       `json:"txIndex"`
	Balance *uint256.Int `json:"balance"`
}

// encodingAccountNonce is the encoding format of NonceChange.
type encodingAccountNonce struct {
	TxIdx uint16 `json:"txIndex"`
	Nonce uint64 `json:"nonce"`
}

// encodingStorageWrite is the encoding format of StorageWrites.
type encodingStorageWrite struct {
	TxIdx      uint16      `json:"txIndex"`
	ValueAfter common.Hash `json:"valueAfter"`
}

// encodingStorageWrite is the encoding format of SlotWrites.
type encodingSlotWrites struct {
	Slot     common.Hash            `json:"slot"`
	Accesses []encodingStorageWrite `json:"accesses"`
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

// AccountAccess is the encoding format of ConstructionAccountAccesses.
type AccountAccess struct {
	Address        common.Address          `json:"address,omitempty"`        // 20-byte Ethereum address
	StorageChanges []encodingSlotWrites    `json:"storageChanges,omitempty"` // Storage changes (slot -> [tx_index -> new_value])
	StorageReads   []common.Hash           `json:"storageReads,omitempty"`   // Read-only storage keys
	BalanceChanges []encodingBalanceChange `json:"balanceChanges,omitempty"` // Balance changes ([tx_index -> post_balance])
	NonceChanges   []encodingAccountNonce  `json:"nonceChanges,omitempty"`   // Nonce changes ([tx_index -> new_nonce])
	CodeChanges    []CodeChange            `json:"code,omitempty"`           // CodeChanges changes ([tx_index -> new_code])
}

// validate converts the account accesses out of encoding format.
// If any of the keys in the encoding object are not ordered according to the
// spec, an error is returned.
func (e *AccountAccess) validate() error {
	// Check the storage write slots are sorted in order
	if !slices.IsSortedFunc(e.StorageChanges, func(a, b encodingSlotWrites) int {
		return bytes.Compare(a.Slot[:], b.Slot[:])
	}) {
		return errors.New("storage writes slots not in lexicographic order")
	}
	for _, write := range e.StorageChanges {
		if err := write.validate(); err != nil {
			return err
		}
	}
	// test case ideas: keys in both read/writes, duplicate keys in either read/writes
	// ensure that the read and write key sets are distinct
	readKeys := make(map[common.Hash]struct{})
	writeKeys := make(map[common.Hash]struct{})
	for _, readKey := range e.StorageReads {
		if _, ok := readKeys[readKey]; ok {
			return errors.New("duplicate read key")
		}
		readKeys[readKey] = struct{}{}
	}
	for _, write := range e.StorageChanges {
		writeKey := write.Slot
		if _, ok := writeKeys[writeKey]; ok {
			return errors.New("duplicate write key")
		}
		writeKeys[writeKey] = struct{}{}
	}

	for readKey := range readKeys {
		if _, ok := writeKeys[readKey]; ok {
			return errors.New("storage key reported in both read/write sets")
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
	for _, storageWrite := range e.StorageChanges {
		res.StorageChanges = append(res.StorageChanges, encodingSlotWrites{
			Slot:     storageWrite.Slot,
			Accesses: slices.Clone(storageWrite.Accesses),
		})
	}
	for _, codeChange := range e.CodeChanges {
		res.CodeChanges = append(res.CodeChanges,
			CodeChange{
				codeChange.TxIdx,
				bytes.Clone(codeChange.Code),
			})
	}
	return res
}

// EncodeRLP returns the RLP-encoded access list
func (c *AccessListBuilder) EncodeRLP(wr io.Writer) error {
	return c.ToEncodingObj().EncodeRLP(wr)
}

var _ rlp.Encoder = &AccessListBuilder{}

// toEncodingObj creates an instance of the ConstructionAccountAccesses of the type that is
// used as input for the encoding.
func (a *ConstructionAccountAccesses) toEncodingObj(addr common.Address) AccountAccess {
	res := AccountAccess{
		Address:        addr,
		StorageChanges: make([]encodingSlotWrites, 0),
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
		res.StorageChanges = append(res.StorageChanges, obj)
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
			Balance: new(uint256.Int).Set(a.BalanceChanges[idx]),
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
func (c *AccessListBuilder) ToEncodingObj() *BlockAccessList {
	var addresses []common.Address
	for addr := range c.FinalizedAccesses {
		addresses = append(addresses, addr)
	}
	slices.SortFunc(addresses, common.Address.Cmp)

	var res BlockAccessList
	for _, addr := range addresses {
		res = append(res, c.FinalizedAccesses[addr].toEncodingObj(addr))
	}
	return &res
}

type ContractCode []byte

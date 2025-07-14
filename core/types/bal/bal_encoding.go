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

//go:generate go run github.com/ethereum/go-ethereum/rlp/rlpgen -out bal_encoding_rlp_generated.go -type BlockAccessList -decoder

// These are objects used as input for the access list encoding. They mirror
// the spec format.

// BlockAccessList is the encoding format of ConstructionBlockAccessList.
type BlockAccessList struct {
	Accesses []AccountAccess `ssz-max:"300000" json:"accesses"`
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

// encodingBalanceChange is the encoding format of BalanceChange.
type encodingBalanceChange struct {
	TxIdx   uint16   `ssz-size:"2" json:"txIndex"`
	Balance [16]byte `ssz-size:"16" json:"balance"`
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
	Code           []CodeChange            `ssz-max:"1" json:"code,omitempty"`                // Code changes ([tx_index -> new_code])
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
	if len(e.Code) == 1 {
		if len(e.Code[0].Code) > params.MaxCodeSize {
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
	if len(e.Code) == 1 {
		res.Code = []CodeChange{
			{
				e.Code[0].TxIndex,
				bytes.Clone(e.Code[0].Code),
			},
		}
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
		Code:           nil,
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
	if a.CodeChange != nil {
		res.Code = []CodeChange{
			{
				a.CodeChange.TxIndex,
				bytes.Clone(a.CodeChange.Code),
			},
		}
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

		if len(accountDiff.Code) > 0 {
			printWithIndent(1, "code:")
			printWithIndent(2, fmt.Sprintf("%d: %x", accountDiff.Code[0].TxIndex, accountDiff.Code[0].Code))
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

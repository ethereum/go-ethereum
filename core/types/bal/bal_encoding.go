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
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

//go:generate go run github.com/ferranbt/fastssz/sszgen  --output bal_encoding_ssz_generated.go --path . --objs encodingStorageWrite,encodingCodeChange,encodingBalanceChange,encodingAccountNonce,encodingAccountAccess,encodingBlockAccessList
//go:generate go run github.com/ethereum/go-ethereum/rlp/rlpgen -out bal_encoding_rlp_generated.go -type encodingBlockAccessList -decoder

// These are objects used as input for the access list encoding. They mirror
// the spec format.

// encodingBlockAccessList is the encoding format of BlockAccessList.
type encodingBlockAccessList struct {
	Accesses []encodingAccountAccess `ssz-max:"300000"`
}

// toBlockAccessList converts out of the encoding format, returning an error if
// values in the encoder object are not properly ordered according to the spec.
func (e *encodingBlockAccessList) toBlockAccessList() (*BlockAccessList, error) {
	var (
		obj  = NewBlockAccessList()
		prev *[20]byte
	)
	for _, entry := range e.Accesses {
		if prev != nil {
			if bytes.Compare(entry.Address[:], (*prev)[:]) <= 0 {
				return nil, fmt.Errorf("block access list accounts not in lexicographic order")
			}
		}
		prev = &entry.Address

		aa, err := entry.toAccountAccess()
		if err != nil {
			return nil, err
		}
		obj.Accounts[entry.Address] = aa
	}
	return &obj, nil
}

// encodingCodeChange is the encoding format of CodeChange.
type encodingCodeChange struct {
	TxIndex uint16
	Code    []byte `ssz-max:"24576"`
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
	TxIdx   uint16   `ssz-size:"2"`
	Balance [16]byte `ssz-size:"16"`
}

// encodingAccountNonce is the encoding format of NonceChange.
type encodingAccountNonce struct {
	TxIdx uint16 `ssz-size:"2"`
	Nonce uint64 `ssz-size:"8"`
}

// encodingStorageWrite is the encoding format of StorageWrites.
type encodingStorageWrite struct {
	TxIdx      uint16
	ValueAfter [32]byte `ssz-size:"32"`
}

// encodingStorageWrite is the encoding format of SlotWrites.
type encodingSlotWrites struct {
	Slot     [32]byte               `ssz-size:"32"`
	Accesses []encodingStorageWrite `ssz-max:"300000"`
}

// toSlotWrites returns an instance of the encoding-representation slot writes in
// working representation.
func (e *encodingSlotWrites) toSlotWrites() (map[uint16]common.Hash, error) {
	var (
		prev *uint16
		obj  = make(map[uint16]common.Hash)
	)
	for _, write := range e.Accesses {
		if prev != nil {
			if *prev >= write.TxIdx {
				return nil, fmt.Errorf("storage write tx indices not in order")
			}
		}
		prev = &write.TxIdx
		obj[write.TxIdx] = write.ValueAfter
	}
	return obj, nil
}

// encodingAccountAccess is the encoding format of AccountAccess.
type encodingAccountAccess struct {
	Address        [20]byte                `ssz-size:"20"`    // 20-byte Ethereum address
	StorageWrites  []encodingSlotWrites    `ssz-max:"300000"` // Storage changes (slot -> [tx_index -> new_value])
	StorageReads   [][32]byte              `ssz-max:"300000"` // Read-only storage keys
	BalanceChanges []encodingBalanceChange `ssz-max:"300000"` // Balance changes ([tx_index -> post_balance])
	NonceChanges   []encodingAccountNonce  `ssz-max:"300000"` // Nonce changes ([tx_index -> new_nonce])
	Code           []encodingCodeChange    `ssz-max:"1"`      // Code changes ([tx_index -> new_code])
}

// toAccountAccess converts the account accesses out of encoding format.
// If any of the keys in the encoding object are not ordered according to the
// spec, an error is returned.
func (e *encodingAccountAccess) toAccountAccess() (*AccountAccess, error) {
	res := AccountAccess{
		StorageWrites:  make(map[common.Hash]map[uint16]common.Hash),
		StorageReads:   make(map[common.Hash]struct{}),
		BalanceChanges: make(map[uint16]*uint256.Int),
		NonceChanges:   make(map[uint16]uint64),
		CodeChange:     nil,
	}

	// Convert slot writes
	var prevSlotWrite *[32]byte
	for _, write := range e.StorageWrites {
		if prevSlotWrite != nil {
			if bytes.Compare((*prevSlotWrite)[:], write.Slot[:]) >= 0 {
				return nil, fmt.Errorf("storage writes slots not in lexicographic order")
			}
		}
		prevSlotWrite = &write.Slot

		wr, err := write.toSlotWrites()
		if err != nil {
			return nil, err
		}
		res.StorageWrites[write.Slot] = wr
	}

	// Convert slot reads
	var prevSlotRead *[32]byte
	for _, read := range e.StorageReads {
		if prevSlotRead != nil {
			if bytes.Compare((*prevSlotRead)[:], read[:]) >= 0 {
				return nil, fmt.Errorf("storage read slots not in lexicographic order")
			}
		}
		prevSlotRead = &read
		res.StorageReads[read] = struct{}{}
	}

	// Convert balance changes
	var prevBalanceIndex *uint16
	for _, balanceChange := range e.BalanceChanges {
		if prevBalanceIndex != nil {
			if *prevBalanceIndex >= balanceChange.TxIdx {
				return nil, fmt.Errorf("balance changes not in ascending order by tx index")
			}
		}
		prevBalanceIndex = &balanceChange.TxIdx
		res.BalanceChanges[balanceChange.TxIdx] = new(uint256.Int).SetBytes(balanceChange.Balance[:])
	}

	// Convert nonce changes
	var prevNonceIndex *uint16
	for _, nonceChange := range e.NonceChanges {
		if prevNonceIndex != nil {
			if *prevNonceIndex >= nonceChange.TxIdx {
				return nil, fmt.Errorf("nonce diffs not in ascending order by tx index")
			}
		}
		prevNonceIndex = &nonceChange.TxIdx
		res.NonceChanges[nonceChange.TxIdx] = nonceChange.Nonce
	}

	// Convert code change
	if len(e.Code) == 1 {
		res.CodeChange = &CodeChange{
			TxIndex: e.Code[0].TxIndex,
			Code:    bytes.Clone(e.Code[0].Code),
		}
	}
	return &res, nil
}

// EncodeRLP returns the SSZ-encoded access list wrapped into RLP bytes.
func (b *BlockAccessList) EncodeRLP(wr io.Writer) error {
	w := rlp.NewEncoderBuffer(wr)
	buf, err := b.encodeSSZ()
	if err != nil {
		return err
	}
	w.WriteBytes(buf)
	return w.Flush()
}

// DecodeRLP decodes the access list
func (b *BlockAccessList) DecodeRLP(s *rlp.Stream) error {
	encBytes, err := s.Bytes()
	if err != nil {
		return err
	}
	return b.decodeSSZ(encBytes)
}

var _ rlp.Encoder = &BlockAccessList{}
var _ rlp.Decoder = &BlockAccessList{}

// toEncodingObj creates an instance of the AccountAccess of the type that is
// used as input for the encoding.
func (a *AccountAccess) toEncodingObj(addr common.Address) encodingAccountAccess {
	res := encodingAccountAccess{
		Address:        addr,
		StorageWrites:  make([]encodingSlotWrites, 0),
		StorageReads:   make([][32]byte, 0),
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
		res.Code = []encodingCodeChange{
			{
				a.CodeChange.TxIndex,
				bytes.Clone(a.CodeChange.Code),
			},
		}
	}
	return res
}

// toEncodingObj returns an instance of the access list expressed as the type
// which is used as input for the encoding/decoding.
func (b *BlockAccessList) toEncodingObj() *encodingBlockAccessList {
	var addresses []common.Address
	for addr := range b.Accounts {
		addresses = append(addresses, addr)
	}
	slices.SortFunc(addresses, common.Address.Cmp)

	var res encodingBlockAccessList
	for _, addr := range addresses {
		res.Accesses = append(res.Accesses, b.Accounts[addr].toEncodingObj(addr))
	}
	return &res
}

func (b *BlockAccessList) encodeSSZ() ([]byte, error) {
	encoderObj := b.toEncodingObj()
	dst, err := encoderObj.MarshalSSZTo(nil)
	if err != nil {
		return nil, err
	}
	return dst, nil
}

func (b *BlockAccessList) decodeSSZ(buf []byte) error {
	var enc encodingBlockAccessList
	if err := enc.UnmarshalSSZ(buf); err != nil {
		return err
	}
	res, err := enc.toBlockAccessList()
	if err != nil {
		return err
	}
	*b = *res
	return nil
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

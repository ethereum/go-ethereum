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
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
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

func (e *BlockAccessList) EncodedSize() int {
	b, err := rlp.EncodeToBytes(e)
	if err != nil {
		// TODO: proper to crit here?
		log.Crit("failed to rlp encode access list", "err", err)
	}
	return len(b)
}

func (e *BlockAccessList) JSONString() string {
	res, _ := json.MarshalIndent(e.StringableRepresentation(), "", "    ")
	return string(res)
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

// TODO: check that no fields are nil in Validate (unless it's valid for them to be nil)
// Validate returns an error if the contents of the access list are not ordered
// according to the spec or any code changes are contained which exceed protocol
// max code size.
func (e BlockAccessList) Validate(blockTxCount int) error {
	if !slices.IsSortedFunc(e, func(a, b AccountAccess) int {
		return bytes.Compare(a.Address[:], b.Address[:])
	}) {
		return errors.New("block access list accounts not in lexicographic order")
	}
	// check that the accounts are unique
	addrs := make(map[common.Address]struct{})
	for _, acct := range e {
		addr := acct.Address
		if _, ok := addrs[addr]; ok {
			return fmt.Errorf("duplicate account in block access list: %x", addr)
		}
		addrs[addr] = struct{}{}
	}

	for _, entry := range e {
		if err := entry.validate(blockTxCount); err != nil {
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
	/*
		bal, err := json.MarshalIndent(e.StringableRepresentation(), "", "    ")
		if err != nil {
			panic(err)
		}
	*/
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
	TxIdx      uint16          `json:"txIndex"`
	ValueAfter *EncodedStorage `json:"valueAfter"`
}

// EncodedStorage can represent either a storage key or value
type EncodedStorage struct {
	inner *uint256.Int
}

var _ rlp.Encoder = &EncodedStorage{}
var _ rlp.Decoder = &EncodedStorage{}

func (e *EncodedStorage) ToHash() common.Hash {
	if e == nil {
		return common.Hash{}
	}
	return e.inner.Bytes32()
}

func newEncodedStorageFromHash(hash common.Hash) *EncodedStorage {
	return &EncodedStorage{
		new(uint256.Int).SetBytes(hash[:]),
	}
}

func (s *EncodedStorage) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}

	str = strings.TrimLeft(str, "0x")
	if len(str) == 0 {
		return nil
	}

	if len(str)%2 == 1 {
		str = "0" + str
	}

	val, err := hex.DecodeString(str)
	if err != nil {
		return err
	}

	if len(val) > 32 {
		return fmt.Errorf("storage key/value cannot be greater than 32 bytes")
	}

	// TODO: check is s == nil ?? should be programmer error

	*s = EncodedStorage{
		inner: new(uint256.Int).SetBytes(val),
	}
	return nil
}

func (s EncodedStorage) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.inner.Hex())
}

func (s *EncodedStorage) EncodeRLP(_w io.Writer) error {
	return s.inner.EncodeRLP(_w)
}

func (s *EncodedStorage) DecodeRLP(dec *rlp.Stream) error {
	if s == nil {
		*s = EncodedStorage{}
	}
	s.inner = uint256.NewInt(0)
	return dec.ReadUint256(s.inner)
}

// encodingStorageWrite is the encoding format of SlotWrites.
type encodingSlotWrites struct {
	Slot     *EncodedStorage        `json:"slot"`
	Accesses []encodingStorageWrite `json:"accesses"`
}

// validate returns an instance of the encoding-representation slot writes in
// working representation.
func (e *encodingSlotWrites) validate(blockTxCount int) error {
	if !slices.IsSortedFunc(e.Accesses, func(a, b encodingStorageWrite) int {
		return cmp.Compare[uint16](a.TxIdx, b.TxIdx)
	}) {
		return errors.New("storage write tx indices not in order")
	}
	// TODO: add test that covers there are actually storage modifications here
	// if there aren't, it should be a bad block
	if len(e.Accesses) == 0 {
		return fmt.Errorf("empty storage writes")
	} else if int(e.Accesses[len(e.Accesses)-1].TxIdx) >= blockTxCount+2 {
		return fmt.Errorf("storage access reported index higher than allowed")
	}
	return nil
}

// AccountAccess is the encoding format of ConstructionAccountAccesses.
type AccountAccess struct {
	Address        common.Address          `json:"address,omitempty"`        // 20-byte Ethereum address
	StorageChanges []encodingSlotWrites    `json:"storageChanges,omitempty"` // EncodedStorage changes (slot -> [tx_index -> new_value])
	StorageReads   []*EncodedStorage       `json:"storageReads,omitempty"`   // Read-only storage keys
	BalanceChanges []encodingBalanceChange `json:"balanceChanges,omitempty"` // Balance changes ([tx_index -> post_balance])
	NonceChanges   []encodingAccountNonce  `json:"nonceChanges,omitempty"`   // Nonce changes ([tx_index -> new_nonce])
	CodeChanges    []CodeChange            `json:"code,omitempty"`           // CodeChanges changes ([tx_index -> new_code])
}

// validate converts the account accesses out of encoding format.
// If any of the keys in the encoding object are not ordered according to the
// spec, an error is returned.
func (e *AccountAccess) validate(blockTxCount int) error {
	// Check the storage write slots are sorted in order
	if !slices.IsSortedFunc(e.StorageChanges, func(a, b encodingSlotWrites) int {
		aHash, bHash := a.Slot.ToHash(), b.Slot.ToHash()
		return bytes.Compare(aHash[:], bHash[:])
	}) {
		return errors.New("storage writes slots not in lexicographic order")
	}
	for _, write := range e.StorageChanges {
		if err := write.validate(blockTxCount); err != nil {
			return err
		}
	}
	readKeys := make(map[common.Hash]struct{})
	writeKeys := make(map[common.Hash]struct{})
	for _, readKey := range e.StorageReads {
		if _, ok := readKeys[readKey.ToHash()]; ok {
			return errors.New("duplicate read key")
		}
		readKeys[readKey.ToHash()] = struct{}{}
	}
	for _, write := range e.StorageChanges {
		writeKey := write.Slot
		if _, ok := writeKeys[writeKey.ToHash()]; ok {
			return errors.New("duplicate write key")
		}
		writeKeys[writeKey.ToHash()] = struct{}{}
	}

	for readKey := range readKeys {
		if _, ok := writeKeys[readKey]; ok {
			return errors.New("storage key reported in both read/write sets")
		}
	}

	// Check the storage read slots are sorted in order
	if !slices.IsSortedFunc(e.StorageReads, func(a, b *EncodedStorage) int {
		aHash, bHash := a.ToHash(), b.ToHash()
		return bytes.Compare(aHash[:], bHash[:])
	}) {
		return errors.New("storage read slots not in lexicographic order")
	}

	// Check the balance changes are sorted in order
	// and that none of them report an index above what is allowed
	if !slices.IsSortedFunc(e.BalanceChanges, func(a, b encodingBalanceChange) int {
		return cmp.Compare[uint16](a.TxIdx, b.TxIdx)
	}) {
		return errors.New("balance changes not in ascending order by tx index")
	}

	if len(e.BalanceChanges) > 0 && int(e.BalanceChanges[len(e.BalanceChanges)-1].TxIdx) > blockTxCount+2 {
		return errors.New("highest balance change index beyond what is allowed")
	}
	// Check the nonce changes are sorted in order
	// and that none of them report an index above what is allowed
	if !slices.IsSortedFunc(e.NonceChanges, func(a, b encodingAccountNonce) int {
		return cmp.Compare[uint16](a.TxIdx, b.TxIdx)
	}) {
		return errors.New("nonce changes not in ascending order by tx index")
	}
	if len(e.CodeChanges) > 0 && int(e.NonceChanges[len(e.NonceChanges)-1].TxIdx) >= blockTxCount+2 {
		return errors.New("highest nonce change index beyond what is allowed")
	}

	// TODO: contact testing team to add a test case which has the code changes out of order,
	// as it wasn't checked here previously
	if !slices.IsSortedFunc(e.CodeChanges, func(a, b CodeChange) int {
		return cmp.Compare[uint16](a.TxIdx, b.TxIdx)
	}) {
		return errors.New("code changes not in ascending order")
	}
	if len(e.CodeChanges) > 0 && int(e.CodeChanges[len(e.CodeChanges)-1].TxIdx) >= blockTxCount+2 {
		return errors.New("highest code change index beyond what is allowed")
	}

	// validate that code changes could plausibly be correct (none exceed
	// max code size of a contract)
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
func (c ConstructionBlockAccessList) EncodeRLP(wr io.Writer) error {
	return c.ToEncodingObj().EncodeRLP(wr)
}

var _ rlp.Encoder = &ConstructionBlockAccessList{}

// toEncodingObj creates an instance of the ConstructionAccountAccesses of the type that is
// used as input for the encoding.
func (a *ConstructionAccountAccesses) toEncodingObj(addr common.Address) AccountAccess {
	res := AccountAccess{
		Address:        addr,
		StorageChanges: make([]encodingSlotWrites, 0),
		StorageReads:   make([]*EncodedStorage, 0),
		BalanceChanges: make([]encodingBalanceChange, 0),
		NonceChanges:   make([]encodingAccountNonce, 0),
		CodeChanges:    make([]CodeChange, 0),
	}

	// Convert write slots
	writeSlots := slices.Collect(maps.Keys(a.StorageWrites))
	slices.SortFunc(writeSlots, common.Hash.Cmp)
	for _, slot := range writeSlots {
		var obj encodingSlotWrites
		obj.Slot = newEncodedStorageFromHash(slot)

		slotWrites := a.StorageWrites[slot]
		obj.Accesses = make([]encodingStorageWrite, 0, len(slotWrites))

		indices := slices.Collect(maps.Keys(slotWrites))
		slices.SortFunc(indices, cmp.Compare[uint16])
		for _, index := range indices {
			obj.Accesses = append(obj.Accesses, encodingStorageWrite{
				TxIdx:      index,
				ValueAfter: newEncodedStorageFromHash(slotWrites[index]),
			})
		}
		res.StorageChanges = append(res.StorageChanges, obj)
	}

	// Convert read slots
	readSlots := slices.Collect(maps.Keys(a.StorageReads))
	slices.SortFunc(readSlots, common.Hash.Cmp)
	for _, slot := range readSlots {
		res.StorageReads = append(res.StorageReads, newEncodedStorageFromHash(slot))
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
func (c ConstructionBlockAccessList) ToEncodingObj() *BlockAccessList {
	var addresses []common.Address
	for addr := range c {
		addresses = append(addresses, addr)
	}
	slices.SortFunc(addresses, common.Address.Cmp)

	var res BlockAccessList
	for _, addr := range addresses {
		res = append(res, c[addr].toEncodingObj(addr))
	}
	return &res
}

type ContractCode []byte

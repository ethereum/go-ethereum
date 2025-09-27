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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/

package pathdb

import (
	"errors"
	"fmt"
	"iter"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// historyType represents the category of historical data.
type historyType uint8

const (
	// typeStateHistory indicates history data related to account or storage changes.
	typeStateHistory historyType = 0

	// typeTrienodeHistory indicates history data related to trie node changes.
	typeTrienodeHistory historyType = 1
)

// String returns the string format representation.
func (h historyType) String() string {
	switch h {
	case typeStateHistory:
		return "state"
	case typeTrienodeHistory:
		return "trienode"
	default:
		return fmt.Sprintf("unknown type: %d", h)
	}
}

// elementType represents the category of state element.
type elementType uint8

const (
	typeAccount  elementType = 0 // represents the account data
	typeStorage  elementType = 1 // represents the storage slot data
	typeTrienode elementType = 2 // represents the trie node data
)

// String returns the string format representation.
func (e elementType) String() string {
	switch e {
	case typeAccount:
		return "account"
	case typeStorage:
		return "storage"
	case typeTrienode:
		return "trienode"
	default:
		return fmt.Sprintf("unknown element type: %d", e)
	}
}

// toHistoryType maps an element type to its corresponding history type.
func toHistoryType(typ elementType) historyType {
	if typ == typeAccount || typ == typeStorage {
		return typeStateHistory
	}
	if typ == typeTrienode {
		return typeTrienodeHistory
	}
	panic(fmt.Sprintf("unknown element type %v", typ))
}

// stateIdent represents the identifier of a state element, which can be
// an account, a storage slot or a trienode.
type stateIdent struct {
	typ elementType

	// The hash of the account address. This is used instead of the raw account
	// address is to align the traversal order with the Merkle-Patricia-Trie.
	addressHash common.Hash

	// The hash of the storage slot key. This is used instead of the raw slot key
	// because, in legacy state histories (prior to the Cancun fork), the slot
	// identifier is the hash of the key, and the original key (preimage) cannot
	// be recovered. To maintain backward compatibility, the key hash is used.
	//
	// Meanwhile, using the storage key hash also preserve the traversal order
	// with Merkle-Patricia-Trie.
	//
	// This field is null if the identifier refers to an account or a trie node.
	storageHash common.Hash

	// The trie node path within the trie.
	//
	// This field is null if the identifier refers to an account or a storage slot.
	// String type is chosen to make stateIdent comparable.
	path string
}

// String returns the string format state identifier.
func (ident stateIdent) String() string {
	if ident.typ == typeAccount {
		return ident.addressHash.Hex()
	}
	if ident.typ == typeStorage {
		return ident.addressHash.Hex() + ident.storageHash.Hex()
	}
	return ident.addressHash.Hex() + ident.path
}

// newAccountIdent constructs a state identifier for an account.
func newAccountIdent(addressHash common.Hash) stateIdent {
	return stateIdent{
		typ:         typeAccount,
		addressHash: addressHash,
	}
}

// newStorageIdent constructs a state identifier for a storage slot.
// The address denotes the address hash of the associated account;
// the storageHash denotes the hash of the raw storage slot key;
func newStorageIdent(addressHash common.Hash, storageHash common.Hash) stateIdent {
	return stateIdent{
		typ:         typeStorage,
		addressHash: addressHash,
		storageHash: storageHash,
	}
}

// newTrienodeIdent constructs a state identifier for a trie node.
// The address denotes the address hash of the associated account;
// the path denotes the path of the node within the trie;
func newTrienodeIdent(addressHash common.Hash, path string) stateIdent {
	return stateIdent{
		typ:         typeTrienode,
		addressHash: addressHash,
		path:        path,
	}
}

// stateIdentQuery is the extension of stateIdent by adding the raw storage key.
type stateIdentQuery struct {
	stateIdent

	address    common.Address
	storageKey common.Hash
}

// newAccountIdentQuery constructs a state identifier for an account.
func newAccountIdentQuery(address common.Address, addressHash common.Hash) stateIdentQuery {
	return stateIdentQuery{
		stateIdent: newAccountIdent(addressHash),
		address:    address,
	}
}

// newStorageIdentQuery constructs a state identifier for a storage slot.
// the address denotes the address of the associated account;
// the addressHash denotes the address hash of the associated account;
// the storageKey denotes the raw storage slot key;
// the storageHash denotes the hash of the raw storage slot key;
func newStorageIdentQuery(address common.Address, addressHash common.Hash, storageKey common.Hash, storageHash common.Hash) stateIdentQuery {
	return stateIdentQuery{
		stateIdent: newStorageIdent(addressHash, storageHash),
		address:    address,
		storageKey: storageKey,
	}
}

// newTrienodeIdentQuery constructs a state identifier for a trie node.
// the addressHash denotes the address hash of the associated account;
// the path denotes the path of the node within the trie;
//
// nolint:unused
func newTrienodeIdentQuery(addrHash common.Hash, path []byte) stateIdentQuery {
	return stateIdentQuery{
		stateIdent: newTrienodeIdent(addrHash, string(path)),
	}
}

// history defines the interface of historical data, shared by stateHistory
// and trienodeHistory.
type history interface {
	// typ returns the historical data type held in the history.
	typ() historyType

	// forEach returns an iterator to traverse the state entries in the history.
	forEach() iter.Seq[stateIdent]
}

var (
	errHeadTruncationOutOfRange = errors.New("history head truncation out of range")
	errTailTruncationOutOfRange = errors.New("history tail truncation out of range")
)

// truncateFromHead removes excess elements from the head of the freezer based
// on the given parameters. It returns the number of items that were removed.
func truncateFromHead(store ethdb.AncientStore, typ historyType, nhead uint64) (int, error) {
	ohead, err := store.Ancients()
	if err != nil {
		return 0, err
	}
	otail, err := store.Tail()
	if err != nil {
		return 0, err
	}
	// Ensure that the truncation target falls within the valid range.
	if ohead < nhead || nhead < otail {
		return 0, fmt.Errorf("%w, %s, tail: %d, head: %d, target: %d", errHeadTruncationOutOfRange, typ, otail, ohead, nhead)
	}
	// Short circuit if nothing to truncate.
	if ohead == nhead {
		return 0, nil
	}
	log.Info("Truncating from head", "type", typ.String(), "ohead", ohead, "tail", otail, "nhead", nhead)

	ohead, err = store.TruncateHead(nhead)
	if err != nil {
		return 0, err
	}
	// Associated root->id mappings are left in the database and wait
	// for overwriting.
	return int(ohead - nhead), nil
}

// truncateFromTail removes excess elements from the end of the freezer based
// on the given parameters. It returns the number of items that were removed.
func truncateFromTail(store ethdb.AncientStore, typ historyType, ntail uint64) (int, error) {
	ohead, err := store.Ancients()
	if err != nil {
		return 0, err
	}
	otail, err := store.Tail()
	if err != nil {
		return 0, err
	}
	// Ensure that the truncation target falls within the valid range.
	if otail > ntail || ntail > ohead {
		return 0, fmt.Errorf("%w, %s, tail: %d, head: %d, target: %d", errTailTruncationOutOfRange, typ, otail, ohead, ntail)
	}
	// Short circuit if nothing to truncate.
	if otail == ntail {
		return 0, nil
	}
	otail, err = store.TruncateTail(ntail)
	if err != nil {
		return 0, err
	}
	// Associated root->id mappings are left in the database.
	return int(ntail - otail), nil
}

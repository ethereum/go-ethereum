// Copyright 2020 The go-ethereum Authors
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

package state

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// verifierStats is a collection of statistics gathered by the verifier
// for logging purposes.
type verifierStats struct {
	start       time.Time // Timestamp when generation started
	lastLog     time.Time
	nodes       uint64 // number of nodes loaded
	accounts    uint64 // Number of accounts loaded
	slots       uint64 // Number of storage slots checked
	codes       uint64
	lastAccount []byte
	path        []byte
}

func (vs *verifierStats) Log(msg string) {
	ctx := []interface{}{
		"elapsed", time.Since(vs.start),
		"nodes", vs.nodes, "accounts", vs.accounts, "slots", vs.slots, "codes", vs.codes,
		"lastAccount", fmt.Sprintf("0x%x", vs.lastAccount),
		"path", fmt.Sprintf("0x%x", vs.path),
	}
	log.Info(msg, ctx...)
	vs.lastLog = time.Now()
}

type pathHash struct {
	path []byte
	hash common.Hash
}

// verifyStorageTrie checks a given trie. If the trie is missing, or the trie
// contains errors, the hashes of all nodes leading to the missing item
// are returned.
// This method may return zero hashes, which means that the storage trie
// itself is missing
func verifyStorageTrie(root common.Hash, db Database, vs *verifierStats) (error, []*pathHash) {

	storageTrie, err := db.OpenStorageTrie(common.Hash{}, root)

	if err != nil {
		// The trie storage root is missing. TODO handle this error
		return fmt.Errorf("Missing storage root: %w", err), nil
	}
	it := storageTrie.NodeIterator(nil)

	for it.Next(true) {
		vs.path = it.Path()
		vs.nodes++
		if time.Since(vs.lastLog) > 8*time.Second {
			vs.Log("Verifying storage trie")
		}
		if it.Leaf() {
			vs.slots++
		}
	}
	if err = it.Error(); err != nil {
		// We have hit an error. Now figure out the parents
		var parents []*pathHash
		path := it.Path()
		log.Error("Storage trie error", "path", fmt.Sprintf("%x", path),
			"hash", it.Hash(), "parent", it.Parent(), "error", err)

		for {
			if ok, _ := trie.Pop(it); !ok {
				break
			}
			parents = append(parents, &pathHash{common.CopyBytes(path), it.Hash()})
			log.Error("Parent ", "hash", it.Hash(), "parent", it.Parent(),
				"path", fmt.Sprintf("0x%x", it.Path()))
		}
		return fmt.Errorf("storage trie error: %w", err), parents
	}

	return nil, nil
}

// Repair returns 'true' if anything was changed
func (s *StateDB) Repair(diskdb ethdb.Database) bool {
	err, hashes := s.Verify(nil)
	if err == nil {
		return false
	}
	msg := fmt.Sprintf(`
The state verification found at least one missing node. In order to perform 
a "healing fast-sync", %d parent nodes needs to be removed from the database. 

Once this is done, you can start geth normally (mode=fast), and geth should finish repairing 
the state trie. 

If you were running an archive node, this operation will most definitely lead to some states
being inaccessible, since the repair will be based off the tip of the chain, not the point at
which your node is currently at.

Do you wish to proceed? 

[y/N] > `, len(hashes))
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(msg)
	text, _ := reader.ReadString('\n')
	if ans := strings.TrimSpace(text); ans != "y" && ans != "Y" {
		return false
	}
	for _, h := range hashes {
		// Delete the hash from the database
		fmt.Printf("Deleting hash %x @ %x\n", h.hash, h.path)
		diskdb.Delete(h.hash[:])
	}
	// Now, we have to fool geth to think it's in the middle of an
	// interrupted fast-sync
	genesis := rawdb.ReadCanonicalHash(diskdb, 0)
	log.Info("Writing genesis as headblock", "hash", genesis)

	// Put the genesis in there
	rawdb.WriteHeadBlockHash(diskdb, genesis)
	return true
}

func (s *StateDB) Verify(start []byte) (error, []*pathHash) {
	log.Info("Starting verification procedure")
	var (
		vs = &verifierStats{
			start:    time.Now(),
			nodes:    0,
			accounts: 0,
			slots:    0,
			path:     []byte{},
		}
		it      = s.trie.NodeIterator(start)
		err     error
		parents []*pathHash
		// Avoid rechecking storage
		checkedStorageTries = make(map[common.Hash]struct{})
	)
	for it.Next(true) {
		vs.path = it.Path()
		vs.nodes++
		vs.nodes++
		if it.Leaf() {
			vs.accounts++
			vs.lastAccount = it.LeafKey()
			// We might have to iterate a storage trie
			//accountHash := common.BytesToHash(it.LeafKey())
			var acc struct {
				Nonce    uint64
				Balance  []byte // big.int can be decoded as byte slice
				Root     common.Hash
				CodeHash []byte
			}
			if err := rlp.DecodeBytes(it.LeafBlob(), &acc); err != nil {
				log.Crit("Invalid account encountered during verification", "err", err)
			}
			if _, checked := checkedStorageTries[acc.Root]; checked {
				continue
			}
			checkedStorageTries[acc.Root] = struct{}{}
			if err, parents = verifyStorageTrie(acc.Root, s.db, vs); err != nil {
				// This account is bad.
				break
			}
			// Check code
			if !bytes.Equal(acc.CodeHash, emptyCodeHash) {
				vs.codes++
				if rawdb.ReadCodeWithPrefix(s.db.TrieDB().DiskDB(), common.BytesToHash(acc.CodeHash)) == nil {
					// Missing code
					log.Error("Missing code", "codehash", acc.CodeHash)
					err = fmt.Errorf("missing code for codehash %x", acc.CodeHash)
					break
				}
			}
		}
		if time.Since(vs.lastLog) > 8*time.Second {
			vs.Log("Verifying state trie")
		}
	}
	if err == nil {
		err = it.Error()
	}
	if err != nil {
		// We have hit an error. Now figure out the parents
		path := it.Path()
		log.Error("Trie error", "path", fmt.Sprintf("%x", path),
			"hash", it.Hash(), "parent", it.Parent(), "error", err)

		for {
			if ok, _ := trie.Pop(it); !ok {
				break
			}
			fmt.Printf("%x\n", it.Hash())
			log.Error("Parent ", "hash", it.Hash(), "parent", it.Parent(),
				"path", fmt.Sprintf("0x%x", it.Path()))
			parents = append(parents, &pathHash{common.CopyBytes(it.Path()), it.Hash()})
		}
		if len(parents) > 0 {
			fmt.Println("Elements that need to be removed:")
			fmt.Println("")
			for _, h := range parents {
				fmt.Printf("%x (@ %x) \n", h.hash, h.path)
			}
		}
		return fmt.Errorf("trie error: %w", err), parents
	}
	vs.Log("Verified state trie")
	return nil, nil
}

type inspectionStats struct {
	start    time.Time // Timestamp when generation started
	lastLog  time.Time
	nodes    uint64 // number of nodes loaded
	items    uint64
	children uint64 // number of children resolved
	current  []byte
}

func (is *inspectionStats) Log(msg string) {
	ctx := []interface{}{
		"elapsed", time.Since(is.start),
		"nodes", is.nodes, "items", is.items, "children", is.children,
		"current", fmt.Sprintf("%x", is.current),
	}
	log.Info(msg, ctx...)
	is.lastLog = time.Now()
}

// decodeNode parses the RLP encoding of a trie node and returns the child hashes
func decodeNode(buf []byte) ([]common.Hash, error) {
	if len(buf) == 0 {
		return nil, errors.New("empty data")
	}
	elems, _, err := rlp.SplitList(buf)
	if err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}
	switch c, _ := rlp.CountValues(elems); c {
	case 2:
		return decodeShort(elems)
	case 17:
		return decodeFull(elems)
	default:
		return nil, fmt.Errorf("invalid number of list elements: %v", c)
	}
}
func decodeShort(elems []byte) ([]common.Hash, error) {
	kbuf, rest, err := rlp.SplitString(elems)
	if err != nil {
		return nil, err
	}
	// Check if it has terminator
	if kbuf[0]&0b100000 != 0 {
		//value node. No children
		return nil, nil
	}
	v, _, err := decodeRef(rest)
	if err != nil {
		return nil, err
	}
	if v != nil {
		return []common.Hash{common.BytesToHash(v)}, nil
	}
	return nil, nil
}

func decodeFull(elems []byte) ([]common.Hash, error) {
	var children []common.Hash
	for i := 0; i < 16; i++ {
		hash, rest, err := decodeRef(elems)
		if err != nil {
			return nil, err
		}
		if hash != nil {
			children = append(children, common.BytesToHash(hash))
		}
		elems = rest
	}
	return children, nil
}

func decodeRef(buf []byte) ([]byte, []byte, error) {
	kind, val, rest, err := rlp.Split(buf)
	if err != nil {
		return nil, buf, err
	}
	switch {
	case kind == rlp.List:
		// 'embedded' node reference..
		return nil, rest, nil
	case kind == rlp.String && len(val) == 0:
		// empty node
		return nil, rest, nil
	case kind == rlp.String && len(val) == 32:
		return val, rest, nil
	default:
		return nil, rest, fmt.Errorf("invalid RLP string size %d (want 0 or 32)", len(val))
	}
}

// InspectDB does an in-depth inspection of the db, in the following manner:
// - For each state entry in the database,
// - Decode the node,
// - Check if the children of that node are present
// This means that the method will basically load every item twice (or more), and
// it can take a long time to execute		"lastAccount", fmt.Sprintf("0x%x", vs.lastAccount),
func InspectDb(db ethdb.Database) error {
	log.Info("Starting verification procedure")
	var (
		is = &inspectionStats{
			start:   time.Now(),
			lastLog: time.Now(),
			nodes:   0,
		}
	)
	it := db.NewIterator(nil, nil)
	defer it.Release()
	// Inspect key-value database first.
	for it.Next() {
		var (
			key = it.Key()
		)
		is.items++
		is.current = key
		if len(key) != common.HashLength {
			continue
		}
		is.nodes++
		// Probably a state node
		// We should be able to decode it as either a full- or a shortnode.
		data := it.Value()
		children, err := decodeNode(data)
		if err != nil {
			return err
		}
		for _, c := range children {
			is.children++
			if _, err := db.Get(c[:]); err != nil {
				return fmt.Errorf("Missing item %x\n", c)
			}
		}
		if time.Since(is.lastLog) > 8*time.Second {
			is.Log("Inspecting state trie")
		}
	}
	is.Log("Inspection done")
	return nil
}

// Copyright 2024 The go-ethereum Authors
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

// nolint:unused
package pathdb

import (
	"bytes"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"maps"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// nodeSet represents a collection of modified trie nodes resulting from a state
// transition, typically corresponding to a block execution. It can also represent
// the combined trie node set from several aggregated state transitions.
type nodeSet struct {
	size         uint64                                    // aggregated size of the trie node
	accountNodes map[string]*trienode.Node                 // account trie nodes, mapped by path
	storageNodes map[common.Hash]map[string]*trienode.Node // storage trie nodes, mapped by owner and path
}

// newNodeSet constructs the set with the provided dirty trie nodes.
func newNodeSet(nodes map[common.Hash]map[string]*trienode.Node) *nodeSet {
	// Don't panic for the lazy callers, initialize the nil map instead
	if nodes == nil {
		nodes = make(map[common.Hash]map[string]*trienode.Node)
	}
	s := &nodeSet{
		accountNodes: make(map[string]*trienode.Node),
		storageNodes: make(map[common.Hash]map[string]*trienode.Node),
	}
	for owner, subset := range nodes {
		if owner == (common.Hash{}) {
			s.accountNodes = subset
		} else {
			s.storageNodes[owner] = subset
		}
	}
	s.computeSize()
	return s
}

// computeSize calculates the database size of the held trie nodes.
func (s *nodeSet) computeSize() {
	var size uint64
	for path, n := range s.accountNodes {
		size += uint64(len(n.Blob) + len(path))
	}
	for _, subset := range s.storageNodes {
		for path, n := range subset {
			size += uint64(common.HashLength + len(n.Blob) + len(path))
		}
	}
	s.size = size
}

// updateSize updates the total cache size by the given delta.
func (s *nodeSet) updateSize(delta int64) {
	size := int64(s.size) + delta
	if size >= 0 {
		s.size = uint64(size)
		return
	}
	log.Error("Nodeset size underflow", "prev", common.StorageSize(s.size), "delta", common.StorageSize(delta))
	s.size = 0
}

// node retrieves the trie node with node path and its trie identifier.
func (s *nodeSet) node(owner common.Hash, path []byte) (*trienode.Node, bool) {
	// Account trie node
	if owner == (common.Hash{}) {
		n, ok := s.accountNodes[string(path)]
		return n, ok
	}
	// Storage trie node
	subset, ok := s.storageNodes[owner]
	if !ok {
		return nil, false
	}
	n, ok := subset[string(path)]
	return n, ok
}

// merge integrates the provided dirty nodes into the set. The provided nodeset
// will remain unchanged, as it may still be referenced by other layers.
func (s *nodeSet) merge(set *nodeSet) {
	var (
		delta     int64   // size difference resulting from node merging
		overwrite counter // counter of nodes being overwritten
	)

	// Merge account nodes
	for path, n := range set.accountNodes {
		if orig, exist := s.accountNodes[path]; !exist {
			delta += int64(len(n.Blob) + len(path))
		} else {
			delta += int64(len(n.Blob) - len(orig.Blob))
			overwrite.add(len(orig.Blob) + len(path))
		}
		s.accountNodes[path] = n
	}

	// Merge storage nodes
	for owner, subset := range set.storageNodes {
		current, exist := s.storageNodes[owner]
		if !exist {
			for path, n := range subset {
				delta += int64(common.HashLength + len(n.Blob) + len(path))
			}
			// Perform a shallow copy of the map for the subset instead of claiming it
			// directly from the provided nodeset to avoid potential concurrent map
			// read/write issues. The nodes belonging to the original diff layer remain
			// accessible even after merging. Therefore, ownership of the nodes map
			// should still belong to the original layer, and any modifications to it
			// should be prevented.
			s.storageNodes[owner] = maps.Clone(subset)
			continue
		}
		for path, n := range subset {
			if orig, exist := current[path]; !exist {
				delta += int64(common.HashLength + len(n.Blob) + len(path))
			} else {
				delta += int64(len(n.Blob) - len(orig.Blob))
				overwrite.add(common.HashLength + len(orig.Blob) + len(path))
			}
			current[path] = n
		}
		s.storageNodes[owner] = current
	}
	overwrite.report(gcTrieNodeMeter, gcTrieNodeBytesMeter)
	s.updateSize(delta)
}

// revertTo merges the provided trie nodes into the set. This should reverse the
// changes made by the most recent state transition.
func (s *nodeSet) revertTo(db ethdb.KeyValueReader, nodes map[common.Hash]map[string]*trienode.Node) {
	var delta int64
	for owner, subset := range nodes {
		if owner == (common.Hash{}) {
			// Account trie nodes
			for path, n := range subset {
				orig, ok := s.accountNodes[path]
				if !ok {
					blob := rawdb.ReadAccountTrieNode(db, []byte(path))
					if bytes.Equal(blob, n.Blob) {
						continue
					}
					panic(fmt.Sprintf("non-existent account node (%v) blob: %v", path, crypto.Keccak256Hash(n.Blob).Hex()))
				}
				s.accountNodes[path] = n
				delta += int64(len(n.Blob)) - int64(len(orig.Blob))
			}
		} else {
			// Storage trie nodes
			current, ok := s.storageNodes[owner]
			if !ok {
				panic(fmt.Sprintf("non-existent subset (%x)", owner))
			}
			for path, n := range subset {
				orig, ok := current[path]
				if !ok {
					blob := rawdb.ReadStorageTrieNode(db, owner, []byte(path))
					if bytes.Equal(blob, n.Blob) {
						continue
					}
					panic(fmt.Sprintf("non-existent storage node (%x %v) blob: %v", owner, path, crypto.Keccak256Hash(n.Blob).Hex()))
				}
				current[path] = n
				delta += int64(len(n.Blob)) - int64(len(orig.Blob))
			}
		}
	}
	s.updateSize(delta)
}

// journalNode represents a trie node persisted in the journal.
type journalNode struct {
	Path []byte // Path of the node in the trie
	Blob []byte // RLP-encoded trie node blob, nil means the node is deleted
}

// journalNodes represents a list trie nodes belong to a single account
// or the main account trie.
type journalNodes struct {
	Owner common.Hash
	Nodes []journalNode
}

// encode serializes the content of trie nodes into the provided writer.
func (s *nodeSet) encode(w io.Writer) error {
	nodes := make([]journalNodes, 0, len(s.storageNodes)+1)

	// Encode account nodes
	if len(s.accountNodes) > 0 {
		entry := journalNodes{Owner: common.Hash{}}
		for path, node := range s.accountNodes {
			entry.Nodes = append(entry.Nodes, journalNode{
				Path: []byte(path),
				Blob: node.Blob,
			})
		}
		nodes = append(nodes, entry)
	}
	// Encode storage nodes
	for owner, subset := range s.storageNodes {
		entry := journalNodes{Owner: owner}
		for path, node := range subset {
			entry.Nodes = append(entry.Nodes, journalNode{
				Path: []byte(path),
				Blob: node.Blob,
			})
		}
		nodes = append(nodes, entry)
	}
	return rlp.Encode(w, nodes)
}

// decode deserializes the content from the rlp stream into the nodeset.
func (s *nodeSet) decode(r *rlp.Stream) error {
	var encoded []journalNodes
	if err := r.Decode(&encoded); err != nil {
		return fmt.Errorf("load nodes: %v", err)
	}
	s.accountNodes = make(map[string]*trienode.Node)
	s.storageNodes = make(map[common.Hash]map[string]*trienode.Node)

	for _, entry := range encoded {
		if entry.Owner == (common.Hash{}) {
			// Account nodes
			for _, n := range entry.Nodes {
				if len(n.Blob) > 0 {
					s.accountNodes[string(n.Path)] = trienode.New(crypto.Keccak256Hash(n.Blob), n.Blob)
				} else {
					s.accountNodes[string(n.Path)] = trienode.NewDeleted()
				}
			}
		} else {
			// Storage nodes
			subset := make(map[string]*trienode.Node)
			for _, n := range entry.Nodes {
				if len(n.Blob) > 0 {
					subset[string(n.Path)] = trienode.New(crypto.Keccak256Hash(n.Blob), n.Blob)
				} else {
					subset[string(n.Path)] = trienode.NewDeleted()
				}
			}
			s.storageNodes[entry.Owner] = subset
		}
	}
	s.computeSize()
	return nil
}

// write flushes nodes into the provided database batch as a whole.
func (s *nodeSet) write(batch ethdb.Batch, clean *fastcache.Cache) int {
	nodes := make(map[common.Hash]map[string]*trienode.Node)
	if len(s.accountNodes) > 0 {
		nodes[common.Hash{}] = s.accountNodes
	}
	for owner, subset := range s.storageNodes {
		nodes[owner] = subset
	}
	return writeNodes(batch, nodes, clean)
}

// reset clears all cached trie node data.
func (s *nodeSet) reset() {
	s.accountNodes = make(map[string]*trienode.Node)
	s.storageNodes = make(map[common.Hash]map[string]*trienode.Node)
	s.size = 0
}

// dbsize returns the approximate size of db write.
func (s *nodeSet) dbsize() int {
	var m int
	m += len(s.accountNodes) * len(rawdb.TrieNodeAccountPrefix) // database key prefix
	for _, nodes := range s.storageNodes {
		m += len(nodes) * (len(rawdb.TrieNodeStoragePrefix)) // database key prefix
	}
	return m + int(s.size)
}

// nodeSetWithOrigin wraps the node set with additional original values of the
// mutated trie nodes.
type nodeSetWithOrigin struct {
	*nodeSet

	// nodeOrigin represents the trie nodes before the state transition. It's keyed
	// by the account address hash and node path. The nil value means the trie node
	// was not present.
	nodeOrigin map[common.Hash]map[string][]byte

	// memory size of the state data (accountNodeOrigin and storageNodeOrigin)
	size uint64
}

// NewNodeSetWithOrigin constructs the state set with the provided data.
func NewNodeSetWithOrigin(nodes map[common.Hash]map[string]*trienode.Node, origins map[common.Hash]map[string][]byte) *nodeSetWithOrigin {
	// Don't panic for the lazy callers, initialize the nil maps instead.
	if origins == nil {
		origins = make(map[common.Hash]map[string][]byte)
	}
	set := &nodeSetWithOrigin{
		nodeSet:    newNodeSet(nodes),
		nodeOrigin: origins,
	}
	set.computeSize()
	return set
}

// computeSize calculates the database size of the held trie nodes.
func (s *nodeSetWithOrigin) computeSize() {
	var size int
	for owner, slots := range s.nodeOrigin {
		prefixLen := common.HashLength
		if owner == (common.Hash{}) {
			prefixLen = 0
		}
		for path, data := range slots {
			size += prefixLen + len(path) + len(data)
		}
	}
	s.size = s.nodeSet.size + uint64(size)
}

// encode serializes the content of node set into the provided writer.
func (s *nodeSetWithOrigin) encode(w io.Writer) error {
	// Encode node set
	if err := s.nodeSet.encode(w); err != nil {
		return err
	}
	// Short circuit if the origins are not tracked
	if len(s.nodeOrigin) == 0 {
		return nil
	}

	// Encode node origins
	nodes := make([]journalNodes, 0, len(s.nodeOrigin))
	for owner, subset := range s.nodeOrigin {
		entry := journalNodes{
			Owner: owner,
			Nodes: make([]journalNode, 0, len(subset)),
		}
		for path, node := range subset {
			entry.Nodes = append(entry.Nodes, journalNode{
				Path: []byte(path),
				Blob: node,
			})
		}
		nodes = append(nodes, entry)
	}
	return rlp.Encode(w, nodes)
}

// hasOrigin returns whether the origin data set exists in the rlp stream.
// It's a workaround for backward compatibility.
func (s *nodeSetWithOrigin) hasOrigin(r *rlp.Stream) (bool, error) {
	kind, _, err := r.Kind()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, err
	}
	// If the type of next element in the RLP stream is:
	// - `rlp.List`: represents the original value of trienodes;
	// - others, like `boolean`: represent a field in the following state data set;
	return kind == rlp.List, nil
}

// decode deserializes the content from the rlp stream into the node set.
func (s *nodeSetWithOrigin) decode(r *rlp.Stream) error {
	if s.nodeSet == nil {
		s.nodeSet = &nodeSet{}
	}
	if err := s.nodeSet.decode(r); err != nil {
		return err
	}

	// Decode node origins
	s.nodeOrigin = make(map[common.Hash]map[string][]byte)
	if hasOrigin, err := s.hasOrigin(r); err != nil {
		return err
	} else if hasOrigin {
		var encoded []journalNodes
		if err := r.Decode(&encoded); err != nil {
			return fmt.Errorf("load nodes: %v", err)
		}
		for _, entry := range encoded {
			subset := make(map[string][]byte, len(entry.Nodes))
			for _, n := range entry.Nodes {
				if len(n.Blob) > 0 {
					subset[string(n.Path)] = n.Blob
				} else {
					subset[string(n.Path)] = nil
				}
			}
			s.nodeOrigin[entry.Owner] = subset
		}
	}
	s.computeSize()
	return nil
}

// encodeNodeCompressed encodes the trie node differences between two consecutive
// versions into byte stream. The format is as below:
//
// - metadata byte layout (1 byte):
//
//	┌──── Bits (from MSB to LSB) ───┐
//	│ 7 │ 6 │ 5 │ 4 │ 3 │ 2 │ 1 │ 0 │
//	└───────────────────────────────┘
//	  │   │   │   │   │   │   │   └─ FlagA: set if value is encoded in compressed format
//	  │   │   │   │   │   │   └───── FlagB: set if no extended bitmap is present after the metadata byte
//	  │   │   │   │   │   └───────── FlagC: bitmap for node (only used when flagB == 1)
//	  │   │   │   │   └───────────── FlagD: bitmap for node (only used when flagB == 1)
//	  │   │   │   └───────────────── FlagE: reserved (marks the presence of the 16th child in a full node)
//	  │   │   └───────────────────── FlagF: reserved
//	  │   └───────────────────────── FlagG: reserved
//	  └───────────────────────────── FlagH: reserved
//
// Note:
// - If flagB is 1, the node refers to a shortNode;
//   - flagC indicates whether the key of the shortNode is recorded.
//   - flagD indicates whether the value of the shortNode is recorded.
//
// - If flagB is 0, the node refers to a fullNode;
//   - each bit in extended bitmap indicates whether the corresponding
//     child have been modified.
//
// Example:
//
// 0b_0000_1011
//
// Bit0=1, Bit1=1 -> node in compressed format, no extended bitmap
// Bit2=0, Bit3=1 -> the key of a short node is not stored; its value is stored.
//
// - 2 bytes extended bitmap (only if the flagB in metadata is 0), each bit
// represents a corresponding child;
//
// - concatenation of original value of modified children along with its size;
func encodeNodeCompressed(addExtension bool, elements [][]byte, indices []int) []byte {
	var (
		enc  []byte
		flag = byte(1) // The compression format indicator
	)
	// Pre-allocate the byte slice for the node encoder
	size := 1
	if addExtension {
		size += 2
	}
	for _, element := range elements {
		size += len(element) + 1
	}
	enc = make([]byte, 0, size)

	if !addExtension {
		flag |= 2 // The embedded bitmap indicator

		// Embedded bitmap
		for _, pos := range indices {
			flag |= 1 << (pos + 2)
		}
		enc = append(enc, flag)
	} else {
		// Extended bitmap
		bitmap := make([]byte, 2) // bitmaps for at most 16 children
		for _, pos := range indices {
			// Children[16] is only theoretically possible in the Merkle-Patricia-trie,
			// in practice this field is never used in the Ethereum case. If it occurs,
			// use the FlagE for marking the presence.
			if pos >= 16 {
				log.Warn("Unexpected 16th child encountered in a full node")
				flag |= 1 << 4 // Use the reserved flagE
				continue
			}
			setBit(bitmap, pos)
		}
		enc = append(enc, flag)
		enc = append(enc, bitmap...)
	}
	for _, element := range elements {
		enc = append(enc, byte(len(element))) // 1 byte is sufficient for element size
		enc = append(enc, element...)
	}
	return enc
}

// encodeNodeFull encodes the full trie node value into byte stream. The format is
// as below:
//
// - metadata byte layout (1 byte): 0b0
// - node value
//
// TODO(rjl493456442) it's not allocation efficient, please improve it.
func encodeNodeFull(value []byte) []byte {
	enc := make([]byte, len(value)+1)
	copy(enc[1:], value)
	return enc
}

// decodeNodeCompressed decodes the byte stream of compressed trie node
// back to the original elements and their indices.
//
// It assumes the byte stream contains a compressed format node.
func decodeNodeCompressed(data []byte) ([][]byte, []int, error) {
	if len(data) < 1 {
		return nil, nil, errors.New("invalid data: too short")
	}
	flag := data[0]
	if flag&byte(1) == 0 {
		return nil, nil, errors.New("invalid data: full node value")
	}
	noExtend := flag&byte(2) != 0

	// Reconstruct indices from bitmap
	var indices []int
	if noExtend {
		if flag&byte(4) != 0 { // flagC
			indices = append(indices, 0)
		}
		if flag&byte(8) != 0 { // flagD
			indices = append(indices, 1)
		}
		data = data[1:]
	} else {
		if len(data) < 3 {
			return nil, nil, errors.New("invalid data: too short")
		}
		bitmap := data[1:3]
		indices = bitPosTwoBytes(bitmap)
		if flag&byte(16) != 0 { // flagE
			indices = append(indices, 16)
			log.Info("Unexpected 16th child encountered in a full node")
		}
		data = data[3:]
	}
	// Reconstruct elements
	elements := make([][]byte, 0, len(indices))
	for i := 0; i < len(indices); i++ {
		if len(data) == 0 {
			return nil, nil, errors.New("invalid data: missing size byte")
		}
		// Read element size
		size := int(data[0])
		data = data[1:]

		// Check if we have enough data for the element
		if len(data) < size {
			return nil, nil, fmt.Errorf("invalid data: expected %d bytes, got %d", size, len(data))
		}
		// Extract element
		if size == 0 {
			elements = append(elements, nil)

			// The zero-size element is practically unexpected, for node deletion
			// the rlp.EmptyString is still expected. Log loudly for the potential
			// programming error.
			log.Error("Empty element from compressed node, please open an issue", "raw", data)
		} else {
			element := make([]byte, size)
			copy(element, data[:size])
			data = data[size:]
			elements = append(elements, element)
		}
	}
	// Check if all data is consumed
	if len(data) != 0 {
		return nil, nil, errors.New("invalid data: trailing bytes")
	}
	return elements, indices, nil
}

// decodeNodeFull decodes the byte stream of full value trie node.
func decodeNodeFull(data []byte) (bool, []byte, error) {
	if len(data) < 1 {
		return false, nil, errors.New("invalid data: too short")
	}
	flag := data[0]
	if flag != byte(0) {
		return false, nil, nil
	}
	return true, data[1:], nil
}

// encodeNodeHistory encodes the history of a node. Typically, the original values
// of dirty nodes serve as the history, but this can lead to significant storage
// overhead.
//
// For full nodes, which often see only a few modified children during state
// transitions, recording the entire child set (up to 16 children at 32 bytes
// each) is inefficient. For short nodes, which often see only the value is
// modified during the state transition, recording the key part is also unnecessary.
// To compress size, we instead record the diff of the node, rather than the
// full value. It's vital to compress the overall trienode history.
//
// However, recovering a node from a series of diffs requires applying multiple
// history records, which is computationally and IO intensive. To mitigate this, we
// periodically record the full value of a node as a checkpoint. The frequency of
// these checkpoints is a tradeoff between the compression rate and read overhead.
func (s *nodeSetWithOrigin) encodeNodeHistory(root common.Hash, rate uint32) (map[common.Hash]map[string][]byte, error) {
	var (
		// the set of all encoded node history elements
		nodes = make(map[common.Hash]map[string][]byte)

		// encodeFullValue determines whether a node should be encoded
		// in full format with a pseudo-random probabilistic algorithm.
		encodeFullValue = func(owner common.Hash, path string) bool {
			// For trie nodes at the first two levels of the account trie, it is very
			// likely that all children are modified within a single state transition.
			// In such cases, do not use diff mode.
			if owner == (common.Hash{}) && len(path) < 2 {
				return true
			}
			h := fnv.New32a()
			h.Write(root.Bytes())
			h.Write(owner.Bytes())
			h.Write([]byte(path))
			return h.Sum32()%rate == 0
		}
	)
	for owner, origins := range s.nodeOrigin {
		var posts map[string]*trienode.Node
		if owner == (common.Hash{}) {
			posts = s.nodeSet.accountNodes
		} else {
			posts = s.nodeSet.storageNodes[owner]
		}
		nodes[owner] = make(map[string][]byte)

		for path, oldvalue := range origins {
			n, exists := posts[path]
			if !exists {
				// something not expected
				return nil, fmt.Errorf("node with origin is not found, %x-%v", owner, []byte(path))
			}
			encodeFull := encodeFullValue(owner, path)
			if !encodeFull {
				// TODO(rjl493456442) the diff-mode reencoding can take non-trivial
				// time, like 1-2ms per block, is there any way to mitigate the overhead?

				// Partial encoding is required, try to find the node diffs and
				// fallback to the full-value encoding if fails.
				//
				// The partial encoding will be failed in these certain cases:
				// - the node is deleted or was not-existent;
				// - the node type has been changed (e.g, from short to full)
				nElem, indices, diffs, err := trie.NodeDifference(oldvalue, n.Blob)
				if err != nil {
					encodeFull = true // fallback to the full node encoding
				} else {
					// Encode the node difference as the history element
					addExt := nElem != 2 // fullNode
					blob := encodeNodeCompressed(addExt, diffs, indices)
					nodes[owner][path] = blob
				}
			}
			if encodeFull {
				// Encode the entire original value as the history element
				nodes[owner][path] = encodeNodeFull(oldvalue)
			}
		}
	}
	return nodes, nil
}

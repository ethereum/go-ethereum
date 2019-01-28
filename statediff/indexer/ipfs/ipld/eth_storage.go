// VulcanizeDB
// Copyright © 2019 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package ipld

import (
	"fmt"

	"github.com/ipfs/go-cid"
	node "github.com/ipfs/go-ipld-format"
	"github.com/multiformats/go-multihash"
)

// EthStorageTrie (eth-storage-trie, codec 0x98), represents
// a node from the storage trie in ethereum.
type EthStorageTrie struct {
	*TrieNode
}

// Static (compile time) check that EthStorageTrie satisfies the node.Node interface.
var _ node.Node = (*EthStorageTrie)(nil)

/*
  INPUT
*/

// FromStorageTrieRLP takes the RLP representation of an ethereum
// storage trie node to return it as an IPLD node for further processing.
func FromStorageTrieRLP(raw []byte) (*EthStorageTrie, error) {
	c, err := RawdataToCid(MEthStorageTrie, raw, multihash.KECCAK_256)
	if err != nil {
		return nil, err
	}

	// Let's run the whole mile and process the nodeKind and
	// its elements, in case somebody would need this function
	// to parse an RLP element from the filesystem
	return DecodeEthStorageTrie(c, raw)
}

/*
  OUTPUT
*/

// DecodeEthStorageTrie returns an EthStorageTrie object from its cid and rawdata.
func DecodeEthStorageTrie(c cid.Cid, b []byte) (*EthStorageTrie, error) {
	tn, err := decodeTrieNode(c, b, decodeEthStorageTrieLeaf)
	if err != nil {
		return nil, err
	}
	return &EthStorageTrie{TrieNode: tn}, nil
}

// decodeEthStorageTrieLeaf parses a eth-tx-trie leaf
// from decoded RLP elements
func decodeEthStorageTrieLeaf(i []interface{}) ([]interface{}, error) {
	return []interface{}{
		i[0].([]byte),
		i[1].([]byte),
	}, nil
}

/*
  Block INTERFACE
*/

// RawData returns the binary of the RLP encode of the storage trie node.
func (st *EthStorageTrie) RawData() []byte {
	return st.rawdata
}

// Cid returns the cid of the storage trie node.
func (st *EthStorageTrie) Cid() cid.Cid {
	return st.cid
}

// String is a helper for output
func (st *EthStorageTrie) String() string {
	return fmt.Sprintf("<EthereumStorageTrie %s>", st.cid)
}

// Loggable returns in a map the type of IPLD Link.
func (st *EthStorageTrie) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"type": "eth-storage-trie",
	}
}

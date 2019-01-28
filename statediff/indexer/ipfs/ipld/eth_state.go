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

	"github.com/ethereum/go-ethereum/rlp"
)

// EthStateTrie (eth-state-trie, codec 0x96), represents
// a node from the satte trie in ethereum.
type EthStateTrie struct {
	*TrieNode
}

// Static (compile time) check that EthStateTrie satisfies the node.Node interface.
var _ node.Node = (*EthStateTrie)(nil)

/*
  INPUT
*/

// FromStateTrieRLP takes the RLP representation of an ethereum
// state trie node to return it as an IPLD node for further processing.
func FromStateTrieRLP(raw []byte) (*EthStateTrie, error) {
	c, err := RawdataToCid(MEthStateTrie, raw, multihash.KECCAK_256)
	if err != nil {
		return nil, err
	}
	// Let's run the whole mile and process the nodeKind and
	// its elements, in case somebody would need this function
	// to parse an RLP element from the filesystem
	return DecodeEthStateTrie(c, raw)
}

/*
  OUTPUT
*/

// DecodeEthStateTrie returns an EthStateTrie object from its cid and rawdata.
func DecodeEthStateTrie(c cid.Cid, b []byte) (*EthStateTrie, error) {
	tn, err := decodeTrieNode(c, b, decodeEthStateTrieLeaf)
	if err != nil {
		return nil, err
	}
	return &EthStateTrie{TrieNode: tn}, nil
}

// decodeEthStateTrieLeaf parses a eth-tx-trie leaf
// from decoded RLP elements
func decodeEthStateTrieLeaf(i []interface{}) ([]interface{}, error) {
	var account EthAccount
	err := rlp.DecodeBytes(i[1].([]byte), &account)
	if err != nil {
		return nil, err
	}
	c, err := RawdataToCid(MEthAccountSnapshot, i[1].([]byte), multihash.KECCAK_256)
	if err != nil {
		return nil, err
	}
	return []interface{}{
		i[0].([]byte),
		&EthAccountSnapshot{
			EthAccount: &account,
			cid:        c,
			rawdata:    i[1].([]byte),
		},
	}, nil
}

/*
  Block INTERFACE
*/

// RawData returns the binary of the RLP encode of the state trie node.
func (st *EthStateTrie) RawData() []byte {
	return st.rawdata
}

// Cid returns the cid of the state trie node.
func (st *EthStateTrie) Cid() cid.Cid {
	return st.cid
}

// String is a helper for output
func (st *EthStateTrie) String() string {
	return fmt.Sprintf("<EthereumStateTrie %s>", st.cid)
}

// Loggable returns in a map the type of IPLD Link.
func (st *EthStateTrie) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"type": "eth-state-trie",
	}
}

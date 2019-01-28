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

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// EthRctTrie (eth-tx-trie codec 0x92) represents
// a node from the transaction trie in ethereum.
type EthRctTrie struct {
	*TrieNode
}

// Static (compile time) check that EthRctTrie satisfies the node.Node interface.
var _ node.Node = (*EthRctTrie)(nil)

/*
 INPUT
*/

// To create a proper trie of the eth-tx-trie objects, it is required
// to input all transactions belonging to a forest in a single step.
// We are adding the transactions, and creating its trie on
// block body parsing time.

/*
  OUTPUT
*/

// DecodeEthRctTrie returns an EthRctTrie object from its cid and rawdata.
func DecodeEthRctTrie(c cid.Cid, b []byte) (*EthRctTrie, error) {
	tn, err := decodeTrieNode(c, b, decodeEthRctTrieLeaf)
	if err != nil {
		return nil, err
	}
	return &EthRctTrie{TrieNode: tn}, nil
}

// decodeEthRctTrieLeaf parses a eth-rct-trie leaf
//from decoded RLP elements
func decodeEthRctTrieLeaf(i []interface{}) ([]interface{}, error) {
	var r types.Receipt
	err := rlp.DecodeBytes(i[1].([]byte), &r)
	if err != nil {
		return nil, err
	}
	c, err := RawdataToCid(MEthTxReceipt, i[1].([]byte), multihash.KECCAK_256)
	if err != nil {
		return nil, err
	}
	return []interface{}{
		i[0].([]byte),
		&EthReceipt{
			Receipt: &r,
			cid:     c,
			rawdata: i[1].([]byte),
		},
	}, nil
}

/*
  Block INTERFACE
*/

// RawData returns the binary of the RLP encode of the transaction.
func (t *EthRctTrie) RawData() []byte {
	return t.rawdata
}

// Cid returns the cid of the transaction.
func (t *EthRctTrie) Cid() cid.Cid {
	return t.cid
}

// String is a helper for output
func (t *EthRctTrie) String() string {
	return fmt.Sprintf("<EthereumRctTrie %s>", t.cid)
}

// Loggable returns in a map the type of IPLD Link.
func (t *EthRctTrie) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"type": "eth-rct-trie",
	}
}

/*
  EthRctTrie functions
*/

// rctTrie wraps a localTrie for use on the receipt trie.
type rctTrie struct {
	*localTrie
}

// newRctTrie initializes and returns a rctTrie.
func newRctTrie() *rctTrie {
	return &rctTrie{
		localTrie: newLocalTrie(),
	}
}

// getNodes invokes the localTrie, which computes the root hash of the
// transaction trie and returns its database keys, to return a slice
// of EthRctTrie nodes.
func (rt *rctTrie) getNodes() []*EthRctTrie {
	keys := rt.getKeys()
	var out []*EthRctTrie
	it := rt.trie.NodeIterator([]byte{})
	for it.Next(true) {

	}
	for _, k := range keys {
		rawdata, err := rt.db.Get(k)
		if err != nil {
			panic(err)
		}
		c, err := RawdataToCid(MEthTxReceiptTrie, rawdata, multihash.KECCAK_256)
		if err != nil {
			return nil
		}
		tn := &TrieNode{
			cid:     c,
			rawdata: rawdata,
		}
		out = append(out, &EthRctTrie{TrieNode: tn})
	}

	return out
}

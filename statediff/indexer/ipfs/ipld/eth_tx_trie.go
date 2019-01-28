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

// EthTxTrie (eth-tx-trie codec 0x92) represents
// a node from the transaction trie in ethereum.
type EthTxTrie struct {
	*TrieNode
}

// Static (compile time) check that EthTxTrie satisfies the node.Node interface.
var _ node.Node = (*EthTxTrie)(nil)

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

// DecodeEthTxTrie returns an EthTxTrie object from its cid and rawdata.
func DecodeEthTxTrie(c cid.Cid, b []byte) (*EthTxTrie, error) {
	tn, err := decodeTrieNode(c, b, decodeEthTxTrieLeaf)
	if err != nil {
		return nil, err
	}
	return &EthTxTrie{TrieNode: tn}, nil
}

// decodeEthTxTrieLeaf parses a eth-tx-trie leaf
//from decoded RLP elements
func decodeEthTxTrieLeaf(i []interface{}) ([]interface{}, error) {
	var t types.Transaction
	err := rlp.DecodeBytes(i[1].([]byte), &t)
	if err != nil {
		return nil, err
	}
	c, err := RawdataToCid(MEthTx, i[1].([]byte), multihash.KECCAK_256)
	if err != nil {
		return nil, err
	}
	return []interface{}{
		i[0].([]byte),
		&EthTx{
			Transaction: &t,
			cid:         c,
			rawdata:     i[1].([]byte),
		},
	}, nil
}

/*
  Block INTERFACE
*/

// RawData returns the binary of the RLP encode of the transaction.
func (t *EthTxTrie) RawData() []byte {
	return t.rawdata
}

// Cid returns the cid of the transaction.
func (t *EthTxTrie) Cid() cid.Cid {
	return t.cid
}

// String is a helper for output
func (t *EthTxTrie) String() string {
	return fmt.Sprintf("<EthereumTxTrie %s>", t.cid)
}

// Loggable returns in a map the type of IPLD Link.
func (t *EthTxTrie) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"type": "eth-tx-trie",
	}
}

/*
  EthTxTrie functions
*/

// txTrie wraps a localTrie for use on the transaction trie.
type txTrie struct {
	*localTrie
}

// newTxTrie initializes and returns a txTrie.
func newTxTrie() *txTrie {
	return &txTrie{
		localTrie: newLocalTrie(),
	}
}

// getNodes invokes the localTrie, which computes the root hash of the
// transaction trie and returns its database keys, to return a slice
// of EthTxTrie nodes.
func (tt *txTrie) getNodes() []*EthTxTrie {
	keys := tt.getKeys()
	var out []*EthTxTrie
	it := tt.trie.NodeIterator([]byte{})
	for it.Next(true) {

	}
	for _, k := range keys {
		rawdata, err := tt.db.Get(k)
		if err != nil {
			panic(err)
		}
		c, err := RawdataToCid(MEthTxTrie, rawdata, multihash.KECCAK_256)
		if err != nil {
			return nil
		}
		tn := &TrieNode{
			cid:     c,
			rawdata: rawdata,
		}
		out = append(out, &EthTxTrie{TrieNode: tn})
	}

	return out
}

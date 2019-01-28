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
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ipfs/go-cid"
	node "github.com/ipfs/go-ipld-format"
)

// EthAccountSnapshot (eth-account-snapshot codec 0x97)
// represents an ethereum account, i.e. a wallet address or
// a smart contract
type EthAccountSnapshot struct {
	*EthAccount

	cid     cid.Cid
	rawdata []byte
}

// EthAccount is the building block of EthAccountSnapshot.
// Or, is the former stripped of its cid and rawdata components.
type EthAccount struct {
	Nonce    uint64
	Balance  *big.Int
	Root     []byte // This is the storage root trie
	CodeHash []byte // This is the hash of the EVM code
}

// Static (compile time) check that EthAccountSnapshot satisfies the
// node.Node interface.
var _ node.Node = (*EthAccountSnapshot)(nil)

/*
  INPUT
*/

// Input should be managed by EthStateTrie

/*
   OUTPUT
*/

// Output should be managed by EthStateTrie

/*
   Block INTERFACE
*/

// RawData returns the binary of the RLP encode of the account snapshot.
func (as *EthAccountSnapshot) RawData() []byte {
	return as.rawdata
}

// Cid returns the cid of the transaction.
func (as *EthAccountSnapshot) Cid() cid.Cid {
	return as.cid
}

// String is a helper for output
func (as *EthAccountSnapshot) String() string {
	return fmt.Sprintf("<EthereumAccountSnapshot %s>", as.cid)
}

// Loggable returns in a map the type of IPLD Link.
func (as *EthAccountSnapshot) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"type": "eth-account-snapshot",
	}
}

/*
   Node INTERFACE
*/

// Resolve resolves a path through this node, stopping at any link boundary
// and returning the object found as well as the remaining path to traverse
func (as *EthAccountSnapshot) Resolve(p []string) (interface{}, []string, error) {
	if len(p) == 0 {
		return as, nil, nil
	}

	if len(p) > 1 {
		return nil, nil, fmt.Errorf("unexpected path elements past %s", p[0])
	}

	switch p[0] {
	case "balance":
		return as.Balance, nil, nil
	case "codeHash":
		return &node.Link{Cid: keccak256ToCid(RawBinary, as.CodeHash)}, nil, nil
	case "nonce":
		return as.Nonce, nil, nil
	case "root":
		return &node.Link{Cid: keccak256ToCid(MEthStorageTrie, as.Root)}, nil, nil
	default:
		return nil, nil, fmt.Errorf("no such link")
	}
}

// Tree lists all paths within the object under 'path', and up to the given depth.
// To list the entire object (similar to `find .`) pass "" and -1
func (as *EthAccountSnapshot) Tree(p string, depth int) []string {
	if p != "" || depth == 0 {
		return nil
	}
	return []string{"balance", "codeHash", "nonce", "root"}
}

// ResolveLink is a helper function that calls resolve and asserts the
// output is a link
func (as *EthAccountSnapshot) ResolveLink(p []string) (*node.Link, []string, error) {
	obj, rest, err := as.Resolve(p)
	if err != nil {
		return nil, nil, err
	}

	if lnk, ok := obj.(*node.Link); ok {
		return lnk, rest, nil
	}

	return nil, nil, fmt.Errorf("resolved item was not a link")
}

// Copy will go away. It is here to comply with the interface.
func (as *EthAccountSnapshot) Copy() node.Node {
	panic("dont use this yet")
}

// Links is a helper function that returns all links within this object
func (as *EthAccountSnapshot) Links() []*node.Link {
	return nil
}

// Stat will go away. It is here to comply with the interface.
func (as *EthAccountSnapshot) Stat() (*node.NodeStat, error) {
	return &node.NodeStat{}, nil
}

// Size will go away. It is here to comply with the interface.
func (as *EthAccountSnapshot) Size() (uint64, error) {
	return 0, nil
}

/*
  EthAccountSnapshot functions
*/

// MarshalJSON processes the transaction into readable JSON format.
func (as *EthAccountSnapshot) MarshalJSON() ([]byte, error) {
	out := map[string]interface{}{
		"balance":  as.Balance,
		"codeHash": keccak256ToCid(RawBinary, as.CodeHash),
		"nonce":    as.Nonce,
		"root":     keccak256ToCid(MEthStorageTrie, as.Root),
	}
	return json.Marshal(out)
}

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
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipfs/go-cid"
	node "github.com/ipfs/go-ipld-format"
	mh "github.com/multiformats/go-multihash"
)

// EthTx (eth-tx codec 0x93) represents an ethereum transaction
type EthTx struct {
	*types.Transaction

	cid     cid.Cid
	rawdata []byte
}

// Static (compile time) check that EthTx satisfies the node.Node interface.
var _ node.Node = (*EthTx)(nil)

/*
  INPUT
*/

// NewEthTx converts a *types.Transaction to an EthTx IPLD node
func NewEthTx(tx *types.Transaction) (*EthTx, error) {
	txRLP, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return nil, err
	}
	c, err := RawdataToCid(MEthTx, txRLP, mh.KECCAK_256)
	if err != nil {
		return nil, err
	}
	return &EthTx{
		Transaction: tx,
		cid:         c,
		rawdata:     txRLP,
	}, nil
}

/*
 OUTPUT
*/

// DecodeEthTx takes a cid and its raw binary data
// from IPFS and returns an EthTx object for further processing.
func DecodeEthTx(c cid.Cid, b []byte) (*EthTx, error) {
	var t *types.Transaction
	if err := rlp.DecodeBytes(b, t); err != nil {
		return nil, err
	}
	return &EthTx{
		Transaction: t,
		cid:         c,
		rawdata:     b,
	}, nil
}

/*
  Block INTERFACE
*/

// RawData returns the binary of the RLP encode of the transaction.
func (t *EthTx) RawData() []byte {
	return t.rawdata
}

// Cid returns the cid of the transaction.
func (t *EthTx) Cid() cid.Cid {
	return t.cid
}

// String is a helper for output
func (t *EthTx) String() string {
	return fmt.Sprintf("<EthereumTx %s>", t.cid)
}

// Loggable returns in a map the type of IPLD Link.
func (t *EthTx) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"type": "eth-tx",
	}
}

/*
  Node INTERFACE
*/

// Resolve resolves a path through this node, stopping at any link boundary
// and returning the object found as well as the remaining path to traverse
func (t *EthTx) Resolve(p []string) (interface{}, []string, error) {
	if len(p) == 0 {
		return t, nil, nil
	}

	if len(p) > 1 {
		return nil, nil, fmt.Errorf("unexpected path elements past %s", p[0])
	}

	switch p[0] {

	case "gas":
		return t.Gas(), nil, nil
	case "gasPrice":
		return t.GasPrice(), nil, nil
	case "input":
		return fmt.Sprintf("%x", t.Data()), nil, nil
	case "nonce":
		return t.Nonce(), nil, nil
	case "r":
		_, r, _ := t.RawSignatureValues()
		return hexutil.EncodeBig(r), nil, nil
	case "s":
		_, _, s := t.RawSignatureValues()
		return hexutil.EncodeBig(s), nil, nil
	case "toAddress":
		return t.To(), nil, nil
	case "v":
		v, _, _ := t.RawSignatureValues()
		return hexutil.EncodeBig(v), nil, nil
	case "value":
		return hexutil.EncodeBig(t.Value()), nil, nil
	default:
		return nil, nil, fmt.Errorf("no such link")
	}
}

// Tree lists all paths within the object under 'path', and up to the given depth.
// To list the entire object (similar to `find .`) pass "" and -1
func (t *EthTx) Tree(p string, depth int) []string {
	if p != "" || depth == 0 {
		return nil
	}
	return []string{"gas", "gasPrice", "input", "nonce", "r", "s", "toAddress", "v", "value"}
}

// ResolveLink is a helper function that calls resolve and asserts the
// output is a link
func (t *EthTx) ResolveLink(p []string) (*node.Link, []string, error) {
	obj, rest, err := t.Resolve(p)
	if err != nil {
		return nil, nil, err
	}

	if lnk, ok := obj.(*node.Link); ok {
		return lnk, rest, nil
	}

	return nil, nil, fmt.Errorf("resolved item was not a link")
}

// Copy will go away. It is here to comply with the interface.
func (t *EthTx) Copy() node.Node {
	panic("implement me")
}

// Links is a helper function that returns all links within this object
func (t *EthTx) Links() []*node.Link {
	return nil
}

// Stat will go away. It is here to comply with the interface.
func (t *EthTx) Stat() (*node.NodeStat, error) {
	return &node.NodeStat{}, nil
}

// Size will go away. It is here to comply with the interface.
func (t *EthTx) Size() (uint64, error) {
	return strconv.ParseUint(t.Transaction.Size().String(), 10, 64)
}

/*
  EthTx functions
*/

// MarshalJSON processes the transaction into readable JSON format.
func (t *EthTx) MarshalJSON() ([]byte, error) {
	v, r, s := t.RawSignatureValues()

	out := map[string]interface{}{
		"gas":       t.Gas(),
		"gasPrice":  hexutil.EncodeBig(t.GasPrice()),
		"input":     fmt.Sprintf("%x", t.Data()),
		"nonce":     t.Nonce(),
		"r":         hexutil.EncodeBig(r),
		"s":         hexutil.EncodeBig(s),
		"toAddress": t.To(),
		"v":         hexutil.EncodeBig(v),
		"value":     hexutil.EncodeBig(t.Value()),
	}
	return json.Marshal(out)
}

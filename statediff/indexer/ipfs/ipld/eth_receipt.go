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

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipfs/go-cid"
	node "github.com/ipfs/go-ipld-format"
	mh "github.com/multiformats/go-multihash"
)

type EthReceipt struct {
	*types.Receipt

	rawdata []byte
	cid     cid.Cid
}

// Static (compile time) check that EthReceipt satisfies the node.Node interface.
var _ node.Node = (*EthReceipt)(nil)

/*
  INPUT
*/

// NewReceipt converts a types.ReceiptForStorage to an EthReceipt IPLD node
func NewReceipt(receipt *types.Receipt) (*EthReceipt, error) {
	receiptRLP, err := rlp.EncodeToBytes(receipt)
	if err != nil {
		return nil, err
	}
	c, err := RawdataToCid(MEthTxReceipt, receiptRLP, mh.KECCAK_256)
	if err != nil {
		return nil, err
	}
	return &EthReceipt{
		Receipt: receipt,
		cid:     c,
		rawdata: receiptRLP,
	}, nil
}

/*
 OUTPUT
*/

// DecodeEthReceipt takes a cid and its raw binary data
// from IPFS and returns an EthTx object for further processing.
func DecodeEthReceipt(c cid.Cid, b []byte) (*EthReceipt, error) {
	var r *types.Receipt
	if err := rlp.DecodeBytes(b, r); err != nil {
		return nil, err
	}
	return &EthReceipt{
		Receipt: r,
		cid:     c,
		rawdata: b,
	}, nil
}

/*
  Block INTERFACE
*/

func (node *EthReceipt) RawData() []byte {
	return node.rawdata
}

func (node *EthReceipt) Cid() cid.Cid {
	return node.cid
}

// String is a helper for output
func (r *EthReceipt) String() string {
	return fmt.Sprintf("<EthereumReceipt %s>", r.cid)
}

// Loggable returns in a map the type of IPLD Link.
func (r *EthReceipt) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"type": "eth-receipt",
	}
}

// Resolve resolves a path through this node, stopping at any link boundary
// and returning the object found as well as the remaining path to traverse
func (r *EthReceipt) Resolve(p []string) (interface{}, []string, error) {
	if len(p) == 0 {
		return r, nil, nil
	}

	if len(p) > 1 {
		return nil, nil, fmt.Errorf("unexpected path elements past %s", p[0])
	}

	switch p[0] {

	case "root":
		return r.PostState, nil, nil
	case "status":
		return r.Status, nil, nil
	case "cumulativeGasUsed":
		return r.CumulativeGasUsed, nil, nil
	case "logsBloom":
		return r.Bloom, nil, nil
	case "logs":
		return r.Logs, nil, nil
	case "transactionHash":
		return r.TxHash, nil, nil
	case "contractAddress":
		return r.ContractAddress, nil, nil
	case "gasUsed":
		return r.GasUsed, nil, nil
	default:
		return nil, nil, fmt.Errorf("no such link")
	}
}

// Tree lists all paths within the object under 'path', and up to the given depth.
// To list the entire object (similar to `find .`) pass "" and -1
func (r *EthReceipt) Tree(p string, depth int) []string {
	if p != "" || depth == 0 {
		return nil
	}
	return []string{"root", "status", "cumulativeGasUsed", "logsBloom", "logs", "transactionHash", "contractAddress", "gasUsed"}
}

// ResolveLink is a helper function that calls resolve and asserts the
// output is a link
func (r *EthReceipt) ResolveLink(p []string) (*node.Link, []string, error) {
	obj, rest, err := r.Resolve(p)
	if err != nil {
		return nil, nil, err
	}

	if lnk, ok := obj.(*node.Link); ok {
		return lnk, rest, nil
	}

	return nil, nil, fmt.Errorf("resolved item was not a link")
}

// Copy will go away. It is here to comply with the Node interface.
func (*EthReceipt) Copy() node.Node {
	panic("implement me")
}

// Links is a helper function that returns all links within this object
func (*EthReceipt) Links() []*node.Link {
	return nil
}

// Stat will go away. It is here to comply with the interface.
func (r *EthReceipt) Stat() (*node.NodeStat, error) {
	return &node.NodeStat{}, nil
}

// Size will go away. It is here to comply with the interface.
func (r *EthReceipt) Size() (uint64, error) {
	return strconv.ParseUint(r.Receipt.Size().String(), 10, 64)
}

/*
  EthReceipt functions
*/

// MarshalJSON processes the receipt into readable JSON format.
func (r *EthReceipt) MarshalJSON() ([]byte, error) {
	out := map[string]interface{}{
		"root":              r.PostState,
		"status":            r.Status,
		"cumulativeGasUsed": r.CumulativeGasUsed,
		"logsBloom":         r.Bloom,
		"logs":              r.Logs,
		"transactionHash":   r.TxHash,
		"contractAddress":   r.ContractAddress,
		"gasUsed":           r.GasUsed,
	}
	return json.Marshal(out)
}

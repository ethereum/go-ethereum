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

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipfs/go-cid"
	node "github.com/ipfs/go-ipld-format"
	mh "github.com/multiformats/go-multihash"
)

// EthHeader (eth-block, codec 0x90), represents an ethereum block header
type EthHeader struct {
	*types.Header

	cid     cid.Cid
	rawdata []byte
}

// Static (compile time) check that EthHeader satisfies the node.Node interface.
var _ node.Node = (*EthHeader)(nil)

/*
  INPUT
*/

// NewEthHeader converts a *types.Header into an EthHeader IPLD node
func NewEthHeader(header *types.Header) (*EthHeader, error) {
	headerRLP, err := rlp.EncodeToBytes(header)
	if err != nil {
		return nil, err
	}
	c, err := RawdataToCid(MEthHeader, headerRLP, mh.KECCAK_256)
	if err != nil {
		return nil, err
	}
	return &EthHeader{
		Header:  header,
		cid:     c,
		rawdata: headerRLP,
	}, nil
}

/*
 OUTPUT
*/

// DecodeEthHeader takes a cid and its raw binary data
// from IPFS and returns an EthTx object for further processing.
func DecodeEthHeader(c cid.Cid, b []byte) (*EthHeader, error) {
	var h *types.Header
	if err := rlp.DecodeBytes(b, h); err != nil {
		return nil, err
	}
	return &EthHeader{
		Header:  h,
		cid:     c,
		rawdata: b,
	}, nil
}

/*
  Block INTERFACE
*/

// RawData returns the binary of the RLP encode of the block header.
func (b *EthHeader) RawData() []byte {
	return b.rawdata
}

// Cid returns the cid of the block header.
func (b *EthHeader) Cid() cid.Cid {
	return b.cid
}

// String is a helper for output
func (b *EthHeader) String() string {
	return fmt.Sprintf("<EthHeader %s>", b.cid)
}

// Loggable returns a map the type of IPLD Link.
func (b *EthHeader) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"type": "eth-block",
	}
}

/*
  Node INTERFACE
*/

// Resolve resolves a path through this node, stopping at any link boundary
// and returning the object found as well as the remaining path to traverse
func (b *EthHeader) Resolve(p []string) (interface{}, []string, error) {
	if len(p) == 0 {
		return b, nil, nil
	}

	first, rest := p[0], p[1:]

	switch first {
	case "parent":
		return &node.Link{Cid: commonHashToCid(MEthHeader, b.ParentHash)}, rest, nil
	case "receipts":
		return &node.Link{Cid: commonHashToCid(MEthTxReceiptTrie, b.ReceiptHash)}, rest, nil
	case "root":
		return &node.Link{Cid: commonHashToCid(MEthStateTrie, b.Root)}, rest, nil
	case "tx":
		return &node.Link{Cid: commonHashToCid(MEthTxTrie, b.TxHash)}, rest, nil
	case "uncles":
		return &node.Link{Cid: commonHashToCid(MEthHeaderList, b.UncleHash)}, rest, nil
	}

	if len(p) != 1 {
		return nil, nil, fmt.Errorf("unexpected path elements past %s", first)
	}

	switch first {
	case "bloom":
		return b.Bloom, nil, nil
	case "coinbase":
		return b.Coinbase, nil, nil
	case "difficulty":
		return b.Difficulty, nil, nil
	case "extra":
		// This is a []byte. By default they are marshalled into Base64.
		return fmt.Sprintf("0x%x", b.Extra), nil, nil
	case "gaslimit":
		return b.GasLimit, nil, nil
	case "gasused":
		return b.GasUsed, nil, nil
	case "mixdigest":
		return b.MixDigest, nil, nil
	case "nonce":
		return b.Nonce, nil, nil
	case "number":
		return b.Number, nil, nil
	case "time":
		return b.Time, nil, nil
	default:
		return nil, nil, fmt.Errorf("no such link")
	}
}

// Tree lists all paths within the object under 'path', and up to the given depth.
// To list the entire object (similar to `find .`) pass "" and -1
func (b *EthHeader) Tree(p string, depth int) []string {
	if p != "" || depth == 0 {
		return nil
	}

	return []string{
		"time",
		"bloom",
		"coinbase",
		"difficulty",
		"extra",
		"gaslimit",
		"gasused",
		"mixdigest",
		"nonce",
		"number",
		"parent",
		"receipts",
		"root",
		"tx",
		"uncles",
	}
}

// ResolveLink is a helper function that allows easier traversal of links through blocks
func (b *EthHeader) ResolveLink(p []string) (*node.Link, []string, error) {
	obj, rest, err := b.Resolve(p)
	if err != nil {
		return nil, nil, err
	}

	if lnk, ok := obj.(*node.Link); ok {
		return lnk, rest, nil
	}

	return nil, nil, fmt.Errorf("resolved item was not a link")
}

// Copy will go away. It is here to comply with the Node interface.
func (b *EthHeader) Copy() node.Node {
	panic("implement me")
}

// Links is a helper function that returns all links within this object
// HINT: Use `ipfs refs <cid>`
func (b *EthHeader) Links() []*node.Link {
	return []*node.Link{
		{Cid: commonHashToCid(MEthHeader, b.ParentHash)},
		{Cid: commonHashToCid(MEthTxReceiptTrie, b.ReceiptHash)},
		{Cid: commonHashToCid(MEthStateTrie, b.Root)},
		{Cid: commonHashToCid(MEthTxTrie, b.TxHash)},
		{Cid: commonHashToCid(MEthHeaderList, b.UncleHash)},
	}
}

// Stat will go away. It is here to comply with the Node interface.
func (b *EthHeader) Stat() (*node.NodeStat, error) {
	return &node.NodeStat{}, nil
}

// Size will go away. It is here to comply with the Node interface.
func (b *EthHeader) Size() (uint64, error) {
	return 0, nil
}

/*
  EthHeader functions
*/

// MarshalJSON processes the block header into readable JSON format,
// converting the right links into their cids, and keeping the original
// hex hash, allowing the user to simplify external queries.
func (b *EthHeader) MarshalJSON() ([]byte, error) {
	out := map[string]interface{}{
		"time":       b.Time,
		"bloom":      b.Bloom,
		"coinbase":   b.Coinbase,
		"difficulty": b.Difficulty,
		"extra":      fmt.Sprintf("0x%x", b.Extra),
		"gaslimit":   b.GasLimit,
		"gasused":    b.GasUsed,
		"mixdigest":  b.MixDigest,
		"nonce":      b.Nonce,
		"number":     b.Number,
		"parent":     commonHashToCid(MEthHeader, b.ParentHash),
		"receipts":   commonHashToCid(MEthTxReceiptTrie, b.ReceiptHash),
		"root":       commonHashToCid(MEthStateTrie, b.Root),
		"tx":         commonHashToCid(MEthTxTrie, b.TxHash),
		"uncles":     commonHashToCid(MEthHeaderList, b.UncleHash),
	}
	return json.Marshal(out)
}

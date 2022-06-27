package trie

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"reflect"
	"unsafe"

	zkt "github.com/scroll-tech/go-ethereum/core/types/zktrie"
)

// NodeType defines the type of node in the MT.
type NodeType byte

const (
	// NodeTypeMiddle indicates the type of middle Node that has children.
	NodeTypeMiddle NodeType = 0
	// NodeTypeLeaf indicates the type of a leaf Node that contains a key &
	// value.
	NodeTypeLeaf NodeType = 1
	// NodeTypeEmpty indicates the type of an empty Node.
	NodeTypeEmpty NodeType = 2

	// DBEntryTypeRoot indicates the type of a DB entry that indicates the
	// current Root of a MerkleTree
	DBEntryTypeRoot NodeType = 3
)

// Node is the struct that represents a node in the MT. The node should not be
// modified after creation because the cached key won't be updated.
type Node struct {
	// Type is the type of node in the tree.
	Type NodeType
	// ChildL is the left child of a middle node.
	ChildL *zkt.Hash
	// ChildR is the right child of a middle node.
	ChildR *zkt.Hash
	// NodeKey is the node's key stored in a leaf node.
	NodeKey *zkt.Hash
	// ValuePreimage can store at most 256 byte32 as fields (represnted by BIG-ENDIAN integer)
	// and the first 24 can be compressed (each bytes32 consider as 2 fields), in hashing the compressed
	// elemments would be calculated first
	ValuePreimage []zkt.Byte32
	// CompressedFlags use each bit for indicating the compressed flag for the first 24 fields
	CompressedFlags uint32
	// key is the cache of entry key used to avoid recalculating
	key *zkt.Hash
	// valueHash is the cache of hashes of valuePreimage, used to avoid recalculating
	valueHash *zkt.Hash
	// KeyPreimage is kept here only for proof
	KeyPreimage *zkt.Byte32
}

// NewNodeLeaf creates a new leaf node.
func NewNodeLeaf(k *zkt.Hash, valueFlags uint32, valuePreimage []zkt.Byte32) *Node {
	return &Node{Type: NodeTypeLeaf, NodeKey: k, CompressedFlags: valueFlags, ValuePreimage: valuePreimage}
}

// NewNodeMiddle creates a new middle node.
func NewNodeMiddle(childL *zkt.Hash, childR *zkt.Hash) *Node {
	return &Node{Type: NodeTypeMiddle, ChildL: childL, ChildR: childR}
}

// NewNodeEmpty creates a new empty node.
func NewNodeEmpty() *Node {
	return &Node{Type: NodeTypeEmpty}
}

// NewNodeFromBytes creates a new node by parsing the input []byte.
func NewNodeFromBytes(b []byte) (*Node, error) {
	if len(b) < 1 {
		return nil, ErrNodeBytesBadSize
	}
	n := Node{Type: NodeType(b[0])}
	b = b[1:]
	switch n.Type {
	case NodeTypeMiddle:
		if len(b) != 2*zkt.ElemBytesLen {
			return nil, ErrNodeBytesBadSize
		}
		n.ChildL, n.ChildR = &zkt.Hash{}, &zkt.Hash{}
		copy(n.ChildL[:], b[:zkt.ElemBytesLen])
		copy(n.ChildR[:], b[zkt.ElemBytesLen:zkt.ElemBytesLen*2])
	case NodeTypeLeaf:
		if len(b) < zkt.ElemBytesLen+4 {
			return nil, ErrNodeBytesBadSize
		}
		n.NodeKey = &zkt.Hash{}
		copy(n.NodeKey[:], b[0:32])
		mark := binary.LittleEndian.Uint32(b[32:36])
		preimageLen := int(mark & 255)
		n.CompressedFlags = mark >> 8
		n.ValuePreimage = make([]zkt.Byte32, preimageLen)
		curPos := 36
		for i := 0; i < preimageLen; i++ {
			copy(n.ValuePreimage[i][:], b[i*32+curPos:(i+1)*32+curPos])
		}
		curPos = 36 + preimageLen*32
		preImageSize := int(b[curPos])
		curPos += 1
		if preImageSize != 0 {
			n.KeyPreimage = new(zkt.Byte32)
			copy(n.KeyPreimage[:], b[curPos:curPos+preImageSize])
		}
	case NodeTypeEmpty:
		break
	default:
		return nil, ErrInvalidNodeFound
	}
	return &n, nil
}

// LeafKey computes the key of a leaf node given the hIndex and hValue of the
// entry of the leaf.
func LeafKey(k, v *zkt.Hash) (*zkt.Hash, error) {
	return zkt.HashElems(big.NewInt(1), k.BigInt(), v.BigInt())
}

// Key computes the key of the node by hashing the content in a specific way
// for each type of node.  This key is used as the hash of the merklee tree for
// each node.
func (n *Node) Key() (*zkt.Hash, error) {
	if n.key == nil { // Cache the key to avoid repeated hash computations.
		// NOTE: We are not using the type to calculate the hash!
		switch n.Type {
		case NodeTypeMiddle: // H(ChildL || ChildR)
			var err error
			n.key, err = zkt.HashElems(n.ChildL.BigInt(), n.ChildR.BigInt())
			if err != nil {
				return nil, err
			}
		case NodeTypeLeaf:
			var err error
			n.valueHash, err = zkt.PreHandlingElems(n.CompressedFlags, n.ValuePreimage)
			if err != nil {
				return nil, err
			}

			n.key, err = LeafKey(n.NodeKey, n.valueHash)
			if err != nil {
				return nil, err
			}

		case NodeTypeEmpty: // Zero
			n.key = &zkt.HashZero
		default:
			n.key = &zkt.HashZero
		}
	}
	return n.key, nil
}

func (n *Node) ValueKey() (*zkt.Hash, error) {
	if _, err := n.Key(); err != nil {
		return nil, err
	}
	return n.valueHash, nil
}

// Data returns the wrapped data inside LeafNode and cast them into bytes
// for other node type it just return nil
func (n *Node) Data() []byte {
	switch n.Type {
	case NodeTypeLeaf:
		var data []byte
		hdata := (*reflect.SliceHeader)(unsafe.Pointer(&data))
		//TODO: uintptr(reflect.ValueOf(n.ValuePreimage).UnsafePointer()) should be more elegant but only available until go 1.18
		hdata.Data = uintptr(unsafe.Pointer(&n.ValuePreimage[0]))
		hdata.Len = 32 * len(n.ValuePreimage)
		hdata.Cap = hdata.Len
		return data
	default:
		return nil
	}
}

// Value returns the value of the node.  This is the content that is stored in
// the backend database.
func (n *Node) Value() []byte {
	switch n.Type {
	case NodeTypeMiddle: // {Type || ChildL || ChildR}
		bytes := []byte{byte(n.Type)}
		bytes = append(bytes, n.ChildL[:]...)
		bytes = append(bytes, n.ChildR[:]...)
		return bytes
	case NodeTypeLeaf: // {Type || Data...}
		bytes := []byte{byte(n.Type)}
		bytes = append(bytes, n.NodeKey[:]...)
		tmp := make([]byte, 4)
		compressedFlag := (n.CompressedFlags << 8) + uint32(len(n.ValuePreimage))
		binary.LittleEndian.PutUint32(tmp, compressedFlag)
		bytes = append(bytes, tmp...)
		for _, elm := range n.ValuePreimage {
			bytes = append(bytes, elm[:]...)
		}
		if n.KeyPreimage != nil {
			bytes = append(bytes, byte(len(n.KeyPreimage)))
			bytes = append(bytes, n.KeyPreimage[:]...)
		} else {
			bytes = append(bytes, 0)
		}

		return bytes
	case NodeTypeEmpty: // { Type }
		return []byte{byte(n.Type)}
	default:
		return []byte{}
	}
}

// String outputs a string representation of a node (different for each type).
func (n *Node) String() string {
	switch n.Type {
	case NodeTypeMiddle: // {Type || ChildL || ChildR}
		return fmt.Sprintf("Middle L:%s R:%s", n.ChildL, n.ChildR)
	case NodeTypeLeaf: // {Type || Data...}
		return fmt.Sprintf("Leaf I:%v Items: %d, First:%v", n.NodeKey, len(n.ValuePreimage), n.ValuePreimage[0])
	case NodeTypeEmpty: // {}
		return "Empty"
	default:
		return "Invalid Node"
	}
}

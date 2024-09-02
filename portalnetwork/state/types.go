package state

import (
	"errors"
	"fmt"

	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
)

var _ common.SSZObj = (*Nibbles)(nil)

type Nibbles struct {
	Nibbles []byte
}

func (n *Nibbles) Serialize(w *codec.EncodingWriter) error {
	if len(n.Nibbles)%2 == 0 {
		err := w.WriteByte(0)
		if err != nil {
			return err
		}

		for i := 0; i < len(n.Nibbles); i += 2 {
			err = w.WriteByte(n.Nibbles[i]<<4 | n.Nibbles[i+1])
			if err != nil {
				return err
			}
		}
	} else {
		err := w.WriteByte(0x10 | n.Nibbles[0])
		if err != nil {
			return err
		}

		for i := 1; i < len(n.Nibbles); i += 2 {
			err = w.WriteByte(n.Nibbles[i]<<4 | n.Nibbles[i+1])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (n *Nibbles) ByteLength() uint64 {
	return uint64(len(n.Nibbles)/2 + 1)
}

func (n *Nibbles) FixedLength() uint64 {
	return 0
}

func (n *Nibbles) Deserialize(dr *codec.DecodingReader) error {
	firstByte, err := dr.ReadByte()
	if err != nil {
		return err
	}

	packedNibbles := make([]byte, dr.Scope())
	_, err = dr.Read(packedNibbles)
	if err != nil {
		return err
	}
	flag, first := unpackNibblePair(firstByte)
	nibbles := make([]byte, 0, 1+2*len(packedNibbles))

	if flag == 0 {
		if first != 0 {
			return fmt.Errorf("nibbles: The lowest 4 bits of the first byte must be 0, but was: %x", first)
		}
	} else if flag == 1 {
		nibbles = append(nibbles, first)
	} else {
		return fmt.Errorf("nibbles: The highest 4 bits of the first byte must be 0 or 1, but was: %x", flag)
	}

	for _, b := range packedNibbles {
		left, right := unpackNibblePair(b)
		nibbles = append(nibbles, left, right)
	}

	unpackedNibbles, err := FromUnpackedNibbles(nibbles)
	if err != nil {
		return err
	}

	*n = *unpackedNibbles
	return nil
}

func (n *Nibbles) HashTreeRoot(h tree.HashFn) tree.Root {
	//TODO implement me
	panic("implement me")
}

func FromUnpackedNibbles(nibbles []byte) (*Nibbles, error) {
	if len(nibbles) > 64 {
		return nil, errors.New("too many nibbles")
	}

	for _, nibble := range nibbles {
		if nibble > 0xf {
			return nil, errors.New("nibble out of range")
		}
	}

	return &Nibbles{Nibbles: nibbles}, nil
}

func unpackNibblePair(pair byte) (byte, byte) {
	return pair >> 4, pair & 0xf
}

// test data from
// https://github.com/ethereum/portal-network-specs/blob/master/state/state-network-test-vectors.md
type AccountTrieNodeKey struct {
	Path     Nibbles
	NodeHash common.Bytes32
}

func (a *AccountTrieNodeKey) Deserialize(dr *codec.DecodingReader) error {
	return dr.Container(
		&a.Path,
		&a.NodeHash,
	)
}

func (a *AccountTrieNodeKey) Serialize(w *codec.EncodingWriter) error {
	return w.Container(
		&a.Path,
		&a.NodeHash,
	)
}

func (a *AccountTrieNodeKey) ByteLength(spec *common.Spec) uint64 {
	return codec.ContainerLength(
		&a.Path,
		&a.NodeHash,
	)
}

func (a *AccountTrieNodeKey) FixedLength(spec *common.Spec) uint64 {
	return 0
}

func (a *AccountTrieNodeKey) HashTreeRoot(spec *common.Spec, hFn tree.HashFn) common.Root {
	return hFn.HashTreeRoot(
		&a.Path,
		&a.NodeHash,
	)
}

type ContractStorageTrieNodeKey struct {
	AddressHash common.Bytes32
	Path        Nibbles
	NodeHash    common.Bytes32
}

func (c *ContractStorageTrieNodeKey) Deserialize(dr *codec.DecodingReader) error {
	return dr.Container(
		&c.AddressHash,
		&c.Path,
		&c.NodeHash,
	)
}

func (c *ContractStorageTrieNodeKey) Serialize(w *codec.EncodingWriter) error {
	return w.Container(
		&c.AddressHash,
		&c.Path,
		&c.NodeHash,
	)
}

func (c *ContractStorageTrieNodeKey) ByteLength(spec *common.Spec) uint64 {
	return codec.ContainerLength(
		&c.AddressHash,
		&c.Path,
		&c.NodeHash,
	)
}

func (c *ContractStorageTrieNodeKey) FixedLength(spec *common.Spec) uint64 {
	return 0
}

func (c *ContractStorageTrieNodeKey) HashTreeRoot(spec *common.Spec, hFn tree.HashFn) common.Root {
	return hFn.HashTreeRoot(
		&c.AddressHash,
		&c.Path,
		&c.NodeHash,
	)
}

type ContractBytecodeKey struct {
	AddressHash common.Bytes32
	CodeHash    common.Bytes32
}

func (c *ContractBytecodeKey) Deserialize(dr *codec.DecodingReader) error {
	return dr.FixedLenContainer(
		&c.AddressHash,
		&c.CodeHash,
	)
}

func (c *ContractBytecodeKey) Serialize(w *codec.EncodingWriter) error {
	return w.FixedLenContainer(
		&c.AddressHash,
		&c.CodeHash,
	)
}

func (c *ContractBytecodeKey) ByteLength(spec *common.Spec) uint64 {
	return codec.ContainerLength(
		&c.AddressHash,
		&c.CodeHash,
	)
}

func (c *ContractBytecodeKey) FixedLength(spec *common.Spec) uint64 {
	return 0
}

func (c *ContractBytecodeKey) HashTreeRoot(spec *common.Spec, hFn tree.HashFn) common.Root {
	return hFn.HashTreeRoot(
		&c.AddressHash,
		&c.CodeHash,
	)
}

const MaxTrieNodeLength = 1026

type EncodedTrieNode []byte

func (e *EncodedTrieNode) Deserialize(dr *codec.DecodingReader) error {
	return dr.ByteList((*[]byte)(e), uint64(MaxTrieNodeLength))
}

func (e EncodedTrieNode) Serialize(w *codec.EncodingWriter) error {
	return w.Write(e)
}

func (e EncodedTrieNode) ByteLength() (out uint64) {
	return uint64(len(e))
}

func (e *EncodedTrieNode) FixedLength() uint64 {
	return 0
}

func (e EncodedTrieNode) HashTreeRoot(hFn tree.HashFn) tree.Root {
	return hFn.ByteListHTR(e, MaxTrieNodeLength)
}

// A content value type, used when retrieving a trie node.
type TrieNode struct {
	Node EncodedTrieNode
}

func (t *TrieNode) Deserialize(dr *codec.DecodingReader) error {
	return dr.Container(
		&t.Node,
	)
}

func (t TrieNode) Serialize(w *codec.EncodingWriter) error {
	return w.Container(&t.Node)
}

func (t TrieNode) ByteLength() (out uint64) {
	return codec.ContainerLength(&t.Node)
}

func (t *TrieNode) FixedLength() uint64 {
	return 0
}

func (t TrieNode) HashTreeRoot(hFn tree.HashFn) tree.Root {
	return hFn.HashTreeRoot(&t.Node)
}

const MaxTrieProofLength = 65

type TrieProof []EncodedTrieNode

func (r *TrieProof) Deserialize(dr *codec.DecodingReader) error {
	return dr.List(func() codec.Deserializable {
		i := len(*r)
		*r = append(*r, EncodedTrieNode{})
		return &((*r)[i])
	}, 0, MaxTrieProofLength)
}

func (r TrieProof) Serialize(w *codec.EncodingWriter) error {
	return w.List(func(i uint64) codec.Serializable {
		return &r[i]
	}, 0, uint64(len(r)))
}

func (r TrieProof) ByteLength() (out uint64) {
	for _, v := range r {
		out += v.ByteLength() + codec.OFFSET_SIZE
	}
	return
}

func (r *TrieProof) FixedLength() uint64 {
	return 0
}

func (r TrieProof) HashTreeRoot(hFn tree.HashFn) common.Root {
	length := uint64(len(r))
	return hFn.ComplexListHTR(func(i uint64) tree.HTR {
		if i < length {
			return &r[i]
		}
		return nil
	}, length, MaxTrieProofLength)
}

const MaxContractBytecodeLength = 32768

type ContractByteCode []byte

func (t *ContractByteCode) Deserialize(dr *codec.DecodingReader) error {
	return dr.ByteList((*[]byte)(t), uint64(MaxContractBytecodeLength))
}

func (t ContractByteCode) Serialize(w *codec.EncodingWriter) error {
	return w.Write(t)
}

func (t ContractByteCode) ByteLength() (out uint64) {
	return uint64(len(t))
}

func (t *ContractByteCode) FixedLength() uint64 {
	return 0
}

func (t ContractByteCode) HashTreeRoot(hFn tree.HashFn) tree.Root {
	return hFn.ByteListHTR(t, MaxContractBytecodeLength)
}

// A content value type, used when retrieving contract's bytecode.
type ContractBytecodeContainer struct {
	Code ContractByteCode
}

func (t *ContractBytecodeContainer) Deserialize(dr *codec.DecodingReader) error {
	return dr.Container(&t.Code)
}

func (t ContractBytecodeContainer) Serialize(w *codec.EncodingWriter) error {
	return w.Container(&t.Code)
}

func (t ContractBytecodeContainer) ByteLength() (out uint64) {
	return codec.ContainerLength(&t.Code)
}

func (t *ContractBytecodeContainer) FixedLength() uint64 {
	return 0
}

func (t ContractBytecodeContainer) HashTreeRoot(hFn tree.HashFn) tree.Root {
	return hFn.HashTreeRoot(t.Code)
}

// A content value type, used when offering a trie node from the account trie.
type AccountTrieNodeWithProof struct {
	/// An proof for the account trie node.
	Proof TrieProof
	/// A block at which the proof is anchored.
	BlockHash common.Bytes32
}

func (a *AccountTrieNodeWithProof) Deserialize(dr *codec.DecodingReader) error {
	return dr.Container(
		&a.Proof,
		&a.BlockHash,
	)
}

func (a *AccountTrieNodeWithProof) Serialize(w *codec.EncodingWriter) error {
	return w.Container(
		&a.Proof,
		&a.BlockHash,
	)
}

func (a *AccountTrieNodeWithProof) ByteLength() uint64 {
	return codec.ContainerLength(
		&a.Proof,
		&a.BlockHash,
	)
}

func (a *AccountTrieNodeWithProof) FixedLength() uint64 {
	return 0
}

func (a *AccountTrieNodeWithProof) HashTreeRoot(hFn tree.HashFn) common.Root {
	return hFn.HashTreeRoot(
		&a.Proof,
		&a.BlockHash,
	)
}

// A content value type, used when offering a trie node from the contract storage trie.
type ContractStorageTrieNodeWithProof struct {
	// A proof for the contract storage trie node.
	StoregeProof TrieProof
	// A proof for the account state.
	AccountProof TrieProof
	// A block at which the proof is anchored.
	BlockHash common.Bytes32
}

func (c *ContractStorageTrieNodeWithProof) Deserialize(dr *codec.DecodingReader) error {
	return dr.Container(
		&c.StoregeProof,
		&c.AccountProof,
		&c.BlockHash,
	)
}

func (c *ContractStorageTrieNodeWithProof) Serialize(w *codec.EncodingWriter) error {
	return w.Container(
		&c.StoregeProof,
		&c.AccountProof,
		&c.BlockHash,
	)
}

func (c *ContractStorageTrieNodeWithProof) ByteLength() uint64 {
	return codec.ContainerLength(
		&c.StoregeProof,
		&c.AccountProof,
		&c.BlockHash,
	)
}

func (c *ContractStorageTrieNodeWithProof) FixedLength() uint64 {
	return 0
}

func (c *ContractStorageTrieNodeWithProof) HashTreeRoot(hFn tree.HashFn) common.Root {
	return hFn.HashTreeRoot(
		&c.StoregeProof,
		&c.AccountProof,
		&c.BlockHash,
	)
}

// A content value type, used when offering contract's bytecode.
type ContractBytecodeWithProof struct {
	// A contract's bytecode.
	Code ContractByteCode
	// A proof for the account state of the corresponding contract.
	AccountProof TrieProof
	// A block at which the proof is anchored.
	BlockHash common.Bytes32
}

func (c *ContractBytecodeWithProof) Deserialize(dr *codec.DecodingReader) error {
	return dr.Container(
		&c.Code,
		&c.AccountProof,
		&c.BlockHash,
	)
}

func (c *ContractBytecodeWithProof) Serialize(w *codec.EncodingWriter) error {
	return w.Container(
		&c.Code,
		&c.AccountProof,
		&c.BlockHash,
	)
}

func (c *ContractBytecodeWithProof) ByteLength() uint64 {
	return codec.ContainerLength(
		&c.Code,
		&c.AccountProof,
		&c.BlockHash,
	)
}

func (c *ContractBytecodeWithProof) FixedLength() uint64 {
	return 0
}

func (c *ContractBytecodeWithProof) HashTreeRoot(hFn tree.HashFn) common.Root {
	return hFn.HashTreeRoot(
		&c.Code,
		&c.AccountProof,
		&c.BlockHash,
	)
}

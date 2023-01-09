package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/bits"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	Txs int = iota
	Recs
	Withd
)

var lookup map[int]common.Hash

func init() {
	// initialize the lookup
	max := 1 << 30
	base := [32]byte{}
	prev := crypto.Keccak256Hash(base[:])
	lookup[1] = prev
	for i := 2; i < max; i = i << 1 {
		prev = crypto.Keccak256Hash(prev.Bytes(), prev.Bytes())
		lookup[i] = prev
	}
}

// Merkleize computes a binary merkle tree of a derivable list.
// It's similar to deriveSha.
func Merkleize(list DerivableList, typ int) common.Hash {
	valueBuf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(valueBuf)

	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)

	var (
		innerLimit int
		outerLimit int
	)
	switch typ {
	case Txs:
		innerLimit = 1 << 30
		outerLimit = 1 << 20
	case Recs:
		// TODO research limits
	case Withd:
		innerLimit = 1 << 2
		outerLimit = 1 << 4
	}

	size := nextPowerOfTwo(uint32(list.Len()))
	roots := make([]common.Hash, size)
	for i := 0; i < list.Len(); i++ {
		valueBuf.Reset()
		list.EncodeIndex(i, valueBuf)
		bytes := valueBuf.Bytes()
		roots[i] = mixinLength(chunkAndHashSSZ(bytes, innerLimit), len(bytes))
	}
	return mixinLength(merkleWithLimit(roots, outerLimit), list.Len())
}

// chunkAndHashSSZ cuts an input into chunks of 32 bytes
// and creates a merkle tree out of them
func chunkAndHashSSZ(input []byte, limit int) common.Hash {
	size := nextPowerOfTwo(uint32(len(input)))
	roots := make([]common.Hash, size)
	for start := 0; start < len(input); start += common.HashLength {
		end := start + common.HashLength
		if end > len(input) {
			end = len(input)
		}
		copy(roots[start/common.HashLength][:], input[start:end])
	}
	return merkleWithLimit(roots, limit)
}

// merkleWithLimit recursively computes a merkle tree with the help of a LUT.
// The limits are set very liberally which means a lot of the tree can be precomputed.
// All data is left aligned in the leaves, so we can recursively compute the left children.
// The right children can be looked up from the table.
func merkleWithLimit(values []common.Hash, limit int) (h common.Hash) {
	if limit == 0 {
		panic("should not happen")
	}
	if len(values) <= limit/2 {
		return fullMerkle(values)
	}
	left := merkleWithLimit(values, limit/2)
	right := lookup[limit/2]

	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)

	sha.Reset()
	sha.Write(left[:])
	sha.Write(right[:])
	sha.Read(h[:])
	return h
}

// fullMerkle computes a full merkle tree root over a set of values.
// The len(values) needs to be a power of two.
func fullMerkle(values []common.Hash) common.Hash {
	// Check if len(values) is power of two
	if len(values)&(len(values)-1) != 0 || len(values) == 0 {
		panic(fmt.Sprintf("not a power of 2: %v", len(values)))
	}
	if len(values) == 1 {
		return values[0]
	}
	round := make([]common.Hash, len(values))
	copy(round, values)
	for i := 0; i < bits.Len(uint(len(values)))-1; i++ {
		for k := 0; k < len(round)/2; k++ {
			round[k] = crypto.Keccak256Hash(round[k*2][:], round[k*2+1][:])
		}
		round = round[0 : len(round)/2]
	}
	return round[0]
}

// mixinLength mixes in the length into a hash.
func mixinLength(root common.Hash, length int) (h common.Hash) {
	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)

	var buf []byte
	binary.LittleEndian.PutUint64(buf, uint64(length))

	sha.Reset()
	sha.Write(root[:])
	sha.Write(buf[:])
	sha.Read(h[:])
	return h
}

func nextPowerOfTwo(v uint32) uint32 {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return v
}

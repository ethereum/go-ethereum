// Copyright 2021 go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"encoding/binary"
	"sync"

	"github.com/crate-crypto/go-ipa/bandersnatch/fr"
	"github.com/ethereum/go-verkle"
	"github.com/holiman/uint256"
)

const (
	VersionLeafKey    = 0
	BalanceLeafKey    = 1
	NonceLeafKey      = 2
	CodeKeccakLeafKey = 3
	CodeSizeLeafKey   = 4
)

var (
	zero                                = uint256.NewInt(0)
	VerkleNodeWidthLog2                 = 8
	HeaderStorageOffset                 = uint256.NewInt(64)
	mainStorageOffsetLshVerkleNodeWidth = new(uint256.Int).Lsh(uint256.NewInt(1), 248-uint(VerkleNodeWidthLog2))
	CodeOffset                          = uint256.NewInt(128)
	MainStorageOffset                   = new(uint256.Int).Lsh(uint256.NewInt(1), 248 /* 8 * 31*/)
	VerkleNodeWidth                     = uint256.NewInt(256)
	codeStorageDelta                    = uint256.NewInt(0).Sub(CodeOffset, HeaderStorageOffset)

	getTreePolyIndex0Point *verkle.Point
)

type PointCache struct {
	cache map[string]*verkle.Point
	lock  sync.RWMutex
}

func NewPointCache() *PointCache {
	return &PointCache{
		cache: make(map[string]*verkle.Point),
	}
}

func (pc *PointCache) GetTreeKeyHeader(addr []byte) *verkle.Point {
	pc.lock.RLock()
	point, ok := pc.cache[string(addr)]
	pc.lock.RUnlock()
	if ok {
		return point
	}

	point = EvaluateAddressPoint(addr)
	pc.lock.Lock()
	pc.cache[string(addr)] = point
	pc.lock.Unlock()
	return point
}

func (pc *PointCache) GetTreeKeyVersionCached(addr []byte) []byte {
	p := pc.GetTreeKeyHeader(addr)
	v := PointToHash(p, VersionLeafKey)
	return v[:]
}

func init() {
	// The byte array is the Marshalled output of the point computed as such:
	//cfg, _ := verkle.GetConfig()
	//verkle.FromLEBytes(&getTreePolyIndex0Fr[0], []byte{2, 64})
	//= cfg.CommitToPoly(getTreePolyIndex0Fr[:], 1)
	getTreePolyIndex0Point = new(verkle.Point)
	err := getTreePolyIndex0Point.SetBytes([]byte{34, 25, 109, 242, 193, 5, 144, 224, 76, 52, 189, 92, 197, 126, 9, 145, 27, 152, 199, 130, 165, 3, 210, 27, 193, 131, 142, 28, 110, 26, 16, 191})
	if err != nil {
		panic(err)
	}
}

// GetTreeKey performs both the work of the spec's get_tree_key function, and that
// of pedersen_hash: it builds the polynomial in pedersen_hash without having to
// create a mostly zero-filled buffer and "type cast" it to a 128-long 16-byte
// array. Since at most the first 5 coefficients of the polynomial will be non-zero,
// these 5 coefficients are created directly.
func GetTreeKey(address []byte, treeIndex *uint256.Int, subIndex byte) []byte {
	if len(address) < 32 {
		var aligned [32]byte
		address = append(aligned[:32-len(address)], address...)
	}

	// poly = [2+256*64, address_le_low, address_le_high, tree_index_le_low, tree_index_le_high]
	var poly [5]fr.Element

	// 32-byte address, interpreted as two little endian
	// 16-byte numbers.
	verkle.FromLEBytes(&poly[1], address[:16])
	verkle.FromLEBytes(&poly[2], address[16:])

	// treeIndex must be interpreted as a 32-byte aligned little-endian integer.
	// e.g: if treeIndex is 0xAABBCC, we need the byte representation to be 0xCCBBAA00...00.
	// poly[3] = LE({CC,BB,AA,00...0}) (16 bytes), poly[4]=LE({00,00,...}) (16 bytes).
	//
	// To avoid unnecessary endianness conversions for go-ipa, we do some trick:
	// - poly[3]'s byte representation is the same as the *top* 16 bytes (trieIndexBytes[16:]) of
	//   32-byte aligned big-endian representation (BE({00,...,AA,BB,CC})).
	// - poly[4]'s byte representation is the same as the *low* 16 bytes (trieIndexBytes[:16]) of
	//   the 32-byte aligned big-endian representation (BE({00,00,...}).
	trieIndexBytes := treeIndex.Bytes32()
	verkle.FromBytes(&poly[3], trieIndexBytes[16:])
	verkle.FromBytes(&poly[4], trieIndexBytes[:16])

	cfg := verkle.GetConfig()
	ret := cfg.CommitToPoly(poly[:], 0)

	// add a constant point corresponding to poly[0]=[2+256*64].
	ret.Add(ret, getTreePolyIndex0Point)

	return PointToHash(ret, subIndex)
}

func GetTreeKeyAccountLeaf(address []byte, leaf byte) []byte {
	return GetTreeKey(address, zero, leaf)
}

func GetTreeKeyVersion(address []byte) []byte {
	return GetTreeKey(address, zero, VersionLeafKey)
}

func GetTreeKeyVersionWithEvaluatedAddress(addrp *verkle.Point) []byte {
	return GetTreeKeyWithEvaluatedAddess(addrp, zero, VersionLeafKey)
}

func GetTreeKeyBalance(address []byte) []byte {
	return GetTreeKey(address, zero, BalanceLeafKey)
}

func GetTreeKeyNonce(address []byte) []byte {
	return GetTreeKey(address, zero, NonceLeafKey)
}

func GetTreeKeyCodeKeccak(address []byte) []byte {
	return GetTreeKey(address, zero, CodeKeccakLeafKey)
}

func GetTreeKeyCodeSize(address []byte) []byte {
	return GetTreeKey(address, zero, CodeSizeLeafKey)
}

func GetTreeKeyCodeChunk(address []byte, chunk *uint256.Int) []byte {
	treeIndex, subIndex := GetTreeKeyCodeChunkIndices(chunk)
	return GetTreeKey(address, treeIndex, subIndex)
}

func GetTreeKeyCodeChunkIndices(chunk *uint256.Int) (*uint256.Int, byte) {
	chunkOffset := new(uint256.Int).Add(CodeOffset, chunk)
	treeIndex := new(uint256.Int).Div(chunkOffset, VerkleNodeWidth)
	subIndexMod := new(uint256.Int).Mod(chunkOffset, VerkleNodeWidth)
	var subIndex byte
	if len(subIndexMod) != 0 {
		subIndex = byte(subIndexMod[0])
	}
	return treeIndex, subIndex
}

func GetTreeKeyCodeChunkWithEvaluatedAddress(addressPoint *verkle.Point, chunk *uint256.Int) []byte {
	chunkOffset := new(uint256.Int).Add(CodeOffset, chunk)
	treeIndex := new(uint256.Int).Div(chunkOffset, VerkleNodeWidth)
	subIndexMod := new(uint256.Int).Mod(chunkOffset, VerkleNodeWidth)
	var subIndex byte
	if len(subIndexMod) != 0 {
		subIndex = byte(subIndexMod[0])
	}
	return GetTreeKeyWithEvaluatedAddess(addressPoint, treeIndex, subIndex)
}

func PointToHash(evaluated *verkle.Point, suffix byte) []byte {
	// The output of Byte() is big engian for banderwagon. This
	// introduces an imbalance in the tree, because hashes are
	// elements of a 253-bit field. This means more than half the
	// tree would be empty. To avoid this problem, use a little
	// endian commitment and chop the MSB.
	retb := evaluated.Bytes()
	for i := 0; i < 16; i++ {
		retb[31-i], retb[i] = retb[i], retb[31-i]
	}
	retb[31] = suffix
	return retb[:]
}

func GetTreeKeyWithEvaluatedAddess(evaluated *verkle.Point, treeIndex *uint256.Int, subIndex byte) []byte {
	var poly [5]fr.Element

	poly[0].SetZero()
	poly[1].SetZero()
	poly[2].SetZero()

	// little-endian, 32-byte aligned treeIndex
	var index [32]byte
	for i := 0; i < len(treeIndex); i++ {
		binary.LittleEndian.PutUint64(index[i*8:(i+1)*8], treeIndex[i])
	}
	verkle.FromLEBytes(&poly[3], index[:16])
	verkle.FromLEBytes(&poly[4], index[16:])

	cfg := verkle.GetConfig()
	ret := cfg.CommitToPoly(poly[:], 0)

	// add the pre-evaluated address
	ret.Add(ret, evaluated)

	return PointToHash(ret, subIndex)
}

func EvaluateAddressPoint(address []byte) *verkle.Point {
	if len(address) < 32 {
		var aligned [32]byte
		address = append(aligned[:32-len(address)], address...)
	}
	var poly [3]fr.Element

	poly[0].SetZero()

	// 32-byte address, interpreted as two little endian
	// 16-byte numbers.
	verkle.FromLEBytes(&poly[1], address[:16])
	verkle.FromLEBytes(&poly[2], address[16:])

	cfg := verkle.GetConfig()
	ret := cfg.CommitToPoly(poly[:], 0)

	// add a constant point
	ret.Add(ret, getTreePolyIndex0Point)

	return ret
}

func GetTreeKeyStorageSlotWithEvaluatedAddress(evaluated *verkle.Point, storageKey []byte) []byte {
	treeIndex, subIndex := GetTreeKeyStorageSlotTreeIndexes(storageKey)
	return GetTreeKeyWithEvaluatedAddess(evaluated, treeIndex, subIndex)
}

func GetTreeKeyStorageSlotTreeIndexes(storageKey []byte) (*uint256.Int, byte) {
	var pos uint256.Int
	pos.SetBytes(storageKey)

	// If the storage slot is in the header, we need to add the header offset.
	if pos.Cmp(codeStorageDelta) < 0 {
		// This addition is always safe; it can't ever overflow since pos<codeStorageDelta.
		pos.Add(HeaderStorageOffset, &pos)

		// In this branch, the tree-index is zero since we're in the account header,
		// and the sub-index is the LSB of the modified storage key.
		return zero, byte(pos[0] & 0xFF)
	}
	// If the storage slot is in the main storage, we need to add the main storage offset.

	// The first MAIN_STORAGE_OFFSET group will see its
	// first 64 slots unreachable. This is either a typo in the
	// spec or intended to conserve the 256-u256
	// aligment. If we decide to ever access these 64
	// slots, uncomment this.
	// // Get the new offset since we now know that we are above 64.
	// pos.Sub(&pos, codeStorageDelta)
	// suffix := byte(pos[0] & 0xFF)
	suffix := storageKey[len(storageKey)-1]

	// We first divide by VerkleNodeWidth to create room to avoid an overflow next.
	pos.Rsh(&pos, uint(VerkleNodeWidthLog2))

	// We add mainStorageOffset/VerkleNodeWidth which can't overflow.
	pos.Add(&pos, mainStorageOffsetLshVerkleNodeWidth)

	// The sub-index is the LSB of the original storage key, since mainStorageOffset
	// doesn't affect this byte, so we can avoid masks or shifts.
	return &pos, suffix
}

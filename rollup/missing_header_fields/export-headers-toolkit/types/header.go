package types

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

const HeaderSizeSerialized = 2
const VanitySize = 32

type Header struct {
	Number     uint64
	Difficulty uint64
	StateRoot  common.Hash
	Coinbase   common.Address
	Nonce      types.BlockNonce
	ExtraData  []byte
}

func NewHeader(number, difficulty uint64, stateRoot common.Hash, coinbase common.Address, nonce types.BlockNonce, extraData []byte) *Header {
	return &Header{
		Number:     number,
		Difficulty: difficulty,
		StateRoot:  stateRoot,
		Coinbase:   coinbase,
		Nonce:      nonce,
		ExtraData:  extraData,
	}
}

func (h *Header) String() string {
	return fmt.Sprintf("%d,%d,%s,%s,%s,%s\n", h.Number, h.Difficulty, h.StateRoot.Hex(), h.Coinbase.Hex(), common.Bytes2Hex(h.Nonce[:]), common.Bytes2Hex(h.ExtraData))
}

func (h *Header) Equal(other *Header) bool {
	if h.Number != other.Number {
		return false
	}
	if h.Difficulty != other.Difficulty {
		return false
	}
	if h.StateRoot != other.StateRoot {
		return false
	}
	if h.Coinbase != other.Coinbase {
		return false
	}
	if h.Nonce != other.Nonce {
		return false
	}
	if !bytes.Equal(h.ExtraData, other.ExtraData) {
		return false
	}
	return true
}

// Bytes returns the byte representation of the header including the initial 2 bytes for the size.
func (h *Header) Bytes() ([]byte, error) {
	size := 8 + 8 + common.HashLength + common.AddressLength + 8 + len(h.ExtraData)

	buf := make([]byte, HeaderSizeSerialized+size)
	binary.BigEndian.PutUint16(buf[:2], uint16(size))
	binary.BigEndian.PutUint64(buf[2:10], h.Number)
	binary.BigEndian.PutUint64(buf[10:18], h.Difficulty)
	copy(buf[18:50], h.StateRoot[:])
	copy(buf[50:70], h.Coinbase[:])
	copy(buf[70:78], h.Nonce[:])
	copy(buf[78:], h.ExtraData)
	return buf, nil
}

func (h *Header) Vanity() [VanitySize]byte {
	return [VanitySize]byte(h.ExtraData[:VanitySize])
}

func (h *Header) Seal() []byte {
	return h.ExtraData[VanitySize:]
}

func (h *Header) SealLen() int {
	return len(h.Seal())
}

// FromBytes reads the header from the byte representation excluding the initial 2 bytes for the size.
func (h *Header) FromBytes(buf []byte) *Header {
	h.Number = binary.BigEndian.Uint64(buf[:8])
	h.Difficulty = binary.BigEndian.Uint64(buf[8:16])
	if len(buf) < 76 {
		panic(fmt.Sprintf("buffer too short for header: %d bytes", len(buf)))
	}
	h.StateRoot = common.BytesToHash(buf[16:48])
	h.Coinbase = common.BytesToAddress(buf[48:68])
	h.Nonce = types.BlockNonce(buf[68:76])
	h.ExtraData = buf[76:]

	return h
}

type HeaderHeap []*Header

func (h HeaderHeap) Len() int            { return len(h) }
func (h HeaderHeap) Less(i, j int) bool  { return h[i].Number < h[j].Number }
func (h HeaderHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *HeaderHeap) Push(x interface{}) { *h = append(*h, x.(*Header)) }
func (h *HeaderHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

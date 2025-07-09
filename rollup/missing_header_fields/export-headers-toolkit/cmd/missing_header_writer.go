package cmd

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"math"
	"os"
	"sort"

	"github.com/scroll-tech/go-ethereum/common"
	coreTypes "github.com/scroll-tech/go-ethereum/core/types"

	"github.com/scroll-tech/go-ethereum/export-headers-toolkit/types"
)

// maxVanityCount is the maximum number of unique vanities that can be represented with a single byte.
const maxVanityCount = math.MaxUint8

type missingHeaderFileWriter struct {
	file   *os.File
	writer *bufio.Writer

	missingHeaderWriter *missingHeaderWriter
}

func newMissingHeaderFileWriter(filename string, seenVanity map[[32]byte]bool) *missingHeaderFileWriter {
	if len(seenVanity) > maxVanityCount {
		log.Fatalf("Number of unique vanities exceeds maximum: %d > %d", len(seenVanity), maxVanityCount)
	}

	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Error creating file: %v", err)
	}

	writer := bufio.NewWriter(file)
	return &missingHeaderFileWriter{
		file:                file,
		writer:              writer,
		missingHeaderWriter: newMissingHeaderWriter(writer, seenVanity),
	}
}

func (m *missingHeaderFileWriter) close() {
	if err := m.writer.Flush(); err != nil {
		log.Fatalf("Error flushing writer: %v", err)
	}
	if err := m.file.Close(); err != nil {
		log.Fatalf("Error closing file: %v", err)
	}
}

type missingHeaderWriter struct {
	writer io.Writer

	sortedVanities    [][32]byte
	sortedVanitiesMap map[[32]byte]int
	seenDifficulty    map[uint64]int
	seenSealLen       map[int]int
}

func newMissingHeaderWriter(writer io.Writer, seenVanity map[[32]byte]bool) *missingHeaderWriter {
	// sort the vanities and assign an index to each so that we can write the index of the vanity in the header
	sortedVanities := make([][32]byte, 0, len(seenVanity))
	for vanity := range seenVanity {
		sortedVanities = append(sortedVanities, vanity)
	}
	sort.Slice(sortedVanities, func(i, j int) bool {
		return bytes.Compare(sortedVanities[i][:], sortedVanities[j][:]) < 0
	})
	sortedVanitiesMap := make(map[[32]byte]int)
	for i, vanity := range sortedVanities {
		sortedVanitiesMap[vanity] = i
	}

	return &missingHeaderWriter{
		writer:            writer,
		sortedVanities:    sortedVanities,
		sortedVanitiesMap: sortedVanitiesMap,
	}
}

func (m *missingHeaderWriter) writeVanities() {
	// write the count of unique vanities
	if _, err := m.writer.Write([]byte{uint8(len(m.sortedVanitiesMap))}); err != nil {
		log.Fatalf("Error writing unique vanity count: %v", err)
	}

	// write the unique vanities
	for _, vanity := range m.sortedVanities {
		if _, err := m.writer.Write(vanity[:]); err != nil {
			log.Fatalf("Error writing vanity: %v", err)
		}
	}
}

func (m *missingHeaderWriter) write(header *types.Header) {
	// 1. prepare the bitmask
	hasCoinbase := header.Coinbase != (common.Address{})
	hasNonce := header.Nonce != (coreTypes.BlockNonce{})

	bits := newBitMask(hasCoinbase, hasNonce, int(header.Difficulty), header.SealLen())
	vanityIndex := m.sortedVanitiesMap[header.Vanity()]

	if vanityIndex >= maxVanityCount {
		log.Fatalf("Vanity index %d exceeds maximum allowed %d", vanityIndex, maxVanityCount-1)
	}

	// 2. write the header: bitmask, optional coinbase, optional nonce, vanity index and seal
	if _, err := m.writer.Write(bits.Bytes()); err != nil {
		log.Fatalf("Error writing bitmask: %v", err)
	}
	if _, err := m.writer.Write([]byte{uint8(vanityIndex)}); err != nil {
		log.Fatalf("Error writing vanity index: %v", err)
	}
	if _, err := m.writer.Write(header.StateRoot[:]); err != nil {
		log.Fatalf("Error writing state root: %v", err)
	}
	if hasCoinbase {
		if _, err := m.writer.Write(header.Coinbase[:]); err != nil {
			log.Fatalf("Error writing coinbase: %v", err)
		}
	}

	if hasNonce {
		if _, err := m.writer.Write(header.Nonce[:]); err != nil {
			log.Fatalf("Error writing nonce: %v", err)
		}
	}
	if _, err := m.writer.Write(header.Seal()); err != nil {
		log.Fatalf("Error writing seal: %v", err)
	}
}

// bitMask is a bitmask that encodes the following information:
//
// bit 4: 1 if the header has a coinbase field
// bit 5: 1 if the header has a nonce field
// bit 6: 0 if difficulty is 2, 1 if difficulty is 1
// bit 7: 0 if seal length is 65, 1 if seal length is 85
type bitMask struct {
	b uint8
}

func newBitMaskFromByte(b uint8) bitMask {
	return bitMask{b}
}

func newBitMask(hasCoinbase bool, hasNonce bool, difficulty int, sealLen int) bitMask {
	b := uint8(0)

	if hasCoinbase {
		b |= 1 << 4
	}

	if hasNonce {
		b |= 1 << 5
	}
	if difficulty == 1 {
		b |= 1 << 6
	} else if difficulty != 2 {
		log.Fatalf("Invalid difficulty: %d", difficulty)
	}

	if sealLen == 85 {
		b |= 1 << 7
	} else if sealLen != 65 {
		log.Fatalf("Invalid seal length: %d", sealLen)
	}

	return bitMask{b}
}

func (b bitMask) difficulty() int {
	val := (b.b >> 6) & 0x01
	if val == 0 {
		return 2
	} else {
		return 1
	}
}

func (b bitMask) sealLen() int {
	val := (b.b >> 7) & 0x01
	if val == 0 {
		return 65
	} else {
		return 85
	}
}

func (b bitMask) hasCoinbase() bool {
	return (b.b>>4)&0x01 == 1
}

func (b bitMask) hasNonce() bool {
	return (b.b>>5)&0x01 == 1
}

func (b bitMask) Bytes() []byte {
	return []byte{b.b}
}

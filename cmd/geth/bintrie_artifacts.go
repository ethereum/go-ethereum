// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/bintrie"
)

// The EIP-8347 distribution artifacts: two byte-canonical files every
// correct producer reproduces bit for bit, so one keccak digest per file
// lets nodes compare sources before downloading.

// snapshotHeaderSize is the fixed artifact header: pbtRoot[32] followed by
// leafCount[8, big-endian].
const snapshotHeaderSize = 40

// snapshotRecord is one artifact leaf: the full tree key and the value as a
// canonical integer (zero values cannot occur).
type snapshotRecord struct {
	Key   []byte
	Value []byte
}

// snapshotWriter streams the PBT snapshot artifact: a placeholder header,
// one RLP record per sorted leaf, the header backpatched at finalize.
type snapshotWriter struct {
	path  string
	f     *os.File
	w     *bufio.Writer
	count uint64
}

// newSnapshotWriter creates the artifact file and reserves its header.
func newSnapshotWriter(path string) (*snapshotWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	w := bufio.NewWriterSize(f, 1<<20)
	if _, err := w.Write(make([]byte, snapshotHeaderSize)); err != nil {
		f.Close()
		return nil, err
	}
	return &snapshotWriter{path: path, f: f, w: w}, nil
}

// add appends one leaf record.
func (sw *snapshotWriter) add(key, value []byte) error {
	if err := rlp.Encode(sw.w, &snapshotRecord{Key: key, Value: common.TrimLeftZeroes(value)}); err != nil {
		return err
	}
	sw.count++
	return nil
}

// finalize backpatches the header, re-reads the artifact for its digest and
// closes the file.
func (sw *snapshotWriter) finalize(root common.Hash) (common.Hash, error) {
	defer sw.f.Close()
	if err := sw.w.Flush(); err != nil {
		return common.Hash{}, err
	}
	var header [snapshotHeaderSize]byte
	copy(header[:32], root[:])
	binary.BigEndian.PutUint64(header[32:], sw.count)
	if _, err := sw.f.WriteAt(header[:], 0); err != nil {
		return common.Hash{}, err
	}
	// The digest names the file; make its bytes durable first.
	if err := sw.f.Sync(); err != nil {
		return common.Hash{}, err
	}
	if _, err := sw.f.Seek(0, io.SeekStart); err != nil {
		return common.Hash{}, err
	}
	hasher := crypto.NewKeccakState()
	if _, err := io.Copy(hasher, bufio.NewReaderSize(sw.f, 1<<20)); err != nil {
		return common.Hash{}, err
	}
	var digest common.Hash
	hasher.Read(digest[:])
	return digest, nil
}

// abort removes a partially written artifact after a failed conversion.
func (sw *snapshotWriter) abort() {
	sw.f.Close()
	os.Remove(sw.path)
}

// preimageRecordHeaderSize is the fixed per-account prefix: address[20]
// followed by slotCount[4, big-endian].
const preimageRecordHeaderSize = common.AddressLength + 4

// preimageFile streams the EIP-8347 preimage file: fixed-width per-account
// records in MPT-path order, keccak256(address) across records and
// keccak256(slotKey) within one. The scan already walks in that order; the
// writer checks the paths it is handed for strict ascent and buffers only the
// open account's slot keys, since slotCount precedes them.
type preimageFile struct {
	path     string
	f        *os.File
	w        *bufio.Writer
	hasher   crypto.KeccakState
	addr     common.Address
	slots    []byte
	prevAddr common.Hash
	prevSlot common.Hash
	open     bool
	accounts uint64
}

// newPreimageFile creates the artifact file.
func newPreimageFile(path string) (*preimageFile, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	pf := &preimageFile{path: path, f: f, hasher: crypto.NewKeccakState()}
	pf.w = bufio.NewWriterSize(io.MultiWriter(f, pf.hasher), 1<<20)
	return pf, nil
}

// beginAccount seals the previous account's record and opens the next one.
// hash is the account's MPT path, keccak256(addr).
func (pf *preimageFile) beginAccount(addr common.Address, hash common.Hash) error {
	if err := pf.sealAccount(); err != nil {
		return err
	}
	if pf.accounts > 0 && bytes.Compare(pf.prevAddr[:], hash[:]) >= 0 {
		return fmt.Errorf("preimage records out of hashed-key order at account %x", addr)
	}
	pf.addr, pf.prevAddr, pf.open = addr, hash, true
	pf.accounts++
	return nil
}

// addSlot records one slot key of the open account; hash is its MPT path,
// keccak256(slotKey).
func (pf *preimageFile) addSlot(slotKey, hash common.Hash) error {
	if len(pf.slots) > 0 && bytes.Compare(pf.prevSlot[:], hash[:]) >= 0 {
		return fmt.Errorf("slot keys out of hashed-key order in account %x", pf.addr)
	}
	pf.slots = append(pf.slots, slotKey[:]...)
	pf.prevSlot = hash
	return nil
}

// sealAccount writes the open account's record.
func (pf *preimageFile) sealAccount() error {
	if !pf.open {
		return nil
	}
	var header [preimageRecordHeaderSize]byte
	copy(header[:], pf.addr[:])
	binary.BigEndian.PutUint32(header[common.AddressLength:], uint32(len(pf.slots)/common.HashLength))
	if _, err := pf.w.Write(header[:]); err != nil {
		return err
	}
	if _, err := pf.w.Write(pf.slots); err != nil {
		return err
	}
	pf.slots, pf.open = pf.slots[:0], false
	return nil
}

// finalize seals the last account and returns the file's digest, hashed in
// the same pass that wrote it.
func (pf *preimageFile) finalize() (common.Hash, error) {
	if err := pf.sealAccount(); err != nil {
		return common.Hash{}, err
	}
	if err := pf.w.Flush(); err != nil {
		return common.Hash{}, err
	}
	if err := pf.f.Sync(); err != nil {
		return common.Hash{}, err
	}
	if err := pf.f.Close(); err != nil {
		return common.Hash{}, err
	}
	var digest common.Hash
	pf.hasher.Read(digest[:])
	return digest, nil
}

// abort removes a partially written artifact after a failed conversion.
func (pf *preimageFile) abort() {
	pf.f.Close()
	os.Remove(pf.path)
}

// snapshotReader streams a PBT snapshot artifact: the claimed root and leaf
// count from the header, then the leaf records with every encoding rule
// enforced - zone-determined key lengths, strictly ascending keys, canonical
// non-zero integer values - hashing the file as it reads.
type snapshotReader struct {
	f       *os.File
	hasher  crypto.KeccakState
	stream  *rlp.Stream
	root    common.Hash
	count   uint64
	decoded uint64
	prevKey []byte
}

// openSnapshot opens the artifact and reads its header.
func openSnapshot(path string) (*snapshotReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	sr := &snapshotReader{f: f, hasher: crypto.NewKeccakState()}
	r := bufio.NewReaderSize(io.TeeReader(f, sr.hasher), 1<<20)

	var header [snapshotHeaderSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		f.Close()
		return nil, fmt.Errorf("truncated snapshot header: %w", err)
	}
	sr.root = common.BytesToHash(header[:32])
	sr.count = binary.BigEndian.Uint64(header[32:])
	// Bound the stream by the bytes that remain: a record's length prefix is
	// attacker-controlled, and rlp allocates against it unless it can check
	// the claim against the input it actually has.
	size, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	sr.stream = rlp.NewStream(r, uint64(max(size.Size()-snapshotHeaderSize, 0)))
	return sr, nil
}

// next returns the following leaf: the full tree key and the value re-padded
// to its 32 bytes. io.EOF ends the stream after exactly leafCount records.
func (sr *snapshotReader) next() ([]byte, [32]byte, error) {
	var (
		rec struct {
			Key   []byte
			Value []byte
		}
		value [32]byte
	)
	if err := sr.stream.Decode(&rec); err == io.EOF {
		if sr.decoded != sr.count {
			return nil, value, fmt.Errorf("snapshot holds %d records, its header claims %d", sr.decoded, sr.count)
		}
		return nil, value, io.EOF
	} else if err != nil {
		return nil, value, fmt.Errorf("snapshot record %d does not decode: %w", sr.decoded, err)
	}
	if sr.decoded == sr.count {
		return nil, value, fmt.Errorf("snapshot holds more records than the %d its header claims", sr.count)
	}
	// The zone byte fixes the key length; reserved zones are invalid.
	var wantLen int
	switch {
	case len(rec.Key) == 0:
		return nil, value, fmt.Errorf("snapshot record %d has an empty key", sr.decoded)
	case rec.Key[0] == bintrie.AccountZone || rec.Key[0] == bintrie.CodeZone:
		wantLen = bintrie.AccountKeyLength
	case rec.Key[0] == bintrie.StorageZone:
		wantLen = bintrie.StorageKeyLength
	default:
		return nil, value, fmt.Errorf("snapshot record %d sits in reserved zone %#x", sr.decoded, rec.Key[0])
	}
	if len(rec.Key) != wantLen {
		return nil, value, fmt.Errorf("snapshot record %d key is %d bytes, zone %#x demands %d", sr.decoded, len(rec.Key), rec.Key[0], wantLen)
	}
	if sr.prevKey != nil && bytes.Compare(sr.prevKey, rec.Key) >= 0 {
		return nil, value, fmt.Errorf("snapshot record %d out of order", sr.decoded)
	}
	sr.prevKey = rec.Key
	if len(rec.Value) == 0 || len(rec.Value) > 32 || rec.Value[0] == 0 {
		return nil, value, fmt.Errorf("snapshot record %d value is not a canonical non-zero integer", sr.decoded)
	}
	copy(value[32-len(rec.Value):], rec.Value)
	sr.decoded++
	return rec.Key, value, nil
}

// digest returns the keccak of everything read; the whole file once next
// returned io.EOF.
func (sr *snapshotReader) digest() common.Hash {
	return common.BytesToHash(sr.hasher.Sum(nil))
}

func (sr *snapshotReader) close() { sr.f.Close() }

// preimageReader streams the EIP-8347 preimage file, hashing it as it reads:
// records strictly ascending by keccak256(address), slot keys strictly
// ascending by keccak256(slotKey), nothing after the last record.
type preimageReader struct {
	f        *os.File
	hasher   crypto.KeccakState
	r        *bufio.Reader
	left     int64
	prevAddr common.Hash
	records  uint64
}

// openPreimages opens the preimage file.
func openPreimages(path string) (*preimageReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	size, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	pr := &preimageReader{f: f, hasher: crypto.NewKeccakState(), left: size.Size()}
	pr.r = bufio.NewReaderSize(io.TeeReader(f, pr.hasher), 1<<20)
	return pr, nil
}

// next returns the following account record: the address and its slot keys.
// io.EOF ends the stream only on a record boundary.
func (pr *preimageReader) next() (common.Address, []common.Hash, error) {
	var header [preimageRecordHeaderSize]byte
	if _, err := io.ReadFull(pr.r, header[:]); err == io.EOF {
		return common.Address{}, nil, io.EOF
	} else if err != nil {
		return common.Address{}, nil, fmt.Errorf("preimage record %d is truncated: %w", pr.records, err)
	}
	pr.left -= preimageRecordHeaderSize

	addr := common.BytesToAddress(header[:common.AddressLength])
	hash := crypto.Keccak256Hash(addr[:])
	if pr.records > 0 && bytes.Compare(pr.prevAddr[:], hash[:]) >= 0 {
		return common.Address{}, nil, fmt.Errorf("preimage record %d is out of hashed-key order", pr.records)
	}
	pr.prevAddr = hash

	// Bound the attacker-controlled count by the bytes left before allocating.
	count := int64(binary.BigEndian.Uint32(header[common.AddressLength:]))
	if count*common.HashLength > pr.left {
		return common.Address{}, nil, fmt.Errorf("preimage record %d claims %d slots past the end of the file", pr.records, count)
	}
	var (
		slots    = make([]common.Hash, count)
		prevSlot common.Hash
	)
	for i := range slots {
		if _, err := io.ReadFull(pr.r, slots[i][:]); err != nil {
			return common.Address{}, nil, fmt.Errorf("preimage record %d is truncated: %w", pr.records, err)
		}
		slotHash := crypto.Keccak256Hash(slots[i][:])
		if i > 0 && bytes.Compare(prevSlot[:], slotHash[:]) >= 0 {
			return common.Address{}, nil, fmt.Errorf("preimage record %d slot keys are out of hashed-key order", pr.records)
		}
		prevSlot = slotHash
	}
	pr.left -= count * common.HashLength
	pr.records++
	return addr, slots, nil
}

// digest returns the keccak of everything read; the whole file once next
// returned io.EOF.
func (pr *preimageReader) digest() common.Hash {
	return common.BytesToHash(pr.hasher.Sum(nil))
}

func (pr *preimageReader) close() { pr.f.Close() }

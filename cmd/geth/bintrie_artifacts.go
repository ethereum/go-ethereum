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
	"slices"

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

// preimageRecord is one preimage-file account: the 20-byte address and its
// slot keys as canonical integers, ascending.
type preimageRecord struct {
	Address  []byte
	SlotKeys [][]byte
}

// preimageFile accumulates the scan's preimage records and writes them out
// address-sorted through the external sorter (the scan walks in hashed-key
// order). By construction the emitted set is exactly the converted one.
type preimageFile struct {
	sorter   *bintrie.RecordSorter
	addr     common.Address
	slots    [][]byte
	accounts uint64
}

// newPreimageFile creates the collector, spilling through tmpDir past budget.
func newPreimageFile(tmpDir string, budget int) *preimageFile {
	return &preimageFile{
		sorter: bintrie.NewRecordSorter(tmpDir, budget, func(key, value []byte) error {
			if len(key) != common.AddressLength {
				return fmt.Errorf("preimage records are keyed by address, got %d bytes", len(key))
			}
			return nil
		}),
	}
}

// beginAccount seals the previous account's record and opens the next.
func (pf *preimageFile) beginAccount(addr common.Address) error {
	if err := pf.sealAccount(); err != nil {
		return err
	}
	pf.addr = addr
	pf.accounts++
	return nil
}

// addSlot records one slot key of the open account.
func (pf *preimageFile) addSlot(slotKey []byte) {
	pf.slots = append(pf.slots, common.CopyBytes(slotKey))
}

// sealAccount sorts the open account's slot keys and buffers its record; a
// record is one RLP value, so its slots are held in memory regardless.
func (pf *preimageFile) sealAccount() error {
	if pf.accounts == 0 {
		return nil
	}
	slices.SortFunc(pf.slots, bytes.Compare)
	rec := preimageRecord{Address: pf.addr[:], SlotKeys: make([][]byte, 0, len(pf.slots))}
	for _, slot := range pf.slots {
		rec.SlotKeys = append(rec.SlotKeys, common.TrimLeftZeroes(slot))
	}
	blob, err := rlp.EncodeToBytes(&rec)
	if err != nil {
		return err
	}
	pf.slots = pf.slots[:0]
	return pf.sorter.Add(pf.addr[:], blob)
}

// write seals the last account, writes the address-sorted records and
// returns the file's digest, hashed in the same pass.
func (pf *preimageFile) write(path string) (common.Hash, error) {
	if err := pf.sealAccount(); err != nil {
		return common.Hash{}, err
	}
	stream, err := pf.sorter.Sort()
	if err != nil {
		return common.Hash{}, err
	}
	f, err := os.Create(path)
	if err != nil {
		return common.Hash{}, err
	}
	defer f.Close()
	var (
		hasher = crypto.NewKeccakState()
		w      = bufio.NewWriterSize(io.MultiWriter(f, hasher), 1<<20)
	)
	for {
		_, blob, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return common.Hash{}, err
		}
		if _, err := w.Write(blob); err != nil {
			return common.Hash{}, err
		}
	}
	if err := w.Flush(); err != nil {
		return common.Hash{}, err
	}
	// The digest names the file; make its bytes durable first.
	if err := f.Sync(); err != nil {
		return common.Hash{}, err
	}
	var digest common.Hash
	hasher.Read(digest[:])
	return digest, nil
}

// close releases the sorter's temporary files.
func (pf *preimageFile) close() {
	pf.sorter.Close()
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
	sr.stream = rlp.NewStream(r, 0)
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
	var d common.Hash
	sr.hasher.Read(d[:])
	return d
}

func (sr *snapshotReader) close() { sr.f.Close() }

// preimageReader streams a preimage file: per-account records in strictly
// ascending address order, slot keys canonical and ascending, hashing the
// file as it reads.
type preimageReader struct {
	f        *os.File
	hasher   crypto.KeccakState
	stream   *rlp.Stream
	prevAddr []byte
	records  uint64
}

// openPreimages opens the preimage file.
func openPreimages(path string) (*preimageReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	pr := &preimageReader{f: f, hasher: crypto.NewKeccakState()}
	pr.stream = rlp.NewStream(bufio.NewReaderSize(io.TeeReader(f, pr.hasher), 1<<20), 0)
	return pr, nil
}

// next returns the following account record: the address and its slot keys
// re-padded to 32 bytes. io.EOF ends the stream.
func (pr *preimageReader) next() (common.Address, []common.Hash, error) {
	var rec struct {
		Address  []byte
		SlotKeys [][]byte
	}
	if err := pr.stream.Decode(&rec); err == io.EOF {
		return common.Address{}, nil, io.EOF
	} else if err != nil {
		return common.Address{}, nil, fmt.Errorf("preimage record %d does not decode: %w", pr.records, err)
	}
	if len(rec.Address) != common.AddressLength {
		return common.Address{}, nil, fmt.Errorf("preimage record %d address is %d bytes, want 20", pr.records, len(rec.Address))
	}
	if pr.prevAddr != nil && bytes.Compare(pr.prevAddr, rec.Address) >= 0 {
		return common.Address{}, nil, fmt.Errorf("preimage record %d out of address order", pr.records)
	}
	pr.prevAddr = rec.Address

	slots := make([]common.Hash, 0, len(rec.SlotKeys))
	for _, enc := range rec.SlotKeys {
		if len(enc) > 32 || (len(enc) > 0 && enc[0] == 0) {
			return common.Address{}, nil, fmt.Errorf("preimage record %d slot key is not a canonical integer", pr.records)
		}
		var slot common.Hash
		copy(slot[32-len(enc):], enc)
		if n := len(slots); n > 0 && bytes.Compare(slots[n-1][:], slot[:]) >= 0 {
			return common.Address{}, nil, fmt.Errorf("preimage record %d slot keys out of order", pr.records)
		}
		slots = append(slots, slot)
	}
	pr.records++
	return common.BytesToAddress(rec.Address), slots, nil
}

// digest returns the keccak of everything read; the whole file once next
// returned io.EOF.
func (pr *preimageReader) digest() common.Hash {
	var d common.Hash
	pr.hasher.Read(d[:])
	return d
}

func (pr *preimageReader) close() { pr.f.Close() }

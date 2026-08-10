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

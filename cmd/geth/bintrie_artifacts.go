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

// The EIP-8347 distribution artifacts. Conversion output is published as two
// byte-canonical files - the PBT snapshot and the preimage file - which every
// correct producer reproduces bit for bit for a given anchor state, so that a
// single keccak digest per file lets nodes compare sources before committing
// to a download. Byte-canonicality is why every encoding choice here is
// pinned: keys at full zone length, values and slot keys as canonical RLP
// integers, records in sorted order, nothing after the last record.

// snapshotHeaderSize is the fixed artifact header: pbtRoot[32] followed by
// leafCount[8, big-endian].
const snapshotHeaderSize = 40

// snapshotRecord is one leaf of the artifact: the full PBT tree key and the
// leaf value as a canonical integer, leading zeros dropped. A zero value
// cannot occur - EIP-8297 keeps zeros out of the tree - so the encoded value
// is never empty.
type snapshotRecord struct {
	Key   []byte
	Value []byte
}

// snapshotWriter streams the PBT snapshot artifact: a placeholder header,
// then one RLP record per leaf in the order the sorted stream yields them,
// then the header backpatched with the root and leaf count once the build
// fixes them. The digest is one sequential re-read at the end; the artifact
// is written once and hashed once, never buffered whole.
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

// add appends one leaf record. Keys arrive from the sorted stream, so the
// artifact inherits its order; values arrive as the tree's full 32 bytes and
// leave as canonical integers.
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

// preimageRecord is one account of the preimage file: the 20-byte address
// and the account's storage-slot keys as canonical integers, ascending. An
// account with no storage carries an empty list.
type preimageRecord struct {
	Address  []byte
	SlotKeys [][]byte
}

// preimageFile accumulates the preimage records of the conversion scan and
// writes them out sorted by address. The scan walks accounts in hashed-key
// order, which is no order at all in address terms, so the records pass
// through the external sorter; each is keyed by its address and carried as
// its final encoding.
//
// The emitted set matches the converted state exactly, by construction: a
// record is added for precisely the accounts and slots the conversion
// derived leaves from, and a preimage miss aborts the conversion before any
// artifact survives.
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

// beginAccount opens the record of one account. The scan visits each account
// exactly once, so the previous record is complete and can be sealed.
func (pf *preimageFile) beginAccount(addr common.Address) error {
	if err := pf.sealAccount(); err != nil {
		return err
	}
	pf.addr = addr
	pf.accounts++
	return nil
}

// addSlot records one storage-slot key of the open account. The slot arrives
// as the stored 32-byte preimage; ordering and trimming happen at sealing.
func (pf *preimageFile) addSlot(slotKey []byte) {
	pf.slots = append(pf.slots, common.CopyBytes(slotKey))
}

// sealAccount encodes and buffers the open account's record, if any. The
// slot keys of one account are sorted here, in memory: a record is a single
// RLP value, so it cannot be assembled without holding the account's slot
// keys anyway, and even the largest mainnet contracts stay within sense.
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

// write seals the last account, sorts the records by address and writes the
// file, hashing it in the same pass - the format has no header, so nothing
// needs backpatching. Returns the digest.
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
	var digest common.Hash
	hasher.Read(digest[:])
	return digest, nil
}

// close releases the sorter's temporary files.
func (pf *preimageFile) close() {
	pf.sorter.Close()
}

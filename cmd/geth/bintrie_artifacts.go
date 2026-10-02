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
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie/bintrie"
)

// The EIP-8347 distribution artifacts: two byte-canonical files every
// correct producer reproduces bit for bit, so one keccak digest per file
// lets nodes compare sources before downloading.

// snapshotOverhead is the snapshot's fixed size: the end tag and the
// pbtRoot[32] trailer. An empty state's snapshot is exactly this.
const snapshotOverhead = 1 + common.HashLength

// The snapshot's record tags. A header record's tag is its account's kind
// (0, 1 or 2); every other record takes one of these. Records run in zone
// order - header records, code groups, then storage records, each a storage
// account followed by its storage groups - and the end tag closes them, the
// claimed root after it.
const (
	tagCodeGroup      = 0x03
	tagStorageAccount = 0x04
	tagStorageSingle  = 0x05 // a storage group of one leaf, without the n byte
	tagStorageGroup   = 0x06 // a storage group of two or more leaves
	tagEnd            = 0x07
)

// The zones, in the order their records appear.
const (
	headerZone = iota
	codeZone
	storageZone
)

// tagZone returns the zone of a record tag below tagEnd.
func tagZone(tag byte) int {
	switch {
	case tag < tagCodeGroup:
		return headerZone
	case tag == tagCodeGroup:
		return codeZone
	default:
		return storageZone
	}
}

// snapshotLeaf is one leaf under a stem: its sub-index and value.
type snapshotLeaf struct {
	sub   byte
	value [32]byte
}

// byteReader is what the record decoders read: the artifact stream, or a
// record the writer checks.
type byteReader interface {
	io.Reader
	io.ByteReader
}

// appendVarint appends the x[≤w] encoding of the big-endian integer v: a
// length byte, then v without its leading zeros.
func appendVarint(dst, v []byte) []byte {
	v = common.TrimLeftZeroes(v)
	return append(append(dst, byte(len(v))), v...)
}

// readVarint reads an x[≤len(dst)] integer into dst, left-padded. A length
// over the width or a leading zero byte is not the canonical encoding.
func readVarint(r byteReader, dst []byte) error {
	n, err := r.ReadByte()
	if err != nil {
		return err
	}
	if int(n) > len(dst) {
		return fmt.Errorf("%d-byte integer in a %d-byte field", n, len(dst))
	}
	pad := len(dst) - int(n)
	clear(dst[:pad])
	if _, err := io.ReadFull(r, dst[pad:]); err != nil {
		return err
	}
	if n > 0 && dst[pad] == 0 {
		return errors.New("integer with a leading zero byte")
	}
	return nil
}

// readValue reads a leaf value, which is never zero: zero leaves are absent.
func readValue(r byteReader, v *[32]byte) error {
	if err := readVarint(r, v[:]); err != nil {
		return err
	}
	if *v == ([32]byte{}) {
		return errors.New("zero leaf value")
	}
	return nil
}

// appendGroup appends group = stemHash[32] | n[1] | (subIndex[1] |
// value[≤32]) * (n+1), or for a single group, which holds one leaf,
// stemHash[32] | subIndex[1] | value[≤32].
func appendGroup(dst []byte, single bool, stemHash []byte, leaves []snapshotLeaf) []byte {
	dst = append(dst, stemHash...)
	if !single {
		dst = append(dst, byte(len(leaves)-1))
	}
	for _, l := range leaves {
		dst = appendVarint(append(dst, l.sub), l.value[:])
	}
	return dst
}

// readGroup reads a group's stem hash into stemHash and appends its leaves
// to dst: n+1 of them, or one for a single group. Sub-index order is left to
// the caller's key-order check.
func readGroup(r byteReader, single bool, stemHash []byte, dst []snapshotLeaf) ([]snapshotLeaf, error) {
	if _, err := io.ReadFull(r, stemHash); err != nil {
		return dst, err
	}
	var n byte
	if !single {
		var err error
		if n, err = r.ReadByte(); err != nil {
			return dst, err
		}
	}
	for range int(n) + 1 {
		sub, err := r.ReadByte()
		if err != nil {
			return dst, err
		}
		dst = append(dst, snapshotLeaf{sub: sub})
		if err := readValue(r, &dst[len(dst)-1].value); err != nil {
			return dst, err
		}
	}
	return dst, nil
}

// headerRecord is an account's header stem in typed form. kind 0 holds no
// code, kind 1 a contract (codeHash, codeSize), kind 2 an EIP-7702
// delegation (target). slots are the header storage, sub the slot number.
type headerRecord struct {
	addrHash [32]byte
	nonce    [8]byte
	balance  [16]byte
	kind     byte
	codeHash [32]byte
	codeSize [4]byte
	target   [common.AddressLength]byte
	slots    []snapshotLeaf
}

// encode appends headerRecord = kind[1] | addressHash[32] | nonce[≤8] |
// balance[≤16] | codeRef | slotCount[1] | (slot[1] | value[≤32]) * slotCount,
// whose kind is its tag.
func (h *headerRecord) encode(dst []byte) []byte {
	dst = append(dst, h.kind)
	dst = append(dst, h.addrHash[:]...)
	dst = appendVarint(dst, h.nonce[:])
	dst = appendVarint(dst, h.balance[:])
	switch h.kind {
	case 1:
		dst = appendVarint(append(dst, h.codeHash[:]...), h.codeSize[:])
	case 2:
		dst = append(dst, h.target[:]...)
	}
	dst = append(dst, byte(len(h.slots)))
	for _, s := range h.slots {
		dst = appendVarint(append(dst, s.sub), s.value[:])
	}
	return dst
}

// decode reads the rest of a header record whose tag, the account's kind,
// has been read, enforcing every rule on its fields. Slot order is left to
// the caller's key-order check.
func (h *headerRecord) decode(r byteReader, kind byte) error {
	h.kind = kind
	if _, err := io.ReadFull(r, h.addrHash[:]); err != nil {
		return err
	}
	if err := readVarint(r, h.nonce[:]); err != nil {
		return err
	}
	if err := readVarint(r, h.balance[:]); err != nil {
		return err
	}
	switch kind {
	case 0:
		if h.nonce == [8]byte{} && h.balance == [16]byte{} {
			return fmt.Errorf("account %x is empty (EIP-7523) and cannot be in a snapshot", h.addrHash)
		}
	case 1:
		if _, err := io.ReadFull(r, h.codeHash[:]); err != nil {
			return err
		}
		if err := readVarint(r, h.codeSize[:]); err != nil {
			return err
		}
		if h.codeSize == [4]byte{} {
			return fmt.Errorf("contract account %x has code size zero", h.addrHash)
		}
	case 2:
		if _, err := io.ReadFull(r, h.target[:]); err != nil {
			return err
		}
	default:
		return fmt.Errorf("account %x has kind %d", h.addrHash, kind)
	}
	n, err := r.ReadByte()
	if err != nil {
		return err
	}
	if n > bintrie.HeaderStorageSlots {
		return fmt.Errorf("account %x has %d header slots", h.addrHash, n)
	}
	h.slots = h.slots[:0]
	for range n {
		slot, err := r.ReadByte()
		if err != nil {
			return err
		}
		if slot >= bintrie.HeaderStorageSlots {
			return fmt.Errorf("account %x has header slot %d", h.addrHash, slot)
		}
		h.slots = append(h.slots, snapshotLeaf{sub: slot})
		if err := readValue(r, &h.slots[len(h.slots)-1].value); err != nil {
			return err
		}
	}
	return nil
}

// derive appends the header stem's leaves, in sub-index order: basic data,
// then the code hash or the delegation, then the header storage.
func (h *headerRecord) derive(dst []snapshotLeaf) []snapshotLeaf {
	basic := snapshotLeaf{sub: bintrie.BasicDataLeafKey}
	switch h.kind {
	case 1:
		copy(basic.value[4:8], h.codeSize[:])
	case 2:
		basic.value[7] = 23
	}
	copy(basic.value[8:16], h.nonce[:])
	copy(basic.value[16:], h.balance[:])
	dst = append(dst, basic)

	switch h.kind {
	case 0:
		dst = append(dst, snapshotLeaf{sub: bintrie.CodeHashLeafKey, value: types.EmptyCodeHash})
	case 1:
		dst = append(dst, snapshotLeaf{sub: bintrie.CodeHashLeafKey, value: h.codeHash})
	case 2:
		leaf := snapshotLeaf{sub: bintrie.DelegationLeafKey}
		copy(leaf.value[copy(leaf.value[:], types.DelegationPrefix):], h.target[:])
		dst = append(dst, leaf)
	}
	for _, s := range h.slots {
		dst = append(dst, snapshotLeaf{sub: bintrie.HeaderStorageOffset + s.sub, value: s.value})
	}
	return dst
}

// snapshotWriter streams the PBT snapshot from the sorted leaves in one
// pass, grouping them by stem into tagged records, and hashes the bytes as
// it writes them.
type snapshotWriter struct {
	path   string
	f      *os.File
	w      *bufio.Writer // feeds the file and the hasher alike
	hasher crypto.KeccakState

	count uint64 // leaves added
	zone  int    // the open stem's zone, -1 before the first

	stem     []byte         // open stem: its leaves' key without the sub-index
	leaves   []snapshotLeaf // the open stem's leaves
	rec      []byte         // one encoded record
	acctOpen bool           // a storage account record has been written,
	acctHash [32]byte       // for this address hash

	hdr, check headerRecord
	derived    []snapshotLeaf
	rd         bytes.Reader
}

// newSnapshotWriter creates the artifact file.
func newSnapshotWriter(path string) (*snapshotWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	hasher := crypto.NewKeccakState()
	return &snapshotWriter{path: path, f: f, w: bufio.NewWriterSize(io.MultiWriter(f, hasher), 1<<20), hasher: hasher, zone: -1}, nil
}

func (sw *snapshotWriter) write(p []byte) error {
	_, err := sw.w.Write(p)
	return err
}

// add takes the next leaf in key order; value is its 32 bytes.
func (sw *snapshotWriter) add(key, value []byte) error {
	if len(key) == 0 || len(value) != 32 || [32]byte(value) == ([32]byte{}) {
		return fmt.Errorf("leaf %x has no non-zero 32-byte value", key)
	}
	sw.count++
	if stem := key[:len(key)-1]; !bytes.Equal(stem, sw.stem) {
		if err := sw.closeStem(); err != nil {
			return err
		}
		if err := sw.openStem(stem); err != nil {
			return err
		}
	}
	sw.leaves = append(sw.leaves, snapshotLeaf{sub: key[len(key)-1], value: [32]byte(value)})
	return nil
}

// openStem starts the next stem. An account's first storage stem writes its
// storage account record ahead of the groups.
func (sw *snapshotWriter) openStem(stem []byte) error {
	var zone int
	switch {
	case len(stem) == bintrie.AccountKeyLength-1 && stem[0] == bintrie.AccountZone:
		zone = headerZone
	case len(stem) == bintrie.CodeKeyLength-1 && stem[0] == bintrie.CodeZone:
		zone = codeZone
	case len(stem) == bintrie.StorageKeyLength-1 && stem[0] == bintrie.StorageZone:
		zone = storageZone
	default:
		return fmt.Errorf("no snapshot record holds stem %x", stem)
	}
	if zone < sw.zone {
		return errors.New("snapshot leaves out of zone order")
	}
	sw.zone = zone
	if zone == storageZone && (!sw.acctOpen || !bytes.Equal(stem[1:33], sw.acctHash[:])) {
		sw.acctOpen = true
		copy(sw.acctHash[:], stem[1:33])
		sw.rec = append(append(sw.rec[:0], tagStorageAccount), sw.acctHash[:]...)
		if err := sw.write(sw.rec); err != nil {
			return err
		}
	}
	sw.stem = append(sw.stem[:0], stem...)
	return nil
}

// closeStem writes the open stem as a record: a header record, a code
// group, or a storage group of the open storage account, tagged by its leaf
// count.
func (sw *snapshotWriter) closeStem() error {
	if len(sw.stem) == 0 {
		return nil
	}
	var err error
	switch sw.zone {
	case headerZone:
		err = sw.writeHeader()
	case codeZone:
		sw.rec = appendGroup(append(sw.rec[:0], tagCodeGroup), false, sw.stem[1:], sw.leaves)
		err = sw.write(sw.rec)
	case storageZone:
		single := len(sw.leaves) == 1
		tag := byte(tagStorageGroup)
		if single {
			tag = tagStorageSingle
		}
		sw.rec = appendGroup(append(sw.rec[:0], tag), single, sw.stem[33:], sw.leaves)
		err = sw.write(sw.rec)
	}
	sw.stem, sw.leaves = sw.stem[:0], sw.leaves[:0]
	return err
}

// writeHeader encodes the open header stem as a record, inferring the kind
// from the leaf present, and writes it only if the record re-derives
// exactly the leaves it was built from.
func (sw *snapshotWriter) writeHeader() error {
	h := &sw.hdr
	copy(h.addrHash[:], sw.stem[1:])
	h.nonce, h.balance, h.codeSize, h.slots = [8]byte{}, [16]byte{}, [4]byte{}, h.slots[:0]
	h.kind = 0xff // no code leaf: fails the decode below
	for _, l := range sw.leaves {
		switch {
		case l.sub == bintrie.BasicDataLeafKey:
			copy(h.codeSize[:], l.value[4:8])
			copy(h.nonce[:], l.value[8:16])
			copy(h.balance[:], l.value[16:])
		case l.sub == bintrie.CodeHashLeafKey:
			h.kind, h.codeHash = 1, l.value
			if h.codeHash == types.EmptyCodeHash {
				h.kind = 0
			}
		case l.sub == bintrie.DelegationLeafKey:
			h.kind = 2
			copy(h.target[:], l.value[len(types.DelegationPrefix):])
		case l.sub >= bintrie.HeaderStorageOffset && l.sub < bintrie.HeaderStorageOffset+bintrie.HeaderStorageSlots:
			h.slots = append(h.slots, snapshotLeaf{sub: l.sub - bintrie.HeaderStorageOffset, value: l.value})
		}
	}
	sw.rec = h.encode(sw.rec[:0])
	sw.rd.Reset(sw.rec[1:])
	if err := sw.check.decode(&sw.rd, sw.rec[0]); err != nil {
		return err
	}
	sw.derived = sw.check.derive(sw.derived[:0])
	if !slices.Equal(sw.derived, sw.leaves) {
		return fmt.Errorf("header leaves of account %x do not re-derive from a typed record", h.addrHash)
	}
	return sw.write(sw.rec)
}

// finalize writes the last stem, the end tag and the root, and returns the
// digest of every byte written, once they are durable.
func (sw *snapshotWriter) finalize(root common.Hash) (common.Hash, error) {
	defer sw.f.Close()
	if err := sw.closeStem(); err != nil {
		return common.Hash{}, err
	}
	if err := sw.write(append([]byte{tagEnd}, root[:]...)); err != nil {
		return common.Hash{}, err
	}
	if err := sw.w.Flush(); err != nil {
		return common.Hash{}, err
	}
	// The digest names the file; make its bytes durable first.
	if err := sw.f.Sync(); err != nil {
		return common.Hash{}, err
	}
	var digest common.Hash
	sw.hasher.Read(digest[:])
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

// snapshotReader streams a PBT snapshot artifact: its tagged records,
// decoded into their derived (key, value) leaves in place with no per-leaf
// allocation, then the end tag and the pbtRoot trailer. Every rule the spec
// places on the format is enforced: the tags, zone order, a storage
// account's groups, the one-leaf group's single encoding, strictly
// ascending storage accounts and group stems, and the six ordering MUSTs,
// which one strictly-ascending check over the derived keys covers.
type snapshotReader struct {
	f      *os.File
	hasher crypto.KeccakState
	br     *bufio.Reader
	root   common.Hash // the claimed pbtRoot, known once next has returned io.EOF
	zone   int         // the last record's zone, -1 before the first
	ended  bool        // the end tag and the root have been read

	pending    []snapshotLeaf
	pendIdx    int
	pendPrefix []byte

	storageOpen   bool     // a storage account has been read,
	storageAddr   [32]byte // this one,
	storageGroups int      // with this many groups so far
	stemHash      [32]byte
	prevStem      [32]byte // the previous group's, in the code zone or the storage account
	havePrevStem  bool

	hdr     headerRecord
	keyBuf  []byte
	prevKey []byte
}

// openSnapshot opens the artifact for next to read record by record.
func openSnapshot(path string) (*snapshotReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	sr := &snapshotReader{f: f, hasher: crypto.NewKeccakState(), zone: -1}
	sr.br = bufio.NewReaderSize(io.TeeReader(f, sr.hasher), 1<<20)
	return sr, nil
}

// fill reads the next record. A header record or a group decodes into
// sr.pending; a storage account only opens its record; the end tag reads the
// root and ends the stream.
func (sr *snapshotReader) fill() error {
	tag, err := sr.br.ReadByte()
	if err == io.EOF {
		return fmt.Errorf("snapshot ends before its end tag: %w", io.ErrUnexpectedEOF)
	} else if err != nil {
		return err
	}
	sr.pending, sr.pendIdx = sr.pending[:0], 0
	if tag > tagEnd {
		return fmt.Errorf("unknown record tag %#x", tag)
	}
	if (tag == tagStorageAccount || tag == tagEnd) && sr.storageOpen && sr.storageGroups == 0 {
		return fmt.Errorf("storage account %x has no storage group", sr.storageAddr)
	}
	if tag == tagEnd {
		if _, err := io.ReadFull(sr.br, sr.root[:]); err != nil {
			return fmt.Errorf("pbtRoot trailer: %w", err)
		}
		sr.ended = true
		return nil
	}
	if zone := tagZone(tag); zone != sr.zone {
		if zone < sr.zone {
			return fmt.Errorf("record tag %#x out of zone order", tag)
		}
		sr.zone, sr.havePrevStem = zone, false
	}
	switch tag {
	case tagCodeGroup:
		if sr.pending, err = readGroup(sr.br, false, sr.stemHash[:], sr.pending); err != nil {
			return fmt.Errorf("code group: %w", err)
		}
		sr.pendPrefix = append(append(sr.pendPrefix[:0], bintrie.CodeZone), sr.stemHash[:]...)
		return sr.stemAscends()
	case tagStorageAccount:
		var addr [32]byte
		if _, err := io.ReadFull(sr.br, addr[:]); err != nil {
			return fmt.Errorf("storage account: %w", err)
		}
		if sr.storageOpen && bytes.Compare(addr[:], sr.storageAddr[:]) <= 0 {
			return fmt.Errorf("storage account %x out of order", addr)
		}
		sr.storageOpen, sr.storageAddr, sr.storageGroups, sr.havePrevStem = true, addr, 0, false
		return nil
	case tagStorageSingle, tagStorageGroup:
		if !sr.storageOpen {
			return errors.New("storage group with no storage account before it")
		}
		single := tag == tagStorageSingle
		if sr.pending, err = readGroup(sr.br, single, sr.stemHash[:], sr.pending); err != nil {
			return fmt.Errorf("storage group of %x: %w", sr.storageAddr, err)
		}
		if !single && len(sr.pending) == 1 {
			return fmt.Errorf("storage group %x holds one leaf but is not tagged %#x", sr.stemHash, tagStorageSingle)
		}
		sr.storageGroups++
		sr.pendPrefix = append(append(append(sr.pendPrefix[:0], bintrie.StorageZone), sr.storageAddr[:]...), sr.stemHash[:]...)
		return sr.stemAscends()
	default: // a header record, tagged by its account's kind
		if err := sr.hdr.decode(sr.br, tag); err != nil {
			return fmt.Errorf("header record: %w", err)
		}
		sr.pendPrefix = append(append(sr.pendPrefix[:0], bintrie.AccountZone), sr.hdr.addrHash[:]...)
		sr.pending = sr.hdr.derive(sr.pending)
		return nil
	}
}

// stemAscends checks the group just read comes strictly after the previous
// one in the code zone or its storage account, so no stem is split over two
// records.
func (sr *snapshotReader) stemAscends() error {
	if sr.havePrevStem && bytes.Compare(sr.stemHash[:], sr.prevStem[:]) <= 0 {
		return fmt.Errorf("group %x out of order", sr.stemHash)
	}
	sr.prevStem, sr.havePrevStem = sr.stemHash, true
	return nil
}

// next returns the following derived leaf: the full tree key and its value,
// decoded in place into reused buffers - a caller that keeps either must
// copy it before calling next again. io.EOF ends the stream once the end tag
// and the root are read and no byte follows them.
func (sr *snapshotReader) next() ([]byte, [32]byte, error) {
	for sr.pendIdx >= len(sr.pending) {
		if sr.ended {
			if b, err := sr.br.ReadByte(); err == nil {
				return nil, [32]byte{}, fmt.Errorf("trailing byte %#x after the pbtRoot trailer", b)
			} else if err != io.EOF {
				return nil, [32]byte{}, err
			}
			return nil, [32]byte{}, io.EOF
		}
		if err := sr.fill(); err != nil {
			return nil, [32]byte{}, err
		}
	}
	leaf := sr.pending[sr.pendIdx]
	sr.pendIdx++
	sr.keyBuf = append(append(sr.keyBuf[:0], sr.pendPrefix...), leaf.sub)
	if sr.prevKey != nil && bytes.Compare(sr.prevKey, sr.keyBuf) >= 0 {
		return nil, [32]byte{}, fmt.Errorf("snapshot leaf %x out of order", sr.keyBuf)
	}
	sr.prevKey = append(sr.prevKey[:0], sr.keyBuf...)
	return sr.keyBuf, leaf.value, nil
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

// next returns the following account record: the address, its keccak digest,
// which the ordering check computes anyway, and its slot keys. io.EOF ends
// the stream only on a record boundary.
func (pr *preimageReader) next() (common.Address, common.Hash, []common.Hash, error) {
	var header [preimageRecordHeaderSize]byte
	if _, err := io.ReadFull(pr.r, header[:]); err == io.EOF {
		return common.Address{}, common.Hash{}, nil, io.EOF
	} else if err != nil {
		return common.Address{}, common.Hash{}, nil, fmt.Errorf("preimage record %d is truncated: %w", pr.records, err)
	}
	pr.left -= preimageRecordHeaderSize

	addr := common.BytesToAddress(header[:common.AddressLength])
	addrHash := crypto.Keccak256Hash(addr[:])
	if pr.records > 0 && bytes.Compare(pr.prevAddr[:], addrHash[:]) >= 0 {
		return common.Address{}, common.Hash{}, nil, fmt.Errorf("preimage record %d is out of hashed-key order", pr.records)
	}
	pr.prevAddr = addrHash

	// Bound the attacker-controlled count by the bytes left before allocating.
	count := int64(binary.BigEndian.Uint32(header[common.AddressLength:]))
	if count*common.HashLength > pr.left {
		return common.Address{}, common.Hash{}, nil, fmt.Errorf("preimage record %d claims %d slots past the end of the file", pr.records, count)
	}
	var (
		slots    = make([]common.Hash, count)
		prevSlot common.Hash
	)
	for i := range slots {
		if _, err := io.ReadFull(pr.r, slots[i][:]); err != nil {
			return common.Address{}, common.Hash{}, nil, fmt.Errorf("preimage record %d is truncated: %w", pr.records, err)
		}
		slotHash := crypto.Keccak256Hash(slots[i][:])
		if i > 0 && bytes.Compare(prevSlot[:], slotHash[:]) >= 0 {
			return common.Address{}, common.Hash{}, nil, fmt.Errorf("preimage record %d slot keys are out of hashed-key order", pr.records)
		}
		prevSlot = slotHash
	}
	pr.left -= count * common.HashLength
	pr.records++
	return addr, addrHash, slots, nil
}

// digest returns the keccak of everything read; the whole file once next
// returned io.EOF.
func (pr *preimageReader) digest() common.Hash {
	return common.BytesToHash(pr.hasher.Sum(nil))
}

func (pr *preimageReader) close() { pr.f.Close() }

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
	"bytes"
	"encoding/binary"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

// The artifacts are checked against parsers and layouts written from
// EIP-8347's rules, not the writers, and must be byte-canonical across runs.

// artifactAlloc covers the artifact encoding decisions: shared code, a
// delegation, zero-tailed code, boundary storage, slot zero, and slots 64 and
// 65 sharing one storage group.
func artifactAlloc() types.GenesisAlloc {
	var (
		shared   = bytes.Repeat([]byte{0x5b}, 40)
		zeroTail = append([]byte{0x60, 0x01, 0x00}, make([]byte, 80)...)
		slot     = func(n int64) common.Hash { return common.BigToHash(big.NewInt(n)) }
	)
	return types.GenesisAlloc{
		common.HexToAddress("0x1000000000000000000000000000000000000001"): {
			Balance: big.NewInt(1), Nonce: 7,
		},
		common.HexToAddress("0x2000000000000000000000000000000000000002"): {
			Balance: big.NewInt(1e15), Nonce: 1, Code: shared,
			Storage: map[common.Hash]common.Hash{
				slot(0):    slot(0x11),
				slot(1):    slot(0x22),
				slot(63):   slot(0x33),
				slot(64):   slot(0x44),
				slot(65):   slot(0x77),
				slot(4096): slot(0x55),
			},
		},
		common.HexToAddress("0x3000000000000000000000000000000000000003"): {
			Balance: big.NewInt(2), Code: shared,
		},
		common.HexToAddress("0x4000000000000000000000000000000000000004"): {
			Balance: big.NewInt(3), Nonce: 2,
			Code: types.AddressToDelegation(common.HexToAddress("0xde1e000000000000000000000000000000000001")),
		},
		common.HexToAddress("0x5000000000000000000000000000000000000005"): {
			Balance: big.NewInt(4), Code: zeroTail,
			Storage: map[common.Hash]common.Hash{slot(70): slot(0x66)},
		},
	}
}

// convertWithArtifacts converts alloc and returns the binary root and the
// two artifact paths.
func convertWithArtifacts(t *testing.T, alloc types.GenesisAlloc, budget int) (common.Hash, string, string) {
	t.Helper()
	dir := t.TempDir()
	var (
		snapPath = filepath.Join(dir, "snapshot.bin")
		prePath  = filepath.Join(dir, "preimages.bin")
	)
	chaindb := rawdb.NewMemoryDatabase()
	srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.Defaults,
	})
	gspec := &core.Genesis{
		Config:  params.TestChainConfig,
		BaseFee: big.NewInt(params.InitialBaseFee),
		Alloc:   alloc,
	}
	root := gspec.MustCommit(chaindb, srcTriedb).Root()
	srcTriedb.Close()

	src := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.ReadOnly,
	})
	defer src.Close()

	binRoot, err := convertState(chaindb, src, root, conversionOptions{
		sortBudget:   budget,
		tmpDir:       dir,
		snapshotPath: snapPath,
		preimagePath: prePath,
	})
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	return binRoot, snapPath, prePath
}

// TestSnapshotArtifactRoundTrip: the snapshot's typed records, decoded here
// by an independent implementation written from EIP-8347's grammar (not
// from the production reader), must derive leaves that rebuild to the root
// the trailer claims, which must be the committed root.
func TestSnapshotArtifactRoundTrip(t *testing.T) {
	// Spill-heavy, so the merged sort path writes the artifact.
	binRoot, snapPath, _ := convertWithArtifacts(t, artifactAlloc(), 512)

	blob, err := os.ReadFile(snapPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(blob) < snapshotOverhead {
		t.Fatalf("snapshot is %d bytes, shorter than its %d-byte overhead", len(blob), snapshotOverhead)
	}
	body, trailer := blob[:len(blob)-common.HashLength], blob[len(blob)-common.HashLength:]
	claimedRoot := common.BytesToHash(trailer)
	if claimedRoot != binRoot {
		t.Fatalf("trailer claims root %x, conversion committed %x", claimedRoot, binRoot)
	}

	r := bytes.NewReader(body)
	readByte := func() byte {
		t.Helper()
		b, err := r.ReadByte()
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	readHash := func() []byte {
		t.Helper()
		h := make([]byte, common.HashLength)
		if _, err := io.ReadFull(r, h); err != nil {
			t.Fatal(err)
		}
		return h
	}
	readVar := func(w int) []byte {
		t.Helper()
		n := readByte()
		if int(n) > w {
			t.Fatalf("%d-byte integer over the %d-byte field", n, w)
		}
		v := make([]byte, n)
		if _, err := io.ReadFull(r, v); err != nil {
			t.Fatal(err)
		}
		if n > 0 && v[0] == 0 {
			t.Fatal("integer carries a leading zero byte")
		}
		padded := make([]byte, w)
		copy(padded[w-len(v):], v)
		return padded
	}
	readValue := func() [32]byte {
		var out [32]byte
		copy(out[:], readVar(32))
		if out == ([32]byte{}) {
			t.Fatal("zero leaf value")
		}
		return out
	}

	var (
		rebuild = bintrie.NewStackBuilder(nil)
		prevKey []byte
		leaves  uint64
		add     = func(key []byte, sub byte, value [32]byte) {
			key = append(append([]byte{}, key...), sub)
			if prevKey != nil && bytes.Compare(prevKey, key) >= 0 {
				t.Fatalf("leaf %x out of order after %x", key, prevKey)
			}
			prevKey = key
			if err := rebuild.Add(key, value[:]); err != nil {
				t.Fatalf("leaf %x does not fold: %v", key, err)
			}
			leaves++
		}
		readEntries = func(key []byte, n int) {
			for range n {
				sub := readByte()
				add(key, sub, readValue())
			}
		}
		zone    = -1   // the last record's zone: 0 header, 1 code, 2 storage
		account []byte // the open storage account
		groups  int    // its storage groups so far
	)
	for tag := readByte(); tag != 0x07; tag = readByte() {
		recZone := 2
		switch {
		case tag <= 0x02:
			recZone = 0
		case tag == 0x03:
			recZone = 1
		case tag > 0x06:
			t.Fatalf("unknown tag %#x", tag)
		}
		if recZone < zone {
			t.Fatalf("tag %#x after a zone-%d record", tag, zone)
		}
		zone = recZone

		switch tag {
		case 0x00, 0x01, 0x02: // a header record, the tag its kind
			key := append([]byte{bintrie.AccountZone}, readHash()...)
			nonce, balance := readVar(8), readVar(16)
			var basic [32]byte
			copy(basic[8:16], nonce)
			copy(basic[16:], balance)
			switch tag {
			case 0x00:
				add(key, bintrie.BasicDataLeafKey, basic)
				add(key, bintrie.CodeHashLeafKey, types.EmptyCodeHash)
			case 0x01:
				codeHash := [32]byte(readHash())
				copy(basic[4:8], readVar(4))
				add(key, bintrie.BasicDataLeafKey, basic)
				add(key, bintrie.CodeHashLeafKey, codeHash)
			case 0x02:
				var target [common.AddressLength]byte
				if _, err := io.ReadFull(r, target[:]); err != nil {
					t.Fatal(err)
				}
				basic[7] = 23
				var deleg [32]byte
				copy(deleg[:], types.DelegationPrefix)
				copy(deleg[len(types.DelegationPrefix):], target[:])
				add(key, bintrie.BasicDataLeafKey, basic)
				add(key, bintrie.DelegationLeafKey, deleg)
			}
			for range readByte() {
				sub := readByte()
				add(key, bintrie.HeaderStorageOffset+sub, readValue())
			}
		case 0x03: // a code group
			key := append([]byte{bintrie.CodeZone}, readHash()...)
			readEntries(key, int(readByte())+1)
		case 0x04: // a storage account, which its groups follow
			if account != nil && groups == 0 {
				t.Fatalf("storage account %x has no storage group", account)
			}
			account, groups = readHash(), 0
		case 0x05, 0x06: // a storage group of one leaf, or of more
			if account == nil {
				t.Fatal("storage group before any storage account")
			}
			key := append(append([]byte{bintrie.StorageZone}, account...), readHash()...)
			n := 1
			if tag == 0x06 {
				if n = int(readByte()) + 1; n == 1 {
					t.Fatal("a one-leaf storage group tagged 0x06")
				}
			}
			readEntries(key, n)
			groups++
		}
	}
	if account != nil && groups == 0 {
		t.Fatalf("storage account %x has no storage group", account)
	}
	if r.Len() != 0 {
		t.Fatalf("%d bytes between the end tag and the pbtRoot trailer", r.Len())
	}
	if leaves == 0 {
		t.Fatal("decoded no leaves")
	}
	if got := rebuild.Finish(); got != claimedRoot {
		t.Fatalf("leaves rebuild to %x, trailer claims %x", got, claimedRoot)
	}
}

// TestPreimageFileLayout: the converter's preimage file is exactly the EIP-8347
// layout of the converted state.
func TestPreimageFileLayout(t *testing.T) {
	alloc := artifactAlloc()
	_, _, prePath := convertWithArtifacts(t, alloc, 512)
	got, err := os.ReadFile(prePath)
	if err != nil {
		t.Fatal(err)
	}
	if want := specPreimageFile(alloc); !bytes.Equal(got, want) {
		t.Fatalf("preimage file is\n%x\nwant\n%x", got, want)
	}
}

// TestArtifactsAreByteCanonical: two conversions of the same state must
// reproduce both files bit for bit - the property the digests stand on.
func TestArtifactsAreByteCanonical(t *testing.T) {
	// Different budgets: the bytes may depend only on the state.
	alloc := artifactAlloc()
	_, snapA, preA := convertWithArtifacts(t, alloc, 0)
	_, snapB, preB := convertWithArtifacts(t, alloc, 512)

	for _, pair := range []struct{ name, a, b string }{
		{"snapshot", snapA, snapB},
		{"preimages", preA, preB},
	} {
		a, err := os.ReadFile(pair.a)
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(pair.b)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a, b) {
			t.Fatalf("two conversions of the same state produced different %s files (%d vs %d bytes)", pair.name, len(a), len(b))
		}
		if len(a) == 0 {
			t.Fatalf("the %s file is empty", pair.name)
		}
	}
}

// specPreimageFile derives the EIP-8347 preimage file from the allocation
// alone, independently of the writer.
func specPreimageFile(alloc types.GenesisAlloc) []byte {
	recs := make([]preRecord, 0, len(alloc))
	for addr, acct := range alloc {
		rec := preRecord{addr: addr}
		for slot := range acct.Storage {
			rec.slots = append(rec.slots, slot)
		}
		recs = append(recs, rec)
	}
	return encodePreimageRecords(recs)
}

// TestArtifactGoldenDigests freezes the artifact byte format: any encoding
// change moves these reference-pinned digests, and moving them breaks every
// existing producer. The contract vector's preimage file must also match the
// layout the state implies, so the golden cannot pin a writer bug, and
// artifactAlloc's snapshot carries every record tag.
func TestArtifactGoldenDigests(t *testing.T) {
	const (
		wantSnapshot  = "0x55243db729ce66fbf98b46e1b7fb86d97c2e6bd8021176051fd2b557fc25799b"
		wantPreimages = "0xf7031eb1bf7682466c33c4a7e0e0616139990ebd4a4cff82f59a1d68b045fdc8"
		wantAllTags   = "0xd2b508c308510ec213f562f60cc417659665cdf8203f577d92203d3153bd25aa"
	)
	found := false
	for _, sv := range loadStateVectors(t) {
		if sv.Name != "contract" {
			continue
		}
		found = true
		alloc := allocOf(t, sv)
		_, snapPath, prePath := convertWithArtifacts(t, alloc, 0)
		snap, err := os.ReadFile(snapPath)
		if err != nil {
			t.Fatal(err)
		}
		pre, err := os.ReadFile(prePath)
		if err != nil {
			t.Fatal(err)
		}
		if want := specPreimageFile(alloc); !bytes.Equal(pre, want) {
			t.Fatalf("preimage file is\n%x\nthe layout the state implies is\n%x", pre, want)
		}
		if got := crypto.Keccak256Hash(snap); got != common.HexToHash(wantSnapshot) {
			t.Fatalf("snapshot digest %x, the recorded golden is %s", got, wantSnapshot)
		}
		if got := crypto.Keccak256Hash(pre); got != common.HexToHash(wantPreimages) {
			t.Fatalf("preimage digest %x, the recorded golden is %s", got, wantPreimages)
		}
	}
	if !found {
		t.Fatal("the contract state vector is gone from the testdata")
	}
	_, snapPath, _ := convertWithArtifacts(t, artifactAlloc(), 0)
	snap, err := os.ReadFile(snapPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := crypto.Keccak256Hash(snap); got != common.HexToHash(wantAllTags) {
		t.Fatalf("artifactAlloc snapshot digest %x, the recorded golden is %s", got, wantAllTags)
	}
}

// TestArtifactReaders pins the production readers against the writers: every
// record accepted, digests matching an independent hash of the files,
// truncation and trailing bytes rejected.
func TestArtifactReaders(t *testing.T) {
	_, snapPath, prePath := convertWithArtifacts(t, artifactAlloc(), 0)

	sr, err := openSnapshot(snapPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sr.close()
	var leaves uint64
	for {
		if _, _, err := sr.next(); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("snapshot reader rejected the writer's output: %v", err)
		}
		leaves++
	}
	if leaves == 0 {
		t.Fatal("read no leaves")
	}
	blob, err := os.ReadFile(snapPath)
	if err != nil {
		t.Fatal(err)
	}
	// Read twice: the importer logs each digest more than once.
	for range 2 {
		if got, want := sr.digest(), crypto.Keccak256Hash(blob); got != want {
			t.Fatalf("snapshot digest %x, file hashes to %x", got, want)
		}
	}

	pr, err := openPreimages(prePath)
	if err != nil {
		t.Fatal(err)
	}
	defer pr.close()
	accounts := 0
	for {
		if _, _, err := pr.next(); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("preimage reader rejected the writer's output: %v", err)
		}
		accounts++
	}
	if accounts != len(artifactAlloc()) {
		t.Fatalf("read %d preimage records, want %d", accounts, len(artifactAlloc()))
	}
	if blob, err = os.ReadFile(prePath); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if got, want := pr.digest(), crypto.Keccak256Hash(blob); got != want {
			t.Fatalf("preimage digest %x, file hashes to %x", got, want)
		}
	}

	// A trailing byte and a truncation must both reject.
	for _, tamper := range []struct {
		name string
		mod  func([]byte) []byte
	}{
		{"trailing byte", func(b []byte) []byte { return append(b, 0x00) }},
		{"truncated", func(b []byte) []byte { return b[:len(b)-1] }},
	} {
		bad := filepath.Join(t.TempDir(), "bad.bin")
		if err := os.WriteFile(bad, tamper.mod(append([]byte{}, blob...)), 0600); err != nil {
			t.Fatal(err)
		}
		pr2, err := openPreimages(bad)
		if err != nil {
			t.Fatal(err)
		}
		for {
			if _, _, err := pr2.next(); err == io.EOF {
				t.Fatalf("a %s preimage file was accepted", tamper.name)
			} else if err != nil {
				break
			}
		}
		pr2.close()
	}
}

// Byte-level builders for the typed snapshot, verbatim and unchecked, so a
// case can inject a fault at exactly one field while the rest stays
// well-formed - the only way to reach the encodings the writer never emits.

// rawVarint returns the raw x[≤w] encoding of v: a length byte, then v
// exactly as given, leading zero and all.
func rawVarint(v ...byte) []byte {
	return append([]byte{byte(len(v))}, v...)
}

// rawEntry is one (subIndex, value) pair inside a group or a header's slot
// list.
type rawEntry struct {
	sub   byte
	value []byte
}

func appendEntries(dst []byte, entries []rawEntry) []byte {
	for _, e := range entries {
		dst = append(dst, e.sub)
		dst = append(dst, rawVarint(e.value...)...)
	}
	return dst
}

// rawHeader builds one header record verbatim, its kind as its tag. codeRef
// is codeHash[32] ‖ rawVarint(codeSize) for kind 1, target[20] for kind 2,
// nil for kind 0.
type rawHeader struct {
	addrHash [32]byte
	nonce    []byte
	balance  []byte
	kind     byte
	codeRef  []byte
	slots    []rawEntry
}

func (h rawHeader) bytes() []byte {
	buf := append([]byte{h.kind}, h.addrHash[:]...)
	buf = append(buf, rawVarint(h.nonce...)...)
	buf = append(buf, rawVarint(h.balance...)...)
	buf = append(buf, h.codeRef...)
	buf = append(buf, byte(len(h.slots)))
	return appendEntries(buf, h.slots)
}

// rawGroup builds one group verbatim: its stem hash, n, then the entries.
type rawGroup struct {
	stemHash [32]byte
	entries  []rawEntry
}

func (g rawGroup) bytes() []byte {
	buf := append([]byte{}, g.stemHash[:]...)
	buf = append(buf, byte(len(g.entries)-1))
	return appendEntries(buf, g.entries)
}

// rawCodeGroup, rawStorageAccount and rawStorageGroup build the other
// records; a storage group takes the tag its leaf count calls for.
func rawCodeGroup(g rawGroup) []byte {
	return append([]byte{tagCodeGroup}, g.bytes()...)
}

func rawStorageAccount(addrHash [32]byte) []byte {
	return append([]byte{tagStorageAccount}, addrHash[:]...)
}

func rawStorageGroup(g rawGroup) []byte {
	if len(g.entries) == 1 {
		return appendEntries(append([]byte{tagStorageSingle}, g.stemHash[:]...), g.entries)
	}
	return append([]byte{tagStorageGroup}, g.bytes()...)
}

// rawSnapshot assembles a whole snapshot file from verbatim records: the
// records, the end tag unless noEnd, then the root, or trailer in its place
// when set, to cut or pad it.
type rawSnapshot struct {
	records []byte
	noEnd   bool
	root    common.Hash
	trailer []byte
}

func (s rawSnapshot) encode() []byte {
	buf := append([]byte{}, s.records...)
	if !s.noEnd {
		buf = append(buf, tagEnd)
	}
	if s.trailer != nil {
		return append(buf, s.trailer...)
	}
	return append(buf, s.root[:]...)
}

func (s rawSnapshot) write(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "snapshot.bin")
	if err := os.WriteFile(path, s.encode(), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// rawPreRecord is one verbatim preimage record; a negative count means
// len(slots).
type rawPreRecord struct {
	addr  []byte
	count int
	slots [][]byte
}

// rawPreimages writes a preimage file from verbatim records, so a case can
// inject bytes the writer can never emit.
func rawPreimages(t *testing.T, recs []rawPreRecord) string {
	t.Helper()
	var buf bytes.Buffer
	for _, rec := range recs {
		count := rec.count
		if count < 0 {
			count = len(rec.slots)
		}
		buf.Write(rec.addr)
		buf.Write(binary.BigEndian.AppendUint32(nil, uint32(count)))
		for _, slot := range rec.slots {
			buf.Write(slot)
		}
	}
	path := filepath.Join(t.TempDir(), "preimages.bin")
	if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestArtifactReadersReject pins the encodings the readers must refuse. They
// are the only way either artifact enters the importer, so a rule enforced
// here is enforced everywhere.
func TestArtifactReadersReject(t *testing.T) {
	addr := common.Address{1}.Bytes()

	t.Run("snapshot", func(t *testing.T) {
		var (
			addrHash1, addrHash2 = [32]byte{1}, [32]byte{2}
			stemHash1, stemHash2 = [32]byte{1}, [32]byte{2}
			codeHash             = bytes.Repeat([]byte{0xaa}, 32)
			one                  = []byte{1} // a canonical non-zero one-byte integer
		)
		basic := func(addrHash [32]byte) rawHeader {
			return rawHeader{addrHash: addrHash, nonce: one, kind: 0}
		}
		header := func(h rawHeader) rawSnapshot {
			return rawSnapshot{records: h.bytes()}
		}
		records := func(parts ...[]byte) rawSnapshot {
			return rawSnapshot{records: bytes.Join(parts, nil)}
		}
		leaf := func(sub byte) rawEntry {
			return rawEntry{sub: sub, value: one}
		}
		group := func(stemHash [32]byte, subs ...byte) rawGroup {
			g := rawGroup{stemHash: stemHash}
			for _, sub := range subs {
				g.entries = append(g.entries, leaf(sub))
			}
			return g
		}
		storage := func(addrHash [32]byte, groups ...rawGroup) []byte {
			buf := rawStorageAccount(addrHash)
			for _, g := range groups {
				buf = append(buf, rawStorageGroup(g)...)
			}
			return buf
		}

		for _, tc := range []struct {
			name    string
			snap    rawSnapshot
			wantErr string
		}{
			// x[≤w] canonicality: leading zero and length above the field.
			{"nonce leading zero", header(rawHeader{addrHash: addrHash1, nonce: []byte{0, 1}, kind: 0}), "leading zero"},
			{"nonce over-long", header(rawHeader{addrHash: addrHash1, nonce: bytes.Repeat([]byte{1}, 9), kind: 0}), "byte field"},
			{"balance leading zero", header(rawHeader{addrHash: addrHash1, nonce: one, balance: []byte{0, 1}, kind: 0}), "leading zero"},
			{"balance over-long", header(rawHeader{addrHash: addrHash1, nonce: one, balance: bytes.Repeat([]byte{1}, 17), kind: 0}), "byte field"},
			{"codeSize leading zero", header(rawHeader{addrHash: addrHash1, nonce: one, kind: 1, codeRef: append(append([]byte{}, codeHash...), rawVarint(0, 1)...)}), "leading zero"},
			{"codeSize over-long", header(rawHeader{addrHash: addrHash1, nonce: one, kind: 1, codeRef: append(append([]byte{}, codeHash...), rawVarint(1, 2, 3, 4, 5)...)}), "byte field"},
			{"value leading zero", header(rawHeader{addrHash: addrHash1, nonce: one, kind: 0, slots: []rawEntry{{sub: 0, value: []byte{0, 1}}}}), "leading zero"},
			{"value over-long", header(rawHeader{addrHash: addrHash1, nonce: one, kind: 0, slots: []rawEntry{{sub: 0, value: bytes.Repeat([]byte{1}, 33)}}}), "byte field"},

			// Tags, empty account, code size, slot range, zero value.
			{"unknown tag", records([]byte{tagEnd + 1}), "unknown record tag"},
			{"kind-0 empty account (EIP-7523)", header(rawHeader{addrHash: addrHash1, kind: 0}), "EIP-7523"},
			{"kind-1 codeSize zero", header(rawHeader{addrHash: addrHash1, nonce: one, kind: 1, codeRef: append(append([]byte{}, codeHash...), rawVarint()...)}), "code size zero"},
			{"header slot at 64", header(rawHeader{addrHash: addrHash1, nonce: one, kind: 0, slots: []rawEntry{{sub: 64, value: one}}}), "header slot"},
			{"zero leaf value", header(rawHeader{addrHash: addrHash1, nonce: one, kind: 0, slots: []rawEntry{{sub: 0}}}), "zero leaf value"},

			// Zone order and the storage record's structure.
			{"code group before a header record", records(rawCodeGroup(group(stemHash1, 0)), basic(addrHash1).bytes()), "zone order"},
			{"storage record before a code group", records(storage(addrHash1, group(stemHash1, 0)), rawCodeGroup(group(stemHash1, 0))), "zone order"},
			{"storage account without a group before the end", records(rawStorageAccount(addrHash1)), "no storage group"},
			{"storage account without a group before the next", records(rawStorageAccount(addrHash1), storage(addrHash2, group(stemHash1, 0))), "no storage group"},
			{"storage group without a storage account", records(rawStorageGroup(group(stemHash1, 0))), "no storage account"},
			{"one-leaf storage group tagged as a larger one", records(rawStorageAccount(addrHash1), []byte{tagStorageGroup}, group(stemHash1, 0).bytes()), "holds one leaf"},

			// Ordering, at each of the six levels the spec names.
			{"header records unsorted", records(basic(addrHash2).bytes(), basic(addrHash1).bytes()), "out of order"},
			{"header records duplicate", records(basic(addrHash1).bytes(), basic(addrHash1).bytes()), "out of order"},
			{"header slots unsorted", header(rawHeader{addrHash: addrHash1, nonce: one, kind: 0, slots: []rawEntry{leaf(5), leaf(3)}}), "out of order"},
			{"header slots duplicate", header(rawHeader{addrHash: addrHash1, nonce: one, kind: 0, slots: []rawEntry{leaf(5), leaf(5)}}), "out of order"},
			{"group entries unsorted", records(rawCodeGroup(group(stemHash1, 5, 3))), "out of order"},
			{"group entries duplicate", records(rawCodeGroup(group(stemHash1, 5, 5))), "out of order"},
			{"code groups unsorted", records(rawCodeGroup(group(stemHash2, 0)), rawCodeGroup(group(stemHash1, 0))), "out of order"},
			{"code groups duplicate", records(rawCodeGroup(group(stemHash1, 0)), rawCodeGroup(group(stemHash1, 0))), "out of order"},
			{"storage records unsorted", records(storage(addrHash2, group(stemHash1, 0)), storage(addrHash1, group(stemHash1, 0))), "out of order"},
			{"storage records duplicate", records(storage(addrHash1, group(stemHash1, 0)), storage(addrHash1, group(stemHash1, 0))), "out of order"},
			{"storage groups within a record unsorted", records(storage(addrHash1, group(stemHash2, 0), group(stemHash1, 0))), "out of order"},
			{"storage groups within a record duplicate", records(storage(addrHash1, group(stemHash1, 0), group(stemHash1, 0))), "out of order"},

			// One stem, or one account's storage, split over two records: the
			// derived leaves still ascend, but the bytes are not the one
			// canonical encoding.
			{"code group split over two records", records(rawCodeGroup(group(stemHash1, 0)), rawCodeGroup(group(stemHash1, 1))), "out of order"},
			{"storage group split over two records", records(storage(addrHash1, group(stemHash1, 0), group(stemHash1, 1))), "out of order"},
			{"storage record split over two records", records(storage(addrHash1, group(stemHash1, 0)), storage(addrHash1, group(stemHash2, 0))), "out of order"},

			// Truncation, the end tag, the trailer.
			{"truncated mid-record", func() rawSnapshot {
				b := basic(addrHash1).bytes()
				return rawSnapshot{records: b[:len(b)-1], noEnd: true, trailer: []byte{}}
			}(), "EOF"},
			{"missing end tag", rawSnapshot{records: basic(addrHash1).bytes(), noEnd: true, trailer: []byte{}}, "end tag"},
			{"pbtRoot truncated", rawSnapshot{trailer: make([]byte, common.HashLength-1)}, "pbtRoot"},
			{"byte after the pbtRoot trailer", rawSnapshot{trailer: make([]byte, common.HashLength+1)}, "trailing byte"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				sr, err := openSnapshot(tc.snap.write(t))
				if err != nil {
					t.Fatal(err)
				}
				defer sr.close()
				var lastErr error
				for {
					if _, _, err := sr.next(); err == io.EOF {
						break
					} else if err != nil {
						lastErr = err
						break
					}
				}
				if lastErr == nil {
					t.Fatal("the reader accepted it")
				}
				if !strings.Contains(lastErr.Error(), tc.wantErr) {
					t.Fatalf("rejection %q does not name the fault %q", lastErr, tc.wantErr)
				}
			})
		}
	})

	t.Run("preimages", func(t *testing.T) {
		// low/high are ordered by hash, not raw bytes.
		lowAddr, highAddr := addr, common.Address{2}.Bytes()
		if bytes.Compare(crypto.Keccak256(lowAddr), crypto.Keccak256(highAddr)) > 0 {
			lowAddr, highAddr = highAddr, lowAddr
		}
		lowSlot, highSlot := common.Hash{31: 1}, common.Hash{31: 2}
		if bytes.Compare(crypto.Keccak256(lowSlot[:]), crypto.Keccak256(highSlot[:])) > 0 {
			lowSlot, highSlot = highSlot, lowSlot
		}
		for _, tc := range []struct {
			name    string
			recs    []rawPreRecord
			wantErr string
		}{
			{"truncated record header", []rawPreRecord{{addr: addr[:19], count: -1}}, "truncated"},
			{"duplicate address", []rawPreRecord{{addr: addr, count: -1}, {addr: addr, count: -1}}, "out of hashed-key order"},
			{"descending addresses", []rawPreRecord{{addr: highAddr, count: -1}, {addr: lowAddr, count: -1}}, "out of hashed-key order"},
			{"descending slots", []rawPreRecord{{addr: addr, count: -1, slots: [][]byte{highSlot[:], lowSlot[:]}}}, "slot keys are out of hashed-key order"},
			{"duplicate slots", []rawPreRecord{{addr: addr, count: -1, slots: [][]byte{lowSlot[:], lowSlot[:]}}}, "slot keys are out of hashed-key order"},
			{"slot count over the file", []rawPreRecord{{addr: addr, count: 4, slots: [][]byte{lowSlot[:]}}}, "past the end of the file"},
			{"truncated slot key", []rawPreRecord{{addr: addr, count: 1, slots: [][]byte{lowSlot[:31]}}}, "past the end of the file"},
			{"slot count over the file after a stored account", []rawPreRecord{{addr: lowAddr, count: -1, slots: [][]byte{lowSlot[:]}}, {addr: highAddr, count: 2, slots: [][]byte{lowSlot[:]}}}, "past the end of the file"},
			{"trailing partial header", []rawPreRecord{{addr: addr, count: -1}, {addr: highAddr[:1], count: -1}}, "truncated"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				pr, err := openPreimages(rawPreimages(t, tc.recs))
				if err != nil {
					t.Fatal(err)
				}
				defer pr.close()
				var lastErr error
				for {
					if _, _, err := pr.next(); err == io.EOF {
						break
					} else if err != nil {
						lastErr = err
						break
					}
				}
				if lastErr == nil {
					t.Fatal("the reader accepted it")
				}
				if !strings.Contains(lastErr.Error(), tc.wantErr) {
					t.Fatalf("rejection %q does not name the fault %q", lastErr, tc.wantErr)
				}
			})
		}
	})
}

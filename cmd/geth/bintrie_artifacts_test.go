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
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

// The artifact round-trips: decoded with an independent parser written
// against EIP-8347's rules, not the writer, and required to be
// byte-canonical across runs.

// artifactAlloc covers the artifact encoding decisions: shared code, a
// delegation, zero-tailed code, boundary storage, slot zero.
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

// mustCanonicalInt asserts the canonical-integer rule and left-pads back to
// 32 bytes.
func mustCanonicalInt(t *testing.T, blob []byte, what string) common.Hash {
	t.Helper()
	if len(blob) > 32 {
		t.Fatalf("%s is %d bytes, over the 32-byte word", what, len(blob))
	}
	if len(blob) > 0 && blob[0] == 0 {
		t.Fatalf("%s %x carries a leading zero byte, which the encoding forbids", what, blob)
	}
	var padded common.Hash
	copy(padded[32-len(blob):], blob)
	return padded
}

// TestSnapshotArtifactRoundTrip: the decoded leaf records must rebuild to
// the root the header claims, which must be the committed root.
func TestSnapshotArtifactRoundTrip(t *testing.T) {
	// Spill-heavy, so the merged sort path writes the artifact.
	binRoot, snapPath, _ := convertWithArtifacts(t, artifactAlloc(), 512)

	f, err := os.Open(snapPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r := bufio.NewReader(f)

	var header [snapshotHeaderSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		t.Fatalf("truncated header: %v", err)
	}
	claimedRoot := common.BytesToHash(header[:32])
	if claimedRoot != binRoot {
		t.Fatalf("header claims root %x, conversion committed %x", claimedRoot, binRoot)
	}
	leafCount := binary.BigEndian.Uint64(header[32:])

	var (
		stream  = rlp.NewStream(r, 0)
		rebuild = bintrie.NewStackBuilder(nil)
		prevKey []byte
		decoded uint64
	)
	for {
		var rec struct {
			Key   []byte
			Value []byte
		}
		if err := stream.Decode(&rec); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("record %d does not decode: %v", decoded, err)
		}
		// The zone byte fixes the key length.
		var wantLen int
		switch {
		case len(rec.Key) == 0:
			t.Fatalf("record %d has an empty key", decoded)
		case rec.Key[0] == 0x00 || rec.Key[0] == 0x01:
			wantLen = 34
		case rec.Key[0] == 0xff:
			wantLen = 66
		default:
			t.Fatalf("record %d sits in reserved zone %#x", decoded, rec.Key[0])
		}
		if len(rec.Key) != wantLen {
			t.Fatalf("record %d key is %d bytes, zone %#x demands %d", decoded, len(rec.Key), rec.Key[0], wantLen)
		}
		if prevKey != nil && bytes.Compare(prevKey, rec.Key) >= 0 {
			t.Fatalf("record %d key %x out of order after %x", decoded, rec.Key, prevKey)
		}
		prevKey = rec.Key
		if len(rec.Value) == 0 {
			t.Fatalf("record %d holds a zero value, which the tree may not contain", decoded)
		}
		value := mustCanonicalInt(t, rec.Value, "leaf value")
		if err := rebuild.Add(rec.Key, value[:]); err != nil {
			t.Fatalf("record %d does not fold: %v", decoded, err)
		}
		decoded++
	}
	if decoded != leafCount {
		t.Fatalf("decoded %d records, header claims %d", decoded, leafCount)
	}
	if got := rebuild.Finish(); got != claimedRoot {
		t.Fatalf("leaves rebuild to %x, header claims %x", got, claimedRoot)
	}
}

// TestPreimageFileRoundTrip: the decoded records must name exactly the
// converted accounts and slots.
func TestPreimageFileRoundTrip(t *testing.T) {
	alloc := artifactAlloc()
	_, _, prePath := convertWithArtifacts(t, alloc, 512)

	f, err := os.Open(prePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var (
		stream   = rlp.NewStream(bufio.NewReader(f), 0)
		got      = make(map[common.Address]map[common.Hash]bool)
		prevAddr []byte
	)
	for {
		var rec struct {
			Address  []byte
			SlotKeys [][]byte
		}
		if err := stream.Decode(&rec); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("record %d does not decode: %v", len(got), err)
		}
		if len(rec.Address) != common.AddressLength {
			t.Fatalf("address %x is %d bytes, the encoding demands 20", rec.Address, len(rec.Address))
		}
		if prevAddr != nil && bytes.Compare(prevAddr, rec.Address) >= 0 {
			t.Fatalf("address %x out of order after %x", rec.Address, prevAddr)
		}
		prevAddr = rec.Address

		slots := make(map[common.Hash]bool, len(rec.SlotKeys))
		var prevSlot *common.Hash
		for _, enc := range rec.SlotKeys {
			slot := mustCanonicalInt(t, enc, "slot key")
			if prevSlot != nil && bytes.Compare(prevSlot[:], slot[:]) >= 0 {
				t.Fatalf("slot %x of %x out of order after %x", slot, rec.Address, *prevSlot)
			}
			prevSlot = &slot
			slots[slot] = true
		}
		got[common.BytesToAddress(rec.Address)] = slots
	}

	// Exactly the allocation, both directions.
	if len(got) != len(alloc) {
		t.Fatalf("decoded %d accounts, converted %d", len(got), len(alloc))
	}
	for addr, acct := range alloc {
		slots, ok := got[addr]
		if !ok {
			t.Fatalf("account %x converted but missing from the preimage file", addr)
		}
		if len(slots) != len(acct.Storage) {
			t.Fatalf("account %x carries %d slot keys, state holds %d", addr, len(slots), len(acct.Storage))
		}
		for slot := range acct.Storage {
			if !slots[slot] {
				t.Fatalf("slot %x of %x converted but missing from the preimage file", slot, addr)
			}
		}
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

// TestArtifactGoldenDigests freezes the artifact byte format: any encoding
// change moves these reference-pinned digests, and moving them breaks every
// existing producer.
func TestArtifactGoldenDigests(t *testing.T) {
	const (
		wantSnapshot  = "0xf10d40938b44dd9b4a27ddf7d265a6cb20b79d2c45118a2c3e114d16e7f251ef"
		wantPreimages = "0x8b6e6a6c99b425ad7806362e4cfeef8ad5cd81b9e944c7656e6335b309e04900"
	)
	for _, sv := range loadStateVectors(t) {
		if sv.Name != "contract" {
			continue
		}
		_, snapPath, prePath := convertWithArtifacts(t, allocOf(t, sv), 0)
		snap, err := os.ReadFile(snapPath)
		if err != nil {
			t.Fatal(err)
		}
		pre, err := os.ReadFile(prePath)
		if err != nil {
			t.Fatal(err)
		}
		if got := crypto.Keccak256Hash(snap); got != common.HexToHash(wantSnapshot) {
			t.Fatalf("snapshot digest %x, the recorded golden is %s", got, wantSnapshot)
		}
		if got := crypto.Keccak256Hash(pre); got != common.HexToHash(wantPreimages) {
			t.Fatalf("preimage digest %x, the recorded golden is %s", got, wantPreimages)
		}
		return
	}
	t.Fatal("the contract state vector is gone from the testdata")
}

// TestArtifactReaders pins the production readers against the writers: every
// record accepted, counts and digests matching an independent hash of the
// files, truncation and trailing bytes rejected.
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
	if leaves != sr.count {
		t.Fatalf("read %d leaves, header claims %d", leaves, sr.count)
	}
	blob, err := os.ReadFile(snapPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sr.digest(), crypto.Keccak256Hash(blob); got != want {
		t.Fatalf("snapshot digest %x, file hashes to %x", got, want)
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
	if got, want := pr.digest(), crypto.Keccak256Hash(blob); got != want {
		t.Fatalf("preimage digest %x, file hashes to %x", got, want)
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

// rawSnapshot writes a snapshot whose records are encoded verbatim, so a case
// can inject an encoding the writers can never emit.
func rawSnapshot(t *testing.T, root common.Hash, count uint64, recs [][2][]byte) string {
	t.Helper()
	var buf bytes.Buffer
	header := make([]byte, snapshotHeaderSize)
	copy(header, root[:])
	binary.BigEndian.PutUint64(header[32:], count)
	buf.Write(header)
	for _, rec := range recs {
		blob, err := rlp.EncodeToBytes([]any{rec[0], rec[1]})
		if err != nil {
			t.Fatal(err)
		}
		buf.Write(blob)
	}
	path := filepath.Join(t.TempDir(), "snapshot.bin")
	if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// rawPreimages writes a preimage file from verbatim records: rec[0] is the
// address, the rest are slot keys.
func rawPreimages(t *testing.T, recs [][][]byte) string {
	t.Helper()
	var buf bytes.Buffer
	for _, rec := range recs {
		blob, err := rlp.EncodeToBytes([]any{rec[0], rec[1:]})
		if err != nil {
			t.Fatal(err)
		}
		buf.Write(blob)
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
	var (
		key  = bintrie.BasicDataKey(common.Address{1})
		next = bintrie.BasicDataKey(common.Address{2})
		good = bytes.Repeat([]byte{1}, 32)
		addr = common.Address{1}.Bytes()
	)
	if bytes.Compare(key, next) > 0 {
		key, next = next, key
	}
	t.Run("snapshot", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			recs    [][2][]byte
			count   uint64
			wantErr string
		}{
			{"leading zero value", [][2][]byte{{key, append([]byte{0}, good[1:]...)}}, 1, "canonical"},
			{"over-long value", [][2][]byte{{key, bytes.Repeat([]byte{1}, 33)}}, 1, "canonical"},
			{"empty value", [][2][]byte{{key, nil}}, 1, "canonical"},
			{"short key in a valid zone", [][2][]byte{{key[:33], good}}, 1, "demands 34"},
			{"long storage key", [][2][]byte{{append([]byte{bintrie.StorageZone}, bytes.Repeat([]byte{1}, 66)...), good}}, 1, "demands 66"},
			{"duplicate key", [][2][]byte{{key, good}, {key, good}}, 2, "out of order"},
			{"descending keys", [][2][]byte{{next, good}, {key, good}}, 2, "out of order"},
			{"fewer records than claimed", [][2][]byte{{key, good}}, 2, "header claims"},
			{"more records than claimed", [][2][]byte{{key, good}, {next, good}}, 1, "more records"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				sr, err := openSnapshot(rawSnapshot(t, common.Hash{}, tc.count, tc.recs))
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

	t.Run("oversized length prefix", func(t *testing.T) {
		// A record whose value claims four gigabytes: rlp must refuse it
		// against the input it actually has rather than allocate for it.
		blob := append([]byte{0xf8, 0x28, 0xa2}, key...)
		blob = append(blob, 0xbb, 0xff, 0xff, 0xff, 0xff)
		path := filepath.Join(t.TempDir(), "snapshot.bin")
		header := make([]byte, snapshotHeaderSize)
		binary.BigEndian.PutUint64(header[32:], 1)
		if err := os.WriteFile(path, append(header, blob...), 0600); err != nil {
			t.Fatal(err)
		}
		sr, err := openSnapshot(path)
		if err != nil {
			t.Fatal(err)
		}
		defer sr.close()
		if _, _, err := sr.next(); err == nil {
			t.Fatal("a four-gigabyte length claim was accepted")
		}
	})

	t.Run("preimages", func(t *testing.T) {
		other := common.Address{2}.Bytes()
		for _, tc := range []struct {
			name    string
			recs    [][][]byte
			wantErr string
		}{
			{"short address", [][][]byte{{addr[:19]}}, "want 20"},
			{"long address", [][][]byte{{append(addr, 0)}}, "want 20"},
			{"duplicate address", [][][]byte{{addr}, {addr}}, "out of address order"},
			{"descending addresses", [][][]byte{{other}, {addr}}, "out of address order"},
			{"leading zero slot", [][][]byte{{addr, {0x00, 0x01}}}, "canonical"},
			{"over-long slot", [][][]byte{{addr, bytes.Repeat([]byte{1}, 33)}}, "canonical"},
			{"descending slots", [][][]byte{{addr, {0x02}, {0x01}}}, "slot keys out of order"},
			{"duplicate slots", [][][]byte{{addr, {0x02}, {0x02}}}, "slot keys out of order"},
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

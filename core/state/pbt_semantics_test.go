// Copyright 2026 The go-ethereum Authors
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

package state

// State-layer semantics of the EIP-8297 tree: zero-as-absence, storage
// emptiness (the EIP-7610 predicate) and account destruction.

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

// newPBTState returns a state database backed by a binary trie.
func newPBTState(t *testing.T) (*StateDB, *triedb.Database) {
	t.Helper()
	disk := rawdb.NewMemoryDatabase()
	db := triedb.NewDatabase(disk, triedb.PBTDefaults)
	sdb, err := New(types.EmptyBinaryHash, NewDatabase(db, nil))
	if err != nil {
		t.Fatal(err)
	}
	return sdb, db
}

// reopenPBT commits the state and returns a fresh StateDB at the new root.
func reopenPBT(t *testing.T, sdb *StateDB, db *triedb.Database, block uint64) *StateDB {
	t.Helper()
	root, err := sdb.Commit(block, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Commit(root, false); err != nil {
		t.Fatal(err)
	}
	next, err := New(root, NewDatabase(db, nil))
	if err != nil {
		t.Fatal(err)
	}
	return next
}

// TestPBTHasStorage covers the storage-emptiness predicate across both homes of
// an account's storage (header slots below 64 and the overflow bucket) and
// across the committed/uncommitted boundary. Under the binary tree there is no
// per-account storage root, so a regression here is invisible to
// GetStorageRoot.
//
// This pins the predicate, not a consensus rule. EIP-7610 rejects deployments
// from a hardcoded address list, so nothing in block processing consults
// HasStorage today.
func TestPBTHasStorage(t *testing.T) {
	var (
		empty    = common.Address{1}
		headerA  = common.Address{2} // slot below 64: lives in the header stem
		overflow = common.Address{3} // slot above 64: lives in the storage bucket
		both     = common.Address{4}
	)
	sdb, db := newPBTState(t)
	for _, addr := range []common.Address{empty, headerA, overflow, both} {
		sdb.CreateAccount(addr)
		sdb.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
	}
	sdb.SetState(headerA, common.Hash{31: 5}, common.Hash{31: 7})
	sdb.SetState(overflow, common.Hash{30: 4}, common.Hash{31: 8}) // slot 1024
	sdb.SetState(both, common.Hash{31: 5}, common.Hash{31: 9})
	sdb.SetState(both, common.Hash{30: 4}, common.Hash{31: 10})

	// Uncommitted writes already make an account non-empty.
	if !sdb.HasStorage(headerA) {
		t.Fatal("uncommitted header slot not seen")
	}
	if !sdb.HasStorage(overflow) {
		t.Fatal("uncommitted overflow slot not seen")
	}
	if sdb.HasStorage(empty) {
		t.Fatal("empty account reported as having storage")
	}

	sdb = reopenPBT(t, sdb, db, 1)

	for _, tc := range []struct {
		addr common.Address
		want bool
	}{
		{empty, false},
		{headerA, true},
		{overflow, true},
		{both, true},
		{common.Address{0xEE}, false}, // never existed
	} {
		if got := sdb.HasStorage(tc.addr); got != tc.want {
			t.Fatalf("HasStorage(%x) = %v, want %v", tc.addr, got, tc.want)
		}
	}
}

// TestPBTAccountDeletion pins that deleting an account removes everything it
// owns in the unified tree - header stem and overflow storage bucket alike -
// so the resulting root equals a state where the account never existed. A
// merkle-patricia state root never contains a destroyed account's storage
// (its trie simply becomes unreachable), so leaving those leaves behind
// would diverge from any conversion of the same state.
func TestPBTAccountDeletion(t *testing.T) {
	var (
		doomed   = common.Address{1}
		survivor = common.Address{2}
	)
	// Reference: only the survivor ever exists.
	ref, _ := newPBTState(t)
	ref.CreateAccount(survivor)
	ref.SetNonce(survivor, 3, tracing.NonceChangeUnspecified)
	ref.SetState(survivor, common.Hash{31: 5}, common.Hash{31: 1})
	ref.SetState(survivor, common.Hash{30: 4}, common.Hash{31: 2})
	refRoot, err := ref.Commit(1, true, false)
	if err != nil {
		t.Fatal(err)
	}

	// The doomed account holds storage in both homes, then is deleted.
	sdb, db := newPBTState(t)
	for _, addr := range []common.Address{doomed, survivor} {
		sdb.CreateAccount(addr)
		sdb.SetNonce(addr, 3, tracing.NonceChangeUnspecified)
		sdb.SetState(addr, common.Hash{31: 5}, common.Hash{31: 1})
		sdb.SetState(addr, common.Hash{30: 4}, common.Hash{31: 2})
	}
	sdb = reopenPBT(t, sdb, db, 1)
	if !sdb.HasStorage(doomed) {
		t.Fatal("setup: doomed account has no storage")
	}

	// Delete through the trie directly: post-EIP-6780 statedb only destroys
	// same-transaction creations, but the tree operation must be correct.
	tr, err := sdb.db.OpenTrie(sdb.originalRoot)
	if err != nil {
		t.Fatal(err)
	}
	bt := tr.(*bintrie.BinaryTrie)
	for _, addr := range []common.Address{doomed, survivor} {
		if err := bt.UpdateAccount(addr, &types.StateAccount{
			Nonce: 3, Balance: uint256.NewInt(0), CodeHash: types.EmptyCodeHash[:],
		}, 0, nil); err != nil {
			t.Fatal(err)
		}
		bt.UpdateStorage(addr, common.Hash{31: 5}.Bytes(), common.Hash{31: 1}.Bytes())
		bt.UpdateStorage(addr, common.Hash{30: 4}.Bytes(), common.Hash{31: 2}.Bytes())
	}
	if err := bt.DeleteAccount(doomed); err != nil {
		t.Fatal(err)
	}
	if got := bt.Hash(); got != refRoot {
		t.Fatalf("deletion left a trace: %x want %x", got, refRoot)
	}
}

// TestPBTCodeShrink pins that clearing an EIP-7702 delegation leaves no trace:
// the tree comes back byte for byte to an account that never carried code.
//
// This is what the delegation leaf makes possible. While a designator was
// ordinary code it was content-addressed by its own hash, so the clearing
// write - under the empty-code hash - addressed a different stem and could not
// reach it. Held in the account's own header it is nobody else's, so the write
// that puts back the code-hash leaf takes it away.
//
// Root equality is the assertion that matters: checking the leaves alone would
// pass on a tree still carrying the old chunk elsewhere.
func TestPBTCodeShrink(t *testing.T) {
	addr := common.Address{1}
	delegation := types.AddressToDelegation(common.Address{9})

	// Reference: the account never carried code.
	ref, _ := newPBTState(t)
	ref.CreateAccount(addr)
	ref.SetNonce(addr, 2, tracing.NonceChangeUnspecified)
	refRoot, err := ref.Commit(1, true, false)
	if err != nil {
		t.Fatal(err)
	}

	// Same account: set a delegation designator, then clear it.
	sdb, db := newPBTState(t)
	sdb.CreateAccount(addr)
	sdb.SetNonce(addr, 2, tracing.NonceChangeUnspecified)
	sdb.SetCode(addr, delegation, tracing.CodeChangeUnspecified)
	sdb = reopenPBT(t, sdb, db, 1)

	// The premise. Without it the clearing below would be asserted against a
	// tree that never held a delegation, and would pass for the wrong reason.
	if got := delegationAt(t, sdb, addr); !bytes.Equal(got, bintrie.EncodeDelegation(delegation)) {
		t.Fatalf("delegation leaf is %x after delegating, want %x", got, bintrie.EncodeDelegation(delegation))
	}
	if got := codeSizeAt(t, sdb, addr); got != 23 {
		t.Fatalf("code size %d while delegated, want 23", got)
	}

	sdb.SetCode(addr, nil, tracing.CodeChangeAuthorizationClear)
	sdb = reopenPBT(t, sdb, db, 2)

	if got := codeSizeAt(t, sdb, addr); got != 0 {
		t.Fatalf("code size %d after clearing the delegation, want 0", got)
	}
	if got := delegationAt(t, sdb, addr); got != nil {
		t.Fatalf("the delegation leaf survived the clear: %x", got)
	}
	if got := codeHashAt(t, sdb, addr); got != types.EmptyCodeHash {
		t.Fatalf("code hash leaf is %x after clearing the delegation, want the empty-code hash", got)
	}
	if sdb.originalRoot != refRoot {
		t.Fatalf("cleared delegation left a trace: %x want %x", sdb.originalRoot, refRoot)
	}
}

// TestPBTDelegationSurvivesBalanceTouch pins that touching a delegated account
// for its balance alone leaves the delegation exactly where it was.
//
// No set-then-clear test can see this. Such a touch leaves dirtyCode false, so
// the write states no size and no designator and the trie must keep both from
// the stem. Rewriting the code leaves from the account instead would put back
// a code-hash leaf and silently un-delegate, on the commonest write there is.
func TestPBTDelegationSurvivesBalanceTouch(t *testing.T) {
	addr := common.Address{1}
	delegation := types.AddressToDelegation(common.Address{9})
	want := bintrie.EncodeDelegation(delegation)

	sdb, db := newPBTState(t)
	sdb.CreateAccount(addr)
	sdb.SetCode(addr, delegation, tracing.CodeChangeUnspecified)
	sdb = reopenPBT(t, sdb, db, 1)

	// Deliberately no code read and no SetCode: only the balance moves.
	sdb.AddBalance(addr, uint256.NewInt(7), tracing.BalanceChangeUnspecified)
	sdb = reopenPBT(t, sdb, db, 2)

	if got := delegationAt(t, sdb, addr); !bytes.Equal(got, want) {
		t.Fatalf("delegation leaf is %x after a balance-only touch, want %x", got, want)
	}
	if got := codeSizeAt(t, sdb, addr); got != 23 {
		t.Fatalf("code size %d after a balance-only touch, want 23", got)
	}
	if got := codeHashAt(t, sdb, addr); got != (common.Hash{}) {
		t.Fatalf("a code-hash leaf appeared beside the delegation: %x", got)
	}
	// The account the rest of geth sees: EIP-161 emptiness, the txpools'
	// delegation probe and EXTCODEHASH all read this field, so a delegated
	// account has to report its indicator's hash however the leaf is stored.
	tr, err := sdb.db.OpenTrie(sdb.originalRoot)
	if err != nil {
		t.Fatal(err)
	}
	acc, err := tr.(*bintrie.BinaryTrie).GetAccount(addr)
	if err != nil {
		t.Fatal(err)
	}
	if wantHash := crypto.Keccak256(delegation); !bytes.Equal(acc.CodeHash, wantHash) {
		t.Fatalf("GetAccount reports code hash %x, want the designator's %x", acc.CodeHash, wantHash)
	}
}

// TestPBTCodeHashIsNotADelegation pins that the discriminator is the
// sub-index, never the value.
//
// EIP #12114 gives delegation its own sub-index because a value discriminator
// would be grindable: a code hash beginning with 0xef0100 costs about 2^24
// offline Keccaks and would read back as a delegation. Nothing here inspects a
// leaf's contents, so such a hash is just a hash - asserted without needing to
// grind a real preimage.
func TestPBTCodeHashIsNotADelegation(t *testing.T) {
	addr := common.Address{1}
	// A code hash that would be read as a delegation by any value-sniffing
	// implementation. No preimage is needed: nothing reads one.
	var codeHash common.Hash
	copy(codeHash[:], types.DelegationPrefix)
	copy(codeHash[3:], common.Address{0xbe, 0xef}.Bytes())

	sdb, _ := newPBTState(t)
	tr, err := sdb.db.OpenTrie(sdb.originalRoot)
	if err != nil {
		t.Fatal(err)
	}
	bt := tr.(*bintrie.BinaryTrie)
	acc := &types.StateAccount{Nonce: 1, Balance: uint256.NewInt(5), CodeHash: codeHash[:], Root: types.EmptyRootHash}
	if err := bt.UpdateAccount(addr, acc, 100, nil); err != nil {
		t.Fatal(err)
	}
	if got, err := bt.GetStemValue(bintrie.DelegationKey(addr)); err != nil {
		t.Fatal(err)
	} else if got != nil {
		t.Fatalf("a code hash beginning with the marker produced a delegation leaf: %x", got)
	}
	if got, err := bt.GetStemValue(bintrie.CodeHashKey(addr)); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(got, codeHash[:]) {
		t.Fatalf("code hash leaf is %x, want %x", got, codeHash)
	}
	got, err := bt.GetAccount(addr)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.CodeHash, codeHash[:]) {
		t.Fatalf("GetAccount read the marker-prefixed hash as %x, want %x", got.CodeHash, codeHash)
	}
}

// TestPBTCodeSizePreserved pins that touching a contract without executing
// it preserves the code size the binary tree packs into basic data. The
// code is loaded lazily, so the naive len(obj.code) reads zero here and the
// account's basic-data leaf would silently lose its code size.
func TestPBTCodeSizePreserved(t *testing.T) {
	addr := common.Address{1}
	code := bytes.Repeat([]byte{0x60, 0x01}, 40) // 80 bytes, spanning chunks

	sdb, db := newPBTState(t)
	sdb.CreateAccount(addr)
	sdb.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
	sdb.SetCode(addr, code, tracing.CodeChangeUnspecified)
	sdb = reopenPBT(t, sdb, db, 1)

	// Touch the account with a plain balance change: the code is never read.
	sdb.AddBalance(addr, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	sdb = reopenPBT(t, sdb, db, 2)

	if got := codeSizeAt(t, sdb, addr); got != uint32(len(code)) {
		t.Fatalf("code size %d after a balance-only touch, want %d", got, len(code))
	}
}

// chunkAt reads a code chunk leaf straight out of the tree, so a test can tell
// "the code is still there" from "the code hash still says it is".
func chunkAt(t *testing.T, sdb *StateDB, codeHash common.Hash, chunk uint64) []byte {
	t.Helper()
	tr, err := sdb.db.OpenTrie(sdb.originalRoot)
	if err != nil {
		t.Fatal(err)
	}
	val, err := tr.(*bintrie.BinaryTrie).GetStemValue(bintrie.CodeChunkKey(codeHash, chunk))
	if err != nil {
		t.Fatal(err)
	}
	return val
}

// codeSizeAt reads the code size the tree packs into an account's basic data.
func codeSizeAt(t *testing.T, sdb *StateDB, addr common.Address) uint32 {
	t.Helper()
	tr, err := sdb.db.OpenTrie(sdb.originalRoot)
	if err != nil {
		t.Fatal(err)
	}
	basic, err := tr.(*bintrie.BinaryTrie).GetStemValue(bintrie.BasicDataKey(addr))
	if err != nil {
		t.Fatal(err)
	}
	if basic == nil {
		t.Fatal("basic data leaf missing")
	}
	_, codeSize, _, _ := bintrie.DecodeBasicData(basic)
	return codeSize
}

// codeHashAt reads an account's code-hash leaf. An absent leaf reads back as
// the zero hash, which no account holding code can produce.
func codeHashAt(t *testing.T, sdb *StateDB, addr common.Address) common.Hash {
	t.Helper()
	tr, err := sdb.db.OpenTrie(sdb.originalRoot)
	if err != nil {
		t.Fatal(err)
	}
	val, err := tr.(*bintrie.BinaryTrie).GetStemValue(bintrie.CodeHashKey(addr))
	if err != nil {
		t.Fatal(err)
	}
	return common.BytesToHash(val)
}

// delegationAt reads an account's delegation leaf, nil when it holds none.
func delegationAt(t *testing.T, sdb *StateDB, addr common.Address) []byte {
	t.Helper()
	tr, err := sdb.db.OpenTrie(sdb.originalRoot)
	if err != nil {
		t.Fatal(err)
	}
	val, err := tr.(*bintrie.BinaryTrie).GetStemValue(bintrie.DelegationKey(addr))
	if err != nil {
		t.Fatal(err)
	}
	return val
}

// assertCodeLeaves checks an account's header against the EIP-8297 transition
// table: the packed code size, and exactly one of the code-hash and delegation
// leaves. A zero wantHash or nil wantDelegation means that leaf must be
// absent. Checking the size alone would pass on a swapped pair.
func assertCodeLeaves(t *testing.T, sdb *StateDB, addr common.Address, wantSize uint32, wantHash common.Hash, wantDelegation []byte) {
	t.Helper()
	if got := codeSizeAt(t, sdb, addr); got != wantSize {
		t.Fatalf("code size %d, want %d", got, wantSize)
	}
	if got := codeHashAt(t, sdb, addr); got != wantHash {
		t.Fatalf("code hash leaf %x, want %x", got, wantHash)
	}
	var want []byte
	if wantDelegation != nil {
		want = bintrie.EncodeDelegation(wantDelegation)
	}
	if got := delegationAt(t, sdb, addr); !bytes.Equal(got, want) {
		t.Fatalf("delegation leaf %x, want %x", got, want)
	}
}

// TestPBTCodeSizeWrites covers the code size across the paths that set it,
// where TestPBTCodeSizePreserved covers the path that must not touch it.
//
// Which branch of UpdateAccount each case reaches is worth stating, because it
// is not what the names suggest. Only setCode sets dirtyCode, so "deploy",
// "delegation set" and "delegation cleared" all report an exact length -
// SetCode(nil) hashes to the empty hash and still sets the flag, so clearing
// takes the exact path too, not preservation. "codeless account" and
// "recreated after destruct" do pass the negative sentinel, but stop at the
// empty-code-hash guard without ever looking at the resident stem. The lookup
// itself is exercised by TestPBTCodeSizePreserved and by the executed-contract
// case below.
func TestPBTCodeSizeWrites(t *testing.T) {
	var (
		addr       = common.Address{1}
		code       = bytes.Repeat([]byte{0x60, 0x01}, 40) // 80 bytes, spanning chunks
		delegation = append([]byte{0xef, 0x01, 0x00}, common.Address{9}.Bytes()...)
	)
	t.Run("codeless account", func(t *testing.T) {
		// Nothing to preserve: the stem is being created by this very write,
		// so the lookup finds no group and the size must come out zero rather
		// than as the sentinel.
		sdb, db := newPBTState(t)
		sdb.CreateAccount(addr)
		sdb.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
		sdb = reopenPBT(t, sdb, db, 1)
		if got := codeSizeAt(t, sdb, addr); got != 0 {
			t.Fatalf("code size %d on an account that never had code, want 0", got)
		}
	})
	t.Run("deploy", func(t *testing.T) {
		// The control for the delegation cases below: ordinary code holds the
		// other leaf of the exclusive pair.
		sdb, db := newPBTState(t)
		sdb.CreateAccount(addr)
		sdb.SetCode(addr, code, tracing.CodeChangeUnspecified)
		sdb = reopenPBT(t, sdb, db, 1)
		assertCodeLeaves(t, sdb, addr, uint32(len(code)), crypto.Keccak256Hash(code), nil)
	})
	t.Run("delegation set", func(t *testing.T) {
		// The indicator becomes the account's code, so the code-hash leaf it
		// held as a codeless account goes and sub 2 arrives in its place.
		sdb, db := newPBTState(t)
		sdb.CreateAccount(addr)
		sdb.SetCode(addr, delegation, tracing.CodeChangeUnspecified)
		sdb = reopenPBT(t, sdb, db, 1)
		assertCodeLeaves(t, sdb, addr, 23, common.Hash{}, delegation)
	})
	t.Run("re-delegated", func(t *testing.T) {
		// A live account replacing its indicator - the one code replacement
		// EIP-8297 allows. The old value must not survive beside the new.
		other := append([]byte{0xef, 0x01, 0x00}, common.Address{10}.Bytes()...)
		sdb, db := newPBTState(t)
		sdb.CreateAccount(addr)
		sdb.SetCode(addr, delegation, tracing.CodeChangeUnspecified)
		sdb = reopenPBT(t, sdb, db, 1)
		sdb.SetCode(addr, other, tracing.CodeChangeAuthorization)
		sdb = reopenPBT(t, sdb, db, 2)
		assertCodeLeaves(t, sdb, addr, 23, common.Hash{}, other)
	})
	t.Run("delegation cleared", func(t *testing.T) {
		// Shrinking to nothing: the tree still holds the old size when the
		// write lands, so this pins that the new value wins. The account ends
		// holding the code-hash leaf again, as the spec requires.
		sdb, db := newPBTState(t)
		sdb.CreateAccount(addr)
		sdb.SetNonce(addr, 2, tracing.NonceChangeUnspecified) // keep it from being swept as empty
		sdb.SetCode(addr, delegation, tracing.CodeChangeUnspecified)
		sdb = reopenPBT(t, sdb, db, 1)
		sdb.SetCode(addr, nil, tracing.CodeChangeAuthorizationClear)
		sdb = reopenPBT(t, sdb, db, 2)
		assertCodeLeaves(t, sdb, addr, 0, types.EmptyCodeHash, nil)
	})
	t.Run("recreated after destruct", func(t *testing.T) {
		// A destroy and a recreate of one address coalesce into a single
		// update mutation, so nothing ever deletes the old stem and the write
		// lands while the dead contract's leaves are still resident. The
		// empty-code-hash guard is what stops its size being inherited;
		// without it this reads the old contract's length.
		sdb, db := newPBTState(t)
		sdb.CreateAccount(addr)
		sdb.SetCode(addr, code, tracing.CodeChangeUnspecified)
		sdb = reopenPBT(t, sdb, db, 1)

		sdb.SelfDestruct(addr)
		sdb.CreateAccount(addr)
		sdb.SetNonce(addr, 7, tracing.NonceChangeUnspecified)
		sdb = reopenPBT(t, sdb, db, 2)
		if got := codeSizeAt(t, sdb, addr); got != 0 {
			t.Fatalf("code size %d after destruct and recreate, want 0", got)
		}
	})
	t.Run("storage override on an unread contract", func(t *testing.T) {
		// The case the codeKnown guard exists for. SetStorage rebuilds the
		// object and re-sets its code from the lazily-loaded field, which is
		// nil when nothing read the bytecode first, so dirtyCode goes true
		// under a real code hash with no blob behind it. The write has to
		// decline it: reporting len(obj.code) would zero the size, leaving an
		// account whose code hash names bytecode its own size says is absent.
		//
		// Only the account's own leaves can regress here: the code chunks are
		// content-addressed, so they survive whatever this account does.
		sdb, db := newPBTState(t)
		sdb.CreateAccount(addr)
		sdb.SetCode(addr, code, tracing.CodeChangeUnspecified)
		sdb = reopenPBT(t, sdb, db, 1)

		// Deliberately no read of the code before the override.
		sdb.SetStorage(addr, map[common.Hash]common.Hash{{31: 5}: {31: 7}})
		sdb = reopenPBT(t, sdb, db, 2)
		if got := codeSizeAt(t, sdb, addr); got != uint32(len(code)) {
			t.Fatalf("code size %d after a storage override, want %d", got, len(code))
		}
		if got, want := codeHashAt(t, sdb, addr), crypto.Keccak256Hash(code); got != want {
			t.Fatalf("code hash leaf is %x after a storage override, want %x", got, want)
		}
	})
	t.Run("executed contract", func(t *testing.T) {
		// The dominant preserve path in production: any ordinary contract
		// call loads the code and writes storage without going near setCode,
		// so dirtyCode stays false and the size is taken from the stem. Every
		// other subtest here reports a length or stops at the guard.
		sdb, db := newPBTState(t)
		sdb.CreateAccount(addr)
		sdb.SetCode(addr, code, tracing.CodeChangeUnspecified)
		sdb = reopenPBT(t, sdb, db, 1)

		if got := sdb.GetCodeSize(addr); got != len(code) { // loads the blob, as a call would
			t.Fatalf("setup: code size %d, want %d", got, len(code))
		}
		sdb.SetState(addr, common.Hash{31: 5}, common.Hash{31: 7})
		sdb.AddBalance(addr, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
		sdb = reopenPBT(t, sdb, db, 2)
		if got := codeSizeAt(t, sdb, addr); got != uint32(len(code)) {
			t.Fatalf("code size %d after an executed contract wrote storage, want %d", got, len(code))
		}
	})
}

// TestPBTPrefetcherWarmsOwners exercises the prefetcher in binary tree mode,
// where a single sub-fetcher serves every account: slot tasks must be
// prefetched under their own owner, and the warm trie must be adopted
// before any writes reach it. Getting either wrong is invisible to the
// state root - the reads simply go to disk - so this asserts the resulting
// root instead, with the prefetcher on and off.
func TestPBTPrefetcherWarmsOwners(t *testing.T) {
	build := func(prefetch bool) common.Hash {
		disk := rawdb.NewMemoryDatabase()
		db := triedb.NewDatabase(disk, triedb.PBTDefaults)
		sdb, err := New(types.EmptyBinaryHash, NewDatabase(db, nil))
		if err != nil {
			t.Fatal(err)
		}
		// Seed several accounts, each with storage in both homes.
		var addrs []common.Address
		for i := byte(1); i <= 6; i++ {
			addr := common.Address{i}
			addrs = append(addrs, addr)
			sdb.CreateAccount(addr)
			sdb.SetNonce(addr, uint64(i), tracing.NonceChangeUnspecified)
			sdb.SetState(addr, common.Hash{31: 5}, common.Hash{31: i})
			sdb.SetState(addr, common.Hash{30: 4}, common.Hash{31: i})
		}
		root, err := sdb.Commit(1, true, false)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Commit(root, false); err != nil {
			t.Fatal(err)
		}

		// Second block: touch every account's storage, optionally warming
		// the trie through the prefetcher first.
		sdb, err = New(root, NewDatabase(db, nil))
		if err != nil {
			t.Fatal(err)
		}
		if prefetch {
			sdb.StartPrefetcher("test", nil)
			defer sdb.StopPrefetcher()
		}
		for i, addr := range addrs {
			sdb.SetState(addr, common.Hash{31: 5}, common.Hash{31: byte(i + 100)})
			sdb.SetState(addr, common.Hash{30: 4}, common.Hash{31: byte(i + 100)})
		}
		next, err := sdb.Commit(2, true, false)
		if err != nil {
			t.Fatal(err)
		}
		return next
	}
	if warm, cold := build(true), build(false); warm != cold {
		t.Fatalf("prefetched root %x differs from cold root %x", warm, cold)
	}
}

// TestPBTProofsVerify pins that proofs taken from a committed binary tree
// verify against its root: inclusion for present account and storage leaves
// in both storage homes, absence for a missing account and an unwritten
// slot, and rejection once a proof node is tampered with. This is the
// machinery behind eth_getProof in binary tree mode.
func TestPBTProofsVerify(t *testing.T) {
	addr := common.Address{1}
	headerSlot := common.Hash{31: 5} // below 64: in the header stem
	bucketSlot := common.Hash{30: 4} // slot 1024: in the storage bucket
	unwritten := common.Hash{31: 63} // never written

	sdb, db := newPBTState(t)
	for i := byte(1); i <= 5; i++ {
		other := common.Address{i}
		sdb.CreateAccount(other)
		sdb.SetNonce(other, uint64(i), tracing.NonceChangeUnspecified)
	}
	sdb.SetState(addr, headerSlot, common.Hash{31: 7})
	sdb.SetState(addr, bucketSlot, common.Hash{31: 8})
	root, err := sdb.Commit(1, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Commit(root, false); err != nil {
		t.Fatal(err)
	}
	sdb, err = New(root, NewDatabase(db, nil))
	if err != nil {
		t.Fatal(err)
	}
	tr, err := sdb.db.OpenTrie(root)
	if err != nil {
		t.Fatal(err)
	}
	bt := tr.(*bintrie.BinaryTrie)

	present := [][]byte{
		bintrie.BasicDataKey(addr),
		bintrie.CodeHashKey(addr),
		bintrie.StorageSlotKey(addr, headerSlot.Bytes()),
		bintrie.StorageSlotKey(addr, bucketSlot.Bytes()),
	}
	for _, key := range present {
		proof := rawdb.NewMemoryDatabase()
		if err := bt.Prove(key, proof); err != nil {
			t.Fatal(err)
		}
		val, err := bintrie.VerifyProof(root, key, proof)
		if err != nil {
			t.Fatalf("proof for %x failed: %v", key, err)
		}
		if val == nil {
			t.Fatalf("proof for %x proved absence of a present leaf", key)
		}
	}
	absent := [][]byte{
		bintrie.BasicDataKey(common.Address{0xEE}),
		bintrie.StorageSlotKey(addr, unwritten.Bytes()),
	}
	for _, key := range absent {
		proof := rawdb.NewMemoryDatabase()
		if err := bt.Prove(key, proof); err != nil {
			t.Fatal(err)
		}
		val, err := bintrie.VerifyProof(root, key, proof)
		if err != nil {
			t.Fatalf("absence proof for %x errored: %v", key, err)
		}
		if val != nil {
			t.Fatalf("absence proof for %x returned a value: %x", key, val)
		}
	}
	// A tampered proof must not verify.
	key := bintrie.BasicDataKey(addr)
	proof := rawdb.NewMemoryDatabase()
	if err := bt.Prove(key, proof); err != nil {
		t.Fatal(err)
	}
	it := proof.NewIterator(nil, nil)
	defer it.Release()
	if !it.Next() {
		t.Fatal("empty proof")
	}
	corrupted := append([]byte{}, it.Value()...)
	corrupted[len(corrupted)-1] ^= 0xff
	if err := proof.Put(it.Key(), corrupted); err != nil {
		t.Fatal(err)
	}
	if _, err := bintrie.VerifyProof(root, key, proof); err == nil {
		t.Fatal("tampered proof verified")
	}
}

// TestPBTSharedCodeChunks pins the content-addressed code zone: two contracts
// deployed with identical bytecode share every chunk leaf, so the second
// deployment adds only its own header stem - two leaves, whatever the code
// length. This is the dedup EIP-8297 exists to provide, and it is invisible
// to the account-level API.
//
// Sharing starts at chunk 0, where it used to start at 128: no part of a
// contract's code is duplicated per holder any more.
func TestPBTSharedCodeChunks(t *testing.T) {
	// Long enough to span several code stems, so the saving is not a rounding
	// effect of one partly-filled group.
	code := bytes.Repeat([]byte{0x5b}, 600*31) // JUMPDEST repeated: no PUSH spans
	var (
		first  = common.Address{1}
		second = common.Address{2}
	)
	countLeaves := func(sdb *StateDB, db *triedb.Database, root common.Hash) int {
		tr, err := sdb.db.OpenTrie(root)
		if err != nil {
			t.Fatal(err)
		}
		it, err := tr.(*bintrie.BinaryTrie).NodeIterator(nil)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for it.Next(true) {
			if it.Leaf() {
				n++
			}
		}
		if err := it.Error(); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// One contract.
	one, onedb := newPBTState(t)
	one.CreateAccount(first)
	one.SetNonce(first, 1, tracing.NonceChangeUnspecified)
	one.SetCode(first, code, tracing.CodeChangeUnspecified)
	one = reopenPBT(t, one, onedb, 1)
	leavesOne := countLeaves(one, onedb, one.originalRoot)

	// The same code deployed twice.
	two, twodb := newPBTState(t)
	for _, addr := range []common.Address{first, second} {
		two.CreateAccount(addr)
		two.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
		two.SetCode(addr, code, tracing.CodeChangeUnspecified)
	}
	two = reopenPBT(t, two, twodb, 1)
	leavesTwo := countLeaves(two, twodb, two.originalRoot)

	// The second contract contributes its own header stem - basic data and
	// code hash - and nothing else. Every chunk is shared.
	perAccount := 2
	if got, want := leavesTwo-leavesOne, perAccount; got != want {
		t.Fatalf("second identical contract added %d leaves, want %d (code chunks not shared)", got, want)
	}
	// Sanity: a chunk really is in the code zone, and its key does not depend
	// on which account holds the code. Chunk 0 is the one that used to be
	// per-account, so it is the one worth naming here.
	codeHash := crypto.Keccak256Hash(code)
	shared := bintrie.CodeChunkKey(codeHash, 0)
	if shared[0] != bintrie.CodeZone {
		t.Fatalf("chunk 0 is not in the code zone: zone byte %#x", shared[0])
	}
	if got := chunkAt(t, two, codeHash, 0); len(got) != 32 {
		t.Fatalf("chunk 0 is %d bytes in the tree, want 32", len(got))
	}
}

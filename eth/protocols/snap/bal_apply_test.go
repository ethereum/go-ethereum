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

package snap

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// buildTestBAL constructs a BlockAccessList from a ConstructionBlockAccessList
// by RLP round-tripping (construction types use unexported encoding types).
func buildTestBAL(t *testing.T, cb *bal.ConstructionBlockAccessList) *bal.BlockAccessList {
	t.Helper()
	var buf bytes.Buffer
	if err := cb.EncodeRLP(&buf); err != nil {
		t.Fatalf("failed to encode BAL: %v", err)
	}
	var b bal.BlockAccessList
	if err := rlp.DecodeBytes(buf.Bytes(), &b); err != nil {
		t.Fatalf("failed to decode BAL: %v", err)
	}
	return &b
}

// applyBAL applies b to the syncer's flat state and commits it, mirroring the
// per-block batch flow used during catch-up: applyAccessList writes into a batch
// that the caller commits.
func applyBAL(t *testing.T, s *syncerV2, b *bal.BlockAccessList) {
	t.Helper()
	batch := s.db.NewBatch()
	if err := s.applyAccessList(b, batch); err != nil {
		t.Fatalf("applyAccessList failed: %v", err)
	}
	if err := batch.Write(); err != nil {
		t.Fatalf("failed to commit BAL batch: %v", err)
	}
}

// TestAccessListVerification checks that verifyAccessList accepts valid BALs
// and rejects tampered ones.
func TestAccessListVerification(t *testing.T) {
	t.Parallel()

	cb := bal.NewConstructionBlockAccessList()
	addr := common.HexToAddress("0x01")
	cb.BalanceChange(0, addr, uint256.NewInt(100))

	b := buildTestBAL(t, cb)
	correctHash := b.Hash()

	// Valid: hash matches header
	header := &types.Header{
		Number:              big.NewInt(1),
		BlockAccessListHash: &correctHash,
	}
	if err := verifyAccessList(b, header); err != nil {
		t.Fatalf("valid access list rejected: %v", err)
	}
	// Invalid: wrong hash in header
	wrongHash := common.HexToHash("0xdead")
	badHeader := &types.Header{
		Number:              big.NewInt(1),
		BlockAccessListHash: &wrongHash,
	}
	if err := verifyAccessList(b, badHeader); err == nil {
		t.Fatal("tampered access list accepted")
	}
	// Invalid: no hash in header
	noHashHeader := &types.Header{
		Number: big.NewInt(1),
	}
	if err := verifyAccessList(b, noHashHeader); err == nil {
		t.Fatal("header without access list hash accepted")
	}
}

// TestAccessListApplication verifies that applyAccessList correctly updates
// flat state (balance, nonce, code, storage) and leaves storageRoot stale.
func TestAccessListApplication(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()
	syncer := newSyncerV2(db, rawdb.HashScheme)
	addr := common.HexToAddress("0x01")
	accountHash := crypto.Keccak256Hash(addr[:])

	// Write an existing account to flat state
	original := types.StateAccount{
		Nonce:    5,
		Balance:  uint256.NewInt(1000),
		Root:     common.HexToHash("0xbeef"), // intentionally non-empty
		CodeHash: types.EmptyCodeHash[:],
	}
	rawdb.WriteAccountSnapshot(db, accountHash, types.SlimAccountRLP(original))

	// Write an existing storage slot. The BAL uses raw slot keys, but the
	// snapshot layer stores slots under keccak256(slot).
	rawSlot := common.HexToHash("0xaa")
	slotHash := crypto.Keccak256Hash(rawSlot[:])
	rawdb.WriteStorageSnapshot(db, accountHash, slotHash, common.HexToHash("0x01").Bytes())

	// Build a BAL that changes balance, nonce, code, and storage
	cb := bal.NewConstructionBlockAccessList()
	cb.BalanceChange(0, addr, uint256.NewInt(2000))
	cb.NonceChange(addr, 0, 6)
	cb.CodeChange(addr, 0, []byte{0x60, 0x00}) // PUSH1 0x00
	cb.StorageWrite(0, addr, rawSlot, common.HexToHash("0x02"))
	b := buildTestBAL(t, cb)
	applyBAL(t, syncer, b)

	// Verify account fields updated
	data := rawdb.ReadAccountSnapshot(db, accountHash)
	if len(data) == 0 {
		t.Fatal("account snapshot missing after apply")
	}
	updated, err := types.FullAccount(data)
	if err != nil {
		t.Fatalf("failed to decode updated account: %v", err)
	}
	if updated.Balance.Cmp(uint256.NewInt(2000)) != 0 {
		t.Errorf("balance wrong: got %v, want 2000", updated.Balance)
	}
	if updated.Nonce != 6 {
		t.Errorf("nonce wrong: got %d, want 6", updated.Nonce)
	}
	wantCodeHash := crypto.Keccak256([]byte{0x60, 0x00})
	if !bytes.Equal(updated.CodeHash, wantCodeHash) {
		t.Errorf("code hash wrong: got %x, want %x", updated.CodeHash, wantCodeHash)
	}

	// Verify code was written
	if code := rawdb.ReadCode(db, common.BytesToHash(wantCodeHash)); !bytes.Equal(code, []byte{0x60, 0x00}) {
		t.Errorf("code wrong: got %x, want 6000", code)
	}

	// Verify storage updated. Slots are stored in the canonical snapshot
	// encoding (RLP of the value with leading zeros trimmed), the same form
	// the download path writes and the trie rebuild consumes.
	storageVal := rawdb.ReadStorageSnapshot(db, accountHash, slotHash)
	wantStorage, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(common.HexToHash("0x02").Bytes()))
	if !bytes.Equal(storageVal, wantStorage) {
		t.Errorf("storage wrong: got %x, want %x", storageVal, wantStorage)
	}

	// Verify storageRoot left stale (unchanged from original)
	if updated.Root != original.Root {
		t.Errorf("storageRoot should be stale: got %v, want %v", updated.Root, original.Root)
	}
}

// TestAccessListApplicationMultiTx verifies that when an account has multiple
// changes at different transaction indices, only the highest index (post-block
// state) is applied.
func TestAccessListApplicationMultiTx(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()
	syncer := newSyncerV2(db, rawdb.HashScheme)
	addr := common.HexToAddress("0x02")
	accountHash := crypto.Keccak256Hash(addr[:])

	// Write initial account
	original := types.StateAccount{
		Nonce:    0,
		Balance:  uint256.NewInt(100),
		Root:     types.EmptyRootHash,
		CodeHash: types.EmptyCodeHash[:],
	}
	rawdb.WriteAccountSnapshot(db, accountHash, types.SlimAccountRLP(original))

	// Build BAL with multiple balance/nonce changes at different tx indices
	cb := bal.NewConstructionBlockAccessList()
	cb.BalanceChange(0, addr, uint256.NewInt(200))  // tx 0
	cb.BalanceChange(3, addr, uint256.NewInt(500))  // tx 3
	cb.BalanceChange(7, addr, uint256.NewInt(9999)) // tx 7 (final)
	cb.NonceChange(addr, 0, 1)                      // tx 0
	cb.NonceChange(addr, 3, 2)                      // tx 3
	cb.NonceChange(addr, 7, 3)                      // tx 7 (final)
	b := buildTestBAL(t, cb)
	applyBAL(t, syncer, b)
	data := rawdb.ReadAccountSnapshot(db, accountHash)
	updated, err := types.FullAccount(data)
	if err != nil {
		t.Fatalf("failed to decode updated account: %v", err)
	}

	// Only the highest tx index values should be applied
	if updated.Balance.Cmp(uint256.NewInt(9999)) != 0 {
		t.Errorf("balance wrong: got %v, want 9999", updated.Balance)
	}
	if updated.Nonce != 3 {
		t.Errorf("nonce wrong: got %d, want 3", updated.Nonce)
	}
}

// TestAccessListApplicationZeroStorage verifies that a BAL slot write with a
// zero post-value deletes the snapshot entry instead of writing 32 zero
// bytes.
func TestAccessListApplicationZeroStorage(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()
	syncer := newSyncerV2(db, rawdb.HashScheme)
	addr := common.HexToAddress("0x06")
	accountHash := crypto.Keccak256Hash(addr[:])

	// Existing account with a non-zero storage slot.
	original := types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(1),
		Root:     types.EmptyRootHash,
		CodeHash: types.EmptyCodeHash[:],
	}
	rawdb.WriteAccountSnapshot(db, accountHash, types.SlimAccountRLP(original))
	rawSlot := common.HexToHash("0xaa")
	slotHash := crypto.Keccak256Hash(rawSlot[:])
	rawdb.WriteStorageSnapshot(db, accountHash, slotHash, common.HexToHash("0x42").Bytes())

	// BAL writes the slot to zero (deletion).
	cb := bal.NewConstructionBlockAccessList()
	cb.StorageWrite(0, addr, rawSlot, common.Hash{})
	b := buildTestBAL(t, cb)
	applyBAL(t, syncer, b)

	if val := rawdb.ReadStorageSnapshot(db, accountHash, slotHash); len(val) != 0 {
		t.Errorf("zeroed slot should have been deleted, got %x", val)
	}
}

// TestAccessListApplicationNewAccount verifies that applyAccessList creates
// new accounts that don't exist in the DB yet.
func TestAccessListApplicationNewAccount(t *testing.T) {
	t.Parallel()

	db := rawdb.NewMemoryDatabase()
	syncer := newSyncerV2(db, rawdb.HashScheme)

	addr := common.HexToAddress("0x03")
	accountHash := crypto.Keccak256Hash(addr[:])

	// Verify account doesn't exist
	if data := rawdb.ReadAccountSnapshot(db, accountHash); len(data) > 0 {
		t.Fatal("account should not exist yet")
	}

	// Build BAL for a new account. BAL uses raw slot keys.
	cb := bal.NewConstructionBlockAccessList()
	cb.BalanceChange(0, addr, uint256.NewInt(42))
	cb.NonceChange(addr, 0, 1)
	rawSlot := common.HexToHash("0xbb")
	cb.StorageWrite(0, addr, rawSlot, common.HexToHash("0xff"))
	b := buildTestBAL(t, cb)
	applyBAL(t, syncer, b)

	// Verify account was created
	data := rawdb.ReadAccountSnapshot(db, accountHash)
	if len(data) == 0 {
		t.Fatal("account should exist after apply")
	}
	account, err := types.FullAccount(data)
	if err != nil {
		t.Fatalf("failed to decode new account: %v", err)
	}
	if account.Balance.Cmp(uint256.NewInt(42)) != 0 {
		t.Errorf("balance wrong: got %v, want 42", account.Balance)
	}
	if account.Nonce != 1 {
		t.Errorf("nonce wrong: got %d, want 1", account.Nonce)
	}
	if account.Root != types.EmptyRootHash {
		t.Errorf("root should be empty for new account: got %v", account.Root)
	}

	// Verify storage was written under keccak256(rawSlot) in the canonical
	// snapshot encoding (RLP of the value with leading zeros trimmed).
	slotHash := crypto.Keccak256Hash(rawSlot[:])
	storageVal := rawdb.ReadStorageSnapshot(db, accountHash, slotHash)
	wantStorage, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(common.HexToHash("0xff").Bytes()))
	if !bytes.Equal(storageVal, wantStorage) {
		t.Errorf("storage wrong: got %x, want %x", storageVal, wantStorage)
	}
}

// TestAccessListApplicationSkipsUnfetched verifies that applyAccessList does
// not write account entries for addresses whose hash falls in a range that
// hasn't been downloaded yet.
func TestAccessListApplicationSkipsUnfetched(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()
	syncer := newSyncerV2(db, rawdb.HashScheme)

	// Pick two addresses and order them by hash.
	addrA := common.HexToAddress("0x01")
	addrB := common.HexToAddress("0x02")
	hashA := crypto.Keccak256Hash(addrA[:])
	hashB := crypto.Keccak256Hash(addrB[:])
	fetchedAddr, fetchedHash := addrA, hashA
	unfetchedAddr, unfetchedHash := addrB, hashB
	if bytes.Compare(hashA[:], hashB[:]) > 0 {
		fetchedAddr, fetchedHash = addrB, hashB
		unfetchedAddr, unfetchedHash = addrA, hashA
	}

	// One remaining task covering [unfetchedHash, MaxHash]: the fetched hash
	// is below Next so isFetched returns true; the unfetched hash equals Next
	// so isFetched returns false.
	syncer.tasks = []*accountTaskV2{{
		Next:           unfetchedHash,
		Last:           common.MaxHash,
		SubTasks:       make(map[common.Hash][]*storageTaskV2),
		stateCompleted: make(map[common.Hash]struct{}),
	}}

	cb := bal.NewConstructionBlockAccessList()
	cb.BalanceChange(0, fetchedAddr, uint256.NewInt(100))
	cb.BalanceChange(0, unfetchedAddr, uint256.NewInt(200))
	b := buildTestBAL(t, cb)

	applyBAL(t, syncer, b)

	// The fetched account should have been written.
	if data := rawdb.ReadAccountSnapshot(db, fetchedHash); len(data) == 0 {
		t.Error("expected fetched account to be written")
	}
	// The unfetched account should not have been touched.
	if data := rawdb.ReadAccountSnapshot(db, unfetchedHash); len(data) != 0 {
		t.Errorf("unfetched account should not be written, got %x", data)
	}
}

// TestAccessListApplicationSkipsUnfetchedStorage verifies that storage writes
// are also skipped when the parent account's hash range isn't downloaded yet.
func TestAccessListApplicationSkipsUnfetchedStorage(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()
	syncer := newSyncerV2(db, rawdb.HashScheme)

	addrA := common.HexToAddress("0x01")
	addrB := common.HexToAddress("0x02")
	hashA := crypto.Keccak256Hash(addrA[:])
	hashB := crypto.Keccak256Hash(addrB[:])

	unfetchedAddr, unfetchedHash := addrB, hashB
	if bytes.Compare(hashA[:], hashB[:]) > 0 {
		unfetchedAddr, unfetchedHash = addrA, hashA
	}

	syncer.tasks = []*accountTaskV2{{
		Next:           unfetchedHash,
		Last:           common.MaxHash,
		SubTasks:       make(map[common.Hash][]*storageTaskV2),
		stateCompleted: make(map[common.Hash]struct{}),
	}}

	// BAL touches an unfetched account with a storage write AND an empty
	// balance mutation. Neither should result in any flat-state writes.
	rawSlot := common.HexToHash("0xaa")
	slotHash := crypto.Keccak256Hash(rawSlot[:])
	cb := bal.NewConstructionBlockAccessList()
	cb.BalanceChange(0, unfetchedAddr, uint256.NewInt(0)) // empty mutation
	cb.StorageWrite(0, unfetchedAddr, rawSlot, common.HexToHash("0xff"))
	b := buildTestBAL(t, cb)

	applyBAL(t, syncer, b)

	if data := rawdb.ReadAccountSnapshot(db, unfetchedHash); len(data) != 0 {
		t.Errorf("unfetched account should not be written, got %x", data)
	}
	if val := rawdb.ReadStorageSnapshot(db, unfetchedHash, slotHash); len(val) != 0 {
		t.Errorf("storage for unfetched account should not be written, got %x", val)
	}
}

// TestAccessListApplicationSameTxCreateDestroy tests the edge case where an
// account is created and self-destructed in the same transaction during the
// pivot gap. Per EIP-7928, such accounts appear in the BAL with a balance
// change to zero but no nonce or code changes. Since the account didn't exist
// at the old pivot and doesn't exist at the new pivot (destroyed),
// applyAccessList should not leave a zero-balance account in the snapshot.
// Per EIP-161, empty accounts (zero balance, zero nonce, no code) must not exist
// in state.
func TestAccessListApplicationSameTxCreateDestroy(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()
	syncer := newSyncerV2(db, rawdb.HashScheme)
	addr := common.HexToAddress("0x04")
	accountHash := crypto.Keccak256Hash(addr[:])

	// Verify account doesn't exist before apply
	if data := rawdb.ReadAccountSnapshot(db, accountHash); len(data) > 0 {
		t.Fatal("account should not exist yet")
	}

	// Build a BAL mimicking same-tx create+destroy: the account appears
	// with a balance change to zero and nothing else.
	cb := bal.NewConstructionBlockAccessList()
	cb.BalanceChange(0, addr, uint256.NewInt(0))
	b := buildTestBAL(t, cb)
	applyBAL(t, syncer, b)

	// Check if applyAccessList created an account.
	data := rawdb.ReadAccountSnapshot(db, accountHash)
	if len(data) > 0 {
		// Account was created
		account, err := types.FullAccount(data)
		if err != nil {
			t.Fatalf("failed to decode account: %v", err)
		}
		t.Errorf("account created for same-tx create+destroy: "+
			"balance=%v, nonce=%d, codeHash=%x, root=%v",
			account.Balance, account.Nonce, account.CodeHash, account.Root)
	}
}

// TestAccessListApplicationDestroyExisting verifies that when a BAL reduces
// an existing flat-state account to nonce=0, balance=0, empty code (the
// pre-funded destruction pattern), applyAccessList deletes the entry rather
// than leaving it zereod.
func TestAccessListApplicationDestroyExisting(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()
	syncer := newSyncerV2(db, rawdb.HashScheme)
	addr := common.HexToAddress("0x05")
	accountHash := crypto.Keccak256Hash(addr[:])

	// Pre-funded account: has balance, no nonce, no code.
	original := types.StateAccount{
		Nonce:    0,
		Balance:  uint256.NewInt(1000),
		Root:     types.EmptyRootHash,
		CodeHash: types.EmptyCodeHash[:],
	}
	rawdb.WriteAccountSnapshot(db, accountHash, types.SlimAccountRLP(original))

	// The BAL zeros the balance. Nonce and code were already empty, so
	// the account ends up fully empty after applying.
	cb := bal.NewConstructionBlockAccessList()
	cb.BalanceChange(0, addr, uint256.NewInt(0))
	b := buildTestBAL(t, cb)
	applyBAL(t, syncer, b)

	if data := rawdb.ReadAccountSnapshot(db, accountHash); len(data) != 0 {
		account, _ := types.FullAccount(data)
		t.Errorf("destroyed account should have been deleted from flat state, "+
			"got balance=%v, nonce=%d, codeHash=%x",
			account.Balance, account.Nonce, account.CodeHash)
	}
}

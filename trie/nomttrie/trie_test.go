package nomttrie

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/nomt/core"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb/nomtdb"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestTrie creates a NomtTrie backed by an in-memory ethdb.
func newTestTrie(t *testing.T) *NomtTrie {
	t.Helper()
	diskdb := rawdb.NewMemoryDatabase()
	backend := nomtdb.New(diskdb, nil)
	t.Cleanup(func() { backend.Close() })

	tr, err := New(common.Hash{}, backend)
	require.NoError(t, err)
	return tr
}

func TestUpdateAndGetAccount(t *testing.T) {
	tr := newTestTrie(t)

	addr := common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	acc := &types.StateAccount{
		Nonce:    42,
		Balance:  uint256.NewInt(1_000_000),
		CodeHash: common.FromHex("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"),
	}

	require.NoError(t, tr.UpdateAccount(addr, acc, 0))

	// Flush to flat state + page tree.
	root := tr.Hash()
	assert.NotEqual(t, common.Hash{}, root, "root should be non-zero after update")

	// Read back from flat state.
	got, err := tr.GetAccount(addr)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, acc.Nonce, got.Nonce)
	assert.Equal(t, acc.Balance.Uint64(), got.Balance.Uint64())
	assert.Equal(t, acc.CodeHash, got.CodeHash)
}

func TestGetAccountNonExistent(t *testing.T) {
	tr := newTestTrie(t)
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")

	got, err := tr.GetAccount(addr)
	require.NoError(t, err)
	assert.Nil(t, got, "nonexistent account should return nil")
}

func TestUpdateAndGetStorage(t *testing.T) {
	tr := newTestTrie(t)

	addr := common.HexToAddress("0xaaaa")
	slot := common.Hex2Bytes(
		"0000000000000000000000000000000000000000000000000000000000000001",
	)
	value := common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000000ff")

	require.NoError(t, tr.UpdateStorage(addr, slot, value))
	tr.Hash()

	got, err := tr.GetStorage(addr, slot)
	require.NoError(t, err)
	assert.Equal(t, value, got)
}

func TestGetStorageNonExistent(t *testing.T) {
	tr := newTestTrie(t)
	addr := common.HexToAddress("0xbbbb")
	slot := make([]byte, 32)

	got, err := tr.GetStorage(addr, slot)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestDeleteStorage(t *testing.T) {
	tr := newTestTrie(t)

	addr := common.HexToAddress("0xcccc")
	slot := make([]byte, 32)
	slot[31] = 1
	value := make([]byte, 32)
	value[31] = 0x42

	// Write then flush.
	require.NoError(t, tr.UpdateStorage(addr, slot, value))
	tr.Hash()

	// Delete then flush.
	require.NoError(t, tr.DeleteStorage(addr, slot))
	tr.Hash()

	// Value should now be 32 zero bytes (not nil).
	got, err := tr.GetStorage(addr, slot)
	require.NoError(t, err)
	assert.Equal(t, make([]byte, bintrie.HashSize), got)
}

func TestDeleteAccountIsNoOp(t *testing.T) {
	tr := newTestTrie(t)
	addr := common.HexToAddress("0xdddd")

	// DeleteAccount should never error.
	require.NoError(t, tr.DeleteAccount(addr))
}

func TestUpdateContractCode(t *testing.T) {
	tr := newTestTrie(t)

	addr := common.HexToAddress("0xeeee")
	code := make([]byte, 100) // 100 bytes of code
	for i := range code {
		code[i] = byte(i)
	}

	require.NoError(t, tr.UpdateContractCode(addr, common.Hash{}, code))

	// Should have queued pending updates: ceil(100/31) = 4 chunks.
	expectedChunks := (len(code) + bintrie.StemSize - 1) / bintrie.StemSize
	codeUpdates := 0
	for _, u := range tr.pending {
		// Code chunks start at suffix derived from offset 128+.
		codeUpdates++
		_ = u
	}
	assert.Equal(t, expectedChunks, codeUpdates)

	// Flush and verify root changes.
	root := tr.Hash()
	assert.NotEqual(t, common.Hash{}, root)
}

func TestHashIdempotent(t *testing.T) {
	tr := newTestTrie(t)

	addr := common.HexToAddress("0x1234")
	acc := &types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(100),
		CodeHash: make([]byte, 32),
	}

	require.NoError(t, tr.UpdateAccount(addr, acc, 0))

	root1 := tr.Hash()
	root2 := tr.Hash()

	assert.Equal(t, root1, root2, "Hash() should be idempotent")
	assert.False(t, tr.dirty, "dirty flag should be cleared after Hash()")
	assert.Empty(t, tr.pending, "pending should be empty after Hash()")
}

func TestHashEmptyTrieIsZero(t *testing.T) {
	tr := newTestTrie(t)
	root := tr.Hash()
	assert.Equal(t, common.Hash{}, root, "empty trie root should be zero")
}

func TestCommitReturnsRootAndNodeSet(t *testing.T) {
	tr := newTestTrie(t)

	addr := common.HexToAddress("0x5678")
	acc := &types.StateAccount{
		Nonce:    5,
		Balance:  uint256.NewInt(999),
		CodeHash: make([]byte, 32),
	}
	require.NoError(t, tr.UpdateAccount(addr, acc, 0))

	root, nodeset := tr.Commit(false)
	assert.NotEqual(t, common.Hash{}, root)
	assert.NotNil(t, nodeset)
}

func TestMultipleAccountsSameBlock(t *testing.T) {
	tr := newTestTrie(t)

	addrs := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
	}

	for i, addr := range addrs {
		acc := &types.StateAccount{
			Nonce:    uint64(i + 1),
			Balance:  uint256.NewInt(uint64((i + 1) * 1000)),
			CodeHash: make([]byte, 32),
		}
		require.NoError(t, tr.UpdateAccount(addr, acc, 0))
	}

	root := tr.Hash()
	assert.NotEqual(t, common.Hash{}, root)

	// Verify all accounts can be read back.
	for i, addr := range addrs {
		got, err := tr.GetAccount(addr)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint64(i+1), got.Nonce)
	}
}

func TestSequentialBlocks(t *testing.T) {
	tr := newTestTrie(t)

	addr := common.HexToAddress("0xabcdef0000000000000000000000000000000000")

	// Block 1: create account.
	acc := &types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(100),
		CodeHash: make([]byte, 32),
	}
	require.NoError(t, tr.UpdateAccount(addr, acc, 0))
	root1 := tr.Hash()
	assert.NotEqual(t, common.Hash{}, root1)

	// Block 2: update balance.
	acc.Nonce = 2
	acc.Balance = uint256.NewInt(200)
	require.NoError(t, tr.UpdateAccount(addr, acc, 0))
	root2 := tr.Hash()
	assert.NotEqual(t, root1, root2, "root should change after update")

	// Verify updated account.
	got, err := tr.GetAccount(addr)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, uint64(2), got.Nonce)
	assert.Equal(t, uint64(200), got.Balance.Uint64())
}

func TestAccountWithStorageAndCode(t *testing.T) {
	tr := newTestTrie(t)
	addr := common.HexToAddress("0xffff")

	// Update account.
	acc := &types.StateAccount{
		Nonce:    10,
		Balance:  uint256.NewInt(5000),
		CodeHash: common.FromHex("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"),
	}
	require.NoError(t, tr.UpdateAccount(addr, acc, 64))

	// Update storage.
	slot := make([]byte, 32)
	slot[31] = 1
	val := make([]byte, 32)
	val[31] = 0x42
	require.NoError(t, tr.UpdateStorage(addr, slot, val))

	// Update code (small contract).
	code := []byte{0x60, 0x00, 0x60, 0x00, 0xFD} // PUSH0 PUSH0 REVERT
	require.NoError(t, tr.UpdateContractCode(addr, common.Hash{}, code))

	root := tr.Hash()
	assert.NotEqual(t, common.Hash{}, root)

	// Verify account.
	got, err := tr.GetAccount(addr)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, uint64(10), got.Nonce)

	// Verify storage.
	gotVal, err := tr.GetStorage(addr, slot)
	require.NoError(t, err)
	assert.Equal(t, val, gotVal)
}

func TestCopyTrieIsIndependent(t *testing.T) {
	tr := newTestTrie(t)

	addr := common.HexToAddress("0x9999")
	acc := &types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(100),
		CodeHash: make([]byte, 32),
	}
	require.NoError(t, tr.UpdateAccount(addr, acc, 0))

	// Copy before flushing.
	tr2 := tr.Copy()
	assert.Equal(t, len(tr.pending), len(tr2.pending))

	// Flush original.
	root1 := tr.Hash()
	assert.NotEqual(t, common.Hash{}, root1)
	assert.Empty(t, tr.pending)

	// Copy should still have pending updates.
	assert.NotEmpty(t, tr2.pending)
	assert.True(t, tr2.dirty)
}

func TestIsVerkle(t *testing.T) {
	tr := newTestTrie(t)
	assert.True(t, tr.IsVerkle())
}

func TestHashProducesCorrectStemHash(t *testing.T) {
	// Verify that a single-account trie produces a root that matches
	// manual stem hash computation.
	tr := newTestTrie(t)

	addr := common.HexToAddress("0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	acc := &types.StateAccount{
		Nonce:    7,
		Balance:  uint256.NewInt(42),
		CodeHash: make([]byte, 32),
	}
	codeLen := 0

	require.NoError(t, tr.UpdateAccount(addr, acc, codeLen))
	root := tr.Hash()

	// Reproduce the expected root manually.
	stem := accountStem(addr)
	basicData := packBasicData(acc, codeLen)
	codeHashVal := make([]byte, bintrie.HashSize)
	copy(codeHashVal, acc.CodeHash)

	var values [core.StemNodeWidth][]byte
	values[bintrie.BasicDataLeafKey] = basicData[:]
	values[bintrie.CodeHashLeafKey] = codeHashVal
	stemHash := core.HashStem(stem, values)

	// A single-stem trie's canonical root equals the stem hash directly,
	// matching bintrie's behavior (StemNode hash IS the root).
	assert.NotEqual(t, common.Hash{}, root)
	assert.Equal(t, common.Hash(stemHash), root,
		"single-stem trie root should equal the stem hash")
}

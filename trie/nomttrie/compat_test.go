package nomttrie

import (
	"encoding/binary"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/nomt/core"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/nomtdb"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newBintrie creates a fresh in-memory BinaryTrie for testing.
func newBintrie(t *testing.T) *bintrie.BinaryTrie {
	t.Helper()
	diskdb := rawdb.NewMemoryDatabase()
	trieDB := triedb.NewDatabase(diskdb, nil)
	t.Cleanup(func() { trieDB.Close() })
	bt, err := bintrie.NewBinaryTrie(types.EmptyRootHash, trieDB)
	require.NoError(t, err)
	return bt
}

// newNomtTrieForCompat creates a NomtTrie with in-memory ethdb.
func newNomtTrieForCompat(t *testing.T) *NomtTrie {
	t.Helper()
	diskdb := rawdb.NewMemoryDatabase()
	backend := nomtdb.New(diskdb, nil)
	t.Cleanup(func() { backend.Close() })

	tr, err := New(common.Hash{}, backend)
	require.NoError(t, err)
	return tr
}

// TestSingleAccountRootMatch verifies that a single account produces
// the same state root on both BinaryTrie and NomtTrie.
func TestSingleAccountRootMatch(t *testing.T) {
	addr := common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	acc := &types.StateAccount{
		Nonce:    42,
		Balance:  uint256.NewInt(1_000_000),
		CodeHash: common.FromHex("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"),
	}

	// BinaryTrie path.
	bt := newBintrie(t)
	require.NoError(t, bt.UpdateAccount(addr, acc, 0))
	binRoot := bt.Hash()

	// NomtTrie path.
	nt := newNomtTrieForCompat(t)
	require.NoError(t, nt.UpdateAccount(addr, acc, 0))
	nomtRoot := nt.Hash()

	t.Logf("bintrie root: %x", binRoot)
	t.Logf("nomt root:    %x", nomtRoot)

	assert.NotEqual(t, common.Hash{}, binRoot)
	assert.NotEqual(t, common.Hash{}, nomtRoot)
	assert.Equal(t, binRoot, nomtRoot, "single-account root must match bintrie")
}

// TestMultiAccountRootMatch tests whether multiple accounts produce
// matching roots between the two trie implementations.
func TestMultiAccountRootMatch(t *testing.T) {
	addrs := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
	}

	bt := newBintrie(t)
	nt := newNomtTrieForCompat(t)

	for i, addr := range addrs {
		acc := &types.StateAccount{
			Nonce:    uint64(i + 1),
			Balance:  uint256.NewInt(uint64((i + 1) * 1000)),
			CodeHash: make([]byte, 32),
		}
		require.NoError(t, bt.UpdateAccount(addr, acc, 0))
		require.NoError(t, nt.UpdateAccount(addr, acc, 0))
	}

	binRoot := bt.Hash()
	nomtRoot := nt.Hash()

	t.Logf("bintrie root: %x", binRoot)
	t.Logf("nomt root:    %x", nomtRoot)

	assert.NotEqual(t, common.Hash{}, binRoot)
	assert.NotEqual(t, common.Hash{}, nomtRoot)
	assert.Equal(t, binRoot, nomtRoot, "multi-account root must match bintrie")
}

// TestStorageRootMatch tests storage slot updates on both tries.
func TestStorageRootMatch(t *testing.T) {
	addr := common.HexToAddress("0xaaaa")
	acc := &types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(100),
		CodeHash: make([]byte, 32),
	}

	slot := common.Hex2Bytes(
		"0000000000000000000000000000000000000000000000000000000000000001",
	)
	val := common.Hex2Bytes("ff")

	bt := newBintrie(t)
	require.NoError(t, bt.UpdateAccount(addr, acc, 0))
	require.NoError(t, bt.UpdateStorage(addr, slot, val))
	binRoot := bt.Hash()

	nt := newNomtTrieForCompat(t)
	require.NoError(t, nt.UpdateAccount(addr, acc, 0))
	require.NoError(t, nt.UpdateStorage(addr, slot, val))
	nomtRoot := nt.Hash()

	t.Logf("bintrie root: %x", binRoot)
	t.Logf("nomt root:    %x", nomtRoot)

	assert.NotEqual(t, common.Hash{}, binRoot)
	assert.NotEqual(t, common.Hash{}, nomtRoot)
	assert.Equal(t, binRoot, nomtRoot, "storage root must match bintrie")
}

// TestCodeChunkRootMatch tests contract code updates on both tries.
func TestCodeChunkRootMatch(t *testing.T) {
	addr := common.HexToAddress("0xbbbb")
	acc := &types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(0),
		CodeHash: common.FromHex("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"),
	}
	code := make([]byte, 100)
	for i := range code {
		code[i] = byte(i)
	}

	bt := newBintrie(t)
	require.NoError(t, bt.UpdateAccount(addr, acc, len(code)))
	require.NoError(t, bt.UpdateContractCode(addr, common.Hash{}, code))
	binRoot := bt.Hash()

	nt := newNomtTrieForCompat(t)
	require.NoError(t, nt.UpdateAccount(addr, acc, len(code)))
	require.NoError(t, nt.UpdateContractCode(addr, common.Hash{}, code))
	nomtRoot := nt.Hash()

	t.Logf("bintrie root: %x", binRoot)
	t.Logf("nomt root:    %x", nomtRoot)

	assert.NotEqual(t, common.Hash{}, binRoot)
	assert.NotEqual(t, common.Hash{}, nomtRoot)
	assert.Equal(t, binRoot, nomtRoot, "code chunk root must match bintrie")
}

// TestNomtTrieDeterministic verifies that the same operations always
// produce the same root hash in NomtTrie.
func TestNomtTrieDeterministic(t *testing.T) {
	makeAndHash := func() common.Hash {
		tr := newNomtTrieForCompat(t)
		addr := common.HexToAddress("0x1234")
		acc := &types.StateAccount{
			Nonce:    7,
			Balance:  uint256.NewInt(42),
			CodeHash: make([]byte, 32),
		}
		require.NoError(t, tr.UpdateAccount(addr, acc, 0))
		return tr.Hash()
	}

	root1 := makeAndHash()
	root2 := makeAndHash()
	assert.Equal(t, root1, root2, "same operations must produce same root")
}

// TestNomtTrieRootChangesOnUpdate verifies that different state changes
// produce different roots.
func TestNomtTrieRootChangesOnUpdate(t *testing.T) {
	addr := common.HexToAddress("0x5678")

	tr1 := newNomtTrieForCompat(t)
	acc1 := &types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(100),
		CodeHash: make([]byte, 32),
	}
	require.NoError(t, tr1.UpdateAccount(addr, acc1, 0))
	root1 := tr1.Hash()

	tr2 := newNomtTrieForCompat(t)
	acc2 := &types.StateAccount{
		Nonce:    2, // different nonce
		Balance:  uint256.NewInt(100),
		CodeHash: make([]byte, 32),
	}
	require.NoError(t, tr2.UpdateAccount(addr, acc2, 0))
	root2 := tr2.Hash()

	assert.NotEqual(t, root1, root2,
		"different state should produce different roots")
}

// TestNomtTrieSequentialConsistency applies two blocks of changes and
// verifies the final root is consistent.
func TestNomtTrieSequentialConsistency(t *testing.T) {
	tr := newNomtTrieForCompat(t)

	addr := common.HexToAddress("0xabcd")
	acc := &types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(1000),
		CodeHash: make([]byte, 32),
	}

	// Block 1.
	require.NoError(t, tr.UpdateAccount(addr, acc, 0))
	root1 := tr.Hash()
	assert.NotEqual(t, common.Hash{}, root1)

	// Block 2: update balance.
	acc.Nonce = 2
	acc.Balance = uint256.NewInt(2000)
	require.NoError(t, tr.UpdateAccount(addr, acc, 0))
	root2 := tr.Hash()

	assert.NotEqual(t, common.Hash{}, root2)
	assert.NotEqual(t, root1, root2)

	// Re-reading the account should show updated values.
	got, err := tr.GetAccount(addr)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, uint64(2), got.Nonce)
	assert.Equal(t, uint64(2000), got.Balance.Uint64())
}

// TestMixedOpsRootMatch performs account, storage, and code updates on
// both tries and compares results.
func TestMixedOpsRootMatch(t *testing.T) {
	addr := common.HexToAddress("0xffff")
	acc := &types.StateAccount{
		Nonce:    10,
		Balance:  uint256.NewInt(5000),
		CodeHash: common.FromHex("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"),
	}
	code := []byte{0x60, 0x00, 0x60, 0x00, 0xFD}
	slot := make([]byte, 32)
	slot[31] = 1
	val := make([]byte, 32)
	val[31] = 0x42

	bt := newBintrie(t)
	require.NoError(t, bt.UpdateAccount(addr, acc, len(code)))
	require.NoError(t, bt.UpdateStorage(addr, slot, val))
	require.NoError(t, bt.UpdateContractCode(addr, common.Hash{}, code))
	binRoot := bt.Hash()

	nt := newNomtTrieForCompat(t)
	require.NoError(t, nt.UpdateAccount(addr, acc, len(code)))
	require.NoError(t, nt.UpdateStorage(addr, slot, val))
	require.NoError(t, nt.UpdateContractCode(addr, common.Hash{}, code))
	nomtRoot := nt.Hash()

	t.Logf("bintrie root: %x", binRoot)
	t.Logf("nomt root:    %x", nomtRoot)

	assert.NotEqual(t, common.Hash{}, binRoot)
	assert.NotEqual(t, common.Hash{}, nomtRoot)
	assert.Equal(t, binRoot, nomtRoot, "mixed-ops root must match bintrie")
}

// TestBintrieRawInsertRootVector validates the known bintrie test vectors
// to confirm our understanding of the expected hashing.
func TestBintrieRawInsertRootVector(t *testing.T) {
	// This test directly uses bintrie's low-level Insert API to verify
	// known test vectors from trie/bintrie/trie_test.go.
	tree := bintrie.NewBinaryNode()

	zeroKey := [bintrie.HashSize]byte{}
	oneKey := common.HexToHash(
		"0101010101010101010101010101010101010101010101010101010101010101",
	)

	var err error
	tree, err = tree.Insert(zeroKey[:], oneKey[:], nil, 0)
	require.NoError(t, err)

	expected := common.HexToHash(
		"aab1060e04cb4f5dc6f697ae93156a95714debbf77d54238766adc5709282b6f",
	)
	assert.Equal(t, expected, tree.Hash(),
		"single entry root should match known test vector")
}

// TestBintrieMerkleizeVector validates the 4-entry merkle test vector.
func TestBintrieMerkleizeVector(t *testing.T) {
	tree := bintrie.NewBinaryNode()
	keys := [][]byte{
		common.HexToHash("0000000000000000000000000000000000000000000000000000000000000000").Bytes(),
		common.HexToHash("8000000000000000000000000000000000000000000000000000000000000000").Bytes(),
		common.HexToHash("0100000000000000000000000000000000000000000000000000000000000000").Bytes(),
		common.HexToHash("8100000000000000000000000000000000000000000000000000000000000000").Bytes(),
	}
	for i, key := range keys {
		var v [bintrie.HashSize]byte
		binary.LittleEndian.PutUint64(v[:8], uint64(i))
		var err error
		tree, err = tree.Insert(key, v[:], nil, 0)
		require.NoError(t, err)
	}

	expected := common.HexToHash(
		"9317155862f7a3867660ddd0966ff799a3d16aa4df1e70a7516eaa4a675191b5",
	)
	assert.Equal(t, expected, tree.Hash(),
		"4-entry merkle root should match known test vector")
}

// buildInternalTreeRoot computes the root hash using BuildInternalTree at
// skip=0, bypassing the depth-7 page walker split.
func buildInternalTreeRoot(kvs []core.StemKeyValue) core.Node {
	sort.Slice(kvs, func(i, j int) bool {
		return kvs[i].Stem != kvs[j].Stem && stemLess(&kvs[i].Stem, &kvs[j].Stem)
	})
	return core.BuildInternalTree(0, kvs, func(_ core.WriteNode) {})
}

// TestBuildInternalTreeSingleStemMatchesBintrie verifies that
// BuildInternalTree(skip=0) with a single stem produces the same root
// as bintrie's InsertValuesAtStem.
func TestBuildInternalTreeSingleStemMatchesBintrie(t *testing.T) {
	var stem core.StemPath
	stem[0] = 0xAA
	stem[1] = 0xBB

	var values [core.StemNodeWidth][]byte
	values[0] = make([]byte, 32)
	values[0][0] = 0x42
	values[1] = make([]byte, 32)
	values[1][31] = 0xFF

	// NOMT path: compute stem hash, then BuildInternalTree at skip=0.
	stemHash := core.HashStem(stem, values)
	nomtRoot := buildInternalTreeRoot([]core.StemKeyValue{
		{Stem: stem, Hash: stemHash},
	})

	// Bintrie path: InsertValuesAtStem on an empty tree.
	tree := bintrie.NewBinaryNode()
	var binValues [bintrie.StemNodeWidth][]byte
	for i, v := range values {
		if v != nil {
			binValues[i] = v
		}
	}
	var err error
	tree, err = tree.InsertValuesAtStem(stem[:], binValues[:], nil, 0)
	require.NoError(t, err)
	binRoot := tree.Hash()

	t.Logf("BuildInternalTree root: %x", nomtRoot)
	t.Logf("bintrie root:           %x", binRoot)

	assert.Equal(t, binRoot, common.Hash(nomtRoot),
		"BuildInternalTree(skip=0) should match bintrie for a single stem")
}

// TestBuildInternalTreeTwoStemsMatchesBintrie verifies root match with two
// stems that diverge early (bit 0).
func TestBuildInternalTreeTwoStemsMatchesBintrie(t *testing.T) {
	var stemA, stemB core.StemPath
	stemA[0] = 0x00 // bit 0 = 0
	stemB[0] = 0x80 // bit 0 = 1

	valA := make([]byte, 32)
	valA[0] = 0x11
	valB := make([]byte, 32)
	valB[0] = 0x22

	var valsA, valsB [core.StemNodeWidth][]byte
	valsA[0] = valA
	valsB[0] = valB

	hashA := core.HashStem(stemA, valsA)
	hashB := core.HashStem(stemB, valsB)

	nomtRoot := buildInternalTreeRoot([]core.StemKeyValue{
		{Stem: stemA, Hash: hashA},
		{Stem: stemB, Hash: hashB},
	})

	// Bintrie: insert both stems.
	tree := bintrie.NewBinaryNode()
	var err error
	tree, err = tree.InsertValuesAtStem(stemA[:], valsA[:], nil, 0)
	require.NoError(t, err)
	tree, err = tree.InsertValuesAtStem(stemB[:], valsB[:], nil, 0)
	require.NoError(t, err)
	binRoot := tree.Hash()

	t.Logf("BuildInternalTree root: %x", nomtRoot)
	t.Logf("bintrie root:           %x", binRoot)

	assert.Equal(t, binRoot, common.Hash(nomtRoot),
		"BuildInternalTree(skip=0) should match bintrie for two diverging stems")
}

// TestBuildInternalTreeLongPrefixMatchesBintrie verifies root match with two
// stems sharing a long common prefix (bits 0-7 identical, diverge at bit 8).
func TestBuildInternalTreeLongPrefixMatchesBintrie(t *testing.T) {
	var stemA, stemB core.StemPath
	stemA[0] = 0xAA // 10101010
	stemA[1] = 0x00 // bit 8 = 0
	stemB[0] = 0xAA // 10101010 (same first byte)
	stemB[1] = 0x80 // bit 8 = 1

	valA := make([]byte, 32)
	valA[0] = 0x33
	valB := make([]byte, 32)
	valB[0] = 0x44

	var valsA, valsB [core.StemNodeWidth][]byte
	valsA[0] = valA
	valsB[0] = valB

	hashA := core.HashStem(stemA, valsA)
	hashB := core.HashStem(stemB, valsB)

	nomtRoot := buildInternalTreeRoot([]core.StemKeyValue{
		{Stem: stemA, Hash: hashA},
		{Stem: stemB, Hash: hashB},
	})

	tree := bintrie.NewBinaryNode()
	var err error
	tree, err = tree.InsertValuesAtStem(stemA[:], valsA[:], nil, 0)
	require.NoError(t, err)
	tree, err = tree.InsertValuesAtStem(stemB[:], valsB[:], nil, 0)
	require.NoError(t, err)
	binRoot := tree.Hash()

	t.Logf("BuildInternalTree root: %x", nomtRoot)
	t.Logf("bintrie root:           %x", binRoot)

	assert.Equal(t, binRoot, common.Hash(nomtRoot),
		"BuildInternalTree(skip=0) should match bintrie for stems with long shared prefix")
}

// TestBuildInternalTreeFourStemsMatchesBintrie validates the 4-stem case
// using the same keys as TestBintrieMerkleizeVector.
func TestBuildInternalTreeFourStemsMatchesBintrie(t *testing.T) {
	keys := [][32]byte{
		common.HexToHash("0000000000000000000000000000000000000000000000000000000000000000"),
		common.HexToHash("8000000000000000000000000000000000000000000000000000000000000000"),
		common.HexToHash("0100000000000000000000000000000000000000000000000000000000000000"),
		common.HexToHash("8100000000000000000000000000000000000000000000000000000000000000"),
	}

	tree := bintrie.NewBinaryNode()
	var kvs []core.StemKeyValue

	for i, key := range keys {
		var v [bintrie.HashSize]byte
		binary.LittleEndian.PutUint64(v[:8], uint64(i))

		// Bintrie: full 32-byte key insert.
		var err error
		tree, err = tree.Insert(key[:], v[:], nil, 0)
		require.NoError(t, err)

		// NOMT: stem is first 31 bytes, suffix is byte 31.
		var stem core.StemPath
		copy(stem[:], key[:31])
		var vals [core.StemNodeWidth][]byte
		vals[key[31]] = v[:]
		kvs = append(kvs, core.StemKeyValue{
			Stem: stem,
			Hash: core.HashStem(stem, vals),
		})
	}

	binRoot := tree.Hash()
	nomtRoot := buildInternalTreeRoot(kvs)

	t.Logf("bintrie root:           %x", binRoot)
	t.Logf("BuildInternalTree root: %x", nomtRoot)

	assert.Equal(t, binRoot, common.Hash(nomtRoot),
		"BuildInternalTree(skip=0) should match bintrie 4-entry merkle vector")
}

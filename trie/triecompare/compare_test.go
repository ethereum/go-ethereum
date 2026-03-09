package triecompare

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/trie/nomttrie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/nomtdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newBintrie(t testing.TB) *bintrie.BinaryTrie {
	t.Helper()
	diskdb := rawdb.NewMemoryDatabase()
	trieDB := triedb.NewDatabase(diskdb, nil)
	t.Cleanup(func() { trieDB.Close() })
	bt, err := bintrie.NewBinaryTrie(types.EmptyRootHash, trieDB)
	require.NoError(t, err)
	return bt
}

func newNomtTrie(t testing.TB) *nomttrie.NomtTrie {
	t.Helper()
	diskdb := rawdb.NewMemoryDatabase()
	backend := nomtdb.New(diskdb, nil)
	t.Cleanup(func() { backend.Close() })

	nt, err := nomttrie.New(common.Hash{}, backend)
	require.NoError(t, err)
	return nt
}

// applyOp applies a single StateOp to both bintrie and nomttrie.
func applyOp(t testing.TB, bt *bintrie.BinaryTrie, nt *nomttrie.NomtTrie, op StateOp) {
	t.Helper()
	switch op.Kind {
	case OpUpdateAccount:
		require.NoError(t, bt.UpdateAccount(op.Address, op.Account, op.CodeLen))
		require.NoError(t, nt.UpdateAccount(op.Address, op.Account, op.CodeLen))
	case OpUpdateStorage:
		require.NoError(t, bt.UpdateStorage(op.Address, op.Slot, op.Value))
		require.NoError(t, nt.UpdateStorage(op.Address, op.Slot, op.Value))
	case OpUpdateCode:
		require.NoError(t, bt.UpdateContractCode(op.Address, common.Hash{}, op.Code))
		require.NoError(t, nt.UpdateContractCode(op.Address, common.Hash{}, op.Code))
	}
}

// ---------------------------------------------------------------------------
// Test configurations
// ---------------------------------------------------------------------------

var (
	smallConfig = StateGenConfig{
		NumAccounts:  100,
		NumContracts: 50,
		MinSlots:     1,
		MaxSlots:     20,
		CodeSize:     128,
		Distribution: PowerLaw,
		Seed:         42,
	}
	mediumConfig = StateGenConfig{
		NumAccounts:  1_000,
		NumContracts: 500,
		MinSlots:     1,
		MaxSlots:     100,
		CodeSize:     256,
		Distribution: PowerLaw,
		Seed:         42,
	}
	largeConfig = StateGenConfig{
		NumAccounts:  10_000,
		NumContracts: 5_000,
		MinSlots:     1,
		MaxSlots:     500,
		CodeSize:     512,
		Distribution: PowerLaw,
		Seed:         42,
	}
)

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestRootEquality generates realistic state at various sizes and verifies
// that bintrie and NOMT produce identical state roots after each block.
func TestRootEquality(t *testing.T) {
	configs := map[string]StateGenConfig{
		"Small": smallConfig,
	}
	if !testing.Short() {
		configs["Medium"] = mediumConfig
		configs["Large"] = largeConfig
	}

	for name, cfg := range configs {
		t.Run(name, func(t *testing.T) {
			blocks := GenerateBlocks(cfg)

			bt := newBintrie(t)
			nt := newNomtTrie(t)

			for blockIdx, ops := range blocks {
				for _, op := range ops {
					applyOp(t, bt, nt, op)
				}
				binRoot := bt.Hash()
				nomtRoot := nt.Hash()

				t.Logf("block %d: %d ops, bintrie=%x nomt=%x",
					blockIdx, len(ops), binRoot[:8], nomtRoot[:8])

				assert.NotEqual(t, common.Hash{}, binRoot,
					"bintrie root should be non-zero at block %d", blockIdx)
				assert.Equal(t, binRoot, nomtRoot,
					"root mismatch at block %d", blockIdx)
			}
		})
	}
}

// TestDeterminism runs the same seed twice and verifies identical roots.
func TestDeterminism(t *testing.T) {
	computeRoot := func() common.Hash {
		blocks := GenerateBlocks(smallConfig)
		nt := newNomtTrie(t)
		bt := newBintrie(t)
		var root common.Hash
		for _, ops := range blocks {
			for _, op := range ops {
				applyOp(t, bt, nt, op)
			}
			root = nt.Hash()
			bt.Hash() // flush bintrie too
		}
		return root
	}

	root1 := computeRoot()
	root2 := computeRoot()
	assert.Equal(t, root1, root2, "same seed must produce same root")
}

// TestDistributionVariants runs Small config with each distribution type
// and verifies matching roots for all variants.
func TestDistributionVariants(t *testing.T) {
	distributions := []struct {
		name string
		dist Distribution
	}{
		{"PowerLaw", PowerLaw},
		{"Uniform", Uniform},
		{"Exponential", Exponential},
	}

	for _, d := range distributions {
		t.Run(d.name, func(t *testing.T) {
			cfg := smallConfig
			cfg.Distribution = d.dist
			cfg.Seed = 123 // same seed for all

			blocks := GenerateBlocks(cfg)

			bt := newBintrie(t)
			nt := newNomtTrie(t)

			var binRoot, nomtRoot common.Hash
			for _, ops := range blocks {
				for _, op := range ops {
					applyOp(t, bt, nt, op)
				}
				binRoot = bt.Hash()
				nomtRoot = nt.Hash()
			}

			t.Logf("dist=%s bintrie=%x nomt=%x", d.name, binRoot[:8], nomtRoot[:8])
			assert.Equal(t, binRoot, nomtRoot,
				"root mismatch with %s distribution", d.name)
		})
	}
}

// TestIncrementalRootEquality hashes after every single operation in the
// first block, catching ordering-sensitive bugs.
func TestIncrementalRootEquality(t *testing.T) {
	if testing.Short() {
		t.Skip("incremental test is slow")
	}

	// Use a smaller config to keep hash-per-op feasible.
	cfg := StateGenConfig{
		NumAccounts:  20,
		NumContracts: 10,
		MinSlots:     1,
		MaxSlots:     5,
		CodeSize:     64,
		Distribution: Uniform,
		Seed:         99,
	}
	blocks := GenerateBlocks(cfg)

	bt := newBintrie(t)
	nt := newNomtTrie(t)

	for i, op := range blocks[0] {
		applyOp(t, bt, nt, op)
		binRoot := bt.Hash()
		nomtRoot := nt.Hash()

		if binRoot != nomtRoot {
			t.Fatalf("root mismatch at op %d (kind=%d addr=%x): bin=%x nomt=%x",
				i, op.Kind, op.Address[:4], binRoot[:8], nomtRoot[:8])
		}
	}
	t.Logf("verified %d incremental hashes match", len(blocks[0]))
}

// TestStorageFootprint populates state and measures serialized node sizes
// for bintrie. NOMT pages are now in ethdb, so only bintrie size is reported.
func TestStorageFootprint(t *testing.T) {
	if testing.Short() {
		t.Skip("storage footprint test requires medium config")
	}

	cfg := mediumConfig
	blocks := GenerateBlocks(cfg)

	bt := newBintrie(t)
	nt := newNomtTrie(t)

	for _, ops := range blocks {
		for _, op := range ops {
			applyOp(t, bt, nt, op)
		}
	}

	// Force both implementations to finalize.
	binRoot := bt.Hash()
	nomtRoot := nt.Hash()
	require.Equal(t, binRoot, nomtRoot, "roots must match before measuring storage")

	// Bintrie: sum serialized node blobs from Commit.
	_, ns := bt.Commit(false)
	binBytes := nodesetBytes(ns)

	t.Logf("bintrie serialized nodes: %s (%d bytes)", humanBytes(binBytes), binBytes)
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkUpdateAccount(b *testing.B) {
	cfg := smallConfig
	blocks := GenerateBlocks(cfg)
	ops := filterOps(blocks[0], OpUpdateAccount)

	b.Run("bintrie", func(b *testing.B) {
		bt := newBintrie(b)
		b.ResetTimer()
		for i := range b.N {
			op := ops[i%len(ops)]
			_ = bt.UpdateAccount(op.Address, op.Account, op.CodeLen)
		}
	})

	b.Run("nomt", func(b *testing.B) {
		nt := newNomtTrie(b)
		b.ResetTimer()
		for i := range b.N {
			op := ops[i%len(ops)]
			_ = nt.UpdateAccount(op.Address, op.Account, op.CodeLen)
		}
	})
}

func BenchmarkUpdateStorage(b *testing.B) {
	cfg := smallConfig
	blocks := GenerateBlocks(cfg)
	ops := filterOps(blocks[0], OpUpdateStorage)

	b.Run("bintrie", func(b *testing.B) {
		bt := newBintrie(b)
		b.ResetTimer()
		for i := range b.N {
			op := ops[i%len(ops)]
			_ = bt.UpdateStorage(op.Address, op.Slot, op.Value)
		}
	})

	b.Run("nomt", func(b *testing.B) {
		nt := newNomtTrie(b)
		b.ResetTimer()
		for i := range b.N {
			op := ops[i%len(ops)]
			_ = nt.UpdateStorage(op.Address, op.Slot, op.Value)
		}
	})
}

func BenchmarkHash(b *testing.B) {
	for _, size := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			cfg := StateGenConfig{
				NumAccounts:  size,
				NumContracts: 0,
				MinSlots:     0,
				MaxSlots:     0,
				CodeSize:     0,
				Distribution: Uniform,
				Seed:         77,
			}
			blocks := GenerateBlocks(cfg)

			b.Run("bintrie", func(b *testing.B) {
				bt := newBintrie(b)
				for _, op := range blocks[0] {
					_ = bt.UpdateAccount(op.Address, op.Account, op.CodeLen)
				}
				bt.Hash() // baseline

				b.ResetTimer()
				for range b.N {
					// Modify one account to dirty the trie.
					op := blocks[0][0]
					op.Account.Nonce++
					_ = bt.UpdateAccount(op.Address, op.Account, op.CodeLen)
					bt.Hash()
				}
			})

			b.Run("nomt", func(b *testing.B) {
				nt := newNomtTrie(b)
				for _, op := range blocks[0] {
					_ = nt.UpdateAccount(op.Address, op.Account, op.CodeLen)
				}
				nt.Hash() // baseline

				b.ResetTimer()
				for range b.N {
					op := blocks[0][0]
					op.Account.Nonce++
					_ = nt.UpdateAccount(op.Address, op.Account, op.CodeLen)
					nt.Hash()
				}
			})
		})
	}
}

func BenchmarkBlockWorkload(b *testing.B) {
	cfg := smallConfig
	blocks := GenerateBlocks(cfg)

	// Use block 1 (mutations) as the repeated workload.
	workload := blocks[1]

	b.Run("bintrie", func(b *testing.B) {
		bt := newBintrie(b)
		// Apply initial state.
		for _, op := range blocks[0] {
			applyOpSingle(b, bt, op)
		}
		bt.Hash()

		b.ResetTimer()
		for range b.N {
			for _, op := range workload {
				applyOpSingle(b, bt, op)
			}
			bt.Hash()
		}
	})

	b.Run("nomt", func(b *testing.B) {
		nt := newNomtTrie(b)
		for _, op := range blocks[0] {
			applyOpSingleNomt(b, nt, op)
		}
		nt.Hash()

		b.ResetTimer()
		for range b.N {
			for _, op := range workload {
				applyOpSingleNomt(b, nt, op)
			}
			nt.Hash()
		}
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// applyOpSingle applies a StateOp to a bintrie only (for benchmarks).
func applyOpSingle(t testing.TB, bt *bintrie.BinaryTrie, op StateOp) {
	t.Helper()
	switch op.Kind {
	case OpUpdateAccount:
		_ = bt.UpdateAccount(op.Address, op.Account, op.CodeLen)
	case OpUpdateStorage:
		_ = bt.UpdateStorage(op.Address, op.Slot, op.Value)
	case OpUpdateCode:
		_ = bt.UpdateContractCode(op.Address, common.Hash{}, op.Code)
	}
}

// applyOpSingleNomt applies a StateOp to a NomtTrie only (for benchmarks).
func applyOpSingleNomt(t testing.TB, nt *nomttrie.NomtTrie, op StateOp) {
	t.Helper()
	switch op.Kind {
	case OpUpdateAccount:
		_ = nt.UpdateAccount(op.Address, op.Account, op.CodeLen)
	case OpUpdateStorage:
		_ = nt.UpdateStorage(op.Address, op.Slot, op.Value)
	case OpUpdateCode:
		_ = nt.UpdateContractCode(op.Address, common.Hash{}, op.Code)
	}
}

// filterOps returns only operations of the given kind.
func filterOps(ops []StateOp, kind OpKind) []StateOp {
	var out []StateOp
	for i := range ops {
		if ops[i].Kind == kind {
			out = append(out, ops[i])
		}
	}
	return out
}

// nodesetBytes sums the serialized blob sizes from a bintrie NodeSet.
func nodesetBytes(ns *trienode.NodeSet) int64 {
	if ns == nil {
		return 0
	}
	var total int64
	for _, node := range ns.Nodes {
		total += int64(len(node.Blob))
	}
	return total
}

// humanBytes formats byte counts for log output.
func humanBytes(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

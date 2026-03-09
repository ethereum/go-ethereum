package db

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/nomt/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	diskdb := rawdb.NewMemoryDatabase()
	db, err := New(diskdb, DefaultConfig())
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestNewClose(t *testing.T) {
	db := newTestDB(t)
	assert.Equal(t, core.Terminator, db.Root())
}

func TestReopenPreservesRoot(t *testing.T) {
	diskdb := rawdb.NewMemoryDatabase()

	db, err := New(diskdb, DefaultConfig())
	require.NoError(t, err)

	newRoot, err := db.Update([]core.StemKeyValue{
		{Stem: makeStem(0x10), Hash: makeHash(0x01)},
	})
	require.NoError(t, err)
	require.False(t, core.IsTerminator(&newRoot))

	// "Reopen" by creating a new DB on the same ethdb.
	db2, err := New(diskdb, DefaultConfig())
	require.NoError(t, err)

	// Root is now persisted in PebbleDB, so it should be recovered.
	assert.Equal(t, newRoot, db2.Root())
}

func TestUpdateSingleKey(t *testing.T) {
	db := newTestDB(t)

	newRoot, err := db.Update([]core.StemKeyValue{
		{Stem: makeStem(0x10), Hash: makeHash(0x42)},
	})
	require.NoError(t, err)

	assert.False(t, core.IsTerminator(&newRoot))
	assert.Equal(t, newRoot, db.Root())
}

func TestUpdateMultipleKeys(t *testing.T) {
	db := newTestDB(t)

	ops := []core.StemKeyValue{
		{Stem: makeStem(0x10), Hash: makeHash(0x01)},
		{Stem: makeStem(0x80), Hash: makeHash(0x02)},
	}

	newRoot, err := db.Update(ops)
	require.NoError(t, err)
	assert.False(t, core.IsTerminator(&newRoot))
}

func TestUpdateDeterministic(t *testing.T) {
	ops := []core.StemKeyValue{
		{Stem: makeStem(0x10), Hash: makeHash(0x01)},
		{Stem: makeStem(0x80), Hash: makeHash(0x02)},
	}

	run := func() core.Node {
		db := newTestDB(t)
		root, err := db.Update(ops)
		require.NoError(t, err)
		return root
	}

	r1 := run()
	r2 := run()
	assert.Equal(t, r1, r2, "same ops should produce same root")
}

func TestUpdateEmptyOps(t *testing.T) {
	db := newTestDB(t)

	root, err := db.Update(nil)
	require.NoError(t, err)
	assert.Equal(t, core.Terminator, root)
}

func TestUpdateSortsByStem(t *testing.T) {
	db := newTestDB(t)

	// Provide stems in reverse order — should still work.
	ops := []core.StemKeyValue{
		{Stem: makeStem(0x80), Hash: makeHash(0x01)},
		{Stem: makeStem(0x10), Hash: makeHash(0x02)},
	}

	root, err := db.Update(ops)
	require.NoError(t, err)
	assert.False(t, core.IsTerminator(&root))
}

func makeStem(b byte) core.StemPath {
	var sp core.StemPath
	for i := range sp {
		sp[i] = b
	}
	return sp
}

func makeHash(b byte) core.Node {
	var h core.Node
	for i := range h {
		h[i] = b ^ byte(i)
	}
	// Ensure non-zero to avoid terminator.
	h[0] |= 0x01
	return h
}

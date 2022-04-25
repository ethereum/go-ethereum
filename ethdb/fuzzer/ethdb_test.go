package main

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
)

func FuzzIterator(f *testing.F, New func() ethdb.KeyValueStore) {
	testcases := []struct {
		content map[string]string
		prefix  string
		start   string
		order   []string
	}{
		// Empty databases should be iterable
		{map[string]string{}, "", "", nil},
		{map[string]string{}, "non-existent-prefix", "", nil},

		// Single-item databases should be iterable
		{map[string]string{"key": "val"}, "", "", []string{"key"}},
		{map[string]string{"key": "val"}, "k", "", []string{"key"}},
		{map[string]string{"key": "val"}, "l", "", nil},

		// Multi-item databases should be fully iterable
		{
			map[string]string{"k1": "v1", "k5": "v5", "k2": "v2", "k4": "v4", "k3": "v3"},
			"", "",
			[]string{"k1", "k2", "k3", "k4", "k5"},
		},
		{
			map[string]string{"k1": "v1", "k5": "v5", "k2": "v2", "k4": "v4", "k3": "v3"},
			"k", "",
			[]string{"k1", "k2", "k3", "k4", "k5"},
		},
		{
			map[string]string{"k1": "v1", "k5": "v5", "k2": "v2", "k4": "v4", "k3": "v3"},
			"l", "",
			nil,
		},
		// Multi-item databases should be prefix-iterable
		{
			map[string]string{
				"ka1": "va1", "ka5": "va5", "ka2": "va2", "ka4": "va4", "ka3": "va3",
				"kb1": "vb1", "kb5": "vb5", "kb2": "vb2", "kb4": "vb4", "kb3": "vb3",
			},
			"ka", "",
			[]string{"ka1", "ka2", "ka3", "ka4", "ka5"},
		},
		{
			map[string]string{
				"ka1": "va1", "ka5": "va5", "ka2": "va2", "ka4": "va4", "ka3": "va3",
				"kb1": "vb1", "kb5": "vb5", "kb2": "vb2", "kb4": "vb4", "kb3": "vb3",
			},
			"kc", "",
			nil,
		},
		// Multi-item databases should be prefix-iterable with start position
		{
			map[string]string{
				"ka1": "va1", "ka5": "va5", "ka2": "va2", "ka4": "va4", "ka3": "va3",
				"kb1": "vb1", "kb5": "vb5", "kb2": "vb2", "kb4": "vb4", "kb3": "vb3",
			},
			"ka", "3",
			[]string{"ka3", "ka4", "ka5"},
		},
		{
			map[string]string{
				"ka1": "va1", "ka5": "va5", "ka2": "va2", "ka4": "va4", "ka3": "va3",
				"kb1": "vb1", "kb5": "vb5", "kb2": "vb2", "kb4": "vb4", "kb3": "vb3",
			},
			"ka", "8",
			nil,
		},
	}

	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, input struct {
		content map[string]string
		prefix  string
		start   string
		order   []string
	}) {
		db := New()

		for key, val := range input.content {
			if err := db.Put([]byte(key), []byte(val)); err != nil {
				t.Fatalf("failed to insert item %s:%s into database: %v", key, val, err)
			}
		}

		// Iterate over the database with the given configs and verify the results
		it, idx := db.NewIterator([]byte(input.prefix), []byte(input.start)), 0
		for it.Next() {
			if len(input.order) <= idx {
				t.Errorf("prefix=%q more items than expected: checking idx=%d (key %q), expecting len=%d", input.prefix, idx, it.Key(), len(input.order))
				break
			}
			if !bytes.Equal(it.Key(), []byte(input.order[idx])) {
				t.Errorf("item %d: key mismatch: have %s, want %s", idx, string(it.Key()), input.order[idx])
			}
			if !bytes.Equal(it.Value(), []byte(input.content[input.order[idx]])) {
				t.Errorf("item %d: value mismatch: have %s, want %s", idx, string(it.Value()), input.content[input.order[idx]])
			}
			idx++
		}
		if err := it.Error(); err != nil {
			t.Errorf("iteration failed: %v", err)
		}
		if idx != len(input.order) {
			t.Errorf("iteration terminated prematurely: have %d, want %d", idx, len(input.order))
		}

		db.Close()
	})
}

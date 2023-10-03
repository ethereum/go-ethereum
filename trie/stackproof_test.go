package trie

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"golang.org/x/crypto/sha3"
	"golang.org/x/exp/slices"
)

func trieWithSmallValues() (*Trie, map[string]*kv) {
	trie := NewEmpty(NewDatabase(rawdb.NewMemoryDatabase(), nil))
	vals := make(map[string]*kv)
	// This loop creates a few dense nodes with small leafs: hence will
	// cause embedded nodes.
	for i := byte(0); i < 100; i++ {
		value := &kv{common.LeftPadBytes([]byte{i}, 32), []byte{i}, false}
		trie.MustUpdate(value.k, value.v)
		vals[string(value.k)] = value
	}
	return trie, vals
}

func TestStRangeProofLeftside(t *testing.T) {
	trie, vals := randomTrie(4096)
	testStRangeProofLeftside(t, trie, vals)
}

func TestStRangeProofLeftsideSmallValues(t *testing.T) {
	trie, vals := trieWithSmallValues()
	testStRangeProofLeftside(t, trie, vals)
}

func testStRangeProofLeftside(t *testing.T, trie *Trie, vals map[string]*kv) {
	var (
		want    = trie.Hash()
		entries []*kv
	)
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	slices.SortFunc(entries, (*kv).cmp)
	for start := 10; start < len(vals); start *= 2 {
		// Set write-fn on both stacktries, to compare outputs
		var (
			haveSponge = &spongeDb{sponge: sha3.NewLegacyKeccak256(), id: "have"}
			wantSponge = &spongeDb{sponge: sha3.NewLegacyKeccak256(), id: "want"}
			proof      = memorydb.New()
		)
		// Provide the proof for the first entry
		if err := trie.Prove(entries[start].k, proof); err != nil {
			t.Fatalf("Failed to prove the first node %v", err)
		}
		// Initiate the stacktrie with the proof
		stTrie, err := newStackTrieFromProof(trie.Hash(), entries[start].k, proof, func(path []byte, hash common.Hash, blob []byte) {
			rawdb.WriteTrieNode(haveSponge, common.Hash{}, path, hash, blob, "path")
		})
		if err != nil {
			t.Fatal(err)
		}
		// Initiate a reference stacktrie without proof (filling manually)
		refTrie := NewStackTrie(nil)
		for i := 0; i <= start; i++ { // do prefill
			k, v := common.CopyBytes(entries[i].k), common.CopyBytes(entries[i].v)
			refTrie.Update(k, v)
		}
		refTrie.writeFn = func(path []byte, hash common.Hash, blob []byte) {
			rawdb.WriteTrieNode(wantSponge, common.Hash{}, path, hash, blob, "path")
		}
		// Feed the remaining values into them both
		for i := start + 1; i < len(vals); i++ {
			stTrie.Update(entries[i].k, common.CopyBytes(entries[i].v))
			refTrie.Update(entries[i].k, common.CopyBytes(entries[i].v))
		}
		// Verify the final trie hash
		if have := stTrie.Hash(); have != want {
			t.Fatalf("wrong hash, have %x want %x\n", have, want)
		}
		if have := refTrie.Hash(); have != want {
			t.Fatalf("wrong hash, have %x want %x\n", have, want)
		}
		// Verify the sequence of committed nodes
		if have, want := haveSponge.sponge.Sum(nil), wantSponge.sponge.Sum(nil); !bytes.Equal(have, want) {
			// Show the journal
			t.Logf("Want:")
			for i, v := range wantSponge.journal {
				t.Logf("op %d: %v", i, v)
			}
			t.Logf("Have:")
			for i, v := range haveSponge.journal {
				t.Logf("op %d: %v", i, v)
			}
			t.Errorf("proof from %d: disk write sequence wrong:\nhave %x want %x\n", start, have, want)
		}
	}
}

func TestStackInsertHash(t *testing.T) {
	trie, vals := randomTrie(4096)
	testStackInsertHash(t, trie, vals)
}

func testStackInsertHash(t *testing.T, trie *Trie, vals map[string]*kv) {
	var (
		entries []*kv
		want    = trie.Hash()
	)
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	slices.SortFunc(entries, (*kv).cmp)
	for start := 10; start < len(vals); start *= 2 {
		var (
			proof = memorydb.New()
		)
		// Provide the proof for the first entry
		if err := trie.Prove(entries[start].k, proof); err != nil {
			t.Fatalf("Failed to prove the first node %v", err)
		}
		// Now we have a proof: use it to initiate the stacktrie
		stTrie, err := newStackTrieFromProof(trie.Hash(), entries[start].k, proof, nil)
		if err != nil {
			t.Fatal(err)
		}
		// Obtain the hashes
		hps, err := iterateProof(trie.Hash(), entries[start].k, false, proof)
		if err != nil {
			t.Fatal(err)
		}
		slices.Reverse(hps)
		// Insert into stacktrie
		for _, hp := range hps {
			//fmt.Printf("%d. Adding hash/val %x: %x\n", i, hp.path, hp.hash)
			stTrie.insert(stTrie.root, hp.path, hp.hash[:], nil, newHashed)
		}
		// Verify the final trie hash
		if have := stTrie.Hash(); have != want {
			t.Fatalf("wrong hash, have %x want %x\n", have, want)
		}
	}
}

func TestStackRangeProof(t *testing.T) {
	trie, vals := randomTrie(4096)
	var entries []*kv
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	slices.SortFunc(entries, (*kv).cmp)
	proof := memorydb.New()
	entries = entries[1000 : len(entries)-1000] // We snip off 1000 entries on either side
	// Provide the proof for both first and last entry
	if err := trie.Prove(entries[0].k, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[len(entries)-1].k, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	testStackRangeProof(t, trie.Hash(), proof, entries)
}

func testStackRangeProof(
	t *testing.T, rootHash common.Hash,
	proof ethdb.KeyValueReader, entries []*kv) {

	var leftBorder = keybytesToHex(entries[0].k)
	var rightBorder = keybytesToHex(entries[len(entries)-1].k)
	writeFn := func(_ common.Hash, path []byte, hash common.Hash, blob []byte) {
		if bytes.HasPrefix(leftBorder, path) {
			fmt.Printf("path %x  tainted left (parent to %x)\n", path, leftBorder)
			return
		}
		if bytes.HasPrefix(rightBorder, path) {
			fmt.Printf("path %x  tainted right (parent to %x)\n", path, rightBorder)
			return
		}
		//fmt.Printf("Committing path %x\n", path)
	}

	// Use the proof initiate the stacktrie with the first entry
	stTrie, err := newStackTrieFromProof(rootHash, entries[0].k, proof, writeFn)
	if err != nil {
		t.Fatal(err)
	}
	// Feed in the standalone values
	for i := 1; i < len(entries); i++ {
		stTrie.Update(entries[i].k, common.CopyBytes(entries[i].v))
	}
	// For the right-hand-side, we need a list of hashes ot inject
	// Obtain the hashes
	hps, err := iterateProof(rootHash, entries[len(entries)-1].k, false, proof)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(hps)
	// Insert into stacktrie
	for _, hp := range hps {
		stTrie.insert(stTrie.root, hp.path, hp.hash[:], nil, newHashed)
	}
	have := stTrie.Hash()
	if have != rootHash {
		t.Fatalf("have %v want %v", have, rootHash)
	}
}

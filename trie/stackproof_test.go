package trie

import (
	"bytes"
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
		t.Logf("Start: %d\nPrev %x\n\nFirst%x\n", start, entries[start-1].k, entries[start].k)
		writeFn := wrapWriteFunction(entries[start].k, entries[len(entries)-1].k, func(path []byte, hash common.Hash, blob []byte) {
			rawdb.WriteTrieNode(haveSponge, common.Hash{}, path, hash, blob, "path")
		})
		stTrie, err := newStackTrieFromProof(trie.Hash(), entries[start].k, proof, writeFn)
		if err != nil {
			t.Fatal(err)
		}
		// Initiate a reference stacktrie without proof (filling manually)
		refTrie := NewStackTrie(nil)
		for i := 0; i <= start; i++ { // do prefill
			k, v := common.CopyBytes(entries[i].k), common.CopyBytes(entries[i].v)
			refTrie.Update(k, v)
		}
		// Determine the origin-border, and lop off the terminator
		hexStart := keybytesToHex(entries[start].k)[:2*len(entries[start].k)]

		w := func(path []byte, hash common.Hash, blob []byte) {
			// the refTrie _might_ have an unhashed sibling still not comitted, in case
			// the proof is between two elements. In that case, it should not be committed,
			// because the proof-initiated one will not have it (only hashed).
			//
			// It might even have a "sibling parent" still uncomitted
			//              1
			//         d                        e
			//  0 1 2 3 4 .. f                  0
			//  a b c d e .. n                  x <-- the one we're about to insert
			//
			// In this case, as soon as we submit 0x...1e0, 0x...0d will be hashed and comitted
			// by the reftrie (but not the proof-initalized one).
			if bytes.Compare(path, hexStart[:len(path)]) < 0 {
				t.Logf("Ignoring path %x", path)
				return
			}
			rawdb.WriteTrieNode(wantSponge, common.Hash{}, path, hash, blob, "path")
		}
		refTrie.writeFn = wrapWriteFunction(entries[start].k, entries[len(entries)-1].k, w)
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
			wantSponge.PrettyPrint(t)
			t.Logf("Have:")
			haveSponge.PrettyPrint(t)
			t.Fatalf("proof from %d: disk write sequence wrong:\nhave %x want %x\n", start, have, want)
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

	writeFn := wrapWriteFunction(entries[0].k, entries[len(entries)-1].k, func(path []byte, hash common.Hash, blob []byte) {})

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

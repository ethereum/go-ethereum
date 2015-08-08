// Copyright 2014 The go-ethereum Authors
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

package trie

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
	mrand "math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

type Db map[string][]byte

func (self Db) Get(k []byte) ([]byte, error) { return self[string(k)], nil }
func (self Db) Put(k, v []byte) error        { self[string(k)] = v; return nil }

// Used for testing
func NewEmpty() *Trie {
	return New(nil, make(Db))
}

func NewEmptySecure() *SecureTrie {
	return NewSecure(nil, make(Db))
}

func TestEmptyTrie(t *testing.T) {
	trie := NewEmpty()
	res := trie.Hash()
	exp := crypto.Sha3(common.Encode(""))
	if !bytes.Equal(res, exp) {
		t.Errorf("expected %x got %x", exp, res)
	}
}

func TestNull(t *testing.T) {
	trie := NewEmpty()

	key := make([]byte, 32)
	value := common.FromHex("0x823140710bf13990e4500136726d8b55")
	trie.Update(key, value)
	value = trie.Get(key)
}

func TestInsert(t *testing.T) {
	trie := NewEmpty()

	trie.UpdateString("doe", "reindeer")
	trie.UpdateString("dog", "puppy")
	trie.UpdateString("dogglesworth", "cat")

	exp := common.Hex2Bytes("8aad789dff2f538bca5d8ea56e8abe10f4c7ba3a5dea95fea4cd6e7c3a1168d3")
	root := trie.Hash()
	if !bytes.Equal(root, exp) {
		t.Errorf("exp %x got %x", exp, root)
	}

	trie = NewEmpty()
	trie.UpdateString("A", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	exp = common.Hex2Bytes("d23786fb4a010da3ce639d66d5e904a11dbc02746d1ce25029e53290cabf28ab")
	root = trie.Hash()
	if !bytes.Equal(root, exp) {
		t.Errorf("exp %x got %x", exp, root)
	}
}

func TestGet(t *testing.T) {
	trie := NewEmpty()

	trie.UpdateString("doe", "reindeer")
	trie.UpdateString("dog", "puppy")
	trie.UpdateString("dogglesworth", "cat")

	res := trie.GetString("dog")
	if !bytes.Equal(res, []byte("puppy")) {
		t.Errorf("expected puppy got %x", res)
	}

	unknown := trie.GetString("unknown")
	if unknown != nil {
		t.Errorf("expected nil got %x", unknown)
	}
}

func TestDelete(t *testing.T) {
	trie := NewEmpty()

	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
	}
	for _, val := range vals {
		if val.v != "" {
			trie.UpdateString(val.k, val.v)
		} else {
			trie.DeleteString(val.k)
		}
	}

	hash := trie.Hash()
	exp := common.Hex2Bytes("5991bb8c6514148a29db676a14ac506cd2cd5775ace63c30a4fe457715e9ac84")
	if !bytes.Equal(hash, exp) {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

func TestEmptyValues(t *testing.T) {
	trie := NewEmpty()

	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
	}
	for _, val := range vals {
		trie.UpdateString(val.k, val.v)
	}

	hash := trie.Hash()
	exp := common.Hex2Bytes("5991bb8c6514148a29db676a14ac506cd2cd5775ace63c30a4fe457715e9ac84")
	if !bytes.Equal(hash, exp) {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

func TestReplication(t *testing.T) {
	trie := NewEmpty()
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
		{"somethingveryoddindeedthis is", "myothernodedata"},
	}
	for _, val := range vals {
		trie.UpdateString(val.k, val.v)
	}
	trie.Commit()

	trie2 := New(trie.Root(), trie.cache.backend)
	if string(trie2.GetString("horse")) != "stallion" {
		t.Error("expected to have horse => stallion")
	}

	hash := trie2.Hash()
	exp := trie.Hash()
	if !bytes.Equal(hash, exp) {
		t.Errorf("root failure. expected %x got %x", exp, hash)
	}

}

func TestReset(t *testing.T) {
	trie := NewEmpty()
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
	}
	for _, val := range vals {
		trie.UpdateString(val.k, val.v)
	}
	trie.Commit()

	before := common.CopyBytes(trie.roothash)
	trie.UpdateString("should", "revert")
	trie.Hash()
	// Should have no effect
	trie.Hash()
	trie.Hash()
	// ###

	trie.Reset()
	after := common.CopyBytes(trie.roothash)

	if !bytes.Equal(before, after) {
		t.Errorf("expected roots to be equal. %x - %x", before, after)
	}
}

func TestParanoia(t *testing.T) {
	t.Skip()
	trie := NewEmpty()

	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
		{"somethingveryoddindeedthis is", "myothernodedata"},
	}
	for _, val := range vals {
		trie.UpdateString(val.k, val.v)
	}
	trie.Commit()

	ok, t2 := ParanoiaCheck(trie, trie.cache.backend)
	if !ok {
		t.Errorf("trie paranoia check failed %x %x", trie.roothash, t2.roothash)
	}
}

// Not an actual test
func TestOutput(t *testing.T) {
	t.Skip()

	base := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	trie := NewEmpty()
	for i := 0; i < 50; i++ {
		trie.UpdateString(fmt.Sprintf("%s%d", base, i), "valueeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	}
	fmt.Println("############################## FULL ################################")
	fmt.Println(trie.root)

	trie.Commit()
	fmt.Println("############################## SMALL ################################")
	trie2 := New(trie.roothash, trie.cache.backend)
	trie2.GetString(base + "20")
	fmt.Println(trie2.root)
}

func BenchmarkGets(b *testing.B) {
	trie := NewEmpty()
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
		{"somethingveryoddindeedthis is", "myothernodedata"},
	}
	for _, val := range vals {
		trie.UpdateString(val.k, val.v)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Get([]byte("horse"))
	}
}

func BenchmarkUpdate(b *testing.B) {
	trie := NewEmpty()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.UpdateString(fmt.Sprintf("aaaaaaaaa%d", i), "value")
	}
	trie.Hash()
}

type kv struct {
	k, v []byte
	t    bool
}

func TestLargeData(t *testing.T) {
	trie := NewEmpty()
	vals := make(map[string]*kv)

	for i := byte(0); i < 255; i++ {
		value := &kv{common.LeftPadBytes([]byte{i}, 32), []byte{i}, false}
		value2 := &kv{common.LeftPadBytes([]byte{10, i}, 32), []byte{i}, false}
		trie.Update(value.k, value.v)
		trie.Update(value2.k, value2.v)
		vals[string(value.k)] = value
		vals[string(value2.k)] = value2
	}

	it := trie.Iterator()
	for it.Next() {
		vals[string(it.Key)].t = true
	}

	var untouched []*kv
	for _, value := range vals {
		if !value.t {
			untouched = append(untouched, value)
		}
	}

	if len(untouched) > 0 {
		t.Errorf("Missed %d nodes", len(untouched))
		for _, value := range untouched {
			t.Error(value)
		}
	}
}

func TestSecureDelete(t *testing.T) {
	trie := NewEmptySecure()

	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
	}
	for _, val := range vals {
		if val.v != "" {
			trie.UpdateString(val.k, val.v)
		} else {
			trie.DeleteString(val.k)
		}
	}

	hash := trie.Hash()
	exp := common.Hex2Bytes("29b235a58c3c25ab83010c327d5932bcf05324b7d6b1185e650798034783ca9d")
	if !bytes.Equal(hash, exp) {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

//------------------------------------------------------------------------------------
// proof tests (and helpers)

func randBytes(n int) []byte {
	r := make([]byte, n)
	crand.Read(r)
	return r
}

func randInt(m int) int {
	return int(mrand.Int31n(int32(m)))
}

// genuinely new byte
func newByte(c byte) byte {
	c2 := byte(randInt(255))
	if c == c2 {
		return newByte(c)
	}
	return c2
}

// genuinely change a byte
func mutateBytes(b []byte) []byte {
	b2 := make([]byte, len(b))
	copy(b2, b)
	b = b2

	// Mutate a single byte
	r := randInt(len(b))
	c := b[r]
	if c == byte(128) || c == byte(32) { // indeterminacy in rlp?
		return mutateBytes(b)
	}
	d := newByte(c)
	b[r] = d
	return b
}

func makeTrieForProofs(n int) (*Trie, map[string]*kv) {
	trie := NewEmpty()
	vals := make(map[string]*kv)

	for i := byte(0); i < 100; i++ {
		value := &kv{common.LeftPadBytes([]byte{i}, 32), []byte{i}, false}
		value2 := &kv{common.LeftPadBytes([]byte{i + 10}, 32), []byte{i}, false}
		trie.Update(value.k, value.v)
		trie.Update(value2.k, value2.v)
		vals[string(value.k)] = value
		vals[string(value2.k)] = value2
	}

	for i := 0; i < n; i++ {
		value := &kv{randBytes(32), randBytes(20), false}
		trie.Update(value.k, value.v)
		vals[string(value.k)] = value
	}

	return trie, vals
}

func TestProof(t *testing.T) {
	trie, vals := makeTrieForProofs(100)
	// prove things are in the tree
	for _, kvv := range vals {
		proof := trie.Prove(kvv.k)
		if proof == nil {
			t.Fatalf("Failed to find key %X while constructing proof", kvv.k)
		}
		proven := proof.Verify(kvv.k, kvv.v, trie.Hash())
		if !proven {
			t.Fatalf("failed to prove key %X", kvv.k)
		}
	}
}

func testBadProof(t *testing.T, trie *Trie, kvv *kv, originalProofBytes []byte) {
	proofBytes := mutateBytes(originalProofBytes)

	proof2 := new(TrieProof)
	if err := rlp.Decode(bytes.NewBuffer(proofBytes), proof2); err == nil {
		proven := proof2.Verify(kvv.k, kvv.v, trie.Hash())
		if proven {
			t.Fatalf("expected proof to fail for %X", kvv.k)
		}
	} else {
		// if we failed to decode, we mutated the rlp too badly.
		// try again
		testBadProof(t, trie, kvv, originalProofBytes)
	}
}

func TestBadProof(t *testing.T) {
	trie, vals := makeTrieForProofs(100)
	for _, kvv := range vals {
		proof := trie.Prove(kvv.k)
		proven := proof.Verify(kvv.k, kvv.v, trie.Hash())
		if !proven {
			t.Fatalf("expected proof not to fail for %X", kvv.k)
		}
		proofBytes := common.Encode(proof)
		testBadProof(t, trie, kvv, proofBytes)
	}
}

func compareProofs(proof, proof2 *TrieProof) error {
	if !bytes.Equal(proof.Key, proof2.Key) {
		return fmt.Errorf("codec error: keys are not same. got %X, expected %X\n", proof2.Key, proof.Key)
	}
	if !bytes.Equal(proof.Value, proof2.Value) {
		return fmt.Errorf("codec error: values are not same. got %X, expected %X\n", proof2.Value, proof.Value)
	}
	if !bytes.Equal(proof.RootHash, proof2.RootHash) {
		return fmt.Errorf("codec error: root hashes are not same. got %X, expected %X\n", proof2.RootHash, proof.RootHash)
	}
	if len(proof.InnerNodes) != len(proof2.InnerNodes) {
		return fmt.Errorf("codec error: wrong number of inner nodes. got %d, expected %d\n", len(proof2.InnerNodes), len(proof.InnerNodes))
	}

	for i, in := range proof.InnerNodes {
		in2 := proof2.InnerNodes[i]
		if !bytes.Equal(in.Key, in2.Key) {
			return fmt.Errorf("codec error: inner keys for node %d are not same. got %X, expected %X\n", i, in2.Key, in.Key)
		}
		for j, n := range in.Nodes {
			if !bytes.Equal(n, in2.Nodes[j]) {
				return fmt.Errorf("codec error: inner nodes %d (%d) are not same. got %X, expected %X\n", i, j, in2.Nodes[j], n)
			}
		}
	}
	return nil
}

func TestProofCodec(t *testing.T) {
	trie, vals := makeTrieForProofs(100)
	for _, kvv := range vals {
		proof := trie.Prove(kvv.k)
		proofBytes := common.Encode(proof)

		proof2 := new(TrieProof)
		if err := rlp.Decode(bytes.NewBuffer(proofBytes), proof2); err != nil {
			t.Fatalf("error decoding proof bytes: %v", err)
		}

		if err := compareProofs(proof, proof2); err != nil {
			t.Fatal(err)
		}

		proven := proof2.Verify(kvv.k, kvv.v, trie.Hash())
		if !proven {
			t.Fatalf("failed to prove key %X", kvv.k)
		}
	}
}

func BenchmarkProof(b *testing.B) {
	trie, vals := makeTrieForProofs(100)

	proofs := make([]*TrieProof, len(vals))

	i := 0
	for _, kvv := range vals {
		proof := trie.Prove(kvv.k)
		if proof == nil {
			b.Fatalf("Failed to find key %X while constructing proof", kvv.k)
		}
		proofs[i] = proof
		i += 1
	}

	N := len(vals)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		im := i % N
		v := proofs[im].Verify(proofs[im].Key, proofs[im].Value, proofs[im].RootHash)
		if !v {
			b.Fatalf("failed to prove key %X", proofs[im].Key)
		}
	}
}

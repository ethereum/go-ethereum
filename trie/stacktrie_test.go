package trie

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

func TestSizeBug(t *testing.T) {
	st := NewStackTrie(nil)
	nt, _ := New(common.Hash{}, NewDatabase(memorydb.New()))

	leaf := common.FromHex("290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563")
	value := common.FromHex("94cf40d0d2b44f2b66e07cace1372ca42b73cf21a3")

	nt.TryUpdate(leaf, value)
	st.TryUpdate(leaf, value)

	if nt.Hash() != st.Hash() {
		t.Fatalf("error %x != %x", st.Hash(), nt.Hash())
	}
}

func TestEmptyBug(t *testing.T) {
	st := NewStackTrie(nil)
	nt, _ := New(common.Hash{}, NewDatabase(memorydb.New()))

	//leaf := common.FromHex("290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563")
	//value := common.FromHex("94cf40d0d2b44f2b66e07cace1372ca42b73cf21a3")
	kvs := []struct {
		K string
		V string
	}{
		{K: "405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace", V: "9496f4ec2bf9dab484cac6be589e8417d84781be08"},
		{K: "40edb63a35fcf86c08022722aa3287cdd36440d671b4918131b2514795fefa9c", V: "01"},
		{K: "b10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6", V: "947a30f7736e48d6599356464ba4c150d8da0302ff"},
		{K: "c2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b", V: "02"},
	}

	for _, kv := range kvs {
		nt.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
		st.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
	}

	if nt.Hash() != st.Hash() {
		t.Fatalf("error %x != %x", st.Hash(), nt.Hash())
	}
}

func TestValLength56(t *testing.T) {
	st := NewStackTrie(nil)
	nt, _ := New(common.Hash{}, NewDatabase(memorydb.New()))

	//leaf := common.FromHex("290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563")
	//value := common.FromHex("94cf40d0d2b44f2b66e07cace1372ca42b73cf21a3")
	kvs := []struct {
		K string
		V string
	}{
		{K: "405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace", V: "1111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111"},
	}

	for _, kv := range kvs {
		nt.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
		st.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
	}

	if nt.Hash() != st.Hash() {
		t.Fatalf("error %x != %x", st.Hash(), nt.Hash())
	}
}

// TestUpdateSmallNodes tests a case where the leaves are small (both key and value),
// which causes a lot of node-within-node. This case was found via fuzzing.
func TestUpdateSmallNodes(t *testing.T) {
	st := NewStackTrie(nil)
	nt, _ := New(common.Hash{}, NewDatabase(memorydb.New()))
	kvs := []struct {
		K string
		V string
	}{
		{"63303030", "3041"}, // stacktrie.Update
		{"65", "3000"},       // stacktrie.Update
	}
	for _, kv := range kvs {
		nt.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
		st.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
	}
	if nt.Hash() != st.Hash() {
		t.Fatalf("error %x != %x", st.Hash(), nt.Hash())
	}
}

// TestUpdateVariableKeys contains a case which stacktrie fails: when keys of different
// sizes are used, and the second one has the same prefix as the first, then the
// stacktrie fails, since it's unable to 'expand' on an already added leaf.
// For all practical purposes, this is fine, since keys are fixed-size length
// in account and storage tries.
//
// The test is marked as 'skipped', and exists just to have the behaviour documented.
// This case was found via fuzzing.
func TestUpdateVariableKeys(t *testing.T) {
	t.SkipNow()
	st := NewStackTrie(nil)
	nt, _ := New(common.Hash{}, NewDatabase(memorydb.New()))
	kvs := []struct {
		K string
		V string
	}{
		{"0x33303534636532393561313031676174", "303030"},
		{"0x3330353463653239356131303167617430", "313131"},
	}
	for _, kv := range kvs {
		nt.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
		st.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
	}
	if nt.Hash() != st.Hash() {
		t.Fatalf("error %x != %x", st.Hash(), nt.Hash())
	}
}

// TestStacktrieNotModifyValues checks that inserting blobs of data into the
// stacktrie does not mutate the blobs
func TestStacktrieNotModifyValues(t *testing.T) {
	st := NewStackTrie(nil)
	{ // Test a very small trie
		// Give it the value as a slice with large backing alloc,
		// so if the stacktrie tries to append, it won't have to realloc
		value := make([]byte, 1, 100)
		value[0] = 0x2
		want := common.CopyBytes(value)
		st.TryUpdate([]byte{0x01}, value)
		st.Hash()
		if have := value; !bytes.Equal(have, want) {
			t.Fatalf("tiny trie: have %#x want %#x", have, want)
		}
		st = NewStackTrie(nil)
	}
	// Test with a larger trie
	keyB := big.NewInt(1)
	keyDelta := big.NewInt(1)
	var vals [][]byte
	getValue := func(i int) []byte {
		if i%2 == 0 { // large
			return crypto.Keccak256(big.NewInt(int64(i)).Bytes())
		} else { //small
			return big.NewInt(int64(i)).Bytes()
		}
	}
	for i := 0; i < 1000; i++ {
		key := common.BigToHash(keyB)
		value := getValue(i)
		st.TryUpdate(key.Bytes(), value)
		vals = append(vals, value)
		keyB = keyB.Add(keyB, keyDelta)
		keyDelta.Add(keyDelta, common.Big1)
	}
	st.Hash()
	for i := 0; i < 1000; i++ {
		want := getValue(i)

		have := vals[i]
		if !bytes.Equal(have, want) {
			t.Fatalf("item %d, have %#x want %#x", i, have, want)
		}

	}
}

// TestStacktrieSerialization tests that the stacktrie works well if we
// serialize/unserialize it a lot
func TestStacktrieSerialization(t *testing.T) {
	var (
		st       = NewStackTrie(nil)
		nt, _    = New(common.Hash{}, NewDatabase(memorydb.New()))
		keyB     = big.NewInt(1)
		keyDelta = big.NewInt(1)
		vals     [][]byte
		keys     [][]byte
	)
	getValue := func(i int) []byte {
		if i%2 == 0 { // large
			return crypto.Keccak256(big.NewInt(int64(i)).Bytes())
		} else { //small
			return big.NewInt(int64(i)).Bytes()
		}
	}
	for i := 0; i < 10; i++ {
		vals = append(vals, getValue(i))
		keys = append(keys, common.BigToHash(keyB).Bytes())
		keyB = keyB.Add(keyB, keyDelta)
		keyDelta.Add(keyDelta, common.Big1)
	}
	for i, k := range keys {
		nt.TryUpdate(k, common.CopyBytes(vals[i]))
	}

	for i, k := range keys {
		blob, err := st.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		newSt, err := NewFromBinary(blob, nil)
		if err != nil {
			t.Fatal(err)
		}
		st = newSt
		st.TryUpdate(k, common.CopyBytes(vals[i]))
	}
	if have, want := st.Hash(), nt.Hash(); have != want {
		t.Fatalf("have %#x want %#x", have, want)
	}
}

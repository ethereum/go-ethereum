package ptrie

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/ethutil"
)

func TestInsert(t *testing.T) {
	trie := NewEmpty()

	trie.UpdateString("doe", "reindeer")
	trie.UpdateString("dog", "puppy")
	trie.UpdateString("dogglesworth", "cat")

	exp := ethutil.Hex2Bytes("8aad789dff2f538bca5d8ea56e8abe10f4c7ba3a5dea95fea4cd6e7c3a1168d3")
	root := trie.Hash()
	if !bytes.Equal(root, exp) {
		t.Errorf("exp %x got %x", exp, root)
	}

	trie = NewEmpty()
	trie.UpdateString("A", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	exp = ethutil.Hex2Bytes("d23786fb4a010da3ce639d66d5e904a11dbc02746d1ce25029e53290cabf28ab")
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
		trie.UpdateString(val.k, val.v)
	}

	hash := trie.Hash()
	exp := ethutil.Hex2Bytes("5991bb8c6514148a29db676a14ac506cd2cd5775ace63c30a4fe457715e9ac84")
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
	trie.Hash()

	trie2 := New(trie.roothash, trie.backend)
	if string(trie2.GetString("horse")) != "stallion" {
		t.Error("expected to have harse => stallion")
	}

	hash := trie2.Hash()
	exp := trie.Hash()
	if !bytes.Equal(hash, exp) {
		t.Errorf("root failure. expected %x got %x", exp, hash)
	}

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
		trie.UpdateString(fmt.Sprintf("aaaaaaaaaaaaaaa%d", j), "value")
	}
	trie.Hash()
}

package trie

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

	trie2 := New(trie.roothash, trie.cache.backend)
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

package trie

import "testing"

func TestIterator(t *testing.T) {
	trie := NewEmpty()
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"dog", "puppy"},
		{"somethingveryoddindeedthis is", "myothernodedata"},
	}
	v := make(map[string]bool)
	for _, val := range vals {
		v[val.k] = false
		trie.UpdateString(val.k, val.v)
	}
	trie.Commit()

	it := trie.Iterator()
	for it.Next() {
		v[string(it.Key)] = true
	}

	for k, found := range v {
		if !found {
			t.Error("iterator didn't find", k)
		}
	}
}

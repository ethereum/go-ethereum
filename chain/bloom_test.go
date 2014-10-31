package chain

import "testing"

func TestBloomFilter(t *testing.T) {
	bf := NewBloomFilter(nil)

	a := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	bf.Set(a)

	b := []byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19}

	if bf.Search(a) == false {
		t.Error("Expected 'a' to yield true using a bloom filter")
	}

	if bf.Search(b) {
		t.Error("Expected 'b' not to field trie using a bloom filter")
	}
}

package bloom

import (
	"testing"
	"time"
)

type hashable struct {
	b []byte
}

func (h hashable) BlockSize() int {
	return len(h.b)
}

func (h hashable) Hash() []byte {
	return h.b
}

func (h hashable) Sum([]byte) []byte {
	return h.b
}

func (h hashable) Sum64() uint64 {
	return 1
}

func (h hashable) Write([]byte) (int, error) {
	return 0, nil
}

func (h hashable) Reset() {}

func (h hashable) Size() int {
	return len(h.b)
}

func TestBloom(t *testing.T) {
	bloom := NewExpiringBloom(2, 10, 10, 10*time.Millisecond)

	testKey := hashable{[]byte{0x01}}
	bloom.Put(testKey)
	if !bloom.Contain(testKey) {
		t.Fatal()
	}
	time.Sleep(10 * time.Millisecond)
	if !bloom.Contain(testKey) {
		t.Fatal()
	}
	time.Sleep(10 * time.Millisecond)
	if bloom.Contain(testKey) {
		t.Fatal()
	}
}

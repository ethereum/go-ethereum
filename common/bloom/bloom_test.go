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
	bloom, _ := NewExpiringBloom(3, 1024, 10*time.Millisecond)

	testKey := hashable{[]byte{0x01}}
	bloom.Put(testKey)
	if !bloom.Contain(testKey) {
		t.Fatal()
	}
	time.Sleep(11 * time.Millisecond)
	if !bloom.Contain(testKey) {
		t.Fatal()
	}
	time.Sleep(11 * time.Millisecond)
	if !bloom.Contain(testKey) {
		t.Fatal()
	}
	time.Sleep(11 * time.Millisecond)
	if bloom.Contain(testKey) {
		t.Fatal()
	}
}

func TestBloom2(t *testing.T) {
	bloom, _ := NewExpiringBloom(3, 1024, 10*time.Second)

	testKey := hashable{[]byte{0x01}}
	// Put key in bloom 0
	bloom.Put(testKey)
	if !bloom.Contain(testKey) {
		t.Fatal()
	}
	// Override bloom 1
	bloom.tick()
	if !bloom.Contain(testKey) {
		t.Fatal()
	}
	// Override bloom 2
	bloom.tick()
	if !bloom.Contain(testKey) {
		t.Fatal()
	}
	// Override bloom 0
	bloom.tick()
	if bloom.Contain(testKey) {
		t.Fatal()
	}
}

func BenchmarkPut(b *testing.B) {
	bloom, _ := NewExpiringBloom(2, 1024, 10*time.Second)

	testKey := hashable{[]byte{0x01}}
	for i := 0; i < b.N; i++ {
		bloom.Put(testKey)
	}
}

func BenchmarkTick(b *testing.B) {
	bloom, _ := NewExpiringBloom(2, 1024, 10*time.Second)

	for i := 0; i < b.N; i++ {
		bloom.tick()
	}
}

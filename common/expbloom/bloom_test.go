package bloom

import (
	"encoding/binary"
	"testing"
	"time"
)

type hashable []byte

func (h hashable) Write(p []byte) (n int, err error) { panic("not implemented") }
func (h hashable) Sum(b []byte) []byte               { panic("not implemented") }
func (h hashable) Reset()                            { panic("not implemented") }
func (h hashable) BlockSize() int                    { panic("not implemented") }
func (h hashable) Size() int                         { return 8 }
func (h hashable) Sum64() uint64 {
	hash := make([]byte, 8)
	copy(hash, h)
	return binary.BigEndian.Uint64(hash[0:8])
}

func TestBloom(t *testing.T) {
	bloom, _ := NewExpiringBloom(3, 1024, 10*time.Millisecond)

	testKey := hashable([]byte{0x01})
	bloom.Add(testKey)
	if !bloom.Contains(testKey) {
		t.Fatal()
	}
	time.Sleep(11 * time.Millisecond)
	if !bloom.Contains(testKey) {
		t.Fatal()
	}
	time.Sleep(11 * time.Millisecond)
	if !bloom.Contains(testKey) {
		t.Fatal()
	}
	time.Sleep(11 * time.Millisecond)
	if bloom.Contains(testKey) {
		t.Fatal()
	}
}

func TestBloom2(t *testing.T) {
	bloom, _ := NewExpiringBloom(3, 1024, 10*time.Second)

	testKey := hashable([]byte{0x01})
	// Add key in bloom 0
	bloom.Add(testKey)
	if !bloom.Contains(testKey) {
		t.Fatal()
	}
	// Override bloom 1
	bloom.tick()
	if !bloom.Contains(testKey) {
		t.Fatal()
	}
	// Override bloom 2
	bloom.tick()
	if !bloom.Contains(testKey) {
		t.Fatal()
	}
	// Override bloom 0
	bloom.tick()
	if bloom.Contains(testKey) {
		t.Fatal()
	}
}

func BenchmarkAdd(b *testing.B) {
	bloom, _ := NewExpiringBloom(2, 1024, 10*time.Second)

	testKey := hashable([]byte{0x01})
	for i := 0; i < b.N; i++ {
		bloom.Add(testKey)
	}
}

func BenchmarkTick(b *testing.B) {
	bloom, _ := NewExpiringBloom(2, 1024, 10*time.Second)

	for i := 0; i < b.N; i++ {
		bloom.tick()
	}
}

package keccak

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	"golang.org/x/crypto/sha3"
)

func TestSum256Empty(t *testing.T) {
	got := Sum256(nil)
	// Known Keccak-256 of empty string.
	want, _ := hex.DecodeString("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")
	if !bytes.Equal(got[:], want) {
		t.Fatalf("Sum256(nil) = %x, want %x", got, want)
	}
}

func TestSum256Hello(t *testing.T) {
	got := Sum256([]byte("hello"))
	want, _ := hex.DecodeString("1c8aff950685c2ed4bc3174f3472287b56d9517b9c948127319a09a7a36deac8")
	if !bytes.Equal(got[:], want) {
		t.Fatalf("Sum256(hello) = %x, want %x", got, want)
	}
}

func TestSum256LargeData(t *testing.T) {
	// Test with data larger than one block (rate=136 bytes).
	data := make([]byte, 500)
	for i := range data {
		data[i] = byte(i)
	}
	got := Sum256(data)
	// Verify against streaming Hasher.
	var h Hasher
	h.Write(data)
	want := h.Sum256()
	if got != want {
		t.Fatalf("Sum256 vs Hasher mismatch: %x vs %x", got, want)
	}
}

func TestHasherStreaming(t *testing.T) {
	data := []byte("hello world, this is a longer test string for streaming keccak")
	// All at once.
	want := Sum256(data)
	// Byte by byte.
	var h Hasher
	for _, b := range data {
		h.Write([]byte{b})
	}
	got := h.Sum256()
	if got != want {
		t.Fatalf("streaming byte-by-byte: %x vs %x", got, want)
	}
}

func TestHasherMultiBlock(t *testing.T) {
	// Test with exactly 2 blocks + partial.
	data := make([]byte, rate*2+50)
	for i := range data {
		data[i] = byte(i * 7)
	}
	want := Sum256(data)
	// Write in chunks of 37 (not aligned to rate).
	var h Hasher
	for i := 0; i < len(data); i += 37 {
		end := i + 37
		if end > len(data) {
			end = len(data)
		}
		h.Write(data[i:end])
	}
	got := h.Sum256()
	if got != want {
		t.Fatalf("multi-block streaming: %x vs %x", got, want)
	}
}

func TestReadMatchesSum256(t *testing.T) {
	// Read of 32 bytes should produce the same result as Sum256.
	data := []byte("hello")
	var h Hasher
	h.Write(data)
	var got [32]byte
	h.Read(got[:])
	want := Sum256(data)
	if got != want {
		t.Fatalf("Read(32) = %x, want %x", got, want)
	}
}

func TestReadMatchesXCrypto(t *testing.T) {
	// Compare Read output against x/crypto/sha3 for various lengths.
	for _, readLen := range []int{32, 64, 136, 200, 500} {
		data := []byte("test data for read comparison")
		ref := sha3.NewLegacyKeccak256()
		ref.Write(data)
		want := make([]byte, readLen)
		ref.(KeccakState).Read(want)

		var h Hasher
		h.Write(data)
		got := make([]byte, readLen)
		h.Read(got)
		if !bytes.Equal(got, want) {
			t.Fatalf("Read(%d) mismatch:\ngot:  %x\nwant: %x", readLen, got, want)
		}
	}
}

func TestReadMultipleCalls(t *testing.T) {
	// Multiple Read calls should produce the same output as one large Read.
	data := []byte("streaming read test")

	// One large read.
	var h1 Hasher
	h1.Write(data)
	all := make([]byte, 300)
	h1.Read(all)

	// Multiple small reads.
	var h2 Hasher
	h2.Write(data)
	var parts []byte
	for i := 0; i < 300; {
		chunk := 37
		if i+chunk > 300 {
			chunk = 300 - i
		}
		buf := make([]byte, chunk)
		h2.Read(buf)
		parts = append(parts, buf...)
		i += chunk
	}
	if !bytes.Equal(all, parts) {
		t.Fatalf("multi-read mismatch:\ngot:  %x\nwant: %x", parts, all)
	}
}

func TestReadEmpty(t *testing.T) {
	// Read from hasher with no data written.
	ref := sha3.NewLegacyKeccak256()
	want := make([]byte, 32)
	ref.(KeccakState).Read(want)

	var h Hasher
	got := make([]byte, 32)
	h.Read(got)
	if !bytes.Equal(got, want) {
		t.Fatalf("Read empty mismatch:\ngot:  %x\nwant: %x", got, want)
	}
}

func TestReadAfterReset(t *testing.T) {
	var h Hasher
	h.Write([]byte("first"))
	h.Read(make([]byte, 32))

	// Reset should allow Write again.
	h.Reset()
	h.Write([]byte("second"))
	got := make([]byte, 32)
	h.Read(got)

	want := Sum256([]byte("second"))
	if !bytes.Equal(got, want[:]) {
		t.Fatalf("Read after Reset mismatch:\ngot:  %x\nwant: %x", got, want)
	}
}

func TestWriteAfterReadPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on Write after Read")
		}
	}()
	var h Hasher
	h.Write([]byte("data"))
	h.Read(make([]byte, 32))
	h.Write([]byte("more")) // should panic
}

func FuzzSum256(f *testing.F) {
	f.Add([]byte(nil))
	f.Add([]byte("hello"))
	f.Add([]byte("hello world, this is a longer test string for streaming keccak"))
	f.Add(make([]byte, rate))
	f.Add(make([]byte, rate+1))
	f.Add(make([]byte, rate*3+50))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Reference: x/crypto NewLegacyKeccak256.
		ref := sha3.NewLegacyKeccak256()
		ref.Write(data)
		want := ref.Sum(nil)

		// Test Sum256.
		got := Sum256(data)
		if !bytes.Equal(got[:], want) {
			t.Fatalf("Sum256 mismatch for len=%d\ngot:  %x\nwant: %x", len(data), got, want)
		}

		// Test streaming Hasher (write all at once).
		var h Hasher
		h.Write(data)
		gotH := h.Sum256()
		if !bytes.Equal(gotH[:], want) {
			t.Fatalf("Hasher mismatch for len=%d\ngot:  %x\nwant: %x", len(data), gotH, want)
		}

		// Test streaming Hasher (byte-by-byte).
		h.Reset()
		for _, b := range data {
			h.Write([]byte{b})
		}
		gotS := h.Sum256()
		if !bytes.Equal(gotS[:], want) {
			t.Fatalf("Hasher byte-by-byte mismatch for len=%d\ngot:  %x\nwant: %x", len(data), gotS, want)
		}

		// Test Read (32 bytes) matches Sum256.
		h.Reset()
		h.Write(data)
		gotRead := make([]byte, 32)
		h.Read(gotRead)
		if !bytes.Equal(gotRead, want) {
			t.Fatalf("Read(32) mismatch for len=%d\ngot:  %x\nwant: %x", len(data), gotRead, want)
		}

		// Test Read (extended output) matches x/crypto.
		ref.Reset()
		ref.Write(data)
		wantExt := make([]byte, 200)
		ref.(KeccakState).Read(wantExt)

		h.Reset()
		h.Write(data)
		gotExt := make([]byte, 200)
		h.Read(gotExt)
		if !bytes.Equal(gotExt, wantExt) {
			t.Fatalf("Read(200) mismatch for len=%d\ngot:  %x\nwant: %x", len(data), gotExt, wantExt)
		}
	})
}

func BenchmarkSum256_500K(b *testing.B) {
	data := make([]byte, 500*1024)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		Sum256(data)
	}
}

// Comparison benchmarks: faster_keccak vs golang.org/x/crypto/sha3.
var benchSizes = []int{32, 128, 256, 1024, 4096, 500 * 1024}

func benchName(size int) string {
	switch {
	case size >= 1024:
		return fmt.Sprintf("%dK", size/1024)
	default:
		return fmt.Sprintf("%dB", size)
	}
}

func BenchmarkFasterKeccak(b *testing.B) {
	for _, size := range benchSizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i)
		}
		b.Run(benchName(size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ReportAllocs()
			for b.Loop() {
				Sum256(data)
			}
		})
	}
}

func BenchmarkXCrypto(b *testing.B) {
	for _, size := range benchSizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i)
		}
		b.Run(benchName(size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ReportAllocs()
			h := sha3.NewLegacyKeccak256()
			for b.Loop() {
				h.Reset()
				h.Write(data)
				h.Sum(nil)
			}
		})
	}
}

func BenchmarkFasterKeccakHasher(b *testing.B) {
	for _, size := range benchSizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i)
		}
		b.Run(benchName(size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ReportAllocs()
			var h Hasher
			for b.Loop() {
				h.Reset()
				h.Write(data)
				h.Sum256()
			}
		})
	}
}

// BenchmarkKeccakStreaming_Sha3 benchmarks the standard sha3 streaming hasher (Reset+Write+Read).
func BenchmarkKeccakStreaming_Sha3(b *testing.B) {
	data := make([]byte, 32)
	for i := range data {
		data[i] = byte(i)
	}
	h := sha3.NewLegacyKeccak256().(KeccakState)
	var buf [32]byte
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		h.Reset()
		h.Write(data)
		h.Read(buf[:])
	}
}
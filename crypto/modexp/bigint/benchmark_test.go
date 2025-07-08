package bigint_test

import (
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/crypto/modexp/bigint"
)

// generateBytes generates random bytes of given length
func generateBytes(b *testing.B, length int) []byte {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		b.Fatal(err)
	}
	// Ensure odd modulus for better performance
	if length > 0 {
		bytes[length-1] |= 0x01
	}
	return bytes
}

// BenchmarkModExp runs benchmarks for different input sizes
func BenchmarkModExp(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"32", 32},
		{"64", 64},
		{"128", 128},
		{"256", 256},
		{"512", 512},
		{"1024", 1024},
		{"2048", 2048},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			base := generateBytes(b, size.size)
			exp := generateBytes(b, size.size)
			mod := generateBytes(b, size.size)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = bigint.ModExp(base, exp, mod)
			}
		})
	}
}

// BenchmarkModExpParallel tests parallel performance
func BenchmarkModExpParallel(b *testing.B) {
	sizes := []int{128, 256, 512}

	for _, size := range sizes {
		base := generateBytes(b, size)
		exp := generateBytes(b, size)
		mod := generateBytes(b, size)

		b.Run(string(rune(size))+"bytes", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, _ = bigint.ModExp(base, exp, mod)
				}
			})
		})
	}
}

// BenchmarkModExpEdgeCases tests special cases
func BenchmarkModExpEdgeCases(b *testing.B) {
	cases := []struct {
		name string
		base []byte
		exp  []byte
		mod  []byte
	}{
		{
			name: "base_is_one",
			base: []byte{0x01},
			exp:  []byte{0xFF, 0xFF, 0xFF, 0xFF},
			mod:  []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			name: "zero_exponent",
			base: []byte{0xFF, 0xFF, 0xFF, 0xFF},
			exp:  []byte{0x00},
			mod:  []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			name: "small_exponent_rsa",
			base: generateBytes(b, 256),
			exp:  []byte{0x01, 0x00, 0x01}, // 65537
			mod:  generateBytes(b, 256),
		},
		{
			name: "empty_base",
			base: []byte{},
			exp:  []byte{0xFF},
			mod:  []byte{0xFF, 0xFF},
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = bigint.ModExp(tc.base, tc.exp, tc.mod)
			}
		})
	}
}

// BenchmarkModExpAllocs measures allocations
func BenchmarkModExpAllocs(b *testing.B) {
	sizes := []int{32, 128, 256, 1024}

	for _, size := range sizes {
		base := generateBytes(b, size)
		exp := generateBytes(b, size)
		mod := generateBytes(b, size)

		b.Run(string(rune(size))+"bytes", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = bigint.ModExp(base, exp, mod)
			}
		})
	}
}
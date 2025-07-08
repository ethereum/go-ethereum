package modexp_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto/modexp/bigint"
	gmpcwrapper "github.com/ethereum/go-ethereum/crypto/modexp/gmp/cwrapper"
	gmpgeneric "github.com/ethereum/go-ethereum/crypto/modexp/gmp/generic"
)

// generateWorstCase generates a byte array with all bits set to 1
func generateWorstCase(size int) []byte {
	result := make([]byte, size)
	for i := range result {
		result[i] = 0xFF
	}
	return result
}

// BenchmarkComprehensive tests with specific bit sizes and worst-case scenarios
func BenchmarkComprehensive(b *testing.B) {
	// Test cases with specific bit sizes
	testCases := []struct {
		name     string
		bits     int
		bytes    int
	}{
		{"1bit", 1, 1},      // 1 bit
		{"8bit", 8, 1},      // 1 byte
		{"16bit", 16, 2},    // 2 bytes
		{"32bit", 32, 4},    // 4 bytes
		{"64bit", 64, 8},    // 8 bytes
		{"128bit", 128, 16}, // 16 bytes
		{"256bit", 256, 32}, // 32 bytes
		{"512bit", 512, 64}, // 64 bytes
		{"1024bit", 1024, 128}, // 128 bytes
		{"2048bit", 2048, 256}, // 256 bytes
		{"4096bit", 4096, 512}, // 512 bytes
		{"8192bit", 8192, 1024}, // 1024 bytes
	}

	implementations := []struct {
		name string
		fn   func([]byte, []byte, []byte) ([]byte, error)
	}{
		{"BigInt", bigint.ModExp},
		{"GMPGeneric", gmpgeneric.ModExp},
		{"GMPCWrapper", gmpcwrapper.ModExp},
	}

	for _, tc := range testCases {
		// Generate worst-case inputs (all bits set)
		base := generateWorstCase(tc.bytes)
		exp := generateWorstCase(tc.bytes)
		mod := generateWorstCase(tc.bytes)
		
		// For 1-bit case, use specific values
		if tc.bits == 1 {
			base = []byte{0x01}
			exp = []byte{0x01}
			mod = []byte{0x03} // Must be > 1 for valid modulo
		}

		for _, impl := range implementations {
			b.Run(tc.name+"/"+impl.name, func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = impl.fn(base, exp, mod)
				}
			})
		}
	}
}

// BenchmarkWorstCaseOnly focuses on worst-case scenarios with maximum values
func BenchmarkWorstCaseOnly(b *testing.B) {
	sizes := []struct {
		name  string
		bytes int
	}{
		{"8bit", 1},
		{"16bit", 2},
		{"32bit", 4},
		{"64bit", 8},
		{"128bit", 16},
		{"256bit", 32},
		{"512bit", 64},
		{"1024bit", 128},
		{"2048bit", 256},
		{"4096bit", 512},
		{"8192bit", 1024},
	}

	implementations := []struct {
		name string
		fn   func([]byte, []byte, []byte) ([]byte, error)
	}{
		{"BigInt", bigint.ModExp},
		{"GMPGeneric", gmpgeneric.ModExp},
		{"GMPCWrapper", gmpcwrapper.ModExp},
	}

	b.Run("AllFF", func(b *testing.B) {
		for _, size := range sizes {
			base := generateWorstCase(size.bytes)
			exp := generateWorstCase(size.bytes)
			mod := generateWorstCase(size.bytes)

			for _, impl := range implementations {
				b.Run(size.name+"/"+impl.name, func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, _ = impl.fn(base, exp, mod)
					}
				})
			}
		}
	})
}

// BenchmarkMemoryProfile shows memory usage for each size
func BenchmarkMemoryProfile(b *testing.B) {
	sizes := []struct {
		name  string
		bytes int
	}{
		{"1B", 1},
		{"2B", 2},
		{"4B", 4},
		{"8B", 8},
		{"16B", 16},
		{"32B", 32},
		{"64B", 64},
		{"128B", 128},
		{"256B", 256},
		{"512B", 512},
		{"1024B", 1024},
	}

	implementations := []struct {
		name string
		fn   func([]byte, []byte, []byte) ([]byte, error)
	}{
		{"BigInt", bigint.ModExp},
		{"GMPGeneric", gmpgeneric.ModExp},
		{"GMPCWrapper", gmpcwrapper.ModExp},
	}

	for _, size := range sizes {
		base := generateWorstCase(size.bytes)
		exp := generateWorstCase(size.bytes)
		mod := generateWorstCase(size.bytes)

		for _, impl := range implementations {
			b.Run(size.name+"/"+impl.name, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					result, _ := impl.fn(base, exp, mod)
					_ = result
				}
			})
		}
	}
}

// BenchmarkComparison provides a direct comparison table
func BenchmarkComparison(b *testing.B) {
	// Prepare test data
	testData := make(map[string]struct{ base, exp, mod []byte })
	
	sizes := []int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024}
	for _, size := range sizes {
		name := ""
		switch {
		case size == 1:
			name = "8bit"
		case size == 2:
			name = "16bit"
		case size == 4:
			name = "32bit"
		case size == 8:
			name = "64bit"
		case size == 16:
			name = "128bit"
		case size == 32:
			name = "256bit"
		case size == 64:
			name = "512bit"
		case size == 128:
			name = "1024bit"
		case size == 256:
			name = "2048bit"
		case size == 512:
			name = "4096bit"
		case size == 1024:
			name = "8192bit"
		}
		
		testData[name] = struct{ base, exp, mod []byte }{
			base: generateWorstCase(size),
			exp:  generateWorstCase(size),
			mod:  generateWorstCase(size),
		}
	}

	// Run benchmarks in a structured way
	for _, sizeName := range []string{"8bit", "16bit", "32bit", "64bit", "128bit", "256bit", "512bit", "1024bit", "2048bit", "4096bit", "8192bit"} {
		data := testData[sizeName]
		
		b.Run(sizeName, func(b *testing.B) {
			b.Run("BigInt", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_, _ = bigint.ModExp(data.base, data.exp, data.mod)
				}
			})
			
			b.Run("GMPGeneric", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_, _ = gmpgeneric.ModExp(data.base, data.exp, data.mod)
				}
			})
			
			b.Run("GMPCWrapper", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_, _ = gmpcwrapper.ModExp(data.base, data.exp, data.mod)
				}
			})
		})
	}
}
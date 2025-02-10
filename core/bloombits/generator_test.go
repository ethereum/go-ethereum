package bloombits

import (
	"bytes"
	crand "crypto/rand"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
)

// Tests that batched bloom bits are correctly rotated from the input bloom
// filters.
func TestGenerator(t *testing.T) {
	// Generate the input and the rotated output
	var input, output [types.BloomBitLength][types.BloomByteLength]byte

	for i := 0; i < types.BloomBitLength; i++ {
		for j := 0; j < types.BloomBitLength; j++ {
			bit := byte(rand.Int() % 2)

			input[i][j/8] |= bit << byte(7-j%8)
			output[types.BloomBitLength-1-j][i/8] |= bit << byte(7-i%8)
		}
	}
	// Crunch the input through the generator and verify the result
	gen, err := NewGenerator(types.BloomBitLength)
	if err != nil {
		t.Fatalf("failed to create bloombit generator: %v", err)
	}
	for i, bloom := range input {
		if err := gen.AddBloom(uint(i), bloom); err != nil {
			t.Fatalf("bloom %d: failed to add: %v", i, err)
		}
	}
	for i, want := range output {
		have, err := gen.Bitset(uint(i))
		if err != nil {
			t.Fatalf("output %d: failed to retrieve bits: %v", i, err)
		}
		if !bytes.Equal(have, want[:]) {
			t.Errorf("output %d: bit vector mismatch have %x, want %x", i, have, want)
		}
	}
}

func BenchmarkGenerator(b *testing.B) {
	var input [types.BloomBitLength][types.BloomByteLength]byte
	b.Run("empty", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Crunch the input through the generator and verify the result
			gen, err := NewGenerator(types.BloomBitLength)
			if err != nil {
				b.Fatalf("failed to create bloombit generator: %v", err)
			}
			for j, bloom := range &input {
				if err := gen.AddBloom(uint(j), bloom); err != nil {
					b.Fatalf("bloom %d: failed to add: %v", i, err)
				}
			}
		}
	})
	for i := 0; i < types.BloomBitLength; i++ {
		crand.Read(input[i][:])
	}
	b.Run("random", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Crunch the input through the generator and verify the result
			gen, err := NewGenerator(types.BloomBitLength)
			if err != nil {
				b.Fatalf("failed to create bloombit generator: %v", err)
			}
			for j, bloom := range &input {
				if err := gen.AddBloom(uint(j), bloom); err != nil {
					b.Fatalf("bloom %d: failed to add: %v", i, err)
				}
			}
		}
	})
}

// TestGeneratorEdgeCases tests edge cases for the bloom generator.
func TestGeneratorEdgeCases(t *testing.T) {
	// Test with zero sections
	_, err := NewGenerator(0)
	if err == nil {
		t.Fatal("expected error for zero sections, got nil")
	}

	// Test with non-multiple of 8 sections
	_, err = NewGenerator(7)
	if err == nil {
		t.Fatal("expected error for non-multiple of 8 sections, got nil")
	}

	// Test with valid sections
	gen, err := NewGenerator(8)
	if err != nil {
		t.Fatalf("failed to create bloombit generator: %v", err)
	}

	// Test adding bloom with unexpected index
	err = gen.AddBloom(1, types.Bloom{})
	if err == nil {
		t.Fatal("expected error for unexpected index, got nil")
	}

	// Test retrieving bitset before fully generated
	_, err = gen.Bitset(0)
	if err == nil {
		t.Fatal("expected error for bloom not fully generated, got nil")
	}

	// Test retrieving bitset with out of bounds index
	gen.AddBloom(0, types.Bloom{})
	_, err = gen.Bitset(types.BloomBitLength)
	if err == nil {
		t.Fatal("expected error for bloom bit out of bounds, got nil")
	}
}

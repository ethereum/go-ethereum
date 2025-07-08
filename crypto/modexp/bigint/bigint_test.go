package bigint

import (
	"bytes"
	"testing"
)

func TestModExp(t *testing.T) {
	tests := []struct {
		name    string
		base    []byte
		exp     []byte
		mod     []byte
		want    []byte
		wantErr bool
	}{
		{
			name: "simple_2^10_mod_1000",
			base: []byte{0x02},       // 2
			exp:  []byte{0x0A},       // 10
			mod:  []byte{0x03, 0xE8}, // 1000
			want: []byte{0x18},       // 24 (1024 mod 1000)
		},
		{
			name: "zero_base",
			base: []byte{0x00},
			exp:  []byte{0x05},
			mod:  []byte{0x07},
			want: []byte{0x00},
		},
		{
			name: "zero_exponent",
			base: []byte{0x05},
			exp:  []byte{0x00},
			mod:  []byte{0x07},
			want: []byte{0x01}, // Any number to the power of 0 is 1
		},
		{
			name: "large_numbers",
			base: []byte{0xFF, 0xFF},
			exp:  []byte{0x02},
			mod:  []byte{0x01, 0x00, 0x00},
			want: []byte{0xFE, 0x01}, // (65535^2) mod 65536 = 65025
		},
		{
			name:    "empty_modulus",
			base:    []byte{0x02},
			exp:     []byte{0x03},
			mod:     []byte{},
			want:    []byte{},
			wantErr: false,
		},
		{
			name:    "zero_modulus",
			base:    []byte{0x02},
			exp:     []byte{0x03},
			mod:     []byte{0x00, 0x00},
			want:    []byte{},
			wantErr: false,
		},
		{
			name: "empty_base_and_exp",
			base: []byte{},
			exp:  []byte{},
			mod:  []byte{0x07},
			want: []byte{0x01}, // 0^0 = 1 in Go's big.Int
		},
		{
			name: "base_equals_one",
			base: []byte{0x01},
			exp:  []byte{0xFF, 0xFF, 0xFF, 0xFF}, // large exponent
			mod:  []byte{0x07},
			want: []byte{0x01}, // 1^anything mod 7 = 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ModExp(tt.base, tt.exp, tt.mod)
			if (err != nil) != tt.wantErr {
				t.Errorf("ModExp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !bytes.Equal(got, tt.want) {
				t.Errorf("ModExp() = %x, want %x", got, tt.want)
			}
		})
	}
}


// TestLargeModExp tests with very large numbers
func TestLargeModExp(t *testing.T) {
	// Test with 2048-bit numbers
	base := make([]byte, 256)
	exp := make([]byte, 256)
	mod := make([]byte, 256)

	// Fill with test data
	for i := range base {
		base[i] = byte(i)
		exp[i] = byte(255 - i)
		mod[i] = 0xFF
	}
	mod[0] = 0x7F // Make sure modulus is not too large

	result, err := ModExp(base, exp, mod)
	if err != nil {
		t.Fatalf("ModExp() with large numbers failed: %v", err)
	}

	if len(result) == 0 {
		t.Error("Expected non-empty result for large numbers")
	}
}

// BenchmarkModExp benchmarks the byte-oriented interface
func BenchmarkModExp(b *testing.B) {
	base := make([]byte, 32)
	exp := make([]byte, 32)
	mod := make([]byte, 32)

	for i := range base {
		base[i] = byte(i * 17)
		exp[i] = byte(i * 31)
		mod[i] = byte(255 - i)
	}
	mod[31] |= 0x01 // Ensure odd modulus

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ModExp(base, exp, mod)
	}
}


package modexp

import (
	"bytes"
	"testing"
)

func TestModExp(t *testing.T) {
	tests := []struct {
		name string
		base []byte
		exp  []byte
		mod  []byte
		want []byte
	}{
		{
			name: "simple",
			base: []byte{0x02},       // 2
			exp:  []byte{0x0A},       // 10
			mod:  []byte{0x03, 0xE8}, // 1000
			want: []byte{0x18},       // 24
		},
		{
			name: "zero_modulus",
			base: []byte{0x02},
			exp:  []byte{0x03},
			mod:  []byte{},
			want: []byte{},
		},
		{
			name: "base_equals_one",
			base: []byte{0x01},
			exp:  []byte{0xFF, 0xFF},
			mod:  []byte{0x07},
			want: []byte{0x01},
		},
		{
			name: "large_numbers",
			base: []byte{0xFF, 0xFF},
			exp:  []byte{0x02},
			mod:  []byte{0x01, 0x00, 0x00},
			want: []byte{0xFE, 0x01},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ModExp(tt.base, tt.exp, tt.mod)
			if err != nil {
				t.Fatalf("ModExp error: %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("ModExp() = %x, want %x", got, tt.want)
			}
		})
	}
}

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
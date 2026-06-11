package serialize

import (
	"math/bits"
	"testing"
)

// TestDeserializeUintWidthBound checks that the variable-length reflect.Uint
// path rejects an encoding wider than the platform uint: such an input would
// overflow the uint64 accumulator and silently truncate the decoded value.
func TestDeserializeUintWidthBound(t *testing.T) {
	width := bits.UintSize / 8 // bytes in a platform uint (4 on 32-bit, 8 on 64-bit)

	// One byte past the platform width: must be rejected.
	over := make([]byte, 1+width+1)
	over[0] = byte(width + 1) // 1-byte length prefix
	for i := 1; i < len(over); i++ {
		over[i] = 0xFF
	}
	var u uint
	if err := Deserialize(NewByteBuffer(over), &u); err == nil {
		t.Fatalf("expected an error for a %d-byte uint encoding (platform width %d bytes)", width+1, width)
	}

	// Exactly the platform width (max uint) still decodes.
	in := make([]byte, 1+width)
	in[0] = byte(width)
	for i := 1; i < len(in); i++ {
		in[i] = 0xFF
	}
	var ok uint
	if err := Deserialize(NewByteBuffer(in), &ok); err != nil {
		t.Fatalf("%d-byte uint rejected: %v", width, err)
	}
	if ok != ^uint(0) {
		t.Fatalf("expected max uint, got %d", ok)
	}
}

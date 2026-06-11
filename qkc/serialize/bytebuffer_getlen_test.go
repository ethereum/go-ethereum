package serialize

import "testing"

// TestGetLenRejectsOverflow pins that getLen never returns a negative or wrapped
// length. A length prefix at/above the platform int width with the high bit set
// previously overflowed the signed accumulator into a negative value, which slid
// past downstream "len <= remaining" checks and panicked (reflect.MakeSlice:
// negative len) on attacker-controlled input.
func TestGetLenRejectsOverflow(t *testing.T) {
	// 8-byte prefix, high bit set: > MaxInt on every platform -> error.
	bb := NewByteBuffer([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	if n, err := bb.getLen(8); err == nil {
		t.Fatalf("expected an error for an over-large 8-byte length prefix, got %d", n)
	}

	// A wide (10-byte) prefix encoding a small value still decodes: leading
	// zeros must not be mistaken for overflow.
	small := NewByteBuffer([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0x12, 0x34})
	if n, err := small.getLen(10); err != nil || n != 0x1234 {
		t.Fatalf("small value with wide prefix: n=%d err=%v, want 4660", n, err)
	}

	// getLen must never return a negative length for any input/width.
	for _, w := range []int{1, 2, 4, 8} {
		b := make([]byte, w)
		for i := range b {
			b[i] = 0xFF
		}
		if n, err := NewByteBuffer(b).getLen(w); err == nil && n < 0 {
			t.Fatalf("getLen(%d) returned negative length %d", w, n)
		}
	}
}

// TestDeserializeWideLenPrefixNoPanic ensures an attacker-controlled over-wide
// slice length prefix yields an error, not a panic.
func TestDeserializeWideLenPrefixNoPanic(t *testing.T) {
	type wide struct {
		V []uint32 `bytesizeofslicelen:"8"`
	}
	var out wide
	err := Deserialize(NewByteBuffer([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}), &out)
	if err == nil {
		t.Fatal("expected an error for an over-large length prefix")
	}
}

package serialize

import "testing"

// TestDeserializeFromBytesRejectsTrailing checks that valid data followed by
// trailing bytes is rejected rather than silently accepted.
func TestDeserializeFromBytesRejectsTrailing(t *testing.T) {
	// uint8 consumes exactly one byte; the second byte is trailing garbage.
	var v uint8
	if err := DeserializeFromBytes([]byte{0x05, 0xff}, &v); err == nil {
		t.Fatal("expected an error for trailing bytes after a complete decode")
	}

	// An exact input still decodes.
	var ok uint8
	if err := DeserializeFromBytes([]byte{0x05}, &ok); err != nil || ok != 5 {
		t.Fatalf("exact input should decode cleanly: err=%v v=%d", err, ok)
	}
}

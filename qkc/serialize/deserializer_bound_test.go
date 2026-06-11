package serialize

import "testing"

// TestDeserializeListBound guards the OOM/DoS vector where a tiny packet claims a
// huge list length: deserializeList must reject a length larger than the
// remaining buffer instead of pre-allocating it.
func TestDeserializeListBound(t *testing.T) {
	type bigList struct {
		V []uint32 `bytesizeofslicelen:"4"`
	}
	// 4-byte length prefix = 0xFFFFFFFF (~4 billion), followed by no element bytes.
	input := unhex("FFFFFFFF")
	var out bigList
	if err := Deserialize(NewByteBuffer(input), &out); err == nil {
		t.Fatal("expected an error for an over-large list length, got nil (would over-allocate ~4B elements)")
	}
}

// TestDeserializeListValidStillDecodes confirms the bound does not reject valid
// input where length <= remaining bytes.
func TestDeserializeListValidStillDecodes(t *testing.T) {
	type list struct {
		V []uint32 `bytesizeofslicelen:"4"`
	}
	// len = 2 (4 bytes), then two uint32s (8 bytes): 0x00000007, 0x00000009.
	input := unhex("000000020000000700000009")
	var out list
	if err := Deserialize(NewByteBuffer(input), &out); err != nil {
		t.Fatalf("valid list rejected: %v", err)
	}
	if len(out.V) != 2 || out.V[0] != 7 || out.V[1] != 9 {
		t.Fatalf("bad decode: %v", out.V)
	}
}

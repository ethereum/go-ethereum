package serialize

import "testing"

type withNilPtr struct {
	P *uint32 `ser:"nil"`
}

// TestDeserializeNilMarkerCanonical checks that the ser:"nil" presence marker
// only accepts 0 or 1. The serializer writes exactly those two values, so any
// other byte is a non-canonical encoding and must be rejected.
func TestDeserializeNilMarkerCanonical(t *testing.T) {
	// marker 0x00 -> nil pointer.
	var nilCase withNilPtr
	if err := Deserialize(NewByteBuffer([]byte{0x00}), &nilCase); err != nil {
		t.Fatalf("marker 0 should decode to nil: %v", err)
	}
	if nilCase.P != nil {
		t.Fatalf("marker 0 should leave pointer nil, got %v", *nilCase.P)
	}

	// marker 0x01 followed by a 4-byte uint32 -> present.
	var present withNilPtr
	if err := Deserialize(NewByteBuffer([]byte{0x01, 0x00, 0x00, 0x00, 0x07}), &present); err != nil {
		t.Fatalf("marker 1 should decode the value: %v", err)
	}
	if present.P == nil || *present.P != 7 {
		t.Fatalf("marker 1 should decode value 7, got %v", present.P)
	}

	// marker 0x05 must be rejected (non-canonical "present" byte).
	var bad withNilPtr
	if err := Deserialize(NewByteBuffer([]byte{0x05, 0x00, 0x00, 0x00, 0x07}), &bad); err == nil {
		t.Fatal("expected an error for a non-0/1 presence marker")
	}
}

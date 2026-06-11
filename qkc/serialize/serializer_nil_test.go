package serialize

import "testing"

// TestSerializeNilValueRejected checks that an untyped nil is rejected with an
// error rather than panicking, symmetric with DeserializeWithTags rejecting a
// nil target. A typed nil pointer is still serializable.
func TestSerializeNilValueRejected(t *testing.T) {
	var w []byte
	if err := Serialize(&w, nil); err == nil {
		t.Fatal("expected an error serializing untyped nil")
	}
	if _, err := SerializeToBytes(nil); err == nil {
		t.Fatal("expected an error from SerializeToBytes(nil)")
	}

	// A typed nil pointer carries a type, so it still serializes (as its
	// element's zero value).
	type foo struct{ X uint32 }
	var p *foo
	w = nil
	if err := Serialize(&w, p); err != nil {
		t.Fatalf("typed nil pointer should serialize: %v", err)
	}
}

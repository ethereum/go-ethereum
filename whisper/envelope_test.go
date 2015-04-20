package whisper

import (
	"bytes"
	"testing"
)

func TestEnvelopeOpen(t *testing.T) {
	payload := []byte("hello world")
	message := NewMessage(payload)

	envelope, err := message.Wrap(DefaultPoW, Options{})
	if err != nil {
		t.Fatalf("failed to wrap message: %v", err)
	}
	opened, err := envelope.Open(nil)
	if err != nil {
		t.Fatalf("failed to open envelope: %v.", err)
	}
	if opened.Flags != message.Flags {
		t.Fatalf("flags mismatch: have %d, want %d", opened.Flags, message.Flags)
	}
	if bytes.Compare(opened.Signature, message.Signature) != 0 {
		t.Fatalf("signature mismatch: have 0x%x, want 0x%x", opened.Signature, message.Signature)
	}
	if bytes.Compare(opened.Payload, message.Payload) != 0 {
		t.Fatalf("payload mismatch: have 0x%x, want 0x%x", opened.Payload, message.Payload)
	}
	if opened.Sent != message.Sent {
		t.Fatalf("send time mismatch: have %d, want %d", opened.Sent, message.Sent)
	}

	if opened.Hash != envelope.Hash() {
		t.Fatalf("message hash mismatch: have 0x%x, want 0x%x", opened.Hash, envelope.Hash())
	}
}

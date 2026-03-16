package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"testing"
)

func TestSignUnsafeCounter(t *testing.T) {
	key, err := ecdsa.GenerateKey(S256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	hash := make([]byte, 32)
	copy(hash, "hi there")

	sig, err := Sign(hash, key)
	if err != nil {
		t.Fatal(err)
	}
	sigUnsafe0, err := SignUnsafe(hash, key, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sig, sigUnsafe0) {
		t.Fatal("counter=0 should match Sign")
	}
	sigUnsafe1a, err := SignUnsafe(hash, key, 1)
	if err != nil {
		t.Fatal(err)
	}
	sigUnsafe1b, err := SignUnsafe(hash, key, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sigUnsafe1a, sigUnsafe1b) {
		t.Fatal("counter=1 signatures not equal")
	}
	if bytes.Equal(sig, sigUnsafe1a) {
		t.Fatal("counter=1 should not match counter=0 signature")
	}
	pub, err := SigToPub(hash, sigUnsafe1a)
	if err != nil {
		t.Fatal(err)
	}
	if pub.X.Cmp(key.PublicKey.X) != 0 || pub.Y.Cmp(key.PublicKey.Y) != 0 {
		t.Fatal("recovered pubkey mismatch")
	}
}

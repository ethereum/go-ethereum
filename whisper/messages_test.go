package whisper

import (
	"bytes"
	"crypto/elliptic"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestSign(t *testing.T) {
	prv, _ := crypto.GenerateKey()
	msg := NewMessage([]byte("hello world"))
	msg.sign(prv)

	pubKey := msg.Recover()
	p1 := elliptic.Marshal(crypto.S256(), prv.PublicKey.X, prv.PublicKey.Y)
	p2 := elliptic.Marshal(crypto.S256(), pubKey.X, pubKey.Y)

	if !bytes.Equal(p1, p2) {
		t.Error("recovered pub key did not match")
	}
}

func TestMessageEncryptDecrypt(t *testing.T) {
	prv1, _ := crypto.GenerateKey()
	prv2, _ := crypto.GenerateKey()

	data := []byte("hello world")
	msg := NewMessage(data)
	envelope, err := msg.Seal(DefaultPow, Opts{
		From: prv1,
		To:   &prv2.PublicKey,
	})
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	msg1, err := envelope.Open(prv2)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	if !bytes.Equal(msg1.Payload, data) {
		fmt.Println("encryption error. data did not match")
		t.FailNow()
	}
}

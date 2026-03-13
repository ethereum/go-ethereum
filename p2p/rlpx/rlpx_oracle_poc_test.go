package rlpx

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
)

func TestHandshakeECIESInvalidCurveOracle(t *testing.T) {
	initKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	respKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	init := handshakeState{
		initiator: true,
		remote:    ecies.ImportECDSAPublic(&respKey.PublicKey),
	}
	authMsg, err := init.makeAuthMsg(initKey)
	if err != nil {
		t.Fatal(err)
	}
	packet, err := init.sealEIP8(authMsg)
	if err != nil {
		t.Fatal(err)
	}

	var recv handshakeState
	if _, err := recv.readMsg(new(authMsgV4), respKey, bytes.NewReader(packet)); err != nil {
		t.Fatalf("expected valid packet to decrypt: %v", err)
	}

	tampered := append([]byte(nil), packet...)
	if len(tampered) < 2+65 {
		t.Fatalf("unexpected packet length %d", len(tampered))
	}
	tampered[2] = 0x04
	for i := 1; i < 65; i++ {
		tampered[2+i] = 0x00
	}

	var recv2 handshakeState
	_, err = recv2.readMsg(new(authMsgV4), respKey, bytes.NewReader(tampered))
	if err == nil {
		t.Fatal("expected decryption failure for invalid curve point")
	}
	if !errors.Is(err, ecies.ErrInvalidPublicKey) {
		t.Fatalf("unexpected error: %v", err)
	}
}

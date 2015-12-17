// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rlpx

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
)

func init() {
	spew.Config.Indent = "\t"
}

func TestSharedSecret(t *testing.T) {
	prv0, _ := crypto.GenerateKey()
	pub0 := &prv0.PublicKey
	prv1, _ := crypto.GenerateKey()
	pub1 := &prv1.PublicKey
	ss0, err := ecies.ImportECDSA(prv0).GenerateShared(ecies.ImportECDSAPublic(pub1), sskLen, sskLen)
	if err != nil {
		return
	}
	ss1, err := ecies.ImportECDSA(prv1).GenerateShared(ecies.ImportECDSAPublic(pub0), sskLen, sskLen)
	if err != nil {
		return
	}
	if !bytes.Equal(ss0, ss1) {
		t.Errorf("secret mismatch")
	}
}

// This test does random V5 handshakes and compares the secrets.
func TestHandshake(t *testing.T) {
	for i := 0; i < 10 && !t.Failed(); i++ {
		start := time.Now()
		doTestHandshake(t)
		t.Logf("%d %v\n", i+1, time.Since(start))
	}
}

func doTestHandshake(t *testing.T) {
	var (
		prv0, _ = crypto.GenerateKey()
		prv1, _ = crypto.GenerateKey()
		p1, p2  = net.Pipe()
		c1      = Server(p1, &Config{Key: prv0})
		c2      = Client(p2, &prv0.PublicKey, &Config{Key: prv1})
	)
	shake := func(conn *Conn, rkey *ecdsa.PublicKey) error {
		defer conn.Close()
		if err := conn.Handshake(); err != nil {
			return err
		}
		if !reflect.DeepEqual(conn.RemoteID(), rkey) {
			return fmt.Errorf("remote ID mismatch: got %v, want: %v", conn.RemoteID(), rkey)
		}
		return nil
	}
	run(t, rig{
		"initiator": func() error { return shake(c1, &prv1.PublicKey) },
		"recipient": func() error { return shake(c2, &prv0.PublicKey) },
	})

	// compare derived secrets
	if !reflect.DeepEqual(c1.rw.egressMac, c2.rw.ingressMac) {
		t.Errorf("egress mac mismatch:\n c1.rw: %#v\n c2.rw: %#v", c1.rw.egressMac, c2.rw.ingressMac)
	}
	if !reflect.DeepEqual(c1.rw.ingressMac, c2.rw.egressMac) {
		t.Errorf("ingress mac mismatch:\n c1.rw: %#v\n c2.rw: %#v", c1.rw.ingressMac, c2.rw.egressMac)
	}
	if !reflect.DeepEqual(c1.rw.enc, c2.rw.dec) {
		t.Errorf("enc cipher mismatch:\n c1.rw: %#v\n c2.rw: %#v", c1.rw.enc, c2.rw.dec)
	}
	if !reflect.DeepEqual(c1.rw.dec, c2.rw.enc) {
		t.Errorf("dec cipher mismatch:\n c1.rw: %#v\n c2.rw: %#v", c1.rw.dec, c2.rw.enc)
	}
}

// This test runs initator/recipient against each other for each test vector.
func TestHandshakeTV(t *testing.T) {
	for i, ht := range handshakeTV {
		p1, p2 := net.Pipe()
		run(t, rig{
			"initiator": func() error { return checkInitiator(p1, ht) },
			"recipient": func() error { return checkRecipient(p2, ht) },
		})
		if t.Failed() {
			t.Fatalf("failed test case %d:\n%s", i, spew.Sdump(ht))
		}
	}
}

// This test runs the encrypted auth packets from the test vectors against
// the recipient code.
func TestHandshakePacketsRecipientTV(t *testing.T) {
	for i, ht := range handshakeTV {
		p1, p2 := net.Pipe()
		run(t, rig{
			"recipient": func() error {
				defer p1.Close()
				return checkRecipient(p1, ht)
			},
			"auth packet send": func() error {
				_, err := p2.Write(ht.encAuth)
				return err
			},
			"authResp packet recv": func() error {
				ioutil.ReadAll(p2)
				return nil
			},
		})
		if t.Failed() {
			t.Fatalf("failed test case %d:\n%s", i, spew.Sdump(ht))
		}
	}
}

// This test runs the encrypted authResp packets from the test vectors against
// the initiator code.
func TestHandshakePacketsInitiatorTV(t *testing.T) {
	for i, ht := range handshakeTV {
		p1, p2 := net.Pipe()
		run(t, rig{
			"initiator": func() error {
				defer p1.Close()
				return checkInitiator(p1, ht)
			},
			"authResp packet send": func() error {
				_, err := p2.Write(ht.encAuthResp)
				return err
			},
			"auth packet recv": func() error {
				ioutil.ReadAll(p2)
				return nil
			},
		})
		if t.Failed() {
			t.Fatalf("failed test case %d:\n%s", i, spew.Sdump(ht))
		}
	}
}

// This test checks that secrets.mac is initialized correctly.
func TestHandshakeDeriveMacTV(t *testing.T) {
	for i, ht := range handshakeTV {
		h := handshake{
			initiator:       true,
			localPrivKey:    ht.initiator.Key,
			remotePub:       ecies.ImportECDSAPublic(&ht.recipient.Key.PublicKey),
			initNonce:       ht.initiatorNonce,
			respNonce:       ht.recipientNonce,
			randomPrivKey:   ecies.ImportECDSA(ht.initiatorEphemeralKey),
			remoteRandomPub: ecies.ImportECDSAPublic(&ht.recipientEphemeralKey.PublicKey),
		}
		vsn, ingress, egress, err := h.deriveSecrets(ht.initiator.ForceV4, ht.encAuth, ht.encAuthResp)
		if err != nil {
			t.Error("deriveSecrets error: %v", err)
		}
		if sum := ingress.mac.Sum(nil); !bytes.Equal(sum, ht.initiatorIngressMacDigest) {
			t.Errorf("ingress mac mismatch: got %x, want %x", sum, ht.initiatorIngressMacDigest)
		}
		if sum := egress.mac.Sum(nil); !bytes.Equal(sum, ht.initiatorEgressMacDigest) {
			t.Errorf("egress mac mismatch: got %x, want %x", sum, ht.initiatorEgressMacDigest)
		}
		if err := ht.checkSecrets(vsn, nil, ingress, egress); err != nil {
			t.Error(err)
		}
		if t.Failed() {
			t.Fatalf("failed test case %d:\n%s", i, spew.Sdump(ht))
		}
	}
}

func checkInitiator(pipe net.Conn, ht handshakeTest) error {
	remotePub := &ht.recipient.Key.PublicKey
	conn := Client(pipe, remotePub, ht.initiator)
	conn.handshakeRand = fakeRandSource{key: ht.initiatorEphemeralKey, nonce: ht.initiatorNonce}
	vsn, ingress, egress, err := conn.initiatorHandshake()
	if err != nil {
		return err
	}
	return ht.checkSecrets(vsn, nil, ingress, egress)
}

func checkRecipient(pipe net.Conn, ht handshakeTest) error {
	conn := Server(pipe, ht.recipient)
	conn.handshakeRand = fakeRandSource{key: ht.recipientEphemeralKey, nonce: ht.recipientNonce}
	vsn, remoteID, ingress, egress, err := conn.recipientHandshake()
	if err != nil {
		return err
	}
	return ht.checkSecrets(vsn, remoteID, egress, ingress)
}

func (ht handshakeTest) checkSecrets(vsn uint, remoteID *ecdsa.PublicKey, ingress, egress secrets) error {
	if remoteID != nil && !reflect.DeepEqual(remoteID, &ht.initiator.Key.PublicKey) {
		return fmt.Errorf("remoteID mismatch:\ngot  %x\nwant %x",
			crypto.FromECDSAPub(remoteID), crypto.FromECDSAPub(&ht.initiator.Key.PublicKey))
	}
	if vsn != ht.negotiatedVersion {
		return fmt.Errorf("version mismatch: got %d, want %d", vsn, ht.negotiatedVersion)
	}
	// Remove the MACs so secrets can be compared with DeepEqual.
	ingress.mac, egress.mac = nil, nil
	if !reflect.DeepEqual(ingress, ht.initiatorIngressSecrets) {
		return fmt.Errorf("initiatorIngressSecrets mismatch:\ngot %swant %s",
			spew.Sdump(ingress), spew.Sdump(ht.initiatorEgressSecrets))
	}
	if !reflect.DeepEqual(egress, ht.initiatorEgressSecrets) {
		return fmt.Errorf("initiatorEgressSecrets mismatch:\ngot %swant %s",
			spew.Sdump(egress), spew.Sdump(ht.initiatorIngressSecrets))
	}
	return nil
}

type fakeRandSource struct {
	key   *ecdsa.PrivateKey
	nonce []byte
}

func (ht fakeRandSource) generateNonce(b []byte) error {
	if len(b) > len(ht.nonce) {
		panic(fmt.Sprintf("requested %d bytes of nonce data, have %d", len(b), len(ht.nonce)))
	}
	copy(b, ht.nonce)
	return nil
}

func (ht fakeRandSource) generateKey() (*ecies.PrivateKey, error) {
	return ecies.ImportECDSA(ht.key), nil
}

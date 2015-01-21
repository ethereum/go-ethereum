package p2p

import (
	"bytes"
	// "crypto/ecdsa"
	// "crypto/elliptic"
	// "crypto/rand"
	"fmt"
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/obscuren/ecies"
)

func TestPublicKeyEncoding(t *testing.T) {
	prv0, _ := crypto.GenerateKey() // = ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	pub0 := &prv0.PublicKey
	pub0s := crypto.FromECDSAPub(pub0)
	pub1, err := ImportPublicKey(pub0s)
	if err != nil {
		t.Errorf("%v", err)
	}
	eciesPub1 := ecies.ImportECDSAPublic(pub1)
	if eciesPub1 == nil {
		t.Errorf("invalid ecdsa public key")
	}
	pub1s, err := ExportPublicKey(pub1)
	if err != nil {
		t.Errorf("%v", err)
	}
	if len(pub1s) != 64 {
		t.Errorf("wrong length expect 64, got", len(pub1s))
	}
	pub2, err := ImportPublicKey(pub1s)
	if err != nil {
		t.Errorf("%v", err)
	}
	pub2s, err := ExportPublicKey(pub2)
	if err != nil {
		t.Errorf("%v", err)
	}
	if !bytes.Equal(pub1s, pub2s) {
		t.Errorf("exports dont match")
	}
	pub2sEC := crypto.FromECDSAPub(pub2)
	if !bytes.Equal(pub0s, pub2sEC) {
		t.Errorf("exports dont match")
	}
}

func TestSharedSecret(t *testing.T) {
	prv0, _ := crypto.GenerateKey() // = ecdsa.GenerateKey(crypto.S256(), rand.Reader)
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
	t.Logf("Secret:\n%v %x\n%v %x", len(ss0), ss0, len(ss0), ss1)
	if !bytes.Equal(ss0, ss1) {
		t.Errorf("dont match :(")
	}
}

func TestCryptoHandshake(t *testing.T) {
	var err error
	var sessionToken []byte
	prv0, _ := crypto.GenerateKey() // = ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	pub0 := &prv0.PublicKey
	prv1, _ := crypto.GenerateKey()
	pub1 := &prv1.PublicKey

	var initiator, receiver *cryptoId
	if initiator, err = newCryptoId(&peerId{crypto.FromECDSA(prv0), crypto.FromECDSAPub(pub0)}); err != nil {
		return
	}
	if receiver, err = newCryptoId(&peerId{crypto.FromECDSA(prv1), crypto.FromECDSAPub(pub1)}); err != nil {
		return
	}

	// simulate handshake by feeding output to input
	// initiator sends handshake 'auth'
	auth, initNonce, randomPrivKey, _, err := initiator.startHandshake(receiver.pubKeyS, sessionToken)
	if err != nil {
		t.Errorf("%v", err)
	}

	// receiver reads auth and responds with response
	response, remoteRecNonce, remoteInitNonce, remoteRandomPrivKey, remoteInitRandomPubKey, err := receiver.respondToHandshake(auth, crypto.FromECDSAPub(pub0), sessionToken)
	if err != nil {
		t.Errorf("%v", err)
	}

	// initiator reads receiver's response and the key exchange completes
	recNonce, remoteRandomPubKey, _, err := initiator.completeHandshake(response)
	if err != nil {
		t.Errorf("%v", err)
	}

	// now both parties should have the same session parameters
	initSessionToken, initSecretRW, err := initiator.newSession(initNonce, recNonce, auth, randomPrivKey, remoteRandomPubKey)
	if err != nil {
		t.Errorf("%v", err)
	}

	recSessionToken, recSecretRW, err := receiver.newSession(remoteInitNonce, remoteRecNonce, auth, remoteRandomPrivKey, remoteInitRandomPubKey)
	if err != nil {
		t.Errorf("%v", err)
	}

	fmt.Printf("\nauth (%v) %x\n\nresp (%v) %x\n\n", len(auth), auth, len(response), response)

	// fmt.Printf("\nauth %x\ninitNonce %x\nresponse%x\nremoteRecNonce %x\nremoteInitNonce %x\nremoteRandomPubKey %x\nrecNonce %x\nremoteInitRandomPubKey %x\ninitSessionToken %x\n\n", auth, initNonce, response, remoteRecNonce, remoteInitNonce, remoteRandomPubKey, recNonce, remoteInitRandomPubKey, initSessionToken)

	if !bytes.Equal(initNonce, remoteInitNonce) {
		t.Errorf("nonces do not match")
	}
	if !bytes.Equal(recNonce, remoteRecNonce) {
		t.Errorf("receiver nonces do not match")
	}
	if !bytes.Equal(initSessionToken, recSessionToken) {
		t.Errorf("session tokens do not match")
	}
	// aesSecret, macSecret, egressMac, ingressMac
	if !bytes.Equal(initSecretRW.aesSecret, recSecretRW.aesSecret) {
		t.Errorf("AES secrets do not match")
	}
	if !bytes.Equal(initSecretRW.macSecret, recSecretRW.macSecret) {
		t.Errorf("macSecrets do not match")
	}
	if !bytes.Equal(initSecretRW.egressMac, recSecretRW.egressMac) {
		t.Errorf("egressMacs do not match")
	}
	if !bytes.Equal(initSecretRW.ingressMac, recSecretRW.ingressMac) {
		t.Errorf("ingressMacs do not match")
	}

}

func TestPeersHandshake(t *testing.T) {
	defer testlog(t).detach()
	var err error
	// var sessionToken []byte
	prv0, _ := crypto.GenerateKey() // = ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	pub0 := &prv0.PublicKey
	prv1, _ := crypto.GenerateKey()
	pub1 := &prv1.PublicKey

	prv0s := crypto.FromECDSA(prv0)
	pub0s := crypto.FromECDSAPub(pub0)
	prv1s := crypto.FromECDSA(prv1)
	pub1s := crypto.FromECDSAPub(pub1)

	conn1, conn2 := net.Pipe()
	initiator := newPeer(conn1, []Protocol{}, nil)
	receiver := newPeer(conn2, []Protocol{}, nil)
	initiator.dialAddr = &peerAddr{IP: net.ParseIP("1.2.3.4"), Port: 2222, Pubkey: pub1s[1:]}
	initiator.ourID = &peerId{prv0s, pub0s}

	// this is cheating. identity of initiator/dialler not available to listener/receiver
	// its public key should be looked up based on IP address
	receiver.identity = initiator.ourID
	receiver.ourID = &peerId{prv1s, pub1s}

	initiator.pubkeyHook = func(*peerAddr) error { return nil }
	receiver.pubkeyHook = func(*peerAddr) error { return nil }

	initiator.cryptoHandshake = true
	receiver.cryptoHandshake = true
	errc0 := make(chan error, 1)
	errc1 := make(chan error, 1)
	go func() {
		_, err := initiator.loop()
		errc0 <- err
	}()
	go func() {
		_, err := receiver.loop()
		errc1 <- err
	}()
	select {
	case err = <-errc0:
		t.Errorf("peer 0 quit with error: %v", err)
	case err = <-errc1:
		t.Errorf("peer 1 quit with error: %v", err)
	}
}

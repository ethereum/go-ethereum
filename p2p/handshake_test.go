package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"net"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

func TestPublicKeyEncoding(t *testing.T) {
	prv0, _ := crypto.GenerateKey() // = ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	pub0 := &prv0.PublicKey
	pub0s := crypto.FromECDSAPub(pub0)
	pub1, err := importPublicKey(pub0s)
	if err != nil {
		t.Errorf("%v", err)
	}
	eciesPub1 := ecies.ImportECDSAPublic(pub1)
	if eciesPub1 == nil {
		t.Errorf("invalid ecdsa public key")
	}
	pub1s, err := exportPublicKey(pub1)
	if err != nil {
		t.Errorf("%v", err)
	}
	if len(pub1s) != 64 {
		t.Errorf("wrong length expect 64, got", len(pub1s))
	}
	pub2, err := importPublicKey(pub1s)
	if err != nil {
		t.Errorf("%v", err)
	}
	pub2s, err := exportPublicKey(pub2)
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
	testCryptoHandshake(newkey(), newkey(), nil, t)
}

func TestCryptoHandshakeWithToken(t *testing.T) {
	sessionToken := make([]byte, shaLen)
	rand.Read(sessionToken)
	testCryptoHandshake(newkey(), newkey(), sessionToken, t)
}

func testCryptoHandshake(prv0, prv1 *ecdsa.PrivateKey, sessionToken []byte, t *testing.T) {
	var err error
	// pub0 := &prv0.PublicKey
	pub1 := &prv1.PublicKey

	// pub0s := crypto.FromECDSAPub(pub0)
	pub1s := crypto.FromECDSAPub(pub1)

	// simulate handshake by feeding output to input
	// initiator sends handshake 'auth'
	auth, initNonce, randomPrivKey, err := authMsg(prv0, pub1s, sessionToken)
	if err != nil {
		t.Errorf("%v", err)
	}
	// t.Logf("-> %v", hexkey(auth))

	// receiver reads auth and responds with response
	response, remoteRecNonce, remoteInitNonce, _, remoteRandomPrivKey, remoteInitRandomPubKey, err := authResp(auth, sessionToken, prv1)
	if err != nil {
		t.Errorf("%v", err)
	}
	// t.Logf("<- %v\n", hexkey(response))

	// initiator reads receiver's response and the key exchange completes
	recNonce, remoteRandomPubKey, _, err := completeHandshake(response, prv0)
	if err != nil {
		t.Errorf("completeHandshake error: %v", err)
	}

	// now both parties should have the same session parameters
	initSessionToken, err := newSession(initNonce, recNonce, randomPrivKey, remoteRandomPubKey)
	if err != nil {
		t.Errorf("newSession error: %v", err)
	}

	recSessionToken, err := newSession(remoteInitNonce, remoteRecNonce, remoteRandomPrivKey, remoteInitRandomPubKey)
	if err != nil {
		t.Errorf("newSession error: %v", err)
	}

	// fmt.Printf("\nauth (%v) %x\n\nresp (%v) %x\n\n", len(auth), auth, len(response), response)

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
}

func TestEncHandshake(t *testing.T) {
	defer testlog(t).detach()

	prv0, _ := crypto.GenerateKey()
	prv1, _ := crypto.GenerateKey()
	pub0s, _ := exportPublicKey(&prv0.PublicKey)
	pub1s, _ := exportPublicKey(&prv1.PublicKey)
	rw0, rw1 := net.Pipe()
	tokens := make(chan []byte)

	go func() {
		token, err := outboundEncHandshake(rw0, prv0, pub1s, nil)
		if err != nil {
			t.Errorf("outbound side error: %v", err)
		}
		tokens <- token
	}()
	go func() {
		token, remotePubkey, err := inboundEncHandshake(rw1, prv1, nil)
		if err != nil {
			t.Errorf("inbound side error: %v", err)
		}
		if !bytes.Equal(remotePubkey, pub0s) {
			t.Errorf("inbound side returned wrong remote pubkey\n  got:  %x\n  want: %x", remotePubkey, pub0s)
		}
		tokens <- token
	}()

	t1, t2 := <-tokens, <-tokens
	if !bytes.Equal(t1, t2) {
		t.Error("session token mismatch")
	}
}

func TestSetupConn(t *testing.T) {
	prv0, _ := crypto.GenerateKey()
	prv1, _ := crypto.GenerateKey()
	node0 := &discover.Node{
		ID:      discover.PubkeyID(&prv0.PublicKey),
		IP:      net.IP{1, 2, 3, 4},
		TCPPort: 33,
	}
	node1 := &discover.Node{
		ID:      discover.PubkeyID(&prv1.PublicKey),
		IP:      net.IP{5, 6, 7, 8},
		TCPPort: 44,
	}
	hs0 := &protoHandshake{
		Version: baseProtocolVersion,
		ID:      node0.ID,
		Caps:    []Cap{{"a", 0}, {"b", 2}},
	}
	hs1 := &protoHandshake{
		Version: baseProtocolVersion,
		ID:      node1.ID,
		Caps:    []Cap{{"c", 1}, {"d", 3}},
	}
	fd0, fd1 := net.Pipe()

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn0, err := setupConn(fd0, prv0, hs0, node1)
		if err != nil {
			t.Errorf("outbound side error: %v", err)
			return
		}
		if conn0.ID != node1.ID {
			t.Errorf("outbound conn id mismatch: got %v, want %v", conn0.ID, node1.ID)
		}
		if !reflect.DeepEqual(conn0.Caps, hs1.Caps) {
			t.Errorf("outbound caps mismatch: got %v, want %v", conn0.Caps, hs1.Caps)
		}
	}()

	conn1, err := setupConn(fd1, prv1, hs1, nil)
	if err != nil {
		t.Fatalf("inbound side error: %v", err)
	}
	if conn1.ID != node0.ID {
		t.Errorf("inbound conn id mismatch: got %v, want %v", conn1.ID, node0.ID)
	}
	if !reflect.DeepEqual(conn1.Caps, hs0.Caps) {
		t.Errorf("inbound caps mismatch: got %v, want %v", conn1.Caps, hs0.Caps)
	}

	<-done
}

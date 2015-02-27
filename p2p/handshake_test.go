package p2p

import (
	"bytes"
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

func TestEncHandshake(t *testing.T) {
	defer testlog(t).detach()

	prv0, _ := crypto.GenerateKey()
	prv1, _ := crypto.GenerateKey()
	rw0, rw1 := net.Pipe()
	secrets := make(chan secrets)

	go func() {
		pub1s, _ := exportPublicKey(&prv1.PublicKey)
		s, err := outboundEncHandshake(rw0, prv0, pub1s, nil)
		if err != nil {
			t.Errorf("outbound side error: %v", err)
		}
		id1 := discover.PubkeyID(&prv1.PublicKey)
		if s.RemoteID != id1 {
			t.Errorf("outbound side remote ID mismatch")
		}
		secrets <- s
	}()
	go func() {
		s, err := inboundEncHandshake(rw1, prv1, nil)
		if err != nil {
			t.Errorf("inbound side error: %v", err)
		}
		id0 := discover.PubkeyID(&prv0.PublicKey)
		if s.RemoteID != id0 {
			t.Errorf("inbound side remote ID mismatch")
		}
		secrets <- s
	}()

	// get computed secrets from both sides
	t1, t2 := <-secrets, <-secrets
	// don't compare remote node IDs
	t1.RemoteID, t2.RemoteID = discover.NodeID{}, discover.NodeID{}
	// flip MACs on one of them so they compare equal
	t1.EgressMAC, t1.IngressMAC = t1.IngressMAC, t1.EgressMAC
	if !reflect.DeepEqual(t1, t2) {
		t.Errorf("secrets mismatch:\n t1: %#v\n t2: %#v", t1, t2)
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

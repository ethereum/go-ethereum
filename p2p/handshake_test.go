package p2p

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

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
	for i := 0; i < 20; i++ {
		start := time.Now()
		if err := testEncHandshake(nil); err != nil {
			t.Fatalf("i=%d %v", i, err)
		}
		t.Logf("(without token) %d %v\n", i+1, time.Since(start))
	}

	for i := 0; i < 20; i++ {
		tok := make([]byte, shaLen)
		rand.Reader.Read(tok)
		start := time.Now()
		if err := testEncHandshake(tok); err != nil {
			t.Fatalf("i=%d %v", i, err)
		}
		t.Logf("(with token) %d %v\n", i+1, time.Since(start))
	}
}

func testEncHandshake(token []byte) error {
	type result struct {
		side string
		s    secrets
		err  error
	}
	var (
		prv0, _  = crypto.GenerateKey()
		prv1, _  = crypto.GenerateKey()
		rw0, rw1 = net.Pipe()
		output   = make(chan result)
	)

	go func() {
		r := result{side: "initiator"}
		defer func() { output <- r }()

		pub1s := discover.PubkeyID(&prv1.PublicKey)
		r.s, r.err = initiatorEncHandshake(rw0, prv0, pub1s, token)
		if r.err != nil {
			return
		}
		id1 := discover.PubkeyID(&prv1.PublicKey)
		if r.s.RemoteID != id1 {
			r.err = fmt.Errorf("remote ID mismatch: got %v, want: %v", r.s.RemoteID, id1)
		}
	}()
	go func() {
		r := result{side: "receiver"}
		defer func() { output <- r }()

		r.s, r.err = receiverEncHandshake(rw1, prv1, token)
		if r.err != nil {
			return
		}
		id0 := discover.PubkeyID(&prv0.PublicKey)
		if r.s.RemoteID != id0 {
			r.err = fmt.Errorf("remote ID mismatch: got %v, want: %v", r.s.RemoteID, id0)
		}
	}()

	// wait for results from both sides
	r1, r2 := <-output, <-output

	if r1.err != nil {
		return fmt.Errorf("%s side error: %v", r1.side, r1.err)
	}
	if r2.err != nil {
		return fmt.Errorf("%s side error: %v", r2.side, r2.err)
	}

	// don't compare remote node IDs
	r1.s.RemoteID, r2.s.RemoteID = discover.NodeID{}, discover.NodeID{}
	// flip MACs on one of them so they compare equal
	r1.s.EgressMAC, r1.s.IngressMAC = r1.s.IngressMAC, r1.s.EgressMAC
	if !reflect.DeepEqual(r1.s, r2.s) {
		return fmt.Errorf("secrets mismatch:\n t1: %#v\n t2: %#v", r1.s, r2.s)
	}
	return nil
}

func TestSetupConn(t *testing.T) {
	prv0, _ := crypto.GenerateKey()
	prv1, _ := crypto.GenerateKey()
	node0 := &discover.Node{
		ID:  discover.PubkeyID(&prv0.PublicKey),
		IP:  net.IP{1, 2, 3, 4},
		TCP: 33,
	}
	node1 := &discover.Node{
		ID:  discover.PubkeyID(&prv1.PublicKey),
		IP:  net.IP{5, 6, 7, 8},
		TCP: 44,
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
		conn0, err := setupConn(fd0, prv0, hs0, node1, false, nil)
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

	conn1, err := setupConn(fd1, prv1, hs1, nil, false, nil)
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

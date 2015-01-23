package p2p

import (
	"bufio"
	"bytes"
	"crypto/rand"
	// "fmt"
	"io"
	"net"
	"testing"
)

func randomKey(i int) (key []byte) {
	key = make([]byte, i)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		panic(err)
	}
	return
}

func TestEncryption(t *testing.T) {
	var args [][]byte = make([][]byte, 4)
	for i, _ := range args {
		args[i] = randomKey(32)
	}
	var pubkey = randomKey(64)

	var caps []interface{}
	for _, p := range []Cap{Cap{"bzz", 0}, Cap{"shh", 1}, Cap{"eth", 2}} {
		caps = append(caps, p)
	}

	var msg0 = NewMsg(handshakeMsg,
		baseProtocolVersion,
		"ethersphere",
		caps,
		3301,
		pubkey,
	)

	var hs handshake

	conn0, conn1 := net.Pipe()
	rw0, err := NewCryptoMsgRW(bufio.NewReader(conn0), conn0, args[0], args[1], args[2], args[3])
	if err != nil {
		return
	}
	messenger0 := NewMessenger(rw0)

	rw1, err := NewCryptoMsgRW(bufio.NewReader(conn1), conn1, args[0], args[1], args[2], args[3])
	if err != nil {
		return
	}
	messenger1 := NewMessenger(rw1)

	messenger0.WriteC() <- msg0

	messenger1.ReadNextC() <- true

	var msg1 Msg
	select {
	case msg1 = <-messenger1.ReadC():

	case err = <-messenger0.ErrorC():
		t.Errorf("unexpected error on initiator%v", err)

	case err = <-messenger1.ErrorC():
		t.Errorf("unexpected error on receiver %v", err)
	}

	if err = msg1.Decode(&hs); err != nil {
		t.Errorf("rlp decoding error: %v", err)
	}

	if //!bytes.Equal(hs.ListenPort, 3301) ||
	hs.ID != "ethersphere" ||
		len(hs.Caps) != 3 ||
		!bytes.Equal(hs.PublicKey(), pubkey) {
		t.Errorf("mismatch")
	}

}

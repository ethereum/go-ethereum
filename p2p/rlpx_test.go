package p2p

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestRlpxFrameFake(t *testing.T) {
	buf := new(bytes.Buffer)
	secret := crypto.Sha3()
	hash := fakeHash([]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	rw := newRlpxFrameRW(buf, secret, hash, hash)

	golden := unhex(`
000006C2808000000000000000000000
01010101010101010101010101010101
08C40102030400000000000000000000
01010101010101010101010101010101
01010101010101010101010101010101
`)

	// Check WriteMsg. This puts a message into the buffer.
	if err := EncodeMsg(rw, 8, []interface{}{1, 2, 3, 4}); err != nil {
		t.Fatalf("WriteMsg error: %v", err)
	}
	written := buf.Bytes()
	if !bytes.Equal(written, golden) {
		t.Fatalf("output mismatch:\n  got:  %x\n  want: %x", written, golden)
	}

	// Check ReadMsg. It reads the message encoded by WriteMsg, which
	// is equivalent to the golden message above.
	msg, err := rw.ReadMsg()
	if err != nil {
		t.Fatalf("ReadMsg error: %v", err)
	}
	if msg.Size != 5 {
		t.Errorf("msg size mismatch: got %d, want %d", msg.Size, 5)
	}
	if msg.Code != 8 {
		t.Errorf("msg code mismatch: got %d, want %d", msg.Code, 8)
	}
	payload, _ := ioutil.ReadAll(msg.Payload)
	wantPayload := unhex("C401020304")
	if !bytes.Equal(payload, wantPayload) {
		t.Errorf("msg payload mismatch:\ngot  %x\nwant %x", payload, wantPayload)
	}
}

type fakeHash []byte

func (fakeHash) Write(p []byte) (int, error) { return len(p), nil }
func (fakeHash) Reset()                      {}
func (fakeHash) BlockSize() int              { return 0 }

func (h fakeHash) Size() int           { return len(h) }
func (h fakeHash) Sum(b []byte) []byte { return append(b, h...) }

func unhex(str string) []byte {
	b, err := hex.DecodeString(strings.Replace(str, "\n", "", -1))
	if err != nil {
		panic(fmt.Sprintf("invalid hex string: %q", str))
	}
	return b
}

func TestRlpxFrameRW(t *testing.T) {
	var (
		macSecret      = make([]byte, 16)
		egressMACinit  = make([]byte, 32)
		ingressMACinit = make([]byte, 32)
	)
	for _, s := range [][]byte{macSecret, egressMACinit, ingressMACinit} {
		rand.Read(s)
	}

	conn := new(bytes.Buffer)

	em1 := sha3.NewKeccak256()
	em1.Write(egressMACinit)
	im1 := sha3.NewKeccak256()
	im1.Write(ingressMACinit)
	rw1 := newRlpxFrameRW(conn, macSecret, em1, im1)

	em2 := sha3.NewKeccak256()
	em2.Write(ingressMACinit)
	im2 := sha3.NewKeccak256()
	im2.Write(egressMACinit)
	rw2 := newRlpxFrameRW(conn, macSecret, em2, im2)

	// send some messages
	for i := 0; i < 10; i++ {
		// write message into conn buffer
		wmsg := []interface{}{"foo", "bar", strings.Repeat("test", i)}
		err := EncodeMsg(rw1, uint64(i), wmsg)
		if err != nil {
			t.Fatalf("WriteMsg error (i=%d): %v", i, err)
		}

		// read message that rw1 just wrote
		msg, err := rw2.ReadMsg()
		if err != nil {
			t.Fatalf("ReadMsg error (i=%d): %v", i, err)
		}
		if msg.Code != uint64(i) {
			t.Fatalf("msg code mismatch: got %d, want %d", msg.Code, i)
		}
		payload, _ := ioutil.ReadAll(msg.Payload)
		wantPayload, _ := rlp.EncodeToBytes(wmsg)
		if !bytes.Equal(payload, wantPayload) {
			t.Fatalf("msg payload mismatch:\ngot  %x\nwant %x", payload, wantPayload)
		}
	}
}

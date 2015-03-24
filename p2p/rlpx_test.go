package p2p

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestRlpxFrameFake(t *testing.T) {
	buf := new(bytes.Buffer)
	hash := fakeHash([]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	rw := newRlpxFrameRW(buf, secrets{
		AES:        crypto.Sha3(),
		MAC:        crypto.Sha3(),
		IngressMAC: hash,
		EgressMAC:  hash,
	})

	golden := unhex(`
00828ddae471818bb0bfa6b551d1cb42
01010101010101010101010101010101
ba628a4ba590cb43f7848f41c4382885
01010101010101010101010101010101
`)

	// Check WriteMsg. This puts a message into the buffer.
	if err := Send(rw, 8, []uint{1, 2, 3, 4}); err != nil {
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

func TestRlpxFrameRW(t *testing.T) {
	var (
		aesSecret      = make([]byte, 16)
		macSecret      = make([]byte, 16)
		egressMACinit  = make([]byte, 32)
		ingressMACinit = make([]byte, 32)
	)
	for _, s := range [][]byte{aesSecret, macSecret, egressMACinit, ingressMACinit} {
		rand.Read(s)
	}
	conn := new(bytes.Buffer)

	s1 := secrets{
		AES:        aesSecret,
		MAC:        macSecret,
		EgressMAC:  sha3.NewKeccak256(),
		IngressMAC: sha3.NewKeccak256(),
	}
	s1.EgressMAC.Write(egressMACinit)
	s1.IngressMAC.Write(ingressMACinit)
	rw1 := newRlpxFrameRW(conn, s1)

	s2 := secrets{
		AES:        aesSecret,
		MAC:        macSecret,
		EgressMAC:  sha3.NewKeccak256(),
		IngressMAC: sha3.NewKeccak256(),
	}
	s2.EgressMAC.Write(ingressMACinit)
	s2.IngressMAC.Write(egressMACinit)
	rw2 := newRlpxFrameRW(conn, s2)

	// send some messages
	for i := 0; i < 10; i++ {
		// write message into conn buffer
		wmsg := []interface{}{"foo", "bar", strings.Repeat("test", i)}
		err := Send(rw1, uint64(i), wmsg)
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

package p2p

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"runtime"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethutil"
)

func TestNewMsg(t *testing.T) {
	msg := NewMsg(3, 1, "000")
	if msg.Code != 3 {
		t.Errorf("incorrect code %d, want %d", msg.Code)
	}
	if msg.Size != 5 {
		t.Errorf("incorrect size %d, want %d", msg.Size, 5)
	}
	pl, _ := ioutil.ReadAll(msg.Payload)
	expect := []byte{0x01, 0x83, 0x30, 0x30, 0x30}
	if !bytes.Equal(pl, expect) {
		t.Errorf("incorrect payload content, got %x, want %x", pl, expect)
	}
}

func TestEncodeDecodeMsg(t *testing.T) {
	msg := NewMsg(3, 1, "000")
	buf := new(bytes.Buffer)
	if err := writeMsg(buf, msg); err != nil {
		t.Fatalf("encodeMsg error: %v", err)
	}
	// t.Logf("encoded: %x", buf.Bytes())

	decmsg, err := readMsg(buf)
	if err != nil {
		t.Fatalf("readMsg error: %v", err)
	}
	if decmsg.Code != 3 {
		t.Errorf("incorrect code %d, want %d", decmsg.Code, 3)
	}
	if decmsg.Size != 5 {
		t.Errorf("incorrect size %d, want %d", decmsg.Size, 5)
	}

	var data struct {
		I uint
		S string
	}
	if err := decmsg.Decode(&data); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if data.I != 1 {
		t.Errorf("incorrect data.I: got %v, expected %d", data.I, 1)
	}
	if data.S != "000" {
		t.Errorf("incorrect data.S: got %q, expected %q", data.S, "000")
	}
}

func TestDecodeRealMsg(t *testing.T) {
	data := ethutil.Hex2Bytes("2240089100000080f87e8002b5457468657265756d282b2b292f5065657220536572766572204f6e652f76302e372e382f52656c656173652f4c696e75782f672b2bc082765fb84086dd80b7aefd6a6d2e3b93f4f300a86bfb6ef7bdc97cb03f793db6bb")
	msg, err := readMsg(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.Code != 0 {
		t.Errorf("incorrect code %d, want %d", msg.Code, 0)
	}
}

func ExampleMsgPipe() {
	rw1, rw2 := MsgPipe()
	go func() {
		EncodeMsg(rw1, 8, []byte{0, 0})
		EncodeMsg(rw1, 5, []byte{1, 1})
		rw1.Close()
	}()

	for {
		msg, err := rw2.ReadMsg()
		if err != nil {
			break
		}
		var data [1][]byte
		msg.Decode(&data)
		fmt.Printf("msg: %d, %x\n", msg.Code, data[0])
	}
	// Output:
	// msg: 8, 0000
	// msg: 5, 0101
}

func TestMsgPipeUnblockWrite(t *testing.T) {
loop:
	for i := 0; i < 100; i++ {
		rw1, rw2 := MsgPipe()
		done := make(chan struct{})
		go func() {
			if err := EncodeMsg(rw1, 1); err == nil {
				t.Error("EncodeMsg returned nil error")
			} else if err != ErrPipeClosed {
				t.Error("EncodeMsg returned wrong error: got %v, want %v", err, ErrPipeClosed)
			}
			close(done)
		}()

		// this call should ensure that EncodeMsg is waiting to
		// deliver sometimes. if this isn't done, Close is likely to
		// be executed before EncodeMsg starts and then we won't test
		// all the cases.
		runtime.Gosched()

		rw2.Close()
		select {
		case <-done:
		case <-time.After(200 * time.Millisecond):
			t.Errorf("write didn't unblock")
			break loop
		}
	}
}

// This test should panic if concurrent close isn't implemented correctly.
func TestMsgPipeConcurrentClose(t *testing.T) {
	rw1, _ := MsgPipe()
	for i := 0; i < 10; i++ {
		go rw1.Close()
	}
}

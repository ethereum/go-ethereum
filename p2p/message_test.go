package p2p

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"runtime"
	"testing"
	"time"
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

func TestEOFSignal(t *testing.T) {
	rb := make([]byte, 10)

	// empty reader
	eof := make(chan struct{}, 1)
	sig := &eofSignal{new(bytes.Buffer), 0, eof}
	if n, err := sig.Read(rb); n != 0 || err != io.EOF {
		t.Errorf("Read returned unexpected values: (%v, %v)", n, err)
	}
	select {
	case <-eof:
	default:
		t.Error("EOF chan not signaled")
	}

	// count before error
	eof = make(chan struct{}, 1)
	sig = &eofSignal{bytes.NewBufferString("aaaaaaaa"), 4, eof}
	if n, err := sig.Read(rb); n != 4 || err != nil {
		t.Errorf("Read returned unexpected values: (%v, %v)", n, err)
	}
	select {
	case <-eof:
	default:
		t.Error("EOF chan not signaled")
	}

	// error before count
	eof = make(chan struct{}, 1)
	sig = &eofSignal{bytes.NewBufferString("aaaa"), 999, eof}
	if n, err := sig.Read(rb); n != 4 || err != nil {
		t.Errorf("Read returned unexpected values: (%v, %v)", n, err)
	}
	if n, err := sig.Read(rb); n != 0 || err != io.EOF {
		t.Errorf("Read returned unexpected values: (%v, %v)", n, err)
	}
	select {
	case <-eof:
	default:
		t.Error("EOF chan not signaled")
	}

	// no signal if neither occurs
	eof = make(chan struct{}, 1)
	sig = &eofSignal{bytes.NewBufferString("aaaaaaaaaaaaaaaaaaaaa"), 999, eof}
	if n, err := sig.Read(rb); n != 10 || err != nil {
		t.Errorf("Read returned unexpected values: (%v, %v)", n, err)
	}
	select {
	case <-eof:
		t.Error("unexpected EOF signal")
	default:
	}
}

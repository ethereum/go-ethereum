package p2p

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/rlp"
)

// parameters for frameRW
const (
	// maximum time allowed for reading a message header.
	// this is effectively the amount of time a connection can be idle.
	frameReadTimeout = 1 * time.Minute

	// maximum time allowed for reading the payload data of a message.
	// this is shorter than (and distinct from) frameReadTimeout because
	// the connection is not considered idle while a message is transferred.
	// this also limits the payload size of messages to how much the connection
	// can transfer within the timeout.
	payloadReadTimeout = 5 * time.Second

	// maximum amount of time allowed for writing a complete message.
	msgWriteTimeout = 5 * time.Second

	// messages smaller than this many bytes will be read at
	// once before passing them to a protocol. this increases
	// concurrency in the processing.
	wholePayloadSize = 64 * 1024
)

// Msg defines the structure of a p2p message.
//
// Note that a Msg can only be sent once since the Payload reader is
// consumed during sending. It is not possible to create a Msg and
// send it any number of times. If you want to reuse an encoded
// structure, encode the payload into a byte array and create a
// separate Msg with a bytes.Reader as Payload for each send.
type Msg struct {
	Code    uint64
	Size    uint32 // size of the paylod
	Payload io.Reader
}

// NewMsg creates an RLP-encoded message with the given code.
func NewMsg(code uint64, params ...interface{}) Msg {
	buf := new(bytes.Buffer)
	for _, p := range params {
		buf.Write(ethutil.Encode(p))
	}
	return Msg{Code: code, Size: uint32(buf.Len()), Payload: buf}
}

func encodePayload(params ...interface{}) []byte {
	buf := new(bytes.Buffer)
	for _, p := range params {
		buf.Write(ethutil.Encode(p))
	}
	return buf.Bytes()
}

// Decode parse the RLP content of a message into
// the given value, which must be a pointer.
//
// For the decoding rules, please see package rlp.
func (msg Msg) Decode(val interface{}) error {
	s := rlp.NewListStream(msg.Payload, uint64(msg.Size))
	if err := s.Decode(val); err != nil {
		return newPeerError(errInvalidMsg, "(code %#x) (size %d) %v", msg.Code, msg.Size, err)
	}
	return nil
}

func (msg Msg) String() string {
	return fmt.Sprintf("msg #%v (%v bytes)", msg.Code, msg.Size)
}

// Discard reads any remaining payload data into a black hole.
func (msg Msg) Discard() error {
	_, err := io.Copy(ioutil.Discard, msg.Payload)
	return err
}

type MsgReader interface {
	ReadMsg() (Msg, error)
}

type MsgWriter interface {
	// WriteMsg sends a message. It will block until the message's
	// Payload has been consumed by the other end.
	//
	// Note that messages can be sent only once because their
	// payload reader is drained.
	WriteMsg(Msg) error
}

// MsgReadWriter provides reading and writing of encoded messages.
// Implementations should ensure that ReadMsg and WriteMsg can be
// called simultaneously from multiple goroutines.
type MsgReadWriter interface {
	MsgReader
	MsgWriter
}

// EncodeMsg writes an RLP-encoded message with the given code and
// data elements.
func EncodeMsg(w MsgWriter, code uint64, data ...interface{}) error {
	return w.WriteMsg(NewMsg(code, data...))
}

// lockedRW wraps a MsgReadWriter with locks around
// ReadMsg and WriteMsg.
type lockedRW struct {
	rmu, wmu sync.Mutex
	wrapped  MsgReadWriter
}

func (rw *lockedRW) ReadMsg() (Msg, error) {
	rw.rmu.Lock()
	defer rw.rmu.Unlock()
	return rw.wrapped.ReadMsg()
}

func (rw *lockedRW) WriteMsg(msg Msg) error {
	rw.wmu.Lock()
	defer rw.wmu.Unlock()
	return rw.wrapped.WriteMsg(msg)
}

// eofSignal wraps a reader with eof signaling. the eof channel is
// closed when the wrapped reader returns an error or when count bytes
// have been read.
type eofSignal struct {
	wrapped io.Reader
	count   uint32 // number of bytes left
	eof     chan<- struct{}
}

// note: when using eofSignal to detect whether a message payload
// has been read, Read might not be called for zero sized messages.
func (r *eofSignal) Read(buf []byte) (int, error) {
	if r.count == 0 {
		if r.eof != nil {
			r.eof <- struct{}{}
			r.eof = nil
		}
		return 0, io.EOF
	}

	max := len(buf)
	if int(r.count) < len(buf) {
		max = int(r.count)
	}
	n, err := r.wrapped.Read(buf[:max])
	r.count -= uint32(n)
	if (err != nil || r.count == 0) && r.eof != nil {
		r.eof <- struct{}{} // tell Peer that msg has been consumed
		r.eof = nil
	}
	return n, err
}

// MsgPipe creates a message pipe. Reads on one end are matched
// with writes on the other. The pipe is full-duplex, both ends
// implement MsgReadWriter.
func MsgPipe() (*MsgPipeRW, *MsgPipeRW) {
	var (
		c1, c2  = make(chan Msg), make(chan Msg)
		closing = make(chan struct{})
		closed  = new(int32)
		rw1     = &MsgPipeRW{c1, c2, closing, closed}
		rw2     = &MsgPipeRW{c2, c1, closing, closed}
	)
	return rw1, rw2
}

// ErrPipeClosed is returned from pipe operations after the
// pipe has been closed.
var ErrPipeClosed = errors.New("p2p: read or write on closed message pipe")

// MsgPipeRW is an endpoint of a MsgReadWriter pipe.
type MsgPipeRW struct {
	w       chan<- Msg
	r       <-chan Msg
	closing chan struct{}
	closed  *int32
}

// WriteMsg sends a messsage on the pipe.
// It blocks until the receiver has consumed the message payload.
func (p *MsgPipeRW) WriteMsg(msg Msg) error {
	if atomic.LoadInt32(p.closed) == 0 {
		consumed := make(chan struct{}, 1)
		msg.Payload = &eofSignal{msg.Payload, msg.Size, consumed}
		select {
		case p.w <- msg:
			if msg.Size > 0 {
				// wait for payload read or discard
				<-consumed
			}
			return nil
		case <-p.closing:
		}
	}
	return ErrPipeClosed
}

// ReadMsg returns a message sent on the other end of the pipe.
func (p *MsgPipeRW) ReadMsg() (Msg, error) {
	if atomic.LoadInt32(p.closed) == 0 {
		select {
		case msg := <-p.r:
			return msg, nil
		case <-p.closing:
		}
	}
	return Msg{}, ErrPipeClosed
}

// Close unblocks any pending ReadMsg and WriteMsg calls on both ends
// of the pipe. They will return ErrPipeClosed. Note that Close does
// not interrupt any reads from a message payload.
func (p *MsgPipeRW) Close() error {
	if atomic.AddInt32(p.closed, 1) != 1 {
		// someone else is already closing
		atomic.StoreInt32(p.closed, 1) // avoid overflow
		return nil
	}
	close(p.closing)
	return nil
}

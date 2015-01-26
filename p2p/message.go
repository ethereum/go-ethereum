package p2p

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/rlp"
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
	Size    uint32 // size of the payload
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

// this should probably use the new rlp encoding
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

func NewMsgFromRLP(size uint32, r rlp.ByteReader) (msg Msg, err error) {
	// decode start of RLP message to get the message code
	posr := &postrack{r, 0}
	s := rlp.NewStream(posr)
	if _, err := s.List(); err != nil {
		return msg, err
	}
	code, err := s.Uint()
	if err != nil {
		return msg, err
	}
	payloadsize := size - posr.p
	return Msg{code, payloadsize, io.LimitReader(r, int64(payloadsize))}, nil
}

/*
MsgReadWriter is an interface for reading and writing messages
It is aware of message structure and knows how to encode/decode
*/

type MsgReader interface {
	ReadMsg() (Msg, error)
}

type MsgWriter interface {
	// WriteMsg sends a message. It will block until the message's
	// Payload has been consumed by the other end.
	//
	// Note that messages can be sent only once.
	WriteMsg(Msg) error
}

// MsgReadWriter provides reading and writing of encoded messages.
type MsgReadWriter interface {
	MsgReader
	MsgWriter
}

// EncodeMsg should be explicit in the interface and should have the MsgWriter
// as receiver

// EncodeMsg writes an RLP-encoded message with the given code and
// data elements.
func EncodeMsg(w MsgWriter, code uint64, data ...interface{}) error {
	return w.WriteMsg(NewMsg(code, data...))
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
		msg.Payload = &eofSignal{msg.Payload, int64(msg.Size), consumed}
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

// eofSignal wraps a reader with eof signaling. the eof channel is
// closed when the wrapped reader returns an error or when count bytes
// have been read.
//
type eofSignal struct {
	wrapped io.Reader
	count   int64
	eof     chan<- struct{}
}

// note: when using eofSignal to detect whether a message payload
// has been read, Read might not be called for zero sized messages.

func (r *eofSignal) Read(buf []byte) (int, error) {
	n, err := r.wrapped.Read(buf)
	r.count -= int64(n)
	if (err != nil || r.count <= 0) && r.eof != nil {
		r.eof <- struct{}{} // tell Peer that msg has been consumed
		r.eof = nil
	}
	return n, err
}

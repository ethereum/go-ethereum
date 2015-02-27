package p2p

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
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

// frameRW is a MsgReadWriter that reads and writes devp2p message frames.
// As required by the interface, ReadMsg and WriteMsg can be called from
// multiple goroutines.
type frameRW struct {
	net.Conn // make Conn methods available. be careful.
	bufconn  *bufio.ReadWriter

	// this channel is used to 'lend' bufconn to a caller of ReadMsg
	// until the message payload has been consumed. the channel
	// receives a value when EOF is reached on the payload, unblocking
	// a pending call to ReadMsg.
	rsync chan struct{}

	// this mutex guards writes to bufconn.
	writeMu sync.Mutex
}

func newFrameRW(conn net.Conn, timeout time.Duration) *frameRW {
	rsync := make(chan struct{}, 1)
	rsync <- struct{}{}
	return &frameRW{
		Conn:    conn,
		bufconn: bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
		rsync:   rsync,
	}
}

var magicToken = []byte{34, 64, 8, 145}

func (rw *frameRW) WriteMsg(msg Msg) error {
	rw.writeMu.Lock()
	defer rw.writeMu.Unlock()
	rw.SetWriteDeadline(time.Now().Add(msgWriteTimeout))
	if err := writeMsg(rw.bufconn, msg); err != nil {
		return err
	}
	return rw.bufconn.Flush()
}

func writeMsg(w io.Writer, msg Msg) error {
	// TODO: handle case when Size + len(code) + len(listhdr) overflows uint32
	code := ethutil.Encode(uint32(msg.Code))
	listhdr := makeListHeader(msg.Size + uint32(len(code)))
	payloadLen := uint32(len(listhdr)) + uint32(len(code)) + msg.Size

	start := make([]byte, 8)
	copy(start, magicToken)
	binary.BigEndian.PutUint32(start[4:], payloadLen)

	for _, b := range [][]byte{start, listhdr, code} {
		if _, err := w.Write(b); err != nil {
			return err
		}
	}
	_, err := io.CopyN(w, msg.Payload, int64(msg.Size))
	return err
}

func makeListHeader(length uint32) []byte {
	if length < 56 {
		return []byte{byte(length + 0xc0)}
	}
	enc := big.NewInt(int64(length)).Bytes()
	lenb := byte(len(enc)) + 0xf7
	return append([]byte{lenb}, enc...)
}

func (rw *frameRW) ReadMsg() (msg Msg, err error) {
	<-rw.rsync // wait until bufconn is ours

	rw.SetReadDeadline(time.Now().Add(frameReadTimeout))

	// read magic and payload size
	start := make([]byte, 8)
	if _, err = io.ReadFull(rw.bufconn, start); err != nil {
		return msg, err
	}
	if !bytes.HasPrefix(start, magicToken) {
		return msg, fmt.Errorf("bad magic token %x", start[:4])
	}
	size := binary.BigEndian.Uint32(start[4:])

	// decode start of RLP message to get the message code
	posr := &postrack{rw.bufconn, 0}
	s := rlp.NewStream(posr)
	if _, err := s.List(); err != nil {
		return msg, err
	}
	msg.Code, err = s.Uint()
	if err != nil {
		return msg, err
	}
	msg.Size = size - posr.p

	rw.SetReadDeadline(time.Now().Add(payloadReadTimeout))

	if msg.Size <= wholePayloadSize {
		// msg is small, read all of it and move on to the next message.
		pbuf := make([]byte, msg.Size)
		if _, err := io.ReadFull(rw.bufconn, pbuf); err != nil {
			return msg, err
		}
		rw.rsync <- struct{}{} // bufconn is available again
		msg.Payload = bytes.NewReader(pbuf)
	} else {
		// lend bufconn to the caller until it has
		// consumed the payload. eofSignal will send a value
		// on rw.rsync when EOF is reached.
		pr := &eofSignal{rw.bufconn, msg.Size, rw.rsync}
		msg.Payload = pr
	}
	return msg, nil
}

// postrack wraps an rlp.ByteReader with a position counter.
type postrack struct {
	r rlp.ByteReader
	p uint32
}

func (r *postrack) Read(buf []byte) (int, error) {
	n, err := r.r.Read(buf)
	r.p += uint32(n)
	return n, err
}

func (r *postrack) ReadByte() (byte, error) {
	b, err := r.r.ReadByte()
	if err == nil {
		r.p++
	}
	return b, err
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

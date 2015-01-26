package p2p

import (
	"bytes"
	"encoding/binary"
	"io"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	// maximum amount of time allowed for reading a message
	msgReadTimeout = 5 * time.Second
	// maximum amount of time allowed for writing a message
	msgWriteTimeout = 5 * time.Second
	// messages smaller than this many bytes will be read at
	// once before passing them to a protocol.
	wholePayloadSize = 64 * 1024
)

var magicToken = []byte{34, 64, 8, 145}

/*
 A MsgChanReadWriter implementation will typically sit on a multiplexed peer connection and runs a single read and a write loop without need to use locking.

 It passes on incoming messages to its channel ReadC()
 the peer runs a dispatch loop that figures out which protocol to forward the message to.

 The channel for outcgoing messages (WriteC) is simply shared between the individual MsgReadWriter instances for each protocol
*/

type MsgChanReadWriter interface {
	ReadC() chan Msg
	WriteC() chan Msg
	ErrorC() chan error
	ReadNextC() chan bool
	Close()
}

type Messenger struct {
	in      chan Msg
	out     chan Msg
	errc    chan error
	unblock chan bool
	rw      MsgReadWriter
}

/*
Messenger is a simple implementation of a read and write loop using a MsgReadWriter to encode/decode individual messages
This MsgReadWriter can implement parsing from/to any kind of packet structure and employ encryption and authentication
*/
func NewMessenger(rw MsgReadWriter) *Messenger {
	self := &Messenger{
		in:      make(chan Msg),
		out:     make(chan Msg),
		errc:    make(chan error),
		unblock: make(chan bool, 1),
		rw:      rw,
	}
	go self.readLoop()
	go self.writeLoop()
	return self
}

func (self *Messenger) Close() {
	close(self.unblock)
	close(self.out)
}

func (self *Messenger) ReadC() chan Msg {
	return self.in
}

func (self *Messenger) WriteC() chan Msg {
	return self.out
}

func (self *Messenger) ErrorC() chan error {
	return self.errc
}

// ReadNextC <- true must be called before the next read is attempted
func (self *Messenger) ReadNextC() chan bool {
	return self.unblock
}

func (self *Messenger) readLoop() {
	for _ = range self.unblock {
		if msg, err := self.rw.ReadMsg(); err != nil {
			self.errc <- err
		} else {
			self.in <- msg
		}
	}
	close(self.errc)
}

func (self *Messenger) writeLoop() {
	for msg := range self.out {
		if err := self.rw.WriteMsg(msg); err != nil {
			self.errc <- newPeerError(errWrite, "%v", err)
		}

	}
}

/*
MsgReadWriter is an interface for reading and writing messages
It is aware of message structure and knows how to encode/decode

MsgRW is a simple encoder implementing MsgReadWriter
It complies with the legacy devp2p packet structure and no encryption or authentication
*/
type MsgRW struct {
	r rlp.ByteReader // this is implemented by bufio.ReadWriter
	// r io.Reader
	w io.Writer
}

func NewMsgRW(r rlp.ByteReader, w io.Writer) (*MsgRW, error) {
	return &MsgRW{
		r: r,
		w: w,
	}, nil
}

func (self *MsgRW) WriteMsg(msg Msg) error {

	// TODO: handle case when Size + len(code) + len(listhdr) overflows uint32
	code := ethutil.Encode(uint32(msg.Code))
	listhdr := makeListHeader(msg.Size + uint32(len(code)))
	payloadLen := uint32(len(listhdr)) + uint32(len(code)) + msg.Size

	start := make([]byte, 8)
	copy(start, magicToken)
	binary.BigEndian.PutUint32(start[4:], payloadLen)

	for _, b := range [][]byte{start, listhdr, code} {
		if _, err := self.w.Write(b); err != nil {
			return err
		}
	}
	_, err := io.CopyN(self.w, msg.Payload, int64(msg.Size))
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

// readMsg reads a message header from r.
// It takes an rlp.ByteReader to ensure that the decoding doesn't buffer.
func (self *MsgRW) ReadMsg() (msg Msg, err error) {

	// read magic and payload size
	start := make([]byte, 8)
	if _, err = io.ReadFull(self.r, start); err != nil {
		return msg, newPeerError(errRead, "%v", err)
	}

	if !bytes.HasPrefix(start, magicToken) {
		return msg, newPeerError(errMagicTokenMismatch, "got %x, want %x", start[:4], magicToken)
	}
	size := binary.BigEndian.Uint32(start[4:])
	return NewMsgFromRLP(size, self.r)
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

// proto will embed the same writer channel as given to the readwriter
// in the legacy code it knows about the code offset
// no need to go through peer for writing , so do not need to embed peer as field
type proto struct {
	name            string
	in, out         chan Msg
	maxcode, offset uint64
}

// WriteMsg proto implements MsgWriter interface
func (rw *proto) WriteMsg(msg Msg) error {
	if msg.Code >= rw.maxcode {
		return newPeerError(errInvalidMsgCode, "not handled")
	}
	msg.Code += rw.offset
	rw.out <- msg
	return nil
}

func (rw *proto) EncodeMsg(code uint64, data ...interface{}) error {
	return rw.WriteMsg(NewMsg(code, data...))
}

func (rw *proto) ReadMsg() (Msg, error) {
	msg, ok := <-rw.in
	if !ok {
		return msg, io.EOF
	}
	msg.Code -= rw.offset
	return msg, nil
}

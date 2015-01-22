package p2p

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"net"
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
 writeMsg, readMsg (makeListHeader) should all be internal to the default legacy
 MsgReadWriter implementation

 A MsgReadWriter implementation will typically sit on a multiplexed peer connection and runs a single read and a write loop without need to use locking.

 It passes on incoming messages to its channel picked up by the interface methods Read
 the peer runs a dispatch loop that figures out which protocol to forward the message to.

  Because framing and header structure will change there will be hardly any overlap with the new code so I do not abstract readers any further.
*/

type MsgChanReadWriter interface {
	ReadC() chan Msg
	WriteC() chan Msg
	ErrorC() chan error
	ReadNextC() chan bool
	Close()
}

type Messenger struct {
	msgReadTimeout  time.Duration
	msgWriteTimeout time.Duration
	in              chan Msg
	out             chan Msg
	errc            chan error
	unblock         chan bool
	conn            net.Conn
	bufconn         *bufio.ReadWriter
}

func NewMessenger(conn net.Conn) *Messenger {
	self := &Messenger{
		msgReadTimeout:  msgReadTimeout,
		msgWriteTimeout: msgWriteTimeout,
		in:              make(chan Msg),
		out:             make(chan Msg),
		errc:            make(chan error),
		unblock:         make(chan bool, 1),
		conn:            conn,
		bufconn:         bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
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

func (self *Messenger) ReadNextC() chan bool {
	return self.unblock
}

func (self *Messenger) readLoop() {
	for _ = range self.unblock {
		self.conn.SetReadDeadline(time.Now().Add(self.msgReadTimeout))
		if msg, err := readMsg(self.bufconn); err != nil {
			self.errc <- err
		} else {
			self.in <- msg
		}
	}
	close(self.errc)
}

func (self *Messenger) writeLoop() {
	for msg := range self.out {
		self.conn.SetWriteDeadline(time.Now().Add(self.msgWriteTimeout))
		if err := writeMsg(self.bufconn, msg); err != nil {
			self.errc <- newPeerError(errWrite, "%v", err)
		}
		self.bufconn.Flush()
	}
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

// readMsg reads a message header from r.
// It takes an rlp.ByteReader to ensure that the decoding doesn't buffer.
func readMsg(r rlp.ByteReader) (msg Msg, err error) {
	// read magic and payload size
	start := make([]byte, 8)
	if _, err = io.ReadFull(r, start); err != nil {
		return msg, newPeerError(errRead, "%v", err)
	}
	if !bytes.HasPrefix(start, magicToken) {
		return msg, newPeerError(errMagicTokenMismatch, "got %x, want %x", start[:4], magicToken)
	}
	size := binary.BigEndian.Uint32(start[4:])

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

// this duplicates functionality of proto.WriteMsg
// if we need this for broadcasting via a server interface then
// simply call the appropriate Write function of the protocol RW
// writeProtoMsg sends the given message on behalf of the given named protocol.
func (p *Peer) writeProtoMsg(protoName string, msg Msg) error {
	p.runlock.RLock()
	proto, ok := p.running[protoName]
	p.runlock.RUnlock()
	if !ok {
		return fmt.Errorf("protocol %s not handled by peer", protoName)
	}
	return proto.WriteMsg(msg)
}

// proto will embed the same writer channel as given to the readwriter
// in the legacy code it knows about the code offset
// no need to go through peer for writing , so do not need to embed peer as field
type proto struct {
	name            string
	in, out         chan Msg
	maxcode, offset uint64
}

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

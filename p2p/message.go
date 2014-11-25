package p2p

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"math/big"

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

// Value returns the decoded RLP payload items in a message.
func (msg Msg) Value() (*ethutil.Value, error) {
	var v []interface{}
	err := msg.Decode(&v)
	return ethutil.NewValue(v), err
}

// Decode parse the RLP content of a message into
// the given value, which must be a pointer.
//
// For the decoding rules, please see package rlp.
func (msg Msg) Decode(val interface{}) error {
	s := rlp.NewListStream(msg.Payload, uint64(msg.Size))
	return s.Decode(val)
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
	// WriteMsg sends an existing message.
	// The Payload reader of the message is consumed.
	// Note that messages can be sent only once.
	WriteMsg(Msg) error

	// EncodeMsg writes an RLP-encoded message with the given
	// code and data elements.
	EncodeMsg(code uint64, data ...interface{}) error
}

// MsgReadWriter provides reading and writing of encoded messages.
type MsgReadWriter interface {
	MsgReader
	MsgWriter
}

// MsgLoop reads messages off the given reader and
// calls the handler function for each decoded message until
// it returns an error or the peer connection is closed.
//
// If a message is larger than the given maximum size,
// MsgLoop returns an appropriate error.
func MsgLoop(r MsgReader, maxsize uint32, f func(code uint64, data *ethutil.Value) error) error {
	for {
		msg, err := r.ReadMsg()
		if err != nil {
			return err
		}
		if msg.Size > maxsize {
			return newPeerError(errInvalidMsg, "size %d exceeds maximum size of %d", msg.Size, maxsize)
		}
		value, err := msg.Value()
		if err != nil {
			return err
		}
		if err := f(msg.Code, value); err != nil {
			return err
		}
	}
}

var magicToken = []byte{34, 64, 8, 145}

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

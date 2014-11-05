package p2p

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"

	"github.com/ethereum/go-ethereum/ethutil"
)

type MsgCode uint64

// Msg defines the structure of a p2p message.
//
// Note that a Msg can only be sent once since the Payload reader is
// consumed during sending. It is not possible to create a Msg and
// send it any number of times. If you want to reuse an encoded
// structure, encode the payload into a byte array and create a
// separate Msg with a bytes.Reader as Payload for each send.
type Msg struct {
	Code    MsgCode
	Size    uint32 // size of the paylod
	Payload io.Reader
}

// NewMsg creates an RLP-encoded message with the given code.
func NewMsg(code MsgCode, params ...interface{}) Msg {
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

// Data returns the decoded RLP payload items in a message.
func (msg Msg) Data() (*ethutil.Value, error) {
	// TODO: avoid copying when we have a better RLP decoder
	buf := new(bytes.Buffer)
	var s []interface{}
	if _, err := buf.ReadFrom(msg.Payload); err != nil {
		return nil, err
	}
	for buf.Len() > 0 {
		s = append(s, ethutil.DecodeWithReader(buf))
	}
	return ethutil.NewValue(s), nil
}

// Discard reads any remaining payload data into a black hole.
func (msg Msg) Discard() error {
	_, err := io.Copy(ioutil.Discard, msg.Payload)
	return err
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

type byteReader interface {
	io.Reader
	io.ByteReader
}

// readMsg reads a message header from r.
func readMsg(r byteReader) (msg Msg, err error) {
	// read magic and payload size
	start := make([]byte, 8)
	if _, err = io.ReadFull(r, start); err != nil {
		return msg, NewPeerError(ReadError, "%v", err)
	}
	if !bytes.HasPrefix(start, magicToken) {
		return msg, NewPeerError(MagicTokenMismatch, "got %x, want %x", start[:4], magicToken)
	}
	size := binary.BigEndian.Uint32(start[4:])

	// decode start of RLP message to get the message code
	_, hdrlen, err := readListHeader(r)
	if err != nil {
		return msg, err
	}
	code, codelen, err := readMsgCode(r)
	if err != nil {
		return msg, err
	}

	rlpsize := size - hdrlen - codelen
	return Msg{
		Code:    code,
		Size:    rlpsize,
		Payload: io.LimitReader(r, int64(rlpsize)),
	}, nil
}

// readListHeader reads an RLP list header from r.
func readListHeader(r byteReader) (len uint64, hdrlen uint32, err error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, 0, err
	}
	if b < 0xC0 {
		return 0, 0, fmt.Errorf("expected list start byte >= 0xC0, got %x", b)
	} else if b < 0xF7 {
		len = uint64(b - 0xc0)
		hdrlen = 1
	} else {
		lenlen := b - 0xF7
		lenbuf := make([]byte, 8)
		if _, err := io.ReadFull(r, lenbuf[8-lenlen:]); err != nil {
			return 0, 0, err
		}
		len = binary.BigEndian.Uint64(lenbuf)
		hdrlen = 1 + uint32(lenlen)
	}
	return len, hdrlen, nil
}

// readUint reads an RLP-encoded unsigned integer from r.
func readMsgCode(r byteReader) (code MsgCode, codelen uint32, err error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, 0, err
	}
	if b < 0x80 {
		return MsgCode(b), 1, nil
	} else if b < 0x89 { // max length for uint64 is 8 bytes
		codelen = uint32(b - 0x80)
		if codelen == 0 {
			return 0, 1, nil
		}
		buf := make([]byte, 8)
		if _, err := io.ReadFull(r, buf[8-codelen:]); err != nil {
			return 0, 0, err
		}
		return MsgCode(binary.BigEndian.Uint64(buf)), codelen, nil
	}
	return 0, 0, fmt.Errorf("bad RLP type for message code: %x", b)
}

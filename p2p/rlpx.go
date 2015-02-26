package p2p

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"errors"
	"hash"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

var (
	zeroHeader = []byte{0xC2, 0x80, 0x80}
	zero16     = make([]byte, 16)
)

type rlpxFrameRW struct {
	conn io.ReadWriter

	macCipher  cipher.Block
	egressMAC  hash.Hash
	ingressMAC hash.Hash
}

func newRlpxFrameRW(conn io.ReadWriter, macSecret []byte, egressMAC, ingressMAC hash.Hash) *rlpxFrameRW {
	cipher, err := aes.NewCipher(macSecret)
	if err != nil {
		panic("invalid macSecret: " + err.Error())
	}
	return &rlpxFrameRW{conn: conn, macCipher: cipher, egressMAC: egressMAC, ingressMAC: ingressMAC}
}

func (rw *rlpxFrameRW) WriteMsg(msg Msg) error {
	ptype, _ := rlp.EncodeToBytes(msg.Code)

	// write header
	headbuf := make([]byte, 32)
	fsize := uint32(len(ptype)) + msg.Size
	putInt24(fsize, headbuf) // TODO: check overflow
	copy(headbuf[3:], zeroHeader)
	copy(headbuf[16:], updateHeaderMAC(rw.egressMAC, rw.macCipher, headbuf[:16]))
	if _, err := rw.conn.Write(headbuf); err != nil {
		return err
	}

	// write frame, updating the egress MAC while writing to conn.
	tee := io.MultiWriter(rw.conn, rw.egressMAC)
	if _, err := tee.Write(ptype); err != nil {
		return err
	}
	if _, err := io.Copy(tee, msg.Payload); err != nil {
		return err
	}
	if padding := fsize % 16; padding > 0 {
		if _, err := tee.Write(zero16[:16-padding]); err != nil {
			return err
		}
	}

	// write packet-mac. egress MAC is up to date because
	// frame content was written to it as well.
	_, err := rw.conn.Write(rw.egressMAC.Sum(nil))
	return err
}

func (rw *rlpxFrameRW) ReadMsg() (msg Msg, err error) {
	// read the header
	headbuf := make([]byte, 32)
	if _, err := io.ReadFull(rw.conn, headbuf); err != nil {
		return msg, err
	}
	fsize := readInt24(headbuf)
	// ignore protocol type for now
	shouldMAC := updateHeaderMAC(rw.ingressMAC, rw.macCipher, headbuf[:16])
	if !hmac.Equal(shouldMAC[:16], headbuf[16:]) {
		return msg, errors.New("bad header MAC")
	}

	// read the frame content
	framebuf := make([]byte, fsize)
	if _, err := io.ReadFull(rw.conn, framebuf); err != nil {
		return msg, err
	}
	rw.ingressMAC.Write(framebuf)
	if padding := fsize % 16; padding > 0 {
		if _, err := io.CopyN(rw.ingressMAC, rw.conn, int64(16-padding)); err != nil {
			return msg, err
		}
	}
	// read and validate frame MAC. we can re-use headbuf for that.
	if _, err := io.ReadFull(rw.conn, headbuf); err != nil {
		return msg, err
	}
	if !hmac.Equal(rw.ingressMAC.Sum(nil), headbuf) {
		return msg, errors.New("bad frame MAC")
	}

	// decode message code
	content := bytes.NewReader(framebuf)
	if err := rlp.Decode(content, &msg.Code); err != nil {
		return msg, err
	}
	msg.Size = uint32(content.Len())
	msg.Payload = content
	return msg, nil
}

func updateHeaderMAC(mac hash.Hash, block cipher.Block, header []byte) []byte {
	aesbuf := make([]byte, aes.BlockSize)
	block.Encrypt(aesbuf, mac.Sum(nil))
	for i := range aesbuf {
		aesbuf[i] ^= header[i]
	}
	mac.Write(aesbuf)
	return mac.Sum(nil)
}

func readInt24(b []byte) uint32 {
	return uint32(b[2]) | uint32(b[1])<<8 | uint32(b[0])<<16
}

func putInt24(v uint32, b []byte) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}

// Copyright 2020 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rlpx

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"errors"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/golang/snappy"
	"hash"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"time"
)

const (
	maxUint24 = ^uint32(0) >> 8
)

// errPlainMessageTooLarge is returned if a decompressed message length exceeds
// the allowed 24 bits (i.e. length >= 16MB).
var errPlainMessageTooLarge = errors.New("message length >= 16MB")

type Rlpx struct { // TODO is this necessary? how to remove it?
	Conn net.Conn
	rmu, wmu sync.Mutex
	RW   *RlpxFrameRW
}

func NewRLPX(conn net.Conn) *Rlpx {
	return &Rlpx{Conn: conn}
}

func (r *Rlpx) Read() (RawRLPXMessage, error)  {
	r.rmu.Lock()
	defer r.rmu.Unlock()

	return r.RW.Read()
}

func (r *Rlpx) Write(msg RawRLPXMessage) error {
	r.wmu.Lock()
	defer r.wmu.Unlock()

	return r.RW.Write(msg)
}

func (r *Rlpx) Close() {
	r.wmu.Lock()
	defer r.wmu.Unlock()

	r.Conn.Close()
}

func (r *Rlpx) SetReadDeadline(timeout time.Duration) {
	r.rmu.Lock()
	defer r.rmu.Unlock()

	r.Conn.SetReadDeadline(time.Now().Add(timeout))
}

func (r *Rlpx) SetWriteDeadline(timeout time.Duration) {
	r.wmu.Lock()
	defer r.wmu.Unlock()

	r.Conn.SetWriteDeadline(time.Now().Add(timeout))
}

var (
	// this is used in place of actual frame header data.
	// TODO: replace this when Msg contains the protocol type code.
	zeroHeader = []byte{0xC2, 0x80, 0x80}
	// sixteen zero bytes
	Zero16 = make([]byte, 16)
)

// RlpxFrameRW implements a simplified version of RLPx framing.
// chunked messages are not supported and all headers are equal to
// zeroHeader.
//
// RlpxFrameRW is not safe for concurrent use from multiple goroutines.
type RlpxFrameRW struct { // TODO THIS SHOULD BE UNEXPORTED
	conn io.ReadWriter
	enc  cipher.Stream
	dec  cipher.Stream

	macCipher  cipher.Block
	egressMAC  hash.Hash
	ingressMAC hash.Hash

	Snappy bool
}

func NewRLPXFrameRW(conn io.ReadWriter, AES, MAC []byte, EgressMAC, IngressMAC hash.Hash) *RlpxFrameRW {
	macc, err := aes.NewCipher(MAC)
	if err != nil {
		panic("invalid MAC secret: " + err.Error())
	}
	encc, err := aes.NewCipher(AES)
	if err != nil {
		panic("invalid AES secret: " + err.Error())
	}
	// we use an all-zeroes IV for AES because the key used
	// for encryption is ephemeral.
	iv := make([]byte, encc.BlockSize())
	return &RlpxFrameRW{
		conn:       conn,
		enc:        cipher.NewCTR(encc, iv),
		dec:        cipher.NewCTR(encc, iv),
		macCipher:  macc,
		egressMAC:  EgressMAC,
		ingressMAC: IngressMAC,
	}
}

func (rw *RlpxFrameRW) IngressMAC() hash.Hash { return rw.ingressMAC }
func (rw *RlpxFrameRW) EgressMAC() hash.Hash  { return rw.egressMAC }
func (rw *RlpxFrameRW) Enc() cipher.Stream    { return rw.enc }
func (rw *RlpxFrameRW) Dec() cipher.Stream    { return rw.dec }

// TODO document
func (rw *RlpxFrameRW) Compress(size uint32, payload io.Reader) (uint32, io.Reader, error) {
	if size > maxUint24 {
		return 0, nil, errPlainMessageTooLarge
	}
	payloadBytes, _ := ioutil.ReadAll(payload)
	payloadBytes = snappy.Encode(nil, payloadBytes)

	compressedLen := uint32(len(payloadBytes))
	compressed := bytes.NewReader(payloadBytes)

	return compressedLen, compressed, nil
}

func (rw *RlpxFrameRW) Write(msg RawRLPXMessage) error {
	ptype, _ := rlp.EncodeToBytes(msg.Code)
	// write header
	headbuf := make([]byte, 32)
	fsize := uint32(len(ptype)) + msg.Size
	if fsize > maxUint24 {
		return errors.New("message size overflows uint24")
	}
	putInt24(fsize, headbuf) // TODO: check overflow
	copy(headbuf[3:], zeroHeader)
	rw.enc.XORKeyStream(headbuf[:16], headbuf[:16]) // first half is now encrypted

	// write header MAC
	copy(headbuf[16:], UpdateMAC(rw.egressMAC, rw.macCipher, headbuf[:16]))
	if _, err := rw.conn.Write(headbuf); err != nil {
		return err
	}

	// write encrypted frame, updating the egress MAC hash with
	// the data written to conn.
	tee := cipher.StreamWriter{S: rw.enc, W: io.MultiWriter(rw.conn, rw.egressMAC)}
	if _, err := tee.Write(ptype); err != nil {
		return err
	}
	if _, err := io.Copy(tee, msg.Payload); err != nil {
		return err
	}
	if padding := fsize % 16; padding > 0 {
		if _, err := tee.Write(Zero16[:16-padding]); err != nil {
			return err
		}
	}

	// write frame MAC. egress MAC hash is up to date because
	// frame content was written to it as well.
	fmacseed := rw.egressMAC.Sum(nil)
	mac := UpdateMAC(rw.egressMAC, rw.macCipher, fmacseed)
	_, err := rw.conn.Write(mac)
	return err
}

func (rw *RlpxFrameRW) Read() (msg RawRLPXMessage, err error) {
	// read the header
	headbuf := make([]byte, 32)
	if _, err := io.ReadFull(rw.conn, headbuf); err != nil {
		return msg, err
	}
	// verify header mac
	shouldMAC := UpdateMAC(rw.ingressMAC, rw.macCipher, headbuf[:16])
	if !hmac.Equal(shouldMAC, headbuf[16:]) {
		return msg, errors.New("bad header MAC")
	}
	rw.dec.XORKeyStream(headbuf[:16], headbuf[:16]) // first half is now decrypted
	fsize := readInt24(headbuf)
	// ignore protocol type for now

	// read the frame content
	var rsize = fsize // frame size rounded up to 16 byte boundary
	if padding := fsize % 16; padding > 0 {
		rsize += 16 - padding
	}
	framebuf := make([]byte, rsize)
	if _, err := io.ReadFull(rw.conn, framebuf); err != nil {
		return msg, err
	}

	// read and validate frame MAC. we can re-use headbuf for that.
	rw.ingressMAC.Write(framebuf)
	fmacseed := rw.ingressMAC.Sum(nil)
	if _, err := io.ReadFull(rw.conn, headbuf[:16]); err != nil {
		return msg, err
	}
	shouldMAC = UpdateMAC(rw.ingressMAC, rw.macCipher, fmacseed)
	if !hmac.Equal(shouldMAC, headbuf[:16]) {
		return msg, errors.New("bad frame MAC")
	}

	// decrypt frame content
	rw.dec.XORKeyStream(framebuf, framebuf)

	// decode message code
	content := bytes.NewReader(framebuf[:fsize])
	if err := rlp.Decode(content, &msg.Code); err != nil {
		return msg, err
	}
	msg.Size = uint32(content.Len())
	msg.Payload = content

	// if snappy is enabled, verify and decompress message
	if rw.Snappy {
		payload, err := ioutil.ReadAll(msg.Payload)
		if err != nil {
			return msg, err
		}
		size, err := snappy.DecodedLen(payload)
		if err != nil {
			return msg, err
		}
		if size > int(maxUint24) {
			return msg, errPlainMessageTooLarge
		}
		payload, err = snappy.Decode(nil, payload)
		if err != nil {
			return msg, err
		}
		msg.Size, msg.Payload = uint32(size), bytes.NewReader(payload)
	}
	return msg, nil
}

// UpdateMAC reseeds the given hash with encrypted seed.
// it returns the first 16 bytes of the hash sum after seeding.
func UpdateMAC(mac hash.Hash, block cipher.Block, seed []byte) []byte {
	aesbuf := make([]byte, aes.BlockSize)
	block.Encrypt(aesbuf, mac.Sum(nil))
	for i := range aesbuf {
		aesbuf[i] ^= seed[i]
	}
	mac.Write(aesbuf)
	return mac.Sum(nil)[:16]
}

func putInt24(v uint32, b []byte) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}

func readInt24(b []byte) uint32 {
	return uint32(b[2]) | uint32(b[1])<<8 | uint32(b[0])<<16
}

package p2p

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"hash"
	"io"

	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/rlp"
)

/*
CryptoMsgRW implements MsgReadWriter a message read writer with encryption and authentication
it is initialised by cryptoId.NewSession() after a successful crypto handshake on the same IO
It uses the legacy devp2p packet structure (temporary)
*/

type CryptoMsgRW struct {
	r                     io.Reader
	w                     io.Writer
	aesSecret, macSecret  []byte
	egressMac, ingressMac []byte
	ingress, egress       hash.Hash
	stream                cipher.Stream
}

func NewCryptoMsgRW(r io.Reader, w io.Writer, aesSecret, macSecret, egressMac, ingressMac []byte) (*CryptoMsgRW, error) {
	self := &CryptoMsgRW{
		r:          r,
		w:          w,
		aesSecret:  aesSecret,
		macSecret:  macSecret,
		egressMac:  egressMac,
		ingressMac: ingressMac,
	}
	block, err := aes.NewCipher(aesSecret)
	if err != nil {
		return nil, err
	}
	self.stream = cipher.NewCTR(block, macSecret[:aes.BlockSize])
	self.egress = hmac.New(sha256.New, egressMac)
	self.ingress = hmac.New(sha256.New, ingressMac)
	return self, nil
}

func (self *CryptoMsgRW) Decrypt(plaintext, ciphertext []byte) (err error) {
	self.stream.XORKeyStream(plaintext, ciphertext)
	self.ingress.Write(plaintext)
	return
}

func (self *CryptoMsgRW) Encrypt(ciphertext, plaintext []byte) (err error) {
	self.stream.XORKeyStream(ciphertext, plaintext)
	self.egress.Write(plaintext)
	return
}

func (self *CryptoMsgRW) WriteMsg(msg Msg) (err error) {
	// TODO: handle case when Size + len(code) + len(listhdr) overflows uint32
	code := ethutil.Encode(uint32(msg.Code))
	listhdr := makeListHeader(msg.Size + uint32(len(code)))
	payloadLen := uint32(len(listhdr)) + uint32(len(code)) + msg.Size

	start := make([]byte, 8)
	copy(start, magicToken)
	binary.BigEndian.PutUint32(start[4:], payloadLen)
	if _, err = self.w.Write(start); err != nil {
		return
	}
	listhdrLen := uint32(len(listhdr))
	codeLen := uint32(len(code))
	ciphertext := make([]byte, listhdrLen+codeLen+msg.Size)
	plaintext := make([]byte, listhdrLen+codeLen+msg.Size)
	copy(plaintext, listhdr)
	copy(plaintext[listhdrLen:], code)
	msg.Payload.Read(plaintext[listhdrLen+codeLen:])
	self.Encrypt(ciphertext, plaintext)
	if _, err = self.w.Write(ciphertext); err != nil {
		return
	}
	mac := self.egress.Sum(nil)
	if _, err = self.w.Write(mac); err != nil {
		return
	}

	return
}

func (self *CryptoMsgRW) ReadMsg() (msg Msg, err error) {
	var size uint32
	if size, err = self.readHeader(); err != nil {
		err = newPeerError(errRead, "%v", err)
		return
	}
	// authenticate size
	var payload rlp.ByteReader
	if payload, err = self.readPayload(size); err != nil {
		err = newPeerError(errRead, "%v", err)
		return
	}
	return NewMsgFromRLP(size, payload)
}

func (self *CryptoMsgRW) readHeader() (size uint32, err error) {
	// read magic and payload size
	start := make([]byte, 8)
	if _, err = io.ReadFull(self.r, start); err != nil {
		err = newPeerError(errRead, "%v", err)
		return
	}
	if !bytes.HasPrefix(start, magicToken) {
		err = newPeerError(errMagicTokenMismatch, "got %x, want %x", start[:4], magicToken)
		return
	}
	// here we could deobfuscate and auth the header...
	size = binary.BigEndian.Uint32(start[4:])
	// here more header type metainfo...
	return
}

func (self *CryptoMsgRW) readPayload(size uint32) (r rlp.ByteReader, err error) {
	plaintext := make([]byte, size)
	ciphertext := make([]byte, size)
	self.r.Read(ciphertext)
	self.Decrypt(plaintext, ciphertext)
	mac := make([]byte, 32)
	if _, err = self.r.Read(mac); err != nil {
		err = newPeerError(errRead, "%v", err)
		return
	}
	var expectedMac = self.ingress.Sum(nil)
	if !hmac.Equal(expectedMac, mac) {
		err = newPeerError(errAuthentication, "ingress incorrect:\nexp %v\ngot %v\n", hexkey(expectedMac), hexkey(mac))
		return
	}
	r = bytes.NewReader(plaintext)
	return
}

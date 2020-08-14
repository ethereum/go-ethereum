package rlpx

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
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

	sskLen = 16                     // ecies.MaxSharedKeyLength(pubKey) / 2
	sigLen = crypto.SignatureLength // elliptic S256
	pubLen = 64                     // 512 bit pubkey in uncompressed representation without format byte
	shaLen = 32                     // hash length (for nonce etc)

	authMsgLen  = sigLen + shaLen + pubLen + shaLen + 1
	authRespLen = pubLen + shaLen + 1

	eciesOverhead = 65 /* pubkey */ + 16 /* IV */ + 32 /* MAC */

	encAuthMsgLen  = authMsgLen + eciesOverhead  // size of encrypted pre-EIP-8 initiator handshake
	encAuthRespLen = authRespLen + eciesOverhead // size of encrypted pre-EIP-8 handshake reply

	// total timeout for encryption handshake and protocol
	// handshake in both directions.
	handshakeTimeout = 5 * time.Second

	// This is the timeout for sending the disconnect reason.
	// This is shorter than the usual timeout because we don't want
	// to wait if the connection is known to be bad anyway.
	discWriteTimeout = 1 * time.Second
)

// errPlainMessageTooLarge is returned if a decompressed message length exceeds
// the allowed 24 bits (i.e. length >= 16MB).
var errPlainMessageTooLarge = errors.New("message length >= 16MB")

// maybe a struct here to encompass what?
// it needs a conn and a read and write lock
// does it need anything else?

// what about the frame ?

type Rlpx struct {
	Conn     net.Conn
	rmu, wmu sync.Mutex
	RW       *RlpxFrameRW
	// TODO probs add frameRW
}

func NewRLPX(conn net.Conn) *Rlpx { // TODO figure out later if it needs an interface?
	// TODO timeouts on the conn can be set on the user-side
	return &Rlpx{Conn: conn}
}

func (r *Rlpx) Read() {
	r.rmu.Lock()
	defer r.rmu.Unlock()
	// TODO timeout for frameread timeout should be set on conn beforehand on user-side

	// TODO call read on frameRW?
}

func (r *Rlpx) Write(msg RawRLPXMessage) error {
	r.wmu.Lock()
	defer r.wmu.Unlock()

	return r.RW.Write(msg)
}

func (r *Rlpx) close(closeCode int, closeMessage string) {
	r.wmu.Lock()
	defer r.wmu.Unlock()

	// TODO nil frameRW check should be done on user side.
	// TODO disc connection reason write should be on user-side
	r.Conn.Close()
}

var (
	// this is used in place of actual frame header data.
	// TODO: replace this when Msg contains the protocol type code.
	zeroHeader = []byte{0xC2, 0x80, 0x80}
	// sixteen zero bytes
	zero16 = make([]byte, 16)
)

// RlpxFrameRW implements a simplified version of RLPx framing.
// chunked messages are not supported and all headers are equal to
// zeroHeader.
//
// RlpxFrameRW is not safe for concurrent use from multiple goroutines.
type RlpxFrameRW struct {
	conn io.ReadWriter
	enc  cipher.Stream
	dec  cipher.Stream

	macCipher  cipher.Block
	egressMAC  hash.Hash
	ingressMAC hash.Hash

	Snappy bool
}

func newRLPXFrameRW(conn io.ReadWriter, AES, MAC []byte, EgressMAC, IngressMAC hash.Hash) *RlpxFrameRW {
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

// TODO i still don't like this func
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
		if _, err := tee.Write(zero16[:16-padding]); err != nil {
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

func (rw *RlpxFrameRW) Read() {

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


package whisper

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	DefaultTtl = 50 * time.Second
)

type Envelope struct {
	Expiry int32 // Whisper protocol specifies int32, really should be int64
	Ttl    int32 // ^^^^^^
	Topics [][]byte
	Data   []byte
	Nonce  uint32

	hash Hash
}

func NewEnvelopeFromReader(reader io.Reader) (*Envelope, error) {
	var envelope Envelope

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)

	h := H(crypto.Sha3(buf.Bytes()))
	if err := rlp.Decode(buf, &envelope); err != nil {
		return nil, err
	}

	envelope.hash = h

	return &envelope, nil
}

func (self *Envelope) Hash() Hash {
	if self.hash == EmptyHash {
		self.hash = H(crypto.Sha3(ethutil.Encode(self)))
	}

	return self.hash
}

func NewEnvelope(ttl time.Duration, topics [][]byte, data *Message) *Envelope {
	exp := time.Now().Add(ttl)

	return &Envelope{int32(exp.Unix()), int32(ttl.Seconds()), topics, data.Bytes(), 0, Hash{}}
}

func (self *Envelope) Seal() {
	self.proveWork(DefaultTtl)
}

func (self *Envelope) proveWork(dura time.Duration) {
	var bestBit int
	d := make([]byte, 64)
	copy(d[:32], ethutil.Encode(self.withoutNonce()))

	then := time.Now().Add(dura).UnixNano()
	for n := uint32(0); time.Now().UnixNano() < then; {
		for i := 0; i < 1024; i++ {
			binary.BigEndian.PutUint32(d[60:], n)

			fbs := ethutil.FirstBitSet(ethutil.BigD(crypto.Sha3(d)))
			if fbs > bestBit {
				bestBit = fbs
				self.Nonce = n
			}

			n++
		}
	}
}

func (self *Envelope) valid() bool {
	d := make([]byte, 64)
	copy(d[:32], ethutil.Encode(self.withoutNonce()))
	binary.BigEndian.PutUint32(d[60:], self.Nonce)
	return ethutil.FirstBitSet(ethutil.BigD(crypto.Sha3(d))) > 0
}

func (self *Envelope) withoutNonce() interface{} {
	return []interface{}{self.Expiry, self.Ttl, ethutil.ByteSliceToInterface(self.Topics), self.Data}
}

func (self *Envelope) RlpData() interface{} {
	return []interface{}{self.Expiry, self.Ttl, ethutil.ByteSliceToInterface(self.Topics), self.Data, self.Nonce}
}

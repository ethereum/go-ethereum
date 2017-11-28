// Copyright 2015 The go-ethereum Authors
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

// Package discover implements the Ethereum Node Record as per https://github.com/ethereum/EIPs/pull/778
package enr

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"net"
	"sort"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/btcsuite/btcd/btcec"
)

// The maximum encoded size of a node record is 300 bytes. Implementations should reject records larger than this size.
const (
	RECORD_MAX_SIZE     = 300
	ID_SECP256k1_KECCAK = "secp256k1-keccak" // "secp256k1-keccak" identity scheme identifier
)

// Pseudo-const identifiers for pre-defined keys
var (
	ID        = []byte(`id`)        // name of identity scheme, e.g. "secp256k1-keccak"
	SECP256K1 = []byte(`secp256k1`) // compressed secp256k1 public key
	IP4       = []byte(`ip4`)       // IPv4 address, 4 bytes
	IP6       = []byte(`ip6`)       // IPv6 address, 16 bytes
	DISCV5    = []byte(`discv5`)    // UDP port for discovery v5
)

type record struct {
	k []byte
	v []byte
}

type ENR struct {
	seq       uint32   // sequence number
	signature []byte   // record's signature
	raw       []byte   // RLP encoded record
	records   []record // list of all key/value pairs, sorted prior to RLP encoding
	dirty     bool     // keeps track if record was modified after it was signed and encoded
}

func NewENR() *ENR {
	return &ENR{
		dirty: true,
	}
}

func (e *ENR) GetID() (string, error) {
	for _, r := range e.records {
		if bytes.Compare(ID, r.k) == 0 {
			return string(r.v), nil
		}
	}

	return "", errors.New("id record does not exist")
}

func (e *ENR) SetID(id string) {
	e.dirty = true
	e.records = append(e.records, record{ID, []byte(id)})
}

func (e *ENR) GetSecp256k1() ([]byte, error) {
	for _, r := range e.records {
		if bytes.Compare(SECP256K1, r.k) == 0 {
			return r.v, nil
		}
	}

	return nil, errors.New("secp256k1 record does not exist")
}

func (e *ENR) SetSecp256k1(pk []byte) {
	e.dirty = true
	e.records = append(e.records, record{SECP256K1, pk})
}

func (e *ENR) GetIPv4() (net.IP, error) {
	for _, r := range e.records {
		if bytes.Compare(IP4, r.k) == 0 {
			if len(r.v) != net.IPv4len {
				return nil, errors.New("wrong ipv4 record length")
			}
			return net.IP(r.v), nil
		}
	}

	return nil, errors.New("ip4 record does not exist")
}

func (e *ENR) SetIPv4(ip net.IP) error {
	e.dirty = true
	ipv4 := ip.To4()
	if ipv4 == nil {
		return errors.New("param is not a valid ipv4 address")
	}
	e.records = append(e.records, record{IP4, ipv4})
	return nil
}

func (e *ENR) GetIPv6() (net.IP, error) {
	for _, r := range e.records {
		if bytes.Compare(IP6, r.k) == 0 {
			if len(r.v) != net.IPv6len {
				return nil, errors.New("wrong ipv6 record length")
			}
			return net.IP(r.v), nil
		}
	}

	return nil, errors.New("ip6 record does not exist")
}

func (e *ENR) SetIPv6(ip net.IP) error {
	if len(ip) != net.IPv6len {
		return errors.New("param length is not equal to 16 bytes")
	}
	e.dirty = true
	e.records = append(e.records, record{IP6, ip})
	return nil
}

func (e *ENR) GetDiscv5() (uint32, error) {
	for _, r := range e.records {
		if bytes.Compare(DISCV5, r.k) == 0 {
			buf := bytes.NewBuffer(r.v)
			var port uint32
			err := binary.Read(buf, binary.BigEndian, &port)
			if err != nil {
				return 0, err
			}

			return port, nil
		}
	}

	return 0, errors.New("secp256k1 record does not exist")
}

func (e *ENR) SetDiscv5(port uint32) error {
	e.dirty = true
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, port)
	if err != nil {
		return err
	}

	e.records = append(e.records, record{DISCV5, buf.Bytes()})

	return nil
}

func (e *ENR) GetRaw(k []byte) ([]byte, error) {
	for _, r := range e.records {
		if bytes.Compare(r.k, k) == 0 {
			return r.v, nil
		}
	}

	return nil, errors.New("record does not exist")
}

func (e *ENR) SetRaw(k []byte, v []byte) {
	e.dirty = true
	e.records = append(e.records, record{k, v})
}

func (e *ENR) Encode() ([]byte, error) {
	if e.dirty {
		return nil, errors.New("record is not signed")
	}
	return e.raw, nil
}

func (e *ENR) NodeAddress() ([]byte, error) {
	pk, err := e.GetSecp256k1()
	if err != nil {
		return nil, err
	}

	digest := crypto.Keccak256Hash(pk)

	return digest.Bytes(), nil
}

func (e *ENR) Decode(data []byte) error {
	if len(data) > RECORD_MAX_SIZE {
		return errors.New("record is too big")
	}

	s := rlp.NewStream(bytes.NewReader(data), RECORD_MAX_SIZE)

	signature, err := s.Bytes()
	if err != nil {
		return err
	}

	// consume the list prefix
	_, err = s.List()
	if err != nil {
		return err
	}

	seq, err := s.Uint()
	if err != nil {
		return err
	}

	var records []record

	// read key/value pairs until we reach rlp.EOL
	for _, _, err = s.Kind(); err == nil; _, _, err = s.Kind() {
		key, err2 := s.Bytes()
		if err2 != nil {
			return err2
		}

		value, err2 := s.Bytes()
		if err2 != nil {
			return err2
		}

		records = append(records, record{k: key, v: value})
	}

	if err != rlp.EOL {
		return err
	}

	e.signature = signature
	e.raw = data
	e.seq = uint32(seq)
	e.records = records

	err = e.verifySignature()
	if err != nil {
		return err
	}

	e.dirty = false

	return nil
}

func (e *ENR) Sign(privkey *ecdsa.PrivateKey) error {
	e.seq = e.seq + 1

	e.SetID(ID_SECP256k1_KECCAK)

	pk := (*btcec.PublicKey)(&privkey.PublicKey).SerializeCompressed()
	e.SetSecp256k1(pk)

	var err error
	e.signature, e.raw, err = e.SignAndEncode(privkey)
	if err != nil {
		return err
	}

	// mark record ready for encoding
	e.dirty = false

	return nil
}

func (e *ENR) SignAndEncode(privkey *ecdsa.PrivateKey) ([]byte, []byte, error) {
	content, err := e.SerialisedContent()
	if err != nil {
		return nil, nil, err
	}

	digest := crypto.Keccak256Hash(content)

	signature, err := crypto.Sign(digest.Bytes(), privkey)
	if err != nil {
		return nil, nil, err
	}

	blob, err := rlp.EncodeToBytes(signature)
	if err != nil {
		return nil, nil, err
	}

	raw := append(blob, content...)

	return signature, raw, nil
}

func (e *ENR) SerialisedContent() ([]byte, error) {
	var buffer bytes.Buffer

	blob, err := rlp.EncodeToBytes(e.seq)
	if err != nil {
		return nil, err
	}

	_, err = buffer.Write(blob)
	if err != nil {
		return nil, err
	}

	sort.Slice(e.records, func(i, j int) bool {
		return bytes.Compare(e.records[i].k, e.records[j].k) < 0
	})

	for _, r := range e.records {
		kk, err := rlp.EncodeToBytes(r.k)
		if err != nil {
			return nil, err
		}

		_, err = buffer.Write(kk)
		if err != nil {
			return nil, err
		}

		vv, err := rlp.EncodeToBytes(r.v)
		if err != nil {
			return nil, err
		}

		_, err = buffer.Write(vv)
		if err != nil {
			return nil, err
		}
	}

	return wrapList(buffer.Bytes()), nil
}

func (e *ENR) verifySignature() error {
	id, err := e.GetID()
	if err != nil {
		return err
	}

	// currently "secp256k1-keccak" is the only known identity scheme
	if id != ID_SECP256k1_KECCAK {
		return errors.New("unknown identity scheme")
	}

	// get publickey from record
	blob, err := e.GetSecp256k1()
	if err != nil {
		return err
	}

	pk, err := btcec.ParsePubKey(blob, btcec.S256())
	if err != nil {
		return err
	}
	pubkey1 := pk.SerializeUncompressed()

	// get publickey from message and signature
	content, err := e.SerialisedContent()
	if err != nil {
		return err
	}

	digest := crypto.Keccak256Hash(content)
	pubkey2, err := crypto.Ecrecover(digest.Bytes(), e.signature)
	if err != nil {
		return err
	}

	if bytes.Compare(pubkey1, pubkey2) != 0 {
		return errors.New("public key mismatch")
	}

	return nil
}

func wrapList(c []byte) []byte {
	head := make([]byte, 9)
	res := rlp.LengthPrefix(head, uint64(len(c)))
	return append(res, c...)
}

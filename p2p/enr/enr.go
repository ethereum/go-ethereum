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
	"errors"
	"io"
	"sort"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/btcsuite/btcd/btcec"
)

var ()

// The maximum encoded size of a node record is 300 bytes. Implementations should reject records larger than this size.
const (
	ID_SECP256k1_KECCAK = "secp256k1-keccak" // "secp256k1-keccak" identity scheme identifier
)

// Key is implemented by known node record key types.
//
// To define a new key that is to be included in a node record,
// create a Go type that satisfies this interface. The type should
// also implement rlp.Decoder if additional checks are needed on the value.
type Key interface {
	ENRKey() string
}

type pair struct {
	k []byte
	v []byte
}

type Record struct {
	seq       uint32 // sequence number
	signature []byte // record's signature
	raw       []byte // RLP encoded record
	pairs     []pair // list of all key/value pairs, sorted prior to RLP encoding
	signed    bool   // keeps track if record was modified after it was signed and encoded
}

func (r Record) Seq() uint32 {
	return r.seq
}

func (r *Record) SetSeq(s uint32) {
	r.signed = false
	r.seq = s
}

func (r *Record) Load(k Key) (bool, error) {
	for _, p := range r.pairs {
		if string(p.k) == k.ENRKey() {
			err := rlp.DecodeBytes(p.v, k)
			return true, err
		}
	}

	return false, errors.New("record does not exist")
}

func (r *Record) Set(k Key) error {
	r.signed = false
	blob, err := rlp.EncodeToBytes(k)
	if err != nil {
		return err
	}
	r.pairs = append(r.pairs, pair{[]byte(k.ENRKey()), blob})
	return nil
}

func (r Record) EncodeRLP(w io.Writer) error {
	if !r.signed {
		return errors.New("record is not signed")
	}

	_, err := w.Write(r.raw)

	return err
}

func (r *Record) DecodeRLP(s *rlp.Stream) error {
	var err error

	r.signature, err = s.Bytes()
	if err != nil {
		return err
	}

	_, err = s.List()
	if err != nil {
		return err
	}

	if err := s.Decode(&r.seq); err != nil {
		return err
	}

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

		r.pairs = append(r.pairs, pair{k: key, v: value})
	}

	if err != rlp.EOL {
		return err
	}

	sigcontent, err := r.serialisedContent()
	if err != nil {
		return err
	}

	// update r.raw
	blob, err := rlp.EncodeToBytes(r.signature)
	if err != nil {
		return err
	}

	r.raw = append(blob, sigcontent...)

	err = r.verifySignature(sigcontent)
	if err != nil {
		return err
	}

	// mark record ready for encoding
	r.signed = true

	return nil
}

func (r Record) Equal(o Record) (bool, error) {
	rr, err := r.serialisedContent()
	if err != nil {
		return false, err
	}

	oo, err := o.serialisedContent()
	if err != nil {
		return false, err
	}

	if bytes.Compare(rr, oo) != 0 {
		return false, nil
	}

	if err := r.verifySignature(rr); err != nil {
		return false, err
	}

	if err := o.verifySignature(oo); err != nil {
		return false, err
	}

	return true, nil
}

func (r *Record) NodeAddr() ([]byte, error) {
	var secp256k1 Secp256k1

	_, err := r.Load(&secp256k1)
	if err != nil {
		return nil, err
	}

	digest := crypto.Keccak256Hash(secp256k1)

	return digest.Bytes(), nil
}

func (r *Record) Sign(privkey *ecdsa.PrivateKey) error {
	r.seq = r.seq + 1

	id := ID(ID_SECP256k1_KECCAK)

	r.Set(id)

	pk := (*btcec.PublicKey)(&privkey.PublicKey).SerializeCompressed()
	secp256k1 := Secp256k1(pk)
	r.Set(secp256k1)

	return r.signAndEncode(privkey)
}

func (r *Record) serialisedContent() ([]byte, error) {
	sort.Slice(r.pairs, func(i, j int) bool {
		return string(r.pairs[i].k) < string(r.pairs[j].k)
	})

	list := []interface{}{r.seq}

	for _, p := range r.pairs {
		list = append(list, p.k, p.v)
	}

	return rlp.EncodeToBytes(list)
}

func (r *Record) signAndEncode(privkey *ecdsa.PrivateKey) error {
	sigcontent, err := r.serialisedContent()
	if err != nil {
		return err
	}

	digest := crypto.Keccak256Hash(sigcontent)

	key := btcec.PrivateKey(*privkey)
	signature, err := key.Sign(digest.Bytes())
	if err != nil {
		return err
	}
	r.signature = signature.Serialize()

	blob, err := rlp.EncodeToBytes(r.signature)
	if err != nil {
		return err
	}

	r.raw = append(blob, sigcontent...)

	// mark record ready for encoding
	r.signed = true

	return nil
}

func (r Record) verifySignature(sigcontent []byte) error {
	var id ID
	_, err := r.Load(&id)
	if err != nil {
		return err
	}

	// currently "secp256k1-keccak" is the only known identity scheme
	if id != ID_SECP256k1_KECCAK {
		return errors.New("unknown identity scheme")
	}

	// get publickey from record
	var secp256k1 Secp256k1
	_, err = r.Load(&secp256k1)
	if err != nil {
		return err
	}

	pk, err := btcec.ParsePubKey(secp256k1, btcec.S256())
	if err != nil {
		return err
	}

	digest := crypto.Keccak256Hash(sigcontent)

	sign, err := btcec.ParseSignature(r.signature, btcec.S256())
	if err != nil {
		return err
	}

	if !sign.Verify(digest.Bytes(), pk) {
		return errors.New("signature is not valid")
	}

	return nil
}

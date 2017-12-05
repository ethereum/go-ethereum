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
	"math/big"
	"sort"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	errNoID           = errors.New("unknown or unspecified identity scheme")
	errInvalidSigsize = errors.New("invalid signature size")
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
	k string
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
	i := sort.Search(len(r.pairs), func(i int) bool { return r.pairs[i].k >= k.ENRKey() })

	if i < len(r.pairs) && r.pairs[i].k == k.ENRKey() {
		return true, rlp.DecodeBytes(r.pairs[i].v, k)
	}

	return false, errors.New("record does not exist")
}

func (r *Record) Set(k Key) error {
	r.signed = false
	blob, err := rlp.EncodeToBytes(k)
	if err != nil {
		return err
	}
	var inserted bool
	for i, p := range r.pairs {
		if p.k == k.ENRKey() {
			// replace value of pair
			r.pairs[i].v = blob
			inserted = true

			break
		} else if p.k > k.ENRKey() {
			// insert pair before i-th elem
			el := pair{k.ENRKey(), blob}

			r.pairs = append(r.pairs, pair{})
			copy(r.pairs[i+1:], r.pairs[i:])
			r.pairs[i] = el

			inserted = true
			break
		}
	}
	if !inserted {
		r.pairs = append(r.pairs, pair{k.ENRKey(), blob})
	}
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

		r.pairs = append(r.pairs, pair{k: string(key), v: value})
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

	pk := btcec.PublicKey(secp256k1)

	digest := crypto.Keccak256Hash(pk.SerializeCompressed())

	return digest.Bytes(), nil
}

func (r *Record) Sign(privkey *ecdsa.PrivateKey) error {
	r.seq = r.seq + 1

	r.Set(ID(ID_SECP256k1_KECCAK))

	pk := (*btcec.PublicKey)(&privkey.PublicKey)
	secp256k1 := Secp256k1(*pk)
	r.Set(secp256k1)

	return r.signAndEncode(privkey)
}

func (r *Record) serialisedContent() ([]byte, error) {
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

	sig, err := (*btcec.PrivateKey)(privkey).Sign(crypto.Keccak256(sigcontent))
	if err != nil {
		return err
	}
	r.signature = encodeCompactSignature(sig)

	blob, err := rlp.EncodeToBytes(r.signature)
	if err != nil {
		return err
	}

	r.raw = append(blob, sigcontent...)

	// mark record ready for encoding
	r.signed = true

	return nil
}

func (r *Record) verifySignature(sigcontent []byte) error {
	// Get identity scheme, public key.
	var id ID
	var secp256k1 Secp256k1
	if _, err := r.Load(&id); err != nil {
		return err
	}
	if id != ID_SECP256k1_KECCAK {
		return errNoID
	}
	if _, err := r.Load(&secp256k1); err != nil {
		return err
	}

	// Verify the signature.
	sig, err := parseCompactSignature(r.signature)
	if err != nil {
		return err
	}
	if !sig.Verify(crypto.Keccak256(sigcontent), (*btcec.PublicKey)(&secp256k1)) {
		return errors.New("signature is not valid")
	}
	return nil
}

func encodeCompactSignature(sig *btcec.Signature) []byte {
	b := make([]byte, 64)
	math.ReadBits(sig.R, b[:32])
	math.ReadBits(sig.S, b[32:])
	return b
}

func parseCompactSignature(sig []byte) (*btcec.Signature, error) {
	if len(sig) != 64 {
		return nil, errInvalidSigsize
	}
	return &btcec.Signature{R: new(big.Int).SetBytes(sig[:32]), S: new(big.Int).SetBytes(sig[32:])}, nil
}

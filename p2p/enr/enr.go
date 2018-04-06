// Copyright 2017 The go-ethereum Authors
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

// Package enr implements Ethereum Node Records as defined in EIP-778. A node record holds
// arbitrary information about a node on the peer-to-peer network.
//
// Records contain named keys. To store and retrieve key/values in a record, use the Entry
// interface.
//
// Records must be signed before transmitting them to another node. Decoding a record verifies
// its signature. When creating a record, set the entries you want, then call Sign to add the
// signature. Modifying a record invalidates the signature.
//
// Package enr supports the "secp256k1-keccak" identity scheme.
package enr

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
)

const SizeLimit = 300 // maximum encoded size of a node record in bytes

const ID_SECP256k1_KECCAK = ID("secp256k1-keccak") // the default identity scheme

var (
	errNoID           = errors.New("unknown or unspecified identity scheme")
	errInvalidSig     = errors.New("invalid signature")
	errNotSorted      = errors.New("record key/value pairs are not sorted by key")
	errDuplicateKey   = errors.New("record contains duplicate key")
	errIncompletePair = errors.New("record contains incomplete k/v pair")
	errTooBig         = fmt.Errorf("record bigger than %d bytes", SizeLimit)
	errEncodeUnsigned = errors.New("can't encode unsigned record")
	errNotFound       = errors.New("no such key in record")
)

// Record represents a node record. The zero value is an empty record.
type Record struct {
	seq       uint64 // sequence number
	signature []byte // the signature
	raw       []byte // RLP encoded record
	pairs     []pair // sorted list of all key/value pairs
}

// pair is a key/value pair in a record.
type pair struct {
	k string
	v rlp.RawValue
}

// Signed reports whether the record has a valid signature.
func (r *Record) Signed() bool {
	return r.signature != nil
}

// Seq returns the sequence number.
func (r *Record) Seq() uint64 {
	return r.seq
}

// SetSeq updates the record sequence number. This invalidates any signature on the record.
// Calling SetSeq is usually not required because signing the redord increments the
// sequence number.
func (r *Record) SetSeq(s uint64) {
	r.signature = nil
	r.raw = nil
	r.seq = s
}

// Load retrieves the value of a key/value pair. The given Entry must be a pointer and will
// be set to the value of the entry in the record.
//
// Errors returned by Load are wrapped in KeyError. You can distinguish decoding errors
// from missing keys using the IsNotFound function.
func (r *Record) Load(e Entry) error {
	i := sort.Search(len(r.pairs), func(i int) bool { return r.pairs[i].k >= e.ENRKey() })
	if i < len(r.pairs) && r.pairs[i].k == e.ENRKey() {
		if err := rlp.DecodeBytes(r.pairs[i].v, e); err != nil {
			return &KeyError{Key: e.ENRKey(), Err: err}
		}
		return nil
	}
	return &KeyError{Key: e.ENRKey(), Err: errNotFound}
}

// Set adds or updates the given entry in the record.
// It panics if the value can't be encoded.
func (r *Record) Set(e Entry) {
	r.signature = nil
	r.raw = nil
	blob, err := rlp.EncodeToBytes(e)
	if err != nil {
		panic(fmt.Errorf("enr: can't encode %s: %v", e.ENRKey(), err))
	}

	i := sort.Search(len(r.pairs), func(i int) bool { return r.pairs[i].k >= e.ENRKey() })

	if i < len(r.pairs) && r.pairs[i].k == e.ENRKey() {
		// element is present at r.pairs[i]
		r.pairs[i].v = blob
		return
	} else if i < len(r.pairs) {
		// insert pair before i-th elem
		el := pair{e.ENRKey(), blob}
		r.pairs = append(r.pairs, pair{})
		copy(r.pairs[i+1:], r.pairs[i:])
		r.pairs[i] = el
		return
	}

	// element should be placed at the end of r.pairs
	r.pairs = append(r.pairs, pair{e.ENRKey(), blob})
}

// EncodeRLP implements rlp.Encoder. Encoding fails if
// the record is unsigned.
func (r Record) EncodeRLP(w io.Writer) error {
	if !r.Signed() {
		return errEncodeUnsigned
	}
	_, err := w.Write(r.raw)
	return err
}

// DecodeRLP implements rlp.Decoder. Decoding verifies the signature.
func (r *Record) DecodeRLP(s *rlp.Stream) error {
	raw, err := s.Raw()
	if err != nil {
		return err
	}
	if len(raw) > SizeLimit {
		return errTooBig
	}

	// Decode the RLP container.
	dec := Record{raw: raw}
	s = rlp.NewStream(bytes.NewReader(raw), 0)
	if _, err := s.List(); err != nil {
		return err
	}
	if err = s.Decode(&dec.signature); err != nil {
		return err
	}
	if err = s.Decode(&dec.seq); err != nil {
		return err
	}
	// The rest of the record contains sorted k/v pairs.
	var prevkey string
	for i := 0; ; i++ {
		var kv pair
		if err := s.Decode(&kv.k); err != nil {
			if err == rlp.EOL {
				break
			}
			return err
		}
		if err := s.Decode(&kv.v); err != nil {
			if err == rlp.EOL {
				return errIncompletePair
			}
			return err
		}
		if i > 0 {
			if kv.k == prevkey {
				return errDuplicateKey
			}
			if kv.k < prevkey {
				return errNotSorted
			}
		}
		dec.pairs = append(dec.pairs, kv)
		prevkey = kv.k
	}
	if err := s.ListEnd(); err != nil {
		return err
	}

	// Verify signature.
	if err = dec.verifySignature(); err != nil {
		return err
	}
	*r = dec
	return nil
}

type s256raw []byte

func (s256raw) ENRKey() string { return "secp256k1" }

// NodeAddr returns the node address. The return value will be nil if the record is
// unsigned.
func (r *Record) NodeAddr() []byte {
	var entry s256raw
	if r.Load(&entry) != nil {
		return nil
	}
	return crypto.Keccak256(entry)
}

// Sign signs the record with the given private key. It updates the record's identity
// scheme, public key and increments the sequence number. Sign returns an error if the
// encoded record is larger than the size limit.
func (r *Record) Sign(privkey *ecdsa.PrivateKey) error {
	r.seq = r.seq + 1
	r.Set(ID_SECP256k1_KECCAK)
	r.Set(Secp256k1(privkey.PublicKey))
	return r.signAndEncode(privkey)
}

func (r *Record) appendPairs(list []interface{}) []interface{} {
	list = append(list, r.seq)
	for _, p := range r.pairs {
		list = append(list, p.k, p.v)
	}
	return list
}

func (r *Record) signAndEncode(privkey *ecdsa.PrivateKey) error {
	// Put record elements into a flat list. Leave room for the signature.
	list := make([]interface{}, 1, len(r.pairs)*2+2)
	list = r.appendPairs(list)

	// Sign the tail of the list.
	h := sha3.NewKeccak256()
	rlp.Encode(h, list[1:])
	sig, err := crypto.Sign(h.Sum(nil), privkey)
	if err != nil {
		return err
	}
	sig = sig[:len(sig)-1] // remove v

	// Put signature in front.
	r.signature, list[0] = sig, sig
	r.raw, err = rlp.EncodeToBytes(list)
	if err != nil {
		return err
	}
	if len(r.raw) > SizeLimit {
		return errTooBig
	}
	return nil
}

func (r *Record) verifySignature() error {
	// Get identity scheme, public key, signature.
	var id ID
	var entry s256raw
	if err := r.Load(&id); err != nil {
		return err
	} else if id != ID_SECP256k1_KECCAK {
		return errNoID
	}
	if err := r.Load(&entry); err != nil {
		return err
	} else if len(entry) != 33 {
		return fmt.Errorf("invalid public key")
	}

	// Verify the signature.
	list := make([]interface{}, 0, len(r.pairs)*2+1)
	list = r.appendPairs(list)
	h := sha3.NewKeccak256()
	rlp.Encode(h, list)
	if !crypto.VerifySignature(entry, h.Sum(nil), r.signature) {
		return errInvalidSig
	}
	return nil
}

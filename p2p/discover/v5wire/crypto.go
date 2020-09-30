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

package v5wire

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"fmt"
	"hash"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

const (
	// Encryption/authentication parameters.
	aesKeySize   = 16
	gcmNonceSize = 12
)

// Nonce represents a nonce used for AES/GCM.
type Nonce [gcmNonceSize]byte

// EncodePubkey encodes a public key.
func EncodePubkey(key *ecdsa.PublicKey) []byte {
	switch key.Curve {
	case crypto.S256():
		return crypto.CompressPubkey(key)
	default:
		panic("unsupported curve " + key.Curve.Params().Name + " in EncodePubkey")
	}
}

// DecodePubkey decodes a public key in compressed format.
func DecodePubkey(curve elliptic.Curve, e []byte) (*ecdsa.PublicKey, error) {
	switch curve {
	case crypto.S256():
		if len(e) != 33 {
			return nil, errors.New("wrong size public key data")
		}
		return crypto.DecompressPubkey(e)
	default:
		return nil, fmt.Errorf("unsupported curve %s in DecodePubkey", curve.Params().Name)
	}
}

// idNonceHash computes the ID signature hash used in the handshake.
func idNonceHash(h hash.Hash, destID enode.ID, iv, authdata, ephkey []byte) []byte {
	h.Reset()
	h.Write([]byte("discovery v5 identity proof"))
	h.Write(iv)
	h.Write(authdata)
	h.Write(ephkey)
	h.Write(destID[:])
	return h.Sum(nil)
}

// makeIDSignature creates the ID nonce signature.
func makeIDSignature(hash hash.Hash, key *ecdsa.PrivateKey, destID enode.ID, ephkey []byte, challenge *Header) ([]byte, error) {
	switch key.Curve {
	case crypto.S256():
		input := idNonceHash(hash, destID, challenge.IV[:], challenge.AuthData, ephkey)
		idsig, err := crypto.Sign(input, key)
		if err != nil {
			return nil, err
		}
		return idsig[:len(idsig)-1], nil // remove recovery ID
	default:
		return nil, fmt.Errorf("unsupported curve %s", key.Curve.Params().Name)
	}
}

// s256raw is an unparsed secp256k1 public key ENR entry.
type s256raw []byte

func (s256raw) ENRKey() string { return "secp256k1" }

// verifyIDSignature checks that signature over idnonce was made by the given node.
func verifyIDSignature(hash hash.Hash, sig []byte, n *enode.Node, destID enode.ID, ephkey []byte, challenge *Header) error {
	switch idscheme := n.Record().IdentityScheme(); idscheme {
	case "v4":
		var pubkey s256raw
		if n.Load(&pubkey) != nil {
			return errors.New("no secp256k1 public key in record")
		}
		input := idNonceHash(hash, destID, challenge.IV[:], challenge.AuthData, ephkey)
		if !crypto.VerifySignature(pubkey, input, sig) {
			return errInvalidNonceSig
		}
		return nil
	default:
		return fmt.Errorf("can't verify ID nonce signature against scheme %q", idscheme)
	}
}

// ecdh creates a shared secret.
func ecdh(privkey *ecdsa.PrivateKey, pubkey *ecdsa.PublicKey) []byte {
	secX, secY := pubkey.ScalarMult(pubkey.X, pubkey.Y, privkey.D.Bytes())
	if secX == nil {
		return nil
	}
	sec := make([]byte, 33)
	sec[0] = 0x02 | byte(secY.Bit(0))
	math.ReadBits(secX, sec[1:])
	return sec
}

// encryptGCM encrypts pt using AES-GCM with the given key and nonce. The ciphertext is
// appended to dest, which must not overlap with plaintext. The resulting ciphertext is 16
// bytes longer than plaintext because it contains an authentication tag.
func encryptGCM(dest, key, nonce, plaintext, authData []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(fmt.Errorf("can't create block cipher: %v", err))
	}
	aesgcm, err := cipher.NewGCMWithNonceSize(block, gcmNonceSize)
	if err != nil {
		panic(fmt.Errorf("can't create GCM: %v", err))
	}
	return aesgcm.Seal(dest, nonce, plaintext, authData), nil
}

// decryptGCM decrypts ct using AES-GCM with the given key and nonce.
func decryptGCM(key, nonce, ct, authData []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("can't create block cipher: %v", err)
	}
	if len(nonce) != gcmNonceSize {
		return nil, fmt.Errorf("invalid GCM nonce size: %d", len(nonce))
	}
	aesgcm, err := cipher.NewGCMWithNonceSize(block, gcmNonceSize)
	if err != nil {
		return nil, fmt.Errorf("can't create GCM: %v", err)
	}
	pt := make([]byte, 0, len(ct))
	return aesgcm.Open(pt, nonce, ct, authData)
}

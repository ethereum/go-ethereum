// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcec

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"io"
)

var (
	// ErrInvalidMAC occurs when Message Authentication Check (MAC) fails
	// during decryption. This happens because of either invalid private key or
	// corrupt ciphertext.
	ErrInvalidMAC = errors.New("invalid mac hash")

	// errInputTooShort occurs when the input ciphertext to the Decrypt
	// function is less than 134 bytes long.
	errInputTooShort = errors.New("ciphertext too short")

	// errUnsupportedCurve occurs when the first two bytes of the encrypted
	// text aren't 0x02CA (= 712 = secp256k1, from OpenSSL).
	errUnsupportedCurve = errors.New("unsupported curve")

	errInvalidXLength = errors.New("invalid X length, must be 32")
	errInvalidYLength = errors.New("invalid Y length, must be 32")
	errInvalidPadding = errors.New("invalid PKCS#7 padding")

	// 0x02CA = 714
	ciphCurveBytes = [2]byte{0x02, 0xCA}
	// 0x20 = 32
	ciphCoordLength = [2]byte{0x00, 0x20}
)

// GenerateSharedSecret generates a shared secret based on a private key and a
// public key using Diffie-Hellman key exchange (ECDH) (RFC 4753).
// RFC5903 Section 9 states we should only return x.
func GenerateSharedSecret(privkey *PrivateKey, pubkey *PublicKey) []byte {
	x, _ := pubkey.Curve.ScalarMult(pubkey.X, pubkey.Y, privkey.D.Bytes())
	return x.Bytes()
}

// Encrypt encrypts data for the target public key using AES-256-CBC. It also
// generates a private key (the pubkey of which is also in the output). The only
// supported curve is secp256k1. The `structure' that it encodes everything into
// is:
//
//	struct {
//		// Initialization Vector used for AES-256-CBC
//		IV [16]byte
//		// Public Key: curve(2) + len_of_pubkeyX(2) + pubkeyX +
//		// len_of_pubkeyY(2) + pubkeyY (curve = 714)
//		PublicKey [70]byte
//		// Cipher text
//		Data []byte
//		// HMAC-SHA-256 Message Authentication Code
//		HMAC [32]byte
//	}
//
// The primary aim is to ensure byte compatibility with Pyelliptic.  Also, refer
// to section 5.8.1 of ANSI X9.63 for rationale on this format.
func Encrypt(pubkey *PublicKey, in []byte) ([]byte, error) {
	ephemeral, err := NewPrivateKey(S256())
	if err != nil {
		return nil, err
	}
	ecdhKey := GenerateSharedSecret(ephemeral, pubkey)
	derivedKey := sha512.Sum512(ecdhKey)
	keyE := derivedKey[:32]
	keyM := derivedKey[32:]

	paddedIn := addPKCSPadding(in)
	// IV + Curve params/X/Y + padded plaintext/ciphertext + HMAC-256
	out := make([]byte, aes.BlockSize+70+len(paddedIn)+sha256.Size)
	iv := out[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	// start writing public key
	pb := ephemeral.PubKey().SerializeUncompressed()
	offset := aes.BlockSize

	// curve and X length
	copy(out[offset:offset+4], append(ciphCurveBytes[:], ciphCoordLength[:]...))
	offset += 4
	// X
	copy(out[offset:offset+32], pb[1:33])
	offset += 32
	// Y length
	copy(out[offset:offset+2], ciphCoordLength[:])
	offset += 2
	// Y
	copy(out[offset:offset+32], pb[33:])
	offset += 32

	// start encryption
	block, err := aes.NewCipher(keyE)
	if err != nil {
		return nil, err
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(out[offset:len(out)-sha256.Size], paddedIn)

	// start HMAC-SHA-256
	hm := hmac.New(sha256.New, keyM)
	hm.Write(out[:len(out)-sha256.Size])          // everything is hashed
	copy(out[len(out)-sha256.Size:], hm.Sum(nil)) // write checksum

	return out, nil
}

// Decrypt decrypts data that was encrypted using the Encrypt function.
func Decrypt(priv *PrivateKey, in []byte) ([]byte, error) {
	// IV + Curve params/X/Y + 1 block + HMAC-256
	if len(in) < aes.BlockSize+70+aes.BlockSize+sha256.Size {
		return nil, errInputTooShort
	}

	// read iv
	iv := in[:aes.BlockSize]
	offset := aes.BlockSize

	// start reading pubkey
	if !bytes.Equal(in[offset:offset+2], ciphCurveBytes[:]) {
		return nil, errUnsupportedCurve
	}
	offset += 2

	if !bytes.Equal(in[offset:offset+2], ciphCoordLength[:]) {
		return nil, errInvalidXLength
	}
	offset += 2

	xBytes := in[offset : offset+32]
	offset += 32

	if !bytes.Equal(in[offset:offset+2], ciphCoordLength[:]) {
		return nil, errInvalidYLength
	}
	offset += 2

	yBytes := in[offset : offset+32]
	offset += 32

	pb := make([]byte, 65)
	pb[0] = byte(0x04) // uncompressed
	copy(pb[1:33], xBytes)
	copy(pb[33:], yBytes)
	// check if (X, Y) lies on the curve and create a Pubkey if it does
	pubkey, err := ParsePubKey(pb, S256())
	if err != nil {
		return nil, err
	}

	// check for cipher text length
	if (len(in)-aes.BlockSize-offset-sha256.Size)%aes.BlockSize != 0 {
		return nil, errInvalidPadding // not padded to 16 bytes
	}

	// read hmac
	messageMAC := in[len(in)-sha256.Size:]

	// generate shared secret
	ecdhKey := GenerateSharedSecret(priv, pubkey)
	derivedKey := sha512.Sum512(ecdhKey)
	keyE := derivedKey[:32]
	keyM := derivedKey[32:]

	// verify mac
	hm := hmac.New(sha256.New, keyM)
	hm.Write(in[:len(in)-sha256.Size]) // everything is hashed
	expectedMAC := hm.Sum(nil)
	if !hmac.Equal(messageMAC, expectedMAC) {
		return nil, ErrInvalidMAC
	}

	// start decryption
	block, err := aes.NewCipher(keyE)
	if err != nil {
		return nil, err
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	// same length as ciphertext
	plaintext := make([]byte, len(in)-offset-sha256.Size)
	mode.CryptBlocks(plaintext, in[offset:len(in)-sha256.Size])

	return removePKCSPadding(plaintext)
}

// Implement PKCS#7 padding with block size of 16 (AES block size).

// addPKCSPadding adds padding to a block of data
func addPKCSPadding(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

// removePKCSPadding removes padding from data that was added with addPKCSPadding
func removePKCSPadding(src []byte) ([]byte, error) {
	length := len(src)
	padLength := int(src[length-1])
	if padLength > aes.BlockSize || length < aes.BlockSize {
		return nil, errInvalidPadding
	}

	return src[:length-padLength], nil
}

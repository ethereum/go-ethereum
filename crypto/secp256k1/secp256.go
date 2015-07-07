// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package secp256k1

// TODO: set USE_SCALAR_4X64 depending on platform?

/*
#cgo CFLAGS: -I./secp256k1
#cgo darwin CFLAGS: -I/usr/local/include
#cgo linux,arm CFLAGS: -I/usr/local/arm/include
#cgo LDFLAGS: -lgmp
#cgo darwin LDFLAGS: -L/usr/local/lib
#cgo linux,arm LDFLAGS: -L/usr/local/arm/lib
#define USE_NUM_GMP
#define USE_FIELD_10X26
#define USE_FIELD_INV_BUILTIN
#define USE_SCALAR_8X32
#define USE_SCALAR_INV_BUILTIN
#define NDEBUG
#include "./secp256k1/src/secp256k1.c"
*/
import "C"

import (
	"bytes"
	"errors"
	"unsafe"

	"github.com/ethereum/go-ethereum/crypto/randentropy"
)

//#define USE_FIELD_5X64

/*
   Todo:
   > Centralize key management in module
   > add pubkey/private key struct
   > Dont let keys leave module; address keys as ints

   > store private keys in buffer and shuffle (deters persistance on swap disc)
   > Byte permutation (changing)
   > xor with chaning random block (to deter scanning memory for 0x63) (stream cipher?)

   On Disk
   > Store keys in wallets
   > use slow key derivation function for wallet encryption key (2 seconds)
*/

func init() {
	//takes 10ms to 100ms
	C.secp256k1_start(3) // SECP256K1_START_SIGN | SECP256K1_START_VERIFY
}

func Stop() {
	C.secp256k1_stop()
}

func GenerateKeyPair() ([]byte, []byte) {

	pubkey_len := C.int(65)
	const seckey_len = 32

	var pubkey []byte = make([]byte, pubkey_len)
	var seckey []byte = randentropy.GetEntropyCSPRNG(seckey_len)

	var pubkey_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&pubkey[0]))
	var seckey_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&seckey[0]))

	ret := C.secp256k1_ec_pubkey_create(
		pubkey_ptr, &pubkey_len,
		seckey_ptr, 0)

	if ret != C.int(1) {
		return GenerateKeyPair() //invalid secret, try again
	}
	return pubkey, seckey
}

func GeneratePubKey(seckey []byte) ([]byte, error) {
	if err := VerifySeckeyValidity(seckey); err != nil {
		return nil, err
	}

	pubkey_len := C.int(65)
	const seckey_len = 32

	var pubkey []byte = make([]byte, pubkey_len)

	var pubkey_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&pubkey[0]))
	var seckey_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&seckey[0]))

	ret := C.secp256k1_ec_pubkey_create(
		pubkey_ptr, &pubkey_len,
		seckey_ptr, 0)

	if ret != C.int(1) {
		return nil, errors.New("Unable to generate pubkey from seckey")
	}

	return pubkey, nil
}

func Sign(msg []byte, seckey []byte) ([]byte, error) {
	nonce := randentropy.GetEntropyCSPRNG(32)

	var sig []byte = make([]byte, 65)
	var recid C.int

	var msg_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&msg[0]))
	var sig_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&sig[0]))
	var seckey_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&seckey[0]))

	var noncefp_ptr = &(*C.secp256k1_nonce_function_default)
	var ndata_ptr = unsafe.Pointer(&nonce[0])

	if C.secp256k1_ec_seckey_verify(seckey_ptr) != C.int(1) {
		return nil, errors.New("Invalid secret key")
	}

	ret := C.secp256k1_ecdsa_sign_compact(
		msg_ptr,
		sig_ptr,
		seckey_ptr,
		noncefp_ptr,
		ndata_ptr,
		&recid)

	sig[64] = byte(int(recid))

	if ret != C.int(1) {
		// nonce invalid, retry
		return Sign(msg, seckey)
	}

	return sig, nil

}

func VerifySeckeyValidity(seckey []byte) error {
	if len(seckey) != 32 {
		return errors.New("priv key is not 32 bytes")
	}
	var seckey_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&seckey[0]))
	ret := C.secp256k1_ec_seckey_verify(seckey_ptr)
	if int(ret) != 1 {
		return errors.New("invalid seckey")
	}
	return nil
}

func VerifyPubkeyValidity(pubkey []byte) error {
	if len(pubkey) != 65 {
		return errors.New("pub key is not 65 bytes")
	}
	var pubkey_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&pubkey[0]))
	ret := C.secp256k1_ec_pubkey_verify(pubkey_ptr, 65)
	if int(ret) != 1 {
		return errors.New("invalid pubkey")
	}

	return nil
}

func VerifySignatureValidity(sig []byte) bool {
	//64+1
	if len(sig) != 65 {
		return false
	}
	//malleability check, highest bit must be 1
	if (sig[32] & 0x80) == 0x80 {
		return false
	}
	//recovery id check
	if sig[64] >= 4 {
		return false
	}

	return true
}

//for compressed signatures, does not need pubkey
func VerifySignature(msg []byte, sig []byte, pubkey1 []byte) error {
	if msg == nil || sig == nil || pubkey1 == nil {
		return errors.New("inputs must be non-nil")
	}
	if len(sig) != 65 {
		return errors.New("invalid signature length")
	}
	if len(pubkey1) != 65 {
		return errors.New("Invalid public key length")
	}

	//to enforce malleability, highest bit of S must be 0
	//S starts at 32nd byte
	if (sig[32] & 0x80) == 0x80 { //highest bit must be 1
		return errors.New("Signature not malleable")
	}

	if sig[64] >= 4 {
		return errors.New("Recover byte invalid")
	}

	// if pubkey recovered, signature valid
	pubkey2, err := RecoverPubkey(msg, sig)
	if err != nil {
		return err
	}
	if len(pubkey2) != 65 {
		return errors.New("Invalid recovered public key length")
	}
	if !bytes.Equal(pubkey1, pubkey2) {
		return errors.New("Public key does not match recovered public key")
	}

	return nil
}

//recovers the public key from the signature
//recovery of pubkey means correct signature
func RecoverPubkey(msg []byte, sig []byte) ([]byte, error) {
	if len(sig) != 65 {
		return nil, errors.New("Invalid signature length")
	}

	var pubkey []byte = make([]byte, 65)

	var msg_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&msg[0]))
	var sig_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&sig[0]))
	var pubkey_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&pubkey[0]))

	var pubkeylen C.int

	ret := C.secp256k1_ecdsa_recover_compact(
		msg_ptr,
		sig_ptr,
		pubkey_ptr,
		&pubkeylen,
		C.int(0),
		C.int(sig[64]),
	)

	if ret == C.int(0) {
		return nil, errors.New("Failed to recover public key")
	} else if pubkeylen != C.int(65) {
		return nil, errors.New("Impossible Error: Invalid recovered public key length")
	} else {
		return pubkey, nil
	}
	return nil, errors.New("Impossible Error: func RecoverPubkey has reached an unreachable state")
}

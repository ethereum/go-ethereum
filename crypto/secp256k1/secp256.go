package secp256k1

/*
#cgo CFLAGS: -std=gnu99 -Wno-error
#cgo darwin CFLAGS: -I/usr/local/include
#cgo LDFLAGS: -lgmp
#cgo darwin LDFLAGS: -L/usr/local/lib
#define USE_FIELD_10X26
#define USE_NUM_GMP
#define USE_FIELD_INV_BUILTIN
#include "./secp256k1/src/secp256k1.c"
*/
import "C"

import (
	"bytes"
	"errors"
	"unsafe"
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
	C.secp256k1_start() //takes 10ms to 100ms
}

func Stop() {
	C.secp256k1_stop()
}

/*
int secp256k1_ecdsa_pubkey_create(
    unsigned char *pubkey, int *pubkeylen,
    const unsigned char *seckey, int compressed);
*/

/** Compute the public key for a secret key.
 *  In:     compressed: whether the computed public key should be compressed
 *          seckey:     pointer to a 32-byte private key.
 *  Out:    pubkey:     pointer to a 33-byte (if compressed) or 65-byte (if uncompressed)
 *                      area to store the public key.
 *          pubkeylen:  pointer to int that will be updated to contains the pubkey's
 *                      length.
 *  Returns: 1: secret was valid, public key stores
 *           0: secret was invalid, try again.
 */

//pubkey, seckey

func GenerateKeyPair() ([]byte, []byte) {

	pubkey_len := C.int(65)
	const seckey_len = 32

	var pubkey []byte = make([]byte, pubkey_len)
	var seckey []byte = RandByte(seckey_len)

	var pubkey_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&pubkey[0]))
	var seckey_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&seckey[0]))

	ret := C.secp256k1_ecdsa_pubkey_create(
		pubkey_ptr, &pubkey_len,
		seckey_ptr, 0)

	if ret != C.int(1) {
		return GenerateKeyPair() //invalid secret, try again
	}
	return pubkey, seckey
}

func GeneratePubKey(seckey []byte) ([]byte, error) {
	pubkey_len := C.int(65)
	const seckey_len = 32

	var pubkey []byte = make([]byte, pubkey_len)

	var pubkey_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&pubkey[0]))
	var seckey_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&seckey[0]))

	ret := C.secp256k1_ecdsa_pubkey_create(
		pubkey_ptr, &pubkey_len,
		seckey_ptr, 0)

	if ret != C.int(1) {
		return nil, errors.New("Unable to generate pubkey from seckey")
	}

	return pubkey, nil
}

/*
*  Create a compact ECDSA signature (64 byte + recovery id).
*  Returns: 1: signature created
*           0: nonce invalid, try another one
*  In:      msg:    the message being signed
*           msglen: the length of the message being signed
*           seckey: pointer to a 32-byte secret key (assumed to be valid)
*           nonce:  pointer to a 32-byte nonce (generated with a cryptographic PRNG)
*  Out:     sig:    pointer to a 64-byte array where the signature will be placed.
*           recid:  pointer to an int, which will be updated to contain the recovery id.
 */

/*
int secp256k1_ecdsa_sign_compact(const unsigned char *msg, int msglen,
                                 unsigned char *sig64,
                                 const unsigned char *seckey,
                                 const unsigned char *nonce,
                                 int *recid);
*/

func Sign(msg []byte, seckey []byte) ([]byte, error) {
	//var nonce []byte = RandByte(32)
	nonce := make([]byte, 32)
	for i := range msg {
		nonce[i] = msg[i] ^ seckey[i]
	}

	var sig []byte = make([]byte, 65)
	var recid C.int

	var msg_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&msg[0]))
	var seckey_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&seckey[0]))
	var nonce_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&nonce[0]))
	var sig_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&sig[0]))

	if C.secp256k1_ecdsa_seckey_verify(seckey_ptr) != C.int(1) {
		return nil, errors.New("Invalid secret key")
	}

	ret := C.secp256k1_ecdsa_sign_compact(
		msg_ptr, C.int(len(msg)),
		sig_ptr,
		seckey_ptr,
		nonce_ptr,
		&recid)

	sig[64] = byte(int(recid))

	if ret != C.int(1) {
		// nonce invalid, retry
		return Sign(msg, seckey)
	}

	return sig, nil

}

/*
* Verify an ECDSA secret key.
*  Returns: 1: secret key is valid
*           0: secret key is invalid
*  In:      seckey: pointer to a 32-byte secret key
 */

func VerifySeckeyValidity(seckey []byte) error {
	if len(seckey) != 32 {
		return errors.New("priv key is not 32 bytes")
	}
	var seckey_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&seckey[0]))
	ret := C.secp256k1_ecdsa_seckey_verify(seckey_ptr)
	if int(ret) != 1 {
		return errors.New("invalid seckey")
	}
	return nil
}

/*
* Validate a public key.
*  Returns: 1: valid public key
*           0: invalid public key
 */

func VerifyPubkeyValidity(pubkey []byte) error {
	if len(pubkey) != 65 {
		return errors.New("pub key is not 65 bytes")
	}
	var pubkey_ptr *C.uchar = (*C.uchar)(unsafe.Pointer(&pubkey[0]))
	ret := C.secp256k1_ecdsa_pubkey_verify(pubkey_ptr, 65)
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

/*
int secp256k1_ecdsa_recover_compact(const unsigned char *msg, int msglen,
                                    const unsigned char *sig64,
                                    unsigned char *pubkey, int *pubkeylen,
                                    int compressed, int recid);
*/

/*
 * Recover an ECDSA public key from a compact signature.
 *  Returns: 1: public key succesfully recovered (which guarantees a correct signature).
 *           0: otherwise.
 *  In:      msg:        the message assumed to be signed
 *           msglen:     the length of the message
 *           compressed: whether to recover a compressed or uncompressed pubkey
 *           recid:      the recovery id (as returned by ecdsa_sign_compact)
 *  Out:     pubkey:     pointer to a 33 or 65 byte array to put the pubkey.
 *           pubkeylen:  pointer to an int that will contain the pubkey length.
 */

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
		msg_ptr, C.int(len(msg)),
		sig_ptr,
		pubkey_ptr, &pubkeylen,
		C.int(0), C.int(sig[64]),
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

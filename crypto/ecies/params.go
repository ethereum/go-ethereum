package ecies

// This file contains parameters for ECIES encryption, specifying the
// symmetric encryption and HMAC parameters.

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
)

// The default curve for this package is the NIST P256 curve, which
// provides security equivalent to AES-128.
var DefaultCurve = elliptic.P256()

var (
	ErrUnsupportedECDHAlgorithm   = fmt.Errorf("ecies: unsupported ECDH algorithm")
	ErrUnsupportedECIESParameters = fmt.Errorf("ecies: unsupported ECIES parameters")
)

type ECIESParams struct {
	Hash      func() hash.Hash // hash function
	hashAlgo  crypto.Hash
	Cipher    func([]byte) (cipher.Block, error) // symmetric cipher
	BlockSize int                                // block size of symmetric cipher
	KeyLen    int                                // length of symmetric key
}

// Standard ECIES parameters:
// * ECIES using AES128 and HMAC-SHA-256-16
// * ECIES using AES256 and HMAC-SHA-256-32
// * ECIES using AES256 and HMAC-SHA-384-48
// * ECIES using AES256 and HMAC-SHA-512-64

var (
	ECIES_AES128_SHA256 = &ECIESParams{
		Hash:      sha256.New,
		hashAlgo:  crypto.SHA256,
		Cipher:    aes.NewCipher,
		BlockSize: aes.BlockSize,
		KeyLen:    16,
	}

	ECIES_AES256_SHA256 = &ECIESParams{
		Hash:      sha256.New,
		hashAlgo:  crypto.SHA256,
		Cipher:    aes.NewCipher,
		BlockSize: aes.BlockSize,
		KeyLen:    32,
	}

	ECIES_AES256_SHA384 = &ECIESParams{
		Hash:      sha512.New384,
		hashAlgo:  crypto.SHA384,
		Cipher:    aes.NewCipher,
		BlockSize: aes.BlockSize,
		KeyLen:    32,
	}

	ECIES_AES256_SHA512 = &ECIESParams{
		Hash:      sha512.New,
		hashAlgo:  crypto.SHA512,
		Cipher:    aes.NewCipher,
		BlockSize: aes.BlockSize,
		KeyLen:    32,
	}
)

var paramsFromCurve = map[elliptic.Curve]*ECIESParams{
	elliptic.P256(): ECIES_AES128_SHA256,
	elliptic.P384(): ECIES_AES256_SHA384,
	elliptic.P521(): ECIES_AES256_SHA512,
}

func AddParamsForCurve(curve elliptic.Curve, params *ECIESParams) {
	paramsFromCurve[curve] = params
}

// ParamsFromCurve selects parameters optimal for the selected elliptic curve.
// Only the curves P256, P384, and P512 are supported.
func ParamsFromCurve(curve elliptic.Curve) (params *ECIESParams) {
	return paramsFromCurve[curve]

	/*
		switch curve {
		case elliptic.P256():
			return ECIES_AES128_SHA256
		case elliptic.P384():
			return ECIES_AES256_SHA384
		case elliptic.P521():
			return ECIES_AES256_SHA512
		default:
			return nil
		}
	*/
}

// ASN.1 encode the ECIES parameters relevant to the encryption operations.
func paramsToASNECIES(params *ECIESParams) (asnParams asnECIESParameters) {
	if nil == params {
		return
	}
	asnParams.KDF = asnNISTConcatenationKDF
	asnParams.MAC = hmacFull
	switch params.KeyLen {
	case 16:
		asnParams.Sym = aes128CTRinECIES
	case 24:
		asnParams.Sym = aes192CTRinECIES
	case 32:
		asnParams.Sym = aes256CTRinECIES
	}
	return
}

// ASN.1 encode the ECIES parameters relevant to ECDH.
func paramsToASNECDH(params *ECIESParams) (algo asnECDHAlgorithm) {
	switch params.hashAlgo {
	case crypto.SHA224:
		algo = dhSinglePass_stdDH_sha224kdf
	case crypto.SHA256:
		algo = dhSinglePass_stdDH_sha256kdf
	case crypto.SHA384:
		algo = dhSinglePass_stdDH_sha384kdf
	case crypto.SHA512:
		algo = dhSinglePass_stdDH_sha512kdf
	}
	return
}

// ASN.1 decode the ECIES parameters relevant to the encryption stage.
func asnECIEStoParams(asnParams asnECIESParameters, params *ECIESParams) {
	if !asnParams.KDF.Cmp(asnNISTConcatenationKDF) {
		params = nil
		return
	} else if !asnParams.MAC.Cmp(hmacFull) {
		params = nil
		return
	}

	switch {
	case asnParams.Sym.Cmp(aes128CTRinECIES):
		params.KeyLen = 16
		params.BlockSize = 16
		params.Cipher = aes.NewCipher
	case asnParams.Sym.Cmp(aes192CTRinECIES):
		params.KeyLen = 24
		params.BlockSize = 16
		params.Cipher = aes.NewCipher
	case asnParams.Sym.Cmp(aes256CTRinECIES):
		params.KeyLen = 32
		params.BlockSize = 16
		params.Cipher = aes.NewCipher
	default:
		params = nil
	}
}

// ASN.1 decode the ECIES parameters relevant to ECDH.
func asnECDHtoParams(asnParams asnECDHAlgorithm, params *ECIESParams) {
	if asnParams.Cmp(dhSinglePass_stdDH_sha224kdf) {
		params.hashAlgo = crypto.SHA224
		params.Hash = sha256.New224
	} else if asnParams.Cmp(dhSinglePass_stdDH_sha256kdf) {
		params.hashAlgo = crypto.SHA256
		params.Hash = sha256.New
	} else if asnParams.Cmp(dhSinglePass_stdDH_sha384kdf) {
		params.hashAlgo = crypto.SHA384
		params.Hash = sha512.New384
	} else if asnParams.Cmp(dhSinglePass_stdDH_sha512kdf) {
		params.hashAlgo = crypto.SHA512
		params.Hash = sha512.New
	} else {
		params = nil
	}
}

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"encoding/hex"
	"encoding/json"
	"errors"

	"code.google.com/p/go-uuid/uuid"
	"code.google.com/p/go.crypto/pbkdf2"
	"code.google.com/p/go.crypto/ripemd160"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/obscuren/ecies"
)

func init() {
	// specify the params for the s256 curve
	ecies.AddParamsForCurve(S256(), ecies.ECIES_AES128_SHA256)
}

func Sha3(data []byte) []byte {
	d := sha3.NewKeccak256()
	d.Write(data)

	return d.Sum(nil)
}

// Creates an ethereum address given the bytes and the nonce
func CreateAddress(b []byte, nonce uint64) []byte {
	return Sha3(ethutil.NewValue([]interface{}{b, nonce}).Encode())[12:]
}

func Sha256(data []byte) []byte {
	hash := sha256.Sum256(data)

	return hash[:]
}

func Ripemd160(data []byte) []byte {
	ripemd := ripemd160.New()
	ripemd.Write(data)

	return ripemd.Sum(nil)
}

func Ecrecover(data []byte) []byte {
	var in = struct {
		hash []byte
		sig  []byte
	}{data[:32], data[32:]}

	r, _ := secp256k1.RecoverPubkey(in.hash, in.sig)

	return r
}

// New methods using proper ecdsa keys from the stdlib
func ToECDSA(prv []byte) *ecdsa.PrivateKey {
	if len(prv) == 0 {
		return nil
	}

	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = S256()
	priv.D = ethutil.BigD(prv)
	priv.PublicKey.X, priv.PublicKey.Y = S256().ScalarBaseMult(prv)
	return priv
}

func FromECDSA(prv *ecdsa.PrivateKey) []byte {
	if prv == nil {
		return nil
	}
	return prv.D.Bytes()
}

func ToECDSAPub(pub []byte) *ecdsa.PublicKey {
	if len(pub) == 0 {
		return nil
	}
	x, y := elliptic.Unmarshal(S256(), pub)
	return &ecdsa.PublicKey{S256(), x, y}
}

func FromECDSAPub(pub *ecdsa.PublicKey) []byte {
	if pub == nil {
		return nil
	}
	return elliptic.Marshal(S256(), pub.X, pub.Y)
}

func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(S256(), rand.Reader)
}

func SigToPub(hash, sig []byte) *ecdsa.PublicKey {
	s := Ecrecover(append(hash, sig...))
	x, y := elliptic.Unmarshal(S256(), s)

	return &ecdsa.PublicKey{S256(), x, y}
}

func Sign(hash []byte, prv *ecdsa.PrivateKey) (sig []byte, err error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash is required to be exactly 32 bytes (%d)", len(hash))
	}

	sig, err = secp256k1.Sign(hash, ethutil.LeftPadBytes(prv.D.Bytes(), prv.Params().BitSize/8))
	return
}

func Encrypt(pub *ecdsa.PublicKey, message []byte) ([]byte, error) {
	return ecies.Encrypt(rand.Reader, ecies.ImportECDSAPublic(pub), message, nil, nil)
}

func Decrypt(prv *ecdsa.PrivateKey, ct []byte) ([]byte, error) {
	key := ecies.ImportECDSA(prv)
	return key.Decrypt(rand.Reader, ct, nil, nil)
}

// creates a Key and stores that in the given KeyStore by decrypting a presale key JSON
func ImportPreSaleKey(keyStore KeyStore2, keyJSON []byte, password string) (*Key, error) {
	key, err := decryptPreSaleKey(keyJSON, password)
	if err != nil {
		return nil, err
	}
	id := uuid.NewRandom()
	key.Id = id
	err = keyStore.StoreKey(key, password)
	return key, err
}

func decryptPreSaleKey(fileContent []byte, password string) (key *Key, err error) {
	preSaleKeyStruct := struct {
		EncSeed string
		EthAddr string
		Email   string
		BtcAddr string
	}{}
	err = json.Unmarshal(fileContent, &preSaleKeyStruct)
	if err != nil {
		return nil, err
	}
	encSeedBytes, err := hex.DecodeString(preSaleKeyStruct.EncSeed)
	iv := encSeedBytes[:16]
	cipherText := encSeedBytes[16:]
	/*
		See https://github.com/ethereum/pyethsaletool

		pyethsaletool generates the encryption key from password by
		2000 rounds of PBKDF2 with HMAC-SHA-256 using password as salt (:().
		16 byte key length within PBKDF2 and resulting key is used as AES key
	*/
	passBytes := []byte(password)
	derivedKey := pbkdf2.Key(passBytes, passBytes, 2000, 16, sha256.New)
	plainText, err := aesCBCDecrypt(derivedKey, cipherText, iv)
	ethPriv := Sha3(plainText)
	ecKey := ToECDSA(ethPriv)
	key = &Key{
		Id:         nil,
		Address:    pubkeyToAddress(ecKey.PublicKey),
		PrivateKey: ecKey,
	}
	derivedAddr := ethutil.Bytes2Hex(key.Address)
	expectedAddr := preSaleKeyStruct.EthAddr
	if derivedAddr != expectedAddr {
		err = errors.New("decrypted addr not equal to expected addr")
	}
	return key, err
}

func aesCBCDecrypt(key []byte, cipherText []byte, iv []byte) (plainText []byte, err error) {
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return plainText, err
	}
	decrypter := cipher.NewCBCDecrypter(aesBlock, iv)
	paddedPlainText := make([]byte, len(cipherText))
	decrypter.CryptBlocks(paddedPlainText, cipherText)
	plainText = PKCS7Unpad(paddedPlainText)
	if plainText == nil {
		err = errors.New("Decryption failed: PKCS7Unpad failed after decryption")
	}
	return plainText, err
}

// From https://leanpub.com/gocrypto/read#leanpub-auto-block-cipher-modes
func PKCS7Pad(in []byte) []byte {
	padding := 16 - (len(in) % 16)
	if padding == 0 {
		padding = 16
	}
	for i := 0; i < padding; i++ {
		in = append(in, byte(padding))
	}
	return in
}

func PKCS7Unpad(in []byte) []byte {
	if len(in) == 0 {
		return nil
	}

	padding := in[len(in)-1]
	if int(padding) > len(in) || padding > aes.BlockSize {
		return nil
	} else if padding == 0 {
		return nil
	}

	for i := len(in) - 1; i > len(in)-int(padding)-1; i-- {
		if in[i] != padding {
			return nil
		}
	}
	return in[:len(in)-int(padding)]
}

func pubkeyToAddress(p ecdsa.PublicKey) []byte {
	pubBytes := FromECDSAPub(&p)
	return Sha3(pubBytes[1:])[12:]
}

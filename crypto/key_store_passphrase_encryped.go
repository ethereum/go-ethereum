/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU Lesser General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU Lesser General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Gustav Simonsson <gustav.simonsson@gmail.com>
 * @date 2015
 *
 */
/*

This key store behaves as KeyStorePlaintextFile with the difference that
the private key is encrypted and encoded as a JSON object within the
key JSON object.

Cryptography:

1. Encryption key is scrypt derived key from user passphrase. Scrypt parameters
   (work factors) [1][2] are defined as constants below.
2. Scrypt salt is 32 random bytes from CSPRNG. It is appended to ciphertext.
3. Checksum is SHA3 of the private key bytes.
4. Plaintext is concatenation of private key bytes and checksum.
5. Encryption algo is AES 256 CBC [3][4]
6. CBC IV is 16 random bytes from CSPRNG. It is appended to ciphertext.
7. Plaintext padding is PKCS #7 [5][6]

Encoding:

1. On disk, ciphertext, salt and IV are encoded as a JSON object.
   cat a key file to see the structure.
2. byte arrays are ASCII HEX encoded as JSON strings.
3. The EC private key bytes are in uncompressed form [7].
   They are a big-endian byte slice of the absolute value of D [8][9].
4. The checksum is the last 32 bytes of the plaintext byte array and the
   private key is the preceeding bytes.

References:

1. http://www.tarsnap.com/scrypt/scrypt-slides.pdf
2. http://stackoverflow.com/questions/11126315/what-are-optimal-scrypt-work-factors
3. http://en.wikipedia.org/wiki/Advanced_Encryption_Standard
4. http://en.wikipedia.org/wiki/Block_cipher_mode_of_operation#Cipher-block_chaining_.28CBC.29
5. https://leanpub.com/gocrypto/read#leanpub-auto-block-cipher-modes
6. http://tools.ietf.org/html/rfc2315
7. http://bitcoin.stackexchange.com/questions/3059/what-is-a-compressed-bitcoin-key
8. http://golang.org/pkg/crypto/ecdsa/#PrivateKey
9. https://golang.org/pkg/math/big/#Int.Bytes

*/

package crypto

import (
	"bytes"
	"code.google.com/p/go-uuid/uuid"
	"code.google.com/p/go.crypto/scrypt"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path"
)

const scryptN int = 262144 // 2^18
const scryptr int = 8
const scryptp int = 1
const scryptdkLen int = 32

type KeyStorePassphrase struct {
	keysDirPath string
}

func (ks KeyStorePassphrase) GenerateNewKey(auth string) (key *Key, err error) {
	key, err = GenerateNewKeyDefault(ks, auth)
	return
}

func (ks KeyStorePassphrase) GetKey(keyId *uuid.UUID, auth string) (key *Key, err error) {
	keyBytes, flags, err := DecryptKey(ks, keyId, auth)
	key = new(Key)
	key.Id = keyId
	copy(key.Flags[:], flags[0:4])
	key.PrivateKey = ToECDSA(keyBytes)
	return
}

func (ks KeyStorePassphrase) StoreKey(key *Key, auth string) (err error) {
	authArray := []byte(auth)
	salt := GetEntropyCSPRNG(32)
	derivedKey, err := scrypt.Key(authArray, salt, scryptN, scryptr, scryptp, scryptdkLen)
	if err != nil {
		return
	}

	keyBytes := FromECDSA(key.PrivateKey)
	keyBytesHash := Sha3(keyBytes)
	toEncrypt := PKCS7Pad(append(keyBytes, keyBytesHash...))

	AES256Block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return
	}

	iv := GetEntropyCSPRNG(aes.BlockSize) // 16
	AES256CBCEncrypter := cipher.NewCBCEncrypter(AES256Block, iv)
	cipherText := make([]byte, len(toEncrypt))
	AES256CBCEncrypter.CryptBlocks(cipherText, toEncrypt)

	cipherStruct := CipherJSON{
		hex.EncodeToString(salt),
		hex.EncodeToString(iv),
		hex.EncodeToString(cipherText),
	}
	keyStruct := KeyProtectedJSON{
		key.Id.String(),
		hex.EncodeToString(key.Flags[:]),
		cipherStruct,
	}
	keyJSON, err := json.Marshal(keyStruct)
	if err != nil {
		return
	}

	err = WriteKeyFile(key.Id.String(), ks.keysDirPath, keyJSON)
	return
}

func (ks KeyStorePassphrase) DeleteKey(keyId *uuid.UUID, auth string) (err error) {
	// only delete if correct passphrase is given
	_, _, err = DecryptKey(ks, keyId, auth)
	if err != nil {
		return
	}

	keyDirPath := path.Join(ks.keysDirPath, keyId.String())
	err = os.RemoveAll(keyDirPath)
	return
}

func DecryptKey(ks KeyStorePassphrase, keyId *uuid.UUID, auth string) (keyBytes []byte, flags []byte, err error) {
	fileContent, err := GetKeyFile(ks.keysDirPath, keyId)
	if err != nil {
		return
	}

	keyProtected := new(KeyProtectedJSON)
	err = json.Unmarshal(fileContent, keyProtected)

	flags, err = hex.DecodeString(keyProtected.Flags)
	if err != nil {
		return
	}

	salt, err := hex.DecodeString(keyProtected.Crypto.Salt)
	if err != nil {
		return
	}

	iv, err := hex.DecodeString(keyProtected.Crypto.IV)
	if err != nil {
		return
	}

	cipherText, err := hex.DecodeString(keyProtected.Crypto.CipherText)
	if err != nil {
		return
	}

	authArray := []byte(auth)
	derivedKey, err := scrypt.Key(authArray, salt, scryptN, scryptr, scryptp, scryptdkLen)
	if err != nil {
		return
	}

	AES256Block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return
	}

	AES256CBCDecrypter := cipher.NewCBCDecrypter(AES256Block, iv)
	paddedPlainText := make([]byte, len(cipherText))
	AES256CBCDecrypter.CryptBlocks(paddedPlainText, cipherText)

	plainText := PKCS7Unpad(paddedPlainText)
	if plainText == nil {
		err = errors.New("Decryption failed: PKCS7Unpad failed after decryption")
		return
	}

	keyBytes = plainText[:len(plainText)-32]
	keyBytesHash := plainText[len(plainText)-32:]
	if !bytes.Equal(Sha3(keyBytes), keyBytesHash) {
		err = errors.New("Decryption failed: checksum mismatch")
		return
	}
	return keyBytes, flags, err
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

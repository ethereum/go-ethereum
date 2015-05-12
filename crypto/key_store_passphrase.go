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

This key store behaves as KeyStorePlain with the difference that
the private key is encrypted and on disk uses another JSON encoding.

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

1. On disk, ciphertext, salt and IV are encoded in a nested JSON object.
   cat a key file to see the structure.
2. byte arrays are base64 JSON strings.
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
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"code.google.com/p/go-uuid/uuid"
	"github.com/ethereum/go-ethereum/crypto/randentropy"
	"golang.org/x/crypto/scrypt"
)

const (
	// 2^18 / 8 / 1 uses 256MB memory and approx 1s CPU time on a modern CPU.
	scryptN     = 1 << 18
	scryptr     = 8
	scryptp     = 1
	scryptdkLen = 32
)

type keyStorePassphrase struct {
	keysDirPath string
}

func NewKeyStorePassphrase(path string) KeyStore2 {
	return &keyStorePassphrase{path}
}

func (ks keyStorePassphrase) GenerateNewKey(rand io.Reader, auth string) (key *Key, err error) {
	return GenerateNewKeyDefault(ks, rand, auth)
}

func (ks keyStorePassphrase) GetKey(keyAddr []byte, auth string) (key *Key, err error) {
	keyBytes, keyId, err := DecryptKey(ks, keyAddr, auth)
	if err != nil {
		return nil, err
	}
	key = &Key{
		Id:         uuid.UUID(keyId),
		Address:    keyAddr,
		PrivateKey: ToECDSA(keyBytes),
	}
	return key, err
}

func (ks keyStorePassphrase) GetKeyAddresses() (addresses [][]byte, err error) {
	return GetKeyAddresses(ks.keysDirPath)
}

func (ks keyStorePassphrase) StoreKey(key *Key, auth string) (err error) {
	authArray := []byte(auth)
	salt := randentropy.GetEntropyMixed(32)
	derivedKey, err := scrypt.Key(authArray, salt, scryptN, scryptr, scryptp, scryptdkLen)
	if err != nil {
		return err
	}

	keyBytes := FromECDSA(key.PrivateKey)
	keyBytesHash := Sha3(keyBytes)
	toEncrypt := PKCS7Pad(append(keyBytes, keyBytesHash...))

	AES256Block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return err
	}

	iv := randentropy.GetEntropyMixed(aes.BlockSize) // 16
	AES256CBCEncrypter := cipher.NewCBCEncrypter(AES256Block, iv)
	cipherText := make([]byte, len(toEncrypt))
	AES256CBCEncrypter.CryptBlocks(cipherText, toEncrypt)

	cipherStruct := cipherJSON{
		salt,
		iv,
		cipherText,
	}
	keyStruct := encryptedKeyJSON{
		key.Id,
		key.Address,
		cipherStruct,
	}
	keyJSON, err := json.Marshal(keyStruct)
	if err != nil {
		return err
	}

	return WriteKeyFile(key.Address, ks.keysDirPath, keyJSON)
}

func (ks keyStorePassphrase) DeleteKey(keyAddr []byte, auth string) (err error) {
	// only delete if correct passphrase is given
	_, _, err = DecryptKey(ks, keyAddr, auth)
	if err != nil {
		return err
	}

	keyDirPath := filepath.Join(ks.keysDirPath, hex.EncodeToString(keyAddr))
	return os.RemoveAll(keyDirPath)
}

func DecryptKey(ks keyStorePassphrase, keyAddr []byte, auth string) (keyBytes []byte, keyId []byte, err error) {
	fileContent, err := GetKeyFile(ks.keysDirPath, keyAddr)
	if err != nil {
		return nil, nil, err
	}

	keyProtected := new(encryptedKeyJSON)
	err = json.Unmarshal(fileContent, keyProtected)

	keyId = keyProtected.Id
	salt := keyProtected.Crypto.Salt
	iv := keyProtected.Crypto.IV
	cipherText := keyProtected.Crypto.CipherText

	authArray := []byte(auth)
	derivedKey, err := scrypt.Key(authArray, salt, scryptN, scryptr, scryptp, scryptdkLen)
	if err != nil {
		return nil, nil, err
	}
	plainText, err := aesCBCDecrypt(derivedKey, cipherText, iv)
	if err != nil {
		return nil, nil, err
	}
	keyBytes = plainText[:len(plainText)-32]
	keyBytesHash := plainText[len(plainText)-32:]
	if !bytes.Equal(Sha3(keyBytes), keyBytesHash) {
		err = errors.New("Decryption failed: checksum mismatch")
		return nil, nil, err
	}
	return keyBytes, keyId, err
}

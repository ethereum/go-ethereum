// Copyright 2014 The go-ethereum Authors
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

package keystore

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pborman/uuid"
)

const (
	version = 3
)

type Key struct {
	Id uuid.UUID // Version 4 "random" for unique id not derived from key data
	// to simplify lookups we also store the address
	Address common.Address
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	PrivateKey *ecdsa.PrivateKey
	// add a second privkey for privary
	PrivateKey2 *ecdsa.PrivateKey
	// compact usechain address format
	UAddress common.UAddress
}

// Used to import and export raw keypair
type keyPair struct {
	D  string `json:"privateKey"`
	D1 string `json:"privateKey1"`
}

type keyStore interface {
	// Loads and decrypts the key from disk.
	GetKey(addr common.Address, filename string, auth string) (*Key, error)
	// Writes and encrypts the key.
	StoreKey(filename string, k *Key, auth string) error
	// Loads an encrypted keyfile from disk
	GetEncryptedKey(addr common.Address, filename string) (*Key, error)
	// Joins filename with the key directory unless it is already absolute.
	JoinPath(filename string) string
}

type plainKeyJSON struct {
	Address    string `json:"address"`
	PrivateKey string `json:"privatekey"`
	Id         string `json:"id"`
	Version    int    `json:"version"`
}

type encryptedKeyJSONV3 struct {
	Address  string     `json:"address"`
	Crypto   CryptoJSON `json:"crypto"`
	Crypto2  CryptoJSON `json:"crypto2"`
	Id       string     `json:"id"`
	Version  int        `json:"version"`
	UAddress string     `json:"uaddress"`
}

type encryptedKeyJSONV1 struct {
	Address string     `json:"address"`
	Crypto  CryptoJSON `json:"crypto"`
	Id      string     `json:"id"`
	Version string     `json:"version"`
}

type CryptoJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherparamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

type cipherparamsJSON struct {
	IV string `json:"iv"`
}

func (k *Key) MarshalJSON() (j []byte, err error) {
	jStruct := plainKeyJSON{
		hex.EncodeToString(k.Address[:]),
		hex.EncodeToString(crypto.FromECDSA(k.PrivateKey)),
		k.Id.String(),
		version,
	}
	j, err = json.Marshal(jStruct)
	return j, err
}

func (k *Key) UnmarshalJSON(j []byte) (err error) {
	keyJSON := new(plainKeyJSON)
	err = json.Unmarshal(j, &keyJSON)
	if err != nil {
		return err
	}

	u := new(uuid.UUID)
	*u = uuid.Parse(keyJSON.Id)
	k.Id = *u
	addr, err := hex.DecodeString(keyJSON.Address)
	if err != nil {
		return err
	}
	privkey, err := crypto.HexToECDSA(keyJSON.PrivateKey)
	if err != nil {
		return err
	}

	k.Address = common.BytesToAddress(addr)
	k.PrivateKey = privkey

	return nil
}

func newKeyFromECDSA(sk1, sk2 *ecdsa.PrivateKey) *Key {
	id := uuid.NewRandom()
	key := &Key{
		Id:          id,
		Address:     crypto.PubkeyToAddress(sk1.PublicKey),
		PrivateKey:  sk1,
		PrivateKey2: sk2,
	}

	updateUaddress(key)
	return key
}

// updateuaddress adds UAddress field to the Key struct
func updateUaddress(k *Key) {
	k.UAddress = *GenerateUaddressFromPK(&k.PrivateKey.PublicKey, &k.PrivateKey2.PublicKey)
}

// ECDSAPKCompression serializes a public key in a 33-byte compressed format from btcec
func ECDSAPKCompression(p *ecdsa.PublicKey) []byte {
	const pubkeyCompressed byte = 0x2
	b := make([]byte, 0, 33)
	format := pubkeyCompressed
	if p.Y.Bit(0) == 1 {
		format |= 0x1
	}
	b = append(b, format)
	b = append(b, math.PaddedBigBytes(p.X, 32)...)
	return b
}

func GenerateUaddressFromPK(A *ecdsa.PublicKey, B *ecdsa.PublicKey) *common.UAddress {
	var tmp common.UAddress
	copy(tmp[:33], ECDSAPKCompression(A))
	copy(tmp[33:], ECDSAPKCompression(B))
	return &tmp
}

// NewKeyForDirectICAP generates a key whose address fits into < 155 bits so it can fit
// into the Direct ICAP spec. for simplicity and easier compatibility with other libs, we
// retry until the first byte is 0.
func NewKeyForDirectICAP(rand io.Reader) *Key {
	randBytes := make([]byte, 64)
	_, err := rand.Read(randBytes)
	if err != nil {
		panic("key generation: could not read from random source: " + err.Error())
	}
	reader := bytes.NewReader(randBytes)
	sk1, err := ecdsa.GenerateKey(crypto.S256(), reader)
	if err != nil {
		panic("key generation: ecdsa.GenerateKey failed: " + err.Error())
	}
	sk2, err := ecdsa.GenerateKey(crypto.S256(), reader)
	if err != nil {
		panic("key generation: ecdsa.GenerateKey failed: " + err.Error())
	}
	key := newKeyFromECDSA(sk1, sk2)
	if !strings.HasPrefix(key.Address.Hex(), "0x00") {
		return NewKeyForDirectICAP(rand)
	}
	return key
}

func newKey(rand io.Reader) (*Key, error) {
	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), rand)
	if err != nil {
		return nil, err
	}

	privateKeyECDSA2, err := ecdsa.GenerateKey(crypto.S256(), rand)
	if err != nil {
		return nil, err
	}

	return newKeyFromECDSA(privateKeyECDSA, privateKeyECDSA2), nil
}

func storeNewKey(ks keyStore, rand io.Reader, auth string) (*Key, accounts.Account, error) {
	key, err := newKey(rand)
	if err != nil {
		return nil, accounts.Account{}, err
	}
	a := accounts.Account{
		Address: key.Address,
		URL:     accounts.URL{Scheme: KeyStoreScheme, Path: ks.JoinPath(keyFileName(key.Address))},
	}
	if err := ks.StoreKey(a.URL.Path, key, auth); err != nil {
		zeroKey(key.PrivateKey)
		return nil, a, err
	}
	return key, a, err
}

func writeTemporaryKeyFile(file string, content []byte) (string, error) {
	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return "", err
	}
	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return "", err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	f.Close()
	return f.Name(), nil
}

func writeKeyFile(file string, content []byte) error {
	name, err := writeTemporaryKeyFile(file, content)
	if err != nil {
		return err
	}
	return os.Rename(name, file)
}

// keyFileName implements the naming convention for keyfiles:
// UTC--<created_at UTC ISO8601>-<address hex>
func keyFileName(keyAddr common.Address) string {
	ts := time.Now().UTC()
	return fmt.Sprintf("UTC--%s--%s", toISO8601(ts), hex.EncodeToString(keyAddr[:]))
}

// GeneratePKPairFromUAddress represents the keystore to retrieve public key-pair from given UAddress
func GeneratePKPairFromUAddress(w []byte) (*ecdsa.PublicKey, *ecdsa.PublicKey, error) {
	if len(w) != common.UAddressLength {
		return nil, nil, ErrUAddressInvalid
	}

	tmp := make([]byte, 33)
	copy(tmp[:], w[:33])
	curve := btcec.S256()
	PK1, err := btcec.ParsePubKey(tmp, curve)
	if err != nil {
		return nil, nil, err
	}

	copy(tmp[:], w[33:])
	PK2, err := btcec.ParsePubKey(tmp, curve)
	if err != nil {
		return nil, nil, err
	}

	return (*ecdsa.PublicKey)(PK1), (*ecdsa.PublicKey)(PK2), nil
}

func UaddrFromUncompressedRawBytes(raw []byte) (*common.UAddress, error) {
	if len(raw) != 32*2*2 {
		return nil, errors.New("invalid uncompressed use address len")
	}

	pub := make([]byte, 65)
	pub[0] = 0x004
	copy(pub[1:], raw[:64])
	A := crypto.ToECDSAPub(pub)
	copy(pub[1:], raw[64:])
	B := crypto.ToECDSAPub(pub)
	return GenerateUaddressFromPK(A, B), nil
}

func UaddrToUncompressedRawBytes(waddr []byte) ([]byte, error) {
	if len(waddr) != common.UAddressLength {
		return nil, ErrUAddressInvalid
	}

	A, B, err := GeneratePKPairFromUAddress(waddr)
	if err != nil {
		return nil, err
	}

	u := make([]byte, 32*2*2)
	ax := math.PaddedBigBytes(A.X, 32)
	ay := math.PaddedBigBytes(A.Y, 32)
	bx := math.PaddedBigBytes(B.X, 32)
	by := math.PaddedBigBytes(B.Y, 32)
	copy(u[0:], ax[:32])
	copy(u[32:], ay[:32])
	copy(u[64:], bx[:32])
	copy(u[96:], by[:32])

	return u, nil
}

func toISO8601(t time.Time) string {
	var tz string
	name, offset := t.Zone()
	if name == "UTC" {
		tz = "Z"
	} else {
		tz = fmt.Sprintf("%03d00", offset/3600)
	}
	return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d.%09d%s",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tz)
}

// LoadECDSAPair loads a secp256k1 private key pair from the given file
func LoadECDSAPair(file string) (*ecdsa.PrivateKey, *ecdsa.PrivateKey, error) {
	// read the given file including private key pair
	kp := keyPair{}

	raw, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, nil, err
	}

	err = json.Unmarshal(raw, &kp)
	if err != nil {
		return nil, nil, err
	}

	// Decode the key pair
	d, err := hex.DecodeString(kp.D)
	if err != nil {
		return nil, nil, err
	}
	d1, err := hex.DecodeString(kp.D1)
	if err != nil {
		return nil, nil, err
	}

	// Generate ecdsa private keys
	sk, err := crypto.ToECDSA(d)
	if err != nil {
		return nil, nil, err
	}

	sk1, err := crypto.ToECDSA(d1)
	if err != nil {
		return nil, nil, err
	}

	return sk, sk1, err
}

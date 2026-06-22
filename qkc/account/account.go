// Ported from github.com/QuarkChain/goquarkchain/account (byte-compatible).
// Adaptation: github.com/pborman/uuid -> github.com/google/uuid (the only uuid
// package available in this module); identical wire/JSON behaviour.

package account

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"golang.org/x/crypto/pbkdf2"
)

// Account include Identity  address and ID
type Account struct {
	Identity   Identity
	QKCAddress Address
	ID         uuid.UUID
}

// EncryptedKeyJSON keystore file included
type EncryptedKeyJSON struct {
	Address string     `json:"address"`
	Crypto  CryptoJSON `json:"crypto"`
	ID      string     `json:"id"`
	Version int        `json:"version"`
}

// CryptoJSON crypto data for keystore file
type CryptoJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherParamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
	Version      int                    `json:"version"`
}

type cipherParamsJSON struct {
	IV string `json:"iv"`
}

func newAccount(identity Identity, address Address) Account {
	return Account{
		ID:         uuid.New(),
		Identity:   identity,
		QKCAddress: address,
	}
}

// NewAccountWithKey create new account with key
func NewAccountWithKey(key Key) (Account, error) {
	identity, err := CreatIdentityFromKey(key)
	if err != nil {
		return Account{}, err
	}

	defaultFullShardKey, err := identity.GetDefaultFullShardKey()
	if err != nil {
		return Account{}, err
	}

	address := CreatAddressFromIdentity(identity, defaultFullShardKey)
	return newAccount(identity, address), nil
}

// NewAccountWithoutKey new account without key,use random key
func NewAccountWithoutKey() (Account, error) {
	identity, err := CreatRandomIdentity()
	if err != nil {
		return Account{}, err
	}

	defaultFullShardKey, err := identity.GetDefaultFullShardKey()
	if err != nil {
		return Account{}, err
	}

	address := CreatAddressFromIdentity(identity, defaultFullShardKey)
	return newAccount(identity, address), nil
}

// Load load a keystore file with password
func Load(path string, password string) (Account, error) {
	jsonData, err := ioutil.ReadFile(path)
	if err != nil {
		return Account{}, err
	}

	var keystoreJSONData EncryptedKeyJSON
	err = json.Unmarshal(jsonData, &keystoreJSONData)
	key, err := DecodeKeyStoreJSON(keystoreJSONData, password)
	if err != nil {
		return Account{}, err
	}

	keyTypeData := BytesToIdentityKey(key)
	if err != nil {
		return Account{}, err
	}
	account, err := NewAccountWithKey(keyTypeData)
	if err != nil {
		return Account{}, err
	}
	if keystoreJSONData.ID != "" {
		account.ID, _ = uuid.Parse(keystoreJSONData.ID)
	}
	return account, nil
}

// DecodeKeyStoreJSON decode key with password ,return plainText to create account
func DecodeKeyStoreJSON(keystoreJSONData EncryptedKeyJSON, password string) ([]byte, error) {
	kdfParams := keystoreJSONData.Crypto.KDFParams
	c := ensureInt(kdfParams[kdfParamsC])
	salt, err := hex.DecodeString(kdfParams[kdfParamsSalt].(string))
	if err != nil {
		return []byte{}, err
	}

	dkLen := ensureInt(kdfParams[kdfParamsPrfDkLen])
	derivedKey := pbkdf2.Key([]byte(password), salt, c, dkLen, sha256.New)
	if len(derivedKey) < 32 { // derived key must be at least 32 bytes long
		return []byte{}, errors.New("derivedkey<32")
	}

	iv, err := hex.DecodeString(keystoreJSONData.Crypto.CipherParams.IV)
	if err != nil {
		return []byte{}, err
	}

	cipherText, err := hex.DecodeString(keystoreJSONData.Crypto.CipherText)
	if err != nil {
		return []byte{}, err
	}

	mac := crypto.Keccak256(derivedKey[16:32], cipherText)
	macJSON, err := hex.DecodeString(keystoreJSONData.Crypto.MAC)
	if err != nil {
		return []byte{}, errors.New("decode Mac failed")
	}
	if !bytes.Equal(mac, macJSON) {
		return []byte{}, errors.New("mac is not match")
	}

	plainText, err := aesCTRXOR(derivedKey[:16], cipherText, iv)
	if err != nil {
		return []byte{}, err
	}
	return plainText, nil
}

// Dump dump a keystore file with it's password
func (Self *Account) Dump(password string, includeAddress bool, write bool, directory string) ([]byte, error) {
	keystoreJSON, err := Self.MakeKeyStoreJSON(password)
	if err != nil {
		return []byte{}, err
	}
	if includeAddress {
		address := Self.Address()
		keystoreJSON.Address = address
	}

	data, err := json.Marshal(keystoreJSON)
	if err != nil {
		return []byte{}, err
	}
	if write {
		if directory == "" {
			directory = DefaultKeyStoreDirectory
		}

		filepath := directory + Self.ID.String() + ".json"
		err := writeKeyFile(filepath, data)
		if err != nil {
			return []byte{}, err
		}
	}
	return data, nil
}

// MakeKeyStoreJSON make encrypt Json depend on it's password
func (Self *Account) MakeKeyStoreJSON(password string) (EncryptedKeyJSON, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return EncryptedKeyJSON{}, errors.New("get salt failed")
	}

	kdfParams := make(map[string]interface{}, 5)
	kdfParams[kdfParamsPrf] = kdfParamsPrfValue
	kdfParams[kdfParamsPrfDkLen] = kdfParamsPrfDkLenValue
	kdfParams[kdfParamsC] = kdfParamsCValue
	kdfParams[kdfParamsSalt] = hex.EncodeToString(salt)
	derivedKey := pbkdf2.Key([]byte(password), salt, kdfParamsCValue, kdfParamsPrfDkLenValue, sha256.New)
	encKey := derivedKey[:16]
	cipherParams := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, cipherParams); err != nil {
		return EncryptedKeyJSON{}, errors.New("get cipherparams failed")
	}

	cipherText, err := aesCTRXOR(encKey, Self.Identity.key.Bytes(), cipherParams)
	if err != nil {
		return EncryptedKeyJSON{}, errors.New("aes error")
	}

	mac := crypto.Keccak256(derivedKey[16:32], cipherText)
	cryptoData := CryptoJSON{
		Cipher:     cryptoCipher,
		CipherText: hex.EncodeToString(cipherText),
		CipherParams: cipherParamsJSON{
			IV: hex.EncodeToString(cipherParams),
		},
		KDF:       cryptoKDF,
		KDFParams: kdfParams,
		MAC:       hex.EncodeToString(mac),
		Version:   cryptoVersion,
	}
	return EncryptedKeyJSON{
		ID:      Self.ID.String(),
		Crypto:  cryptoData,
		Version: jsonVersion,
	}, nil
}

// Address return it's real address
func (Self *Account) Address() string {
	return Self.QKCAddress.ToHex()
}

// PrivateKey return it's key
func (Self *Account) PrivateKey() string {
	return hex.EncodeToString(Self.Identity.key.Bytes())
}

// UUID return it's uuid
func (Self *Account) UUID() uuid.UUID {
	return Self.ID
}

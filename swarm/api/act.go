package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/sctx"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"golang.org/x/crypto/scrypt"
	"golang.org/x/crypto/sha3"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	ErrDecrypt                = errors.New("cant decrypt - forbidden")
	ErrUnknownAccessType      = errors.New("unknown access type (or not implemented)")
	ErrDecryptDomainForbidden = errors.New("decryption request domain forbidden - can only decrypt on localhost")
	AllowedDecryptDomains     = []string{
		"localhost",
		"127.0.0.1",
	}
)

const EmptyCredentials = ""

type AccessEntry struct {
	Type      AccessType
	Publisher string
	Salt      []byte
	Act       string
	KdfParams *KdfParams
}

type DecryptFunc func(*ManifestEntry) error

func (a *AccessEntry) MarshalJSON() (out []byte, err error) {

	return json.Marshal(struct {
		Type      AccessType `json:"type,omitempty"`
		Publisher string     `json:"publisher,omitempty"`
		Salt      string     `json:"salt,omitempty"`
		Act       string     `json:"act,omitempty"`
		KdfParams *KdfParams `json:"kdf_params,omitempty"`
	}{
		Type:      a.Type,
		Publisher: a.Publisher,
		Salt:      hex.EncodeToString(a.Salt),
		Act:       a.Act,
		KdfParams: a.KdfParams,
	})

}

func (a *AccessEntry) UnmarshalJSON(value []byte) error {
	v := struct {
		Type      AccessType `json:"type,omitempty"`
		Publisher string     `json:"publisher,omitempty"`
		Salt      string     `json:"salt,omitempty"`
		Act       string     `json:"act,omitempty"`
		KdfParams *KdfParams `json:"kdf_params,omitempty"`
	}{}

	err := json.Unmarshal(value, &v)
	if err != nil {
		return err
	}
	a.Act = v.Act
	a.KdfParams = v.KdfParams
	a.Publisher = v.Publisher
	a.Salt, err = hex.DecodeString(v.Salt)
	if err != nil {
		return err
	}
	if len(a.Salt) != 32 {
		return errors.New("salt should be 32 bytes long")
	}
	a.Type = v.Type
	return nil
}

type KdfParams struct {
	N int `json:"n"`
	P int `json:"p"`
	R int `json:"r"`
}

type AccessType string

const AccessTypePass = AccessType("pass")
const AccessTypePK = AccessType("pk")
const AccessTypeACT = AccessType("act")

// NewAccessEntryPassword creates a manifest AccessEntry in order to create an ACT protected by a password
func NewAccessEntryPassword(salt []byte, kdfParams *KdfParams) (*AccessEntry, error) {
	if len(salt) != 32 {
		return nil, fmt.Errorf("salt should be 32 bytes long")
	}
	return &AccessEntry{
		Type:      AccessTypePass,
		Salt:      salt,
		KdfParams: kdfParams,
	}, nil
}

// NewAccessEntryPK creates a manifest AccessEntry in order to create an ACT protected by a pair of Elliptic Curve keys
func NewAccessEntryPK(publisher string, salt []byte) (*AccessEntry, error) {
	if len(publisher) != 66 {
		return nil, fmt.Errorf("publisher should be 66 characters long, got %d", len(publisher))
	}
	if len(salt) != 32 {
		return nil, fmt.Errorf("salt should be 32 bytes long")
	}
	return &AccessEntry{
		Type:      AccessTypePK,
		Publisher: publisher,
		Salt:      salt,
	}, nil
}

// NewAccessEntryACT creates a manifest AccessEntry in order to create an ACT protected by a combination of EC keys and passwords
func NewAccessEntryACT(publisher string, salt []byte, act string) (*AccessEntry, error) {
	if len(salt) != 32 {
		return nil, fmt.Errorf("salt should be 32 bytes long")
	}
	if len(publisher) != 66 {
		return nil, fmt.Errorf("publisher should be 66 characters long")
	}

	return &AccessEntry{
		Type:      AccessTypeACT,
		Publisher: publisher,
		Salt:      salt,
		Act:       act,
		KdfParams: DefaultKdfParams,
	}, nil
}

// NOOPDecrypt is a generic decrypt function that is passed into the API in places where real ACT decryption capabilities are
// either unwanted, or alternatively, cannot be implemented in the immediate scope
func NOOPDecrypt(*ManifestEntry) error {
	return nil
}

var DefaultKdfParams = NewKdfParams(262144, 1, 8)

// NewKdfParams returns a KdfParams struct with the given scrypt params
func NewKdfParams(n, p, r int) *KdfParams {

	return &KdfParams{
		N: n,
		P: p,
		R: r,
	}
}

// NewSessionKeyPassword creates a session key based on a shared secret (password) and the given salt
// and kdf parameters in the access entry
func NewSessionKeyPassword(password string, accessEntry *AccessEntry) ([]byte, error) {
	if accessEntry.Type != AccessTypePass && accessEntry.Type != AccessTypeACT {
		return nil, errors.New("incorrect access entry type")

	}
	return sessionKeyPassword(password, accessEntry.Salt, accessEntry.KdfParams)
}

func sessionKeyPassword(password string, salt []byte, kdfParams *KdfParams) ([]byte, error) {
	return scrypt.Key(
		[]byte(password),
		salt,
		kdfParams.N,
		kdfParams.R,
		kdfParams.P,
		32,
	)
}

// NewSessionKeyPK creates a new ACT Session Key using an ECDH shared secret for the given key pair and the given salt value
func NewSessionKeyPK(private *ecdsa.PrivateKey, public *ecdsa.PublicKey, salt []byte) ([]byte, error) {
	granteePubEcies := ecies.ImportECDSAPublic(public)
	privateKey := ecies.ImportECDSA(private)

	bytes, err := privateKey.GenerateShared(granteePubEcies, 16, 16)
	if err != nil {
		return nil, err
	}
	bytes = append(salt, bytes...)
	sessionKey := crypto.Keccak256(bytes)
	return sessionKey, nil
}

func (a *API) doDecrypt(ctx context.Context, credentials string, pk *ecdsa.PrivateKey) DecryptFunc {
	return func(m *ManifestEntry) error {
		if m.Access == nil {
			return nil
		}

		allowed := false
		requestDomain := sctx.GetHost(ctx)
		for _, v := range AllowedDecryptDomains {
			if strings.Contains(requestDomain, v) {
				allowed = true
			}
		}

		if !allowed {
			return ErrDecryptDomainForbidden
		}

		switch m.Access.Type {
		case "pass":
			if credentials != "" {
				key, err := NewSessionKeyPassword(credentials, m.Access)
				if err != nil {
					return err
				}

				ref, err := hex.DecodeString(m.Hash)
				if err != nil {
					return err
				}

				enc := NewRefEncryption(len(ref) - 8)
				decodedRef, err := enc.Decrypt(ref, key)
				if err != nil {
					return ErrDecrypt
				}

				m.Hash = hex.EncodeToString(decodedRef)
				m.Access = nil
				return nil
			}
			return ErrDecrypt
		case "pk":
			publisherBytes, err := hex.DecodeString(m.Access.Publisher)
			if err != nil {
				return ErrDecrypt
			}
			publisher, err := crypto.DecompressPubkey(publisherBytes)
			if err != nil {
				return ErrDecrypt
			}
			key, err := NewSessionKeyPK(pk, publisher, m.Access.Salt)
			if err != nil {
				return ErrDecrypt
			}
			ref, err := hex.DecodeString(m.Hash)
			if err != nil {
				return err
			}

			enc := NewRefEncryption(len(ref) - 8)
			decodedRef, err := enc.Decrypt(ref, key)
			if err != nil {
				return ErrDecrypt
			}

			m.Hash = hex.EncodeToString(decodedRef)
			m.Access = nil
			return nil
		case "act":
			var (
				sessionKey []byte
				err        error
			)

			publisherBytes, err := hex.DecodeString(m.Access.Publisher)
			if err != nil {
				return ErrDecrypt
			}
			publisher, err := crypto.DecompressPubkey(publisherBytes)
			if err != nil {
				return ErrDecrypt
			}

			sessionKey, err = NewSessionKeyPK(pk, publisher, m.Access.Salt)
			if err != nil {
				return ErrDecrypt
			}

			found, ciphertext, decryptionKey, err := a.getACTDecryptionKey(ctx, storage.Address(common.Hex2Bytes(m.Access.Act)), sessionKey)
			if err != nil {
				return err
			}
			if !found {
				// try to fall back to password
				if credentials != "" {
					sessionKey, err = NewSessionKeyPassword(credentials, m.Access)
					if err != nil {
						return err
					}
					found, ciphertext, decryptionKey, err = a.getACTDecryptionKey(ctx, storage.Address(common.Hex2Bytes(m.Access.Act)), sessionKey)
					if err != nil {
						return err
					}
					if !found {
						return ErrDecrypt
					}
				} else {
					return ErrDecrypt
				}
			}
			enc := NewRefEncryption(len(ciphertext) - 8)
			decodedRef, err := enc.Decrypt(ciphertext, decryptionKey)
			if err != nil {
				return ErrDecrypt
			}

			ref, err := hex.DecodeString(m.Hash)
			if err != nil {
				return err
			}

			enc = NewRefEncryption(len(ref) - 8)
			decodedMainRef, err := enc.Decrypt(ref, decodedRef)
			if err != nil {
				return ErrDecrypt
			}
			m.Hash = hex.EncodeToString(decodedMainRef)
			m.Access = nil
			return nil
		}
		return ErrUnknownAccessType
	}
}

func (a *API) getACTDecryptionKey(ctx context.Context, actManifestAddress storage.Address, sessionKey []byte) (found bool, ciphertext, decryptionKey []byte, err error) {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(append(sessionKey, 0))
	lookupKey := hasher.Sum(nil)
	hasher.Reset()

	hasher.Write(append(sessionKey, 1))
	accessKeyDecryptionKey := hasher.Sum(nil)
	hasher.Reset()

	lk := hex.EncodeToString(lookupKey)
	list, err := a.GetManifestList(ctx, NOOPDecrypt, actManifestAddress, lk)
	if err != nil {
		return false, nil, nil, err
	}
	for _, v := range list.Entries {
		if v.Path == lk {
			cipherTextBytes, err := hex.DecodeString(v.Hash)
			if err != nil {
				return false, nil, nil, err
			}
			return true, cipherTextBytes, accessKeyDecryptionKey, nil
		}
	}
	return false, nil, nil, nil
}

func GenerateAccessControlManifest(ctx *cli.Context, ref string, accessKey []byte, ae *AccessEntry) (*Manifest, error) {
	refBytes, err := hex.DecodeString(ref)
	if err != nil {
		return nil, err
	}
	// encrypt ref with accessKey
	enc := NewRefEncryption(len(refBytes))
	encrypted, err := enc.Encrypt(refBytes, accessKey)
	if err != nil {
		return nil, err
	}

	m := &Manifest{
		Entries: []ManifestEntry{
			{
				Hash:        hex.EncodeToString(encrypted),
				ContentType: ManifestType,
				ModTime:     time.Now(),
				Access:      ae,
			},
		},
	}

	return m, nil
}

// DoPK is a helper function to the CLI API that handles the entire business logic for
// creating a session key and access entry given the cli context, ec keys and salt
func DoPK(ctx *cli.Context, privateKey *ecdsa.PrivateKey, granteePublicKey string, salt []byte) (sessionKey []byte, ae *AccessEntry, err error) {
	if granteePublicKey == "" {
		return nil, nil, errors.New("need a grantee Public Key")
	}
	b, err := hex.DecodeString(granteePublicKey)
	if err != nil {
		log.Error("error decoding grantee public key", "err", err)
		return nil, nil, err
	}

	granteePub, err := crypto.DecompressPubkey(b)
	if err != nil {
		log.Error("error decompressing grantee public key", "err", err)
		return nil, nil, err
	}

	sessionKey, err = NewSessionKeyPK(privateKey, granteePub, salt)
	if err != nil {
		log.Error("error getting session key", "err", err)
		return nil, nil, err
	}

	ae, err = NewAccessEntryPK(hex.EncodeToString(crypto.CompressPubkey(&privateKey.PublicKey)), salt)
	if err != nil {
		log.Error("error generating access entry", "err", err)
		return nil, nil, err
	}

	return sessionKey, ae, nil
}

// DoACT is a helper function to the CLI API that handles the entire business logic for
// creating a access key, access entry and ACT manifest (including uploading it) given the cli context, ec keys, password grantees and salt
func DoACT(ctx *cli.Context, privateKey *ecdsa.PrivateKey, salt []byte, grantees []string, encryptPasswords []string) (accessKey []byte, ae *AccessEntry, actManifest *Manifest, err error) {
	if len(grantees) == 0 && len(encryptPasswords) == 0 {
		return nil, nil, nil, errors.New("did not get any grantee public keys or any encryption passwords")
	}

	publisherPub := hex.EncodeToString(crypto.CompressPubkey(&privateKey.PublicKey))
	grantees = append(grantees, publisherPub)

	accessKey = make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	if _, err := io.ReadFull(rand.Reader, accessKey); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}

	lookupPathEncryptedAccessKeyMap := make(map[string]string)
	i := 0
	for _, v := range grantees {
		i++
		if v == "" {
			return nil, nil, nil, errors.New("need a grantee Public Key")
		}
		b, err := hex.DecodeString(v)
		if err != nil {
			log.Error("error decoding grantee public key", "err", err)
			return nil, nil, nil, err
		}

		granteePub, err := crypto.DecompressPubkey(b)
		if err != nil {
			log.Error("error decompressing grantee public key", "err", err)
			return nil, nil, nil, err
		}
		sessionKey, err := NewSessionKeyPK(privateKey, granteePub, salt)
		if err != nil {
			return nil, nil, nil, err
		}

		hasher := sha3.NewLegacyKeccak256()
		hasher.Write(append(sessionKey, 0))
		lookupKey := hasher.Sum(nil)

		hasher.Reset()
		hasher.Write(append(sessionKey, 1))

		accessKeyEncryptionKey := hasher.Sum(nil)

		enc := NewRefEncryption(len(accessKey))
		encryptedAccessKey, err := enc.Encrypt(accessKey, accessKeyEncryptionKey)
		if err != nil {
			return nil, nil, nil, err
		}
		lookupPathEncryptedAccessKeyMap[hex.EncodeToString(lookupKey)] = hex.EncodeToString(encryptedAccessKey)
	}

	for _, pass := range encryptPasswords {
		sessionKey, err := sessionKeyPassword(pass, salt, DefaultKdfParams)
		if err != nil {
			return nil, nil, nil, err
		}
		hasher := sha3.NewLegacyKeccak256()
		hasher.Write(append(sessionKey, 0))
		lookupKey := hasher.Sum(nil)

		hasher.Reset()
		hasher.Write(append(sessionKey, 1))

		accessKeyEncryptionKey := hasher.Sum(nil)

		enc := NewRefEncryption(len(accessKey))
		encryptedAccessKey, err := enc.Encrypt(accessKey, accessKeyEncryptionKey)
		if err != nil {
			return nil, nil, nil, err
		}
		lookupPathEncryptedAccessKeyMap[hex.EncodeToString(lookupKey)] = hex.EncodeToString(encryptedAccessKey)
	}

	m := &Manifest{
		Entries: []ManifestEntry{},
	}

	for k, v := range lookupPathEncryptedAccessKeyMap {
		m.Entries = append(m.Entries, ManifestEntry{
			Path:        k,
			Hash:        v,
			ContentType: "text/plain",
		})
	}

	ae, err = NewAccessEntryACT(hex.EncodeToString(crypto.CompressPubkey(&privateKey.PublicKey)), salt, "")
	if err != nil {
		return nil, nil, nil, err
	}

	return accessKey, ae, m, nil
}

// DoPassword is a helper function to the CLI API that handles the entire business logic for
// creating a session key and an access entry given the cli context, password and salt.
// By default - DefaultKdfParams are used as the scrypt params
func DoPassword(ctx *cli.Context, password string, salt []byte) (sessionKey []byte, ae *AccessEntry, err error) {
	ae, err = NewAccessEntryPassword(salt, DefaultKdfParams)
	if err != nil {
		return nil, nil, err
	}

	sessionKey, err = NewSessionKeyPassword(password, ae)
	if err != nil {
		return nil, nil, err
	}
	return sessionKey, ae, nil
}

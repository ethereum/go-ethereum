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
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/sctx"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"golang.org/x/crypto/scrypt"
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

const EMPTY_CREDENTIALS = ""

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
	}, nil
}

func NOOPDecrypt(*ManifestEntry) error {
	return nil
}

var DefaultKdfParams = NewKdfParams(262144, 1, 8)

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
	if accessEntry.Type != AccessTypePass {
		return nil, errors.New("incorrect access entry type")
	}
	return scrypt.Key(
		[]byte(password),
		accessEntry.Salt,
		accessEntry.KdfParams.N,
		accessEntry.KdfParams.R,
		accessEntry.KdfParams.P,
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

func (a *API) NodeSessionKey(privateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey, salt []byte) ([]byte, error) {
	return NewSessionKeyPK(privateKey, publicKey, salt)
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
			key, err := a.NodeSessionKey(pk, publisher, m.Access.Salt)
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
			publisherBytes, err := hex.DecodeString(m.Access.Publisher)
			if err != nil {
				return ErrDecrypt
			}
			publisher, err := crypto.DecompressPubkey(publisherBytes)
			if err != nil {
				return ErrDecrypt
			}

			sessionKey, err := a.NodeSessionKey(pk, publisher, m.Access.Salt)
			if err != nil {
				return ErrDecrypt
			}

			hasher := sha3.NewKeccak256()
			hasher.Write(append(sessionKey, 0))
			lookupKey := hasher.Sum(nil)

			hasher.Reset()

			hasher.Write(append(sessionKey, 1))
			accessKeyDecryptionKey := hasher.Sum(nil)

			lk := hex.EncodeToString(lookupKey)
			list, err := a.GetManifestList(ctx, NOOPDecrypt, storage.Address(common.Hex2Bytes(m.Access.Act)), lk)

			found := ""
			for _, v := range list.Entries {
				if v.Path == lk {
					found = v.Hash
				}
			}

			if found == "" {
				return ErrDecrypt
			}

			v, err := hex.DecodeString(found)
			if err != nil {
				return err
			}
			enc := NewRefEncryption(len(v) - 8)
			decodedRef, err := enc.Decrypt(v, accessKeyDecryptionKey)
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

func DoPKNew(ctx *cli.Context, privateKey *ecdsa.PrivateKey, granteePublicKey string, salt []byte) (sessionKey []byte, ae *AccessEntry, err error) {
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

func DoACTNew(ctx *cli.Context, privateKey *ecdsa.PrivateKey, salt []byte, grantees []string) (accessKey []byte, ae *AccessEntry, actManifest *Manifest, err error) {
	if len(grantees) == 0 {
		return nil, nil, nil, errors.New("did not get any grantee public keys")
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

		hasher := sha3.NewKeccak256()
		hasher.Write(append(sessionKey, 0))
		lookupKey := hasher.Sum(nil)

		hasher.Reset()
		hasher.Write(append(sessionKey, 1))

		accessKeyEncryptionKey := hasher.Sum(nil)

		enc := NewRefEncryption(len(accessKey))
		encryptedAccessKey, err := enc.Encrypt(accessKey, accessKeyEncryptionKey)

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

func DoPasswordNew(ctx *cli.Context, password string, salt []byte) (sessionKey []byte, ae *AccessEntry, err error) {
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

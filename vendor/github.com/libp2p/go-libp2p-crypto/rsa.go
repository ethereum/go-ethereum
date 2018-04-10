package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"errors"

	pb "github.com/libp2p/go-libp2p-crypto/pb"

	proto "github.com/gogo/protobuf/proto"
	sha256 "github.com/minio/sha256-simd"
)

type RsaPrivateKey struct {
	sk *rsa.PrivateKey
	pk *rsa.PublicKey
}

type RsaPublicKey struct {
	k *rsa.PublicKey
}

func (pk *RsaPublicKey) Verify(data, sig []byte) (bool, error) {
	hashed := sha256.Sum256(data)
	err := rsa.VerifyPKCS1v15(pk.k, crypto.SHA256, hashed[:], sig)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (pk *RsaPublicKey) Bytes() ([]byte, error) {
	b, err := x509.MarshalPKIXPublicKey(pk.k)
	if err != nil {
		return nil, err
	}

	pbmes := new(pb.PublicKey)
	typ := pb.KeyType_RSA
	pbmes.Type = &typ
	pbmes.Data = b
	return proto.Marshal(pbmes)
}

func (pk *RsaPublicKey) Encrypt(b []byte) ([]byte, error) {
	return rsa.EncryptPKCS1v15(rand.Reader, pk.k, b)
}

// Equals checks whether this key is equal to another
func (pk *RsaPublicKey) Equals(k Key) bool {
	return KeyEqual(pk, k)
}

func (sk *RsaPrivateKey) Sign(message []byte) ([]byte, error) {
	hashed := sha256.Sum256(message)
	return rsa.SignPKCS1v15(rand.Reader, sk.sk, crypto.SHA256, hashed[:])
}

func (sk *RsaPrivateKey) GetPublic() PubKey {
	if sk.pk == nil {
		sk.pk = &sk.sk.PublicKey
	}
	return &RsaPublicKey{sk.pk}
}

func (sk *RsaPrivateKey) Decrypt(b []byte) ([]byte, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, sk.sk, b)
}

func (sk *RsaPrivateKey) Bytes() ([]byte, error) {
	b := x509.MarshalPKCS1PrivateKey(sk.sk)
	pbmes := new(pb.PrivateKey)
	typ := pb.KeyType_RSA
	pbmes.Type = &typ
	pbmes.Data = b
	return proto.Marshal(pbmes)
}

// Equals checks whether this key is equal to another
func (sk *RsaPrivateKey) Equals(k Key) bool {
	return KeyEqual(sk, k)
}

func UnmarshalRsaPrivateKey(b []byte) (PrivKey, error) {
	sk, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		return nil, err
	}
	return &RsaPrivateKey{sk: sk}, nil
}

func MarshalRsaPrivateKey(k *RsaPrivateKey) []byte {
	return x509.MarshalPKCS1PrivateKey(k.sk)
}

func UnmarshalRsaPublicKey(b []byte) (PubKey, error) {
	pub, err := x509.ParsePKIXPublicKey(b)
	if err != nil {
		return nil, err
	}
	pk, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("Not actually an rsa public key.")
	}
	return &RsaPublicKey{pk}, nil
}

func MarshalRsaPublicKey(k *RsaPublicKey) ([]byte, error) {
	return x509.MarshalPKIXPublicKey(k.k)
}

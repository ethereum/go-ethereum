package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"

	"code.google.com/p/go.crypto/ripemd160"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/obscuren/ecies"
	"github.com/obscuren/secp256k1-go"
	"github.com/obscuren/sha3"
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
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = S256()
	priv.D = ethutil.BigD(prv)
	priv.PublicKey.X, priv.PublicKey.Y = S256().ScalarBaseMult(prv)
	return priv
}

func FromECDSA(prv *ecdsa.PrivateKey) []byte {
	return prv.D.Bytes()
}

func PubToECDSA(pub []byte) *ecdsa.PublicKey {
	x, y := elliptic.Unmarshal(S256(), pub)
	return &ecdsa.PublicKey{S256(), x, y}
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
	sig, err = secp256k1.Sign(hash, prv.D.Bytes())
	return
}

func Encrypt(pub *ecdsa.PublicKey, message []byte) ([]byte, error) {
	return ecies.Encrypt(rand.Reader, ecies.ImportECDSAPublic(pub), message, nil, nil)
}

func Decrypt(prv *ecdsa.PrivateKey, ct []byte) ([]byte, error) {
	key := ecies.ImportECDSA(prv)
	return key.Decrypt(rand.Reader, ct, nil, nil)
}

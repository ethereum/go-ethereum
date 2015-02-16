package ecies

import (
	"bytes"
	"crypto"
	"crypto/elliptic"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"hash"
	"math/big"
)

var (
	secgScheme     = []int{1, 3, 132, 1}
	shaScheme      = []int{2, 16, 840, 1, 101, 3, 4, 2}
	ansiX962Scheme = []int{1, 2, 840, 10045}
	x963Scheme     = []int{1, 2, 840, 63, 0}
)

var ErrInvalidPrivateKey = fmt.Errorf("ecies: invalid private key")

func doScheme(base, v []int) asn1.ObjectIdentifier {
	var oidInts asn1.ObjectIdentifier
	oidInts = append(oidInts, base...)
	return append(oidInts, v...)
}

// curve OID code taken from crypto/x509, including
//	- oidNameCurve*
//	- namedCurveFromOID
//	- oidFromNamedCurve
// RFC 5480, 2.1.1.1. Named Curve
//
// secp224r1 OBJECT IDENTIFIER ::= {
//   iso(1) identified-organization(3) certicom(132) curve(0) 33 }
//
// secp256r1 OBJECT IDENTIFIER ::= {
//   iso(1) member-body(2) us(840) ansi-X9-62(10045) curves(3)
//   prime(1) 7 }
//
// secp384r1 OBJECT IDENTIFIER ::= {
//   iso(1) identified-organization(3) certicom(132) curve(0) 34 }
//
// secp521r1 OBJECT IDENTIFIER ::= {
//   iso(1) identified-organization(3) certicom(132) curve(0) 35 }
//
// NB: secp256r1 is equivalent to prime256v1
type secgNamedCurve asn1.ObjectIdentifier

var (
	secgNamedCurveP224 = secgNamedCurve{1, 3, 132, 0, 33}
	secgNamedCurveP256 = secgNamedCurve{1, 2, 840, 10045, 3, 1, 7}
	secgNamedCurveP384 = secgNamedCurve{1, 3, 132, 0, 34}
	secgNamedCurveP521 = secgNamedCurve{1, 3, 132, 0, 35}
	rawCurveP224       = []byte{6, 5, 4, 3, 1, 2, 9, 4, 0, 3, 3}
	rawCurveP256       = []byte{6, 8, 4, 2, 1, 3, 4, 7, 2, 2, 0, 6, 6, 1, 3, 1, 7}
	rawCurveP384       = []byte{6, 5, 4, 3, 1, 2, 9, 4, 0, 3, 4}
	rawCurveP521       = []byte{6, 5, 4, 3, 1, 2, 9, 4, 0, 3, 5}
)

func rawCurve(curve elliptic.Curve) []byte {
	switch curve {
	case elliptic.P224():
		return rawCurveP224
	case elliptic.P256():
		return rawCurveP256
	case elliptic.P384():
		return rawCurveP384
	case elliptic.P521():
		return rawCurveP521
	default:
		return nil
	}
}

func (curve secgNamedCurve) Equal(curve2 secgNamedCurve) bool {
	if len(curve) != len(curve2) {
		return false
	}
	for i, _ := range curve {
		if curve[i] != curve2[i] {
			return false
		}
	}
	return true
}

func namedCurveFromOID(curve secgNamedCurve) elliptic.Curve {
	switch {
	case curve.Equal(secgNamedCurveP224):
		return elliptic.P224()
	case curve.Equal(secgNamedCurveP256):
		return elliptic.P256()
	case curve.Equal(secgNamedCurveP384):
		return elliptic.P384()
	case curve.Equal(secgNamedCurveP521):
		return elliptic.P521()
	}
	return nil
}

func oidFromNamedCurve(curve elliptic.Curve) (secgNamedCurve, bool) {
	switch curve {
	case elliptic.P224():
		return secgNamedCurveP224, true
	case elliptic.P256():
		return secgNamedCurveP256, true
	case elliptic.P384():
		return secgNamedCurveP384, true
	case elliptic.P521():
		return secgNamedCurveP521, true
	}

	return nil, false
}

// asnAlgorithmIdentifier represents the ASN.1 structure of the same name. See RFC
// 5280, section 4.1.1.2.
type asnAlgorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"optional"`
}

func (a asnAlgorithmIdentifier) Cmp(b asnAlgorithmIdentifier) bool {
	if len(a.Algorithm) != len(b.Algorithm) {
		return false
	}
	for i, _ := range a.Algorithm {
		if a.Algorithm[i] != b.Algorithm[i] {
			return false
		}
	}
	return true
}

type asnHashFunction asnAlgorithmIdentifier

var (
	oidSHA1   = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 26}
	oidSHA224 = doScheme(shaScheme, []int{4})
	oidSHA256 = doScheme(shaScheme, []int{1})
	oidSHA384 = doScheme(shaScheme, []int{2})
	oidSHA512 = doScheme(shaScheme, []int{3})
)

func hashFromOID(oid asn1.ObjectIdentifier) func() hash.Hash {
	switch {
	case oid.Equal(oidSHA1):
		return sha1.New
	case oid.Equal(oidSHA224):
		return sha256.New224
	case oid.Equal(oidSHA256):
		return sha256.New
	case oid.Equal(oidSHA384):
		return sha512.New384
	case oid.Equal(oidSHA512):
		return sha512.New
	}
	return nil
}

func oidFromHash(hash crypto.Hash) (asn1.ObjectIdentifier, bool) {
	switch hash {
	case crypto.SHA1:
		return oidSHA1, true
	case crypto.SHA224:
		return oidSHA224, true
	case crypto.SHA256:
		return oidSHA256, true
	case crypto.SHA384:
		return oidSHA384, true
	case crypto.SHA512:
		return oidSHA512, true
	default:
		return nil, false
	}
}

var (
	asnAlgoSHA1 = asnHashFunction{
		Algorithm: oidSHA1,
	}
	asnAlgoSHA224 = asnHashFunction{
		Algorithm: oidSHA224,
	}
	asnAlgoSHA256 = asnHashFunction{
		Algorithm: oidSHA256,
	}
	asnAlgoSHA384 = asnHashFunction{
		Algorithm: oidSHA384,
	}
	asnAlgoSHA512 = asnHashFunction{
		Algorithm: oidSHA512,
	}
)

// type ASNasnSubjectPublicKeyInfo struct {
//
// }
//

type asnSubjectPublicKeyInfo struct {
	Algorithm   asn1.ObjectIdentifier
	PublicKey   asn1.BitString
	Supplements ecpksSupplements `asn1:"optional"`
}

type asnECPKAlgorithms struct {
	Type asn1.ObjectIdentifier
}

var idPublicKeyType = doScheme(ansiX962Scheme, []int{2})
var idEcPublicKey = doScheme(idPublicKeyType, []int{1})
var idEcPublicKeySupplemented = doScheme(idPublicKeyType, []int{0})

func curveToRaw(curve elliptic.Curve) (rv asn1.RawValue, ok bool) {
	switch curve {
	case elliptic.P224(), elliptic.P256(), elliptic.P384(), elliptic.P521():
		raw := rawCurve(curve)
		return asn1.RawValue{
			Tag:       30,
			Bytes:     raw[2:],
			FullBytes: raw,
		}, true
	default:
		return rv, false
	}
}

func asnECPublicKeyType(curve elliptic.Curve) (algo asnAlgorithmIdentifier, ok bool) {
	raw, ok := curveToRaw(curve)
	if !ok {
		return
	} else {
		return asnAlgorithmIdentifier{Algorithm: idEcPublicKey,
			Parameters: raw}, true
	}
}

type asnECPrivKeyVer int

var asnECPrivKeyVer1 asnECPrivKeyVer = 1

type asnPrivateKey struct {
	Version asnECPrivKeyVer
	Private []byte
	Curve   secgNamedCurve `asn1:"optional"`
	Public  asn1.BitString
}

var asnECDH = doScheme(secgScheme, []int{12})

type asnECDHAlgorithm asnAlgorithmIdentifier

var (
	dhSinglePass_stdDH_sha1kdf = asnECDHAlgorithm{
		Algorithm: doScheme(x963Scheme, []int{2}),
	}
	dhSinglePass_stdDH_sha256kdf = asnECDHAlgorithm{
		Algorithm: doScheme(secgScheme, []int{11, 1}),
	}
	dhSinglePass_stdDH_sha384kdf = asnECDHAlgorithm{
		Algorithm: doScheme(secgScheme, []int{11, 2}),
	}
	dhSinglePass_stdDH_sha224kdf = asnECDHAlgorithm{
		Algorithm: doScheme(secgScheme, []int{11, 0}),
	}
	dhSinglePass_stdDH_sha512kdf = asnECDHAlgorithm{
		Algorithm: doScheme(secgScheme, []int{11, 3}),
	}
)

func (a asnECDHAlgorithm) Cmp(b asnECDHAlgorithm) bool {
	if len(a.Algorithm) != len(b.Algorithm) {
		return false
	}
	for i, _ := range a.Algorithm {
		if a.Algorithm[i] != b.Algorithm[i] {
			return false
		}
	}
	return true
}

// asnNISTConcatenation is the only supported KDF at this time.
type asnKeyDerivationFunction asnAlgorithmIdentifier

var asnNISTConcatenationKDF = asnKeyDerivationFunction{
	Algorithm: doScheme(secgScheme, []int{17, 1}),
}

func (a asnKeyDerivationFunction) Cmp(b asnKeyDerivationFunction) bool {
	if len(a.Algorithm) != len(b.Algorithm) {
		return false
	}
	for i, _ := range a.Algorithm {
		if a.Algorithm[i] != b.Algorithm[i] {
			return false
		}
	}
	return true
}

var eciesRecommendedParameters = doScheme(secgScheme, []int{7})
var eciesSpecifiedParameters = doScheme(secgScheme, []int{8})

type asnECIESParameters struct {
	KDF asnKeyDerivationFunction     `asn1:"optional"`
	Sym asnSymmetricEncryption       `asn1:"optional"`
	MAC asnMessageAuthenticationCode `asn1:"optional"`
}

type asnSymmetricEncryption asnAlgorithmIdentifier

var (
	aes128CTRinECIES = asnSymmetricEncryption{
		Algorithm: doScheme(secgScheme, []int{21, 0}),
	}
	aes192CTRinECIES = asnSymmetricEncryption{
		Algorithm: doScheme(secgScheme, []int{21, 1}),
	}
	aes256CTRinECIES = asnSymmetricEncryption{
		Algorithm: doScheme(secgScheme, []int{21, 2}),
	}
)

func (a asnSymmetricEncryption) Cmp(b asnSymmetricEncryption) bool {
	if len(a.Algorithm) != len(b.Algorithm) {
		return false
	}
	for i, _ := range a.Algorithm {
		if a.Algorithm[i] != b.Algorithm[i] {
			return false
		}
	}
	return true
}

type asnMessageAuthenticationCode asnAlgorithmIdentifier

var (
	hmacFull = asnMessageAuthenticationCode{
		Algorithm: doScheme(secgScheme, []int{22}),
	}
)

func (a asnMessageAuthenticationCode) Cmp(b asnMessageAuthenticationCode) bool {
	if len(a.Algorithm) != len(b.Algorithm) {
		return false
	}
	for i, _ := range a.Algorithm {
		if a.Algorithm[i] != b.Algorithm[i] {
			return false
		}
	}
	return true
}

type ecpksSupplements struct {
	ECDomain      secgNamedCurve
	ECCAlgorithms eccAlgorithmSet
}

type eccAlgorithmSet struct {
	ECDH  asnECDHAlgorithm   `asn1:"optional"`
	ECIES asnECIESParameters `asn1:"optional"`
}

func marshalSubjectPublicKeyInfo(pub *PublicKey) (subj asnSubjectPublicKeyInfo, err error) {
	subj.Algorithm = idEcPublicKeySupplemented
	curve, ok := oidFromNamedCurve(pub.Curve)
	if !ok {
		err = ErrInvalidPublicKey
		return
	}
	subj.Supplements.ECDomain = curve
	if pub.Params != nil {
		subj.Supplements.ECCAlgorithms.ECDH = paramsToASNECDH(pub.Params)
		subj.Supplements.ECCAlgorithms.ECIES = paramsToASNECIES(pub.Params)
	}
	pubkey := elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	subj.PublicKey = asn1.BitString{
		BitLength: len(pubkey) * 8,
		Bytes:     pubkey,
	}
	return
}

// Encode a public key to DER format.
func MarshalPublic(pub *PublicKey) ([]byte, error) {
	subj, err := marshalSubjectPublicKeyInfo(pub)
	if err != nil {
		return nil, err
	}
	return asn1.Marshal(subj)
}

// Decode a DER-encoded public key.
func UnmarshalPublic(in []byte) (pub *PublicKey, err error) {
	var subj asnSubjectPublicKeyInfo

	if _, err = asn1.Unmarshal(in, &subj); err != nil {
		return
	}
	if !subj.Algorithm.Equal(idEcPublicKeySupplemented) {
		err = ErrInvalidPublicKey
		return
	}
	pub = new(PublicKey)
	pub.Curve = namedCurveFromOID(subj.Supplements.ECDomain)
	x, y := elliptic.Unmarshal(pub.Curve, subj.PublicKey.Bytes)
	if x == nil {
		err = ErrInvalidPublicKey
		return
	}
	pub.X = x
	pub.Y = y
	pub.Params = new(ECIESParams)
	asnECIEStoParams(subj.Supplements.ECCAlgorithms.ECIES, pub.Params)
	asnECDHtoParams(subj.Supplements.ECCAlgorithms.ECDH, pub.Params)
	if pub.Params == nil {
		if pub.Params = ParamsFromCurve(pub.Curve); pub.Params == nil {
			err = ErrInvalidPublicKey
		}
	}
	return
}

func marshalPrivateKey(prv *PrivateKey) (ecprv asnPrivateKey, err error) {
	ecprv.Version = asnECPrivKeyVer1
	ecprv.Private = prv.D.Bytes()

	var ok bool
	ecprv.Curve, ok = oidFromNamedCurve(prv.PublicKey.Curve)
	if !ok {
		err = ErrInvalidPrivateKey
		return
	}

	var pub []byte
	if pub, err = MarshalPublic(&prv.PublicKey); err != nil {
		return
	} else {
		ecprv.Public = asn1.BitString{
			BitLength: len(pub) * 8,
			Bytes:     pub,
		}
	}
	return
}

// Encode a private key to DER format.
func MarshalPrivate(prv *PrivateKey) ([]byte, error) {
	ecprv, err := marshalPrivateKey(prv)
	if err != nil {
		return nil, err
	}
	return asn1.Marshal(ecprv)
}

// Decode a private key from a DER-encoded format.
func UnmarshalPrivate(in []byte) (prv *PrivateKey, err error) {
	var ecprv asnPrivateKey

	if _, err = asn1.Unmarshal(in, &ecprv); err != nil {
		return
	} else if ecprv.Version != asnECPrivKeyVer1 {
		err = ErrInvalidPrivateKey
		return
	}

	privateCurve := namedCurveFromOID(ecprv.Curve)
	if privateCurve == nil {
		err = ErrInvalidPrivateKey
		return
	}

	prv = new(PrivateKey)
	prv.D = new(big.Int).SetBytes(ecprv.Private)

	if pub, err := UnmarshalPublic(ecprv.Public.Bytes); err != nil {
		return nil, err
	} else {
		prv.PublicKey = *pub
	}

	return
}

// Export a public key to PEM format.
func ExportPublicPEM(pub *PublicKey) (out []byte, err error) {
	der, err := MarshalPublic(pub)
	if err != nil {
		return
	}

	var block pem.Block
	block.Type = "ELLIPTIC CURVE PUBLIC KEY"
	block.Bytes = der

	buf := new(bytes.Buffer)
	err = pem.Encode(buf, &block)
	if err != nil {
		return
	} else {
		out = buf.Bytes()
	}
	return
}

// Export a private key to PEM format.
func ExportPrivatePEM(prv *PrivateKey) (out []byte, err error) {
	der, err := MarshalPrivate(prv)
	if err != nil {
		return
	}

	var block pem.Block
	block.Type = "ELLIPTIC CURVE PRIVATE KEY"
	block.Bytes = der

	buf := new(bytes.Buffer)
	err = pem.Encode(buf, &block)
	if err != nil {
		return
	} else {
		out = buf.Bytes()
	}
	return
}

// Import a PEM-encoded public key.
func ImportPublicPEM(in []byte) (pub *PublicKey, err error) {
	p, _ := pem.Decode(in)
	if p == nil || p.Type != "ELLIPTIC CURVE PUBLIC KEY" {
		return nil, ErrInvalidPublicKey
	}

	pub, err = UnmarshalPublic(p.Bytes)
	return
}

// Import a PEM-encoded private key.
func ImportPrivatePEM(in []byte) (prv *PrivateKey, err error) {
	p, _ := pem.Decode(in)
	if p == nil || p.Type != "ELLIPTIC CURVE PRIVATE KEY" {
		return nil, ErrInvalidPrivateKey
	}

	prv, err = UnmarshalPrivate(p.Bytes)
	return
}

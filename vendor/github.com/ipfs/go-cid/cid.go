// Package cid implements the Content-IDentifiers specification
// (https://github.com/ipld/cid) in Go. CIDs are
// self-describing content-addressed identifiers useful for
// distributed information systems. CIDs are used in the IPFS
// (https://ipfs.io) project ecosystem.
//
// CIDs have two major versions. A CIDv0 corresponds to a multihash of type
// DagProtobuf, is deprecated and exists for compatibility reasons. Usually,
// CIDv1 should be used.
//
// A CIDv1 has four parts:
//
//     <cidv1> ::= <multibase-prefix><cid-version><multicodec-packed-content-type><multihash-content-address>
//
// As shown above, the CID implementation relies heavily on Multiformats,
// particularly Multibase
// (https://github.com/multiformats/go-multibase), Multicodec
// (https://github.com/multiformats/multicodec) and Multihash
// implementations (https://github.com/multiformats/go-multihash).
package cid

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	mbase "github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"
)

// UnsupportedVersionString just holds an error message
const UnsupportedVersionString = "<unsupported cid version>"

var (
	// ErrVarintBuffSmall means that a buffer passed to the cid parser was not
	// long enough, or did not contain an invalid cid
	ErrVarintBuffSmall = errors.New("reading varint: buffer too small")

	// ErrVarintTooBig means that the varint in the given cid was above the
	// limit of 2^64
	ErrVarintTooBig = errors.New("reading varint: varint bigger than 64bits" +
		" and not supported")

	// ErrCidTooShort means that the cid passed to decode was not long
	// enough to be a valid Cid
	ErrCidTooShort = errors.New("cid too short")

	// ErrInvalidEncoding means that selected encoding is not supported
	// by this Cid version
	ErrInvalidEncoding = errors.New("invalid base encoding")
)

// These are multicodec-packed content types. The should match
// the codes described in the authoritative document:
// https://github.com/multiformats/multicodec/blob/master/table.csv
const (
	Raw = 0x55

	DagProtobuf = 0x70
	DagCBOR     = 0x71

	GitRaw = 0x78

	EthBlock           = 0x90
	EthBlockList       = 0x91
	EthTxTrie          = 0x92
	EthTx              = 0x93
	EthTxReceiptTrie   = 0x94
	EthTxReceipt       = 0x95
	EthStateTrie       = 0x96
	EthAccountSnapshot = 0x97
	EthStorageTrie     = 0x98
	BitcoinBlock       = 0xb0
	BitcoinTx          = 0xb1
	ZcashBlock         = 0xc0
	ZcashTx            = 0xc1
	DecredBlock        = 0xe0
	DecredTx           = 0xe1
	DashBlock          = 0xf0
	DashTx             = 0xf1
)

// Codecs maps the name of a codec to its type
var Codecs = map[string]uint64{
	"v0":                   DagProtobuf,
	"raw":                  Raw,
	"protobuf":             DagProtobuf,
	"cbor":                 DagCBOR,
	"git-raw":              GitRaw,
	"eth-block":            EthBlock,
	"eth-block-list":       EthBlockList,
	"eth-tx-trie":          EthTxTrie,
	"eth-tx":               EthTx,
	"eth-tx-receipt-trie":  EthTxReceiptTrie,
	"eth-tx-receipt":       EthTxReceipt,
	"eth-state-trie":       EthStateTrie,
	"eth-account-snapshot": EthAccountSnapshot,
	"eth-storage-trie":     EthStorageTrie,
	"bitcoin-block":        BitcoinBlock,
	"bitcoin-tx":           BitcoinTx,
	"zcash-block":          ZcashBlock,
	"zcash-tx":             ZcashTx,
	"decred-block":         DecredBlock,
	"decred-tx":            DecredTx,
	"dash-block":           DashBlock,
	"dash-tx":              DashTx,
}

// CodecToStr maps the numeric codec to its name
var CodecToStr = map[uint64]string{
	Raw:                "raw",
	DagProtobuf:        "protobuf",
	DagCBOR:            "cbor",
	GitRaw:             "git-raw",
	EthBlock:           "eth-block",
	EthBlockList:       "eth-block-list",
	EthTxTrie:          "eth-tx-trie",
	EthTx:              "eth-tx",
	EthTxReceiptTrie:   "eth-tx-receipt-trie",
	EthTxReceipt:       "eth-tx-receipt",
	EthStateTrie:       "eth-state-trie",
	EthAccountSnapshot: "eth-account-snapshot",
	EthStorageTrie:     "eth-storage-trie",
	BitcoinBlock:       "bitcoin-block",
	BitcoinTx:          "bitcoin-tx",
	ZcashBlock:         "zcash-block",
	ZcashTx:            "zcash-tx",
	DecredBlock:        "decred-block",
	DecredTx:           "decred-tx",
	DashBlock:          "dash-block",
	DashTx:             "dash-tx",
}

// NewCidV0 returns a Cid-wrapped multihash.
// They exist to allow IPFS to work with Cids while keeping
// compatibility with the plain-multihash format used used in IPFS.
// NewCidV1 should be used preferentially.
func NewCidV0(mhash mh.Multihash) Cid {
	// Need to make sure hash is valid for CidV0 otherwise we will
	// incorrectly detect it as CidV1 in the Version() method
	dec, err := mh.Decode(mhash)
	if err != nil {
		panic(err)
	}
	if dec.Code != mh.SHA2_256 || dec.Length != 32 {
		panic("invalid hash for cidv0")
	}
	return Cid{string(mhash)}
}

// NewCidV1 returns a new Cid using the given multicodec-packed
// content type.
func NewCidV1(codecType uint64, mhash mh.Multihash) Cid {
	hashlen := len(mhash)
	// two 8 bytes (max) numbers plus hash
	buf := make([]byte, 2*binary.MaxVarintLen64+hashlen)
	n := binary.PutUvarint(buf, 1)
	n += binary.PutUvarint(buf[n:], codecType)
	cn := copy(buf[n:], mhash)
	if cn != hashlen {
		panic("copy hash length is inconsistent")
	}

	return Cid{string(buf[:n+hashlen])}
}

var _ encoding.BinaryMarshaler = Cid{}
var _ encoding.BinaryUnmarshaler = (*Cid)(nil)
var _ encoding.TextMarshaler = Cid{}
var _ encoding.TextUnmarshaler = (*Cid)(nil)

// Cid represents a self-describing content addressed
// identifier. It is formed by a Version, a Codec (which indicates
// a multicodec-packed content type) and a Multihash.
type Cid struct{ str string }

// Undef can be used to represent a nil or undefined Cid, using Cid{}
// directly is also acceptable.
var Undef = Cid{}

// Defined returns true if a Cid is defined
// Calling any other methods on an undefined Cid will result in
// undefined behavior.
func (c Cid) Defined() bool {
	return c.str != ""
}

// Parse is a short-hand function to perform Decode, Cast etc... on
// a generic interface{} type.
func Parse(v interface{}) (Cid, error) {
	switch v2 := v.(type) {
	case string:
		if strings.Contains(v2, "/ipfs/") {
			return Decode(strings.Split(v2, "/ipfs/")[1])
		}
		return Decode(v2)
	case []byte:
		return Cast(v2)
	case mh.Multihash:
		return NewCidV0(v2), nil
	case Cid:
		return v2, nil
	default:
		return Undef, fmt.Errorf("can't parse %+v as Cid", v2)
	}
}

// Decode parses a Cid-encoded string and returns a Cid object.
// For CidV1, a Cid-encoded string is primarily a multibase string:
//
//     <multibase-type-code><base-encoded-string>
//
// The base-encoded string represents a:
//
// <version><codec-type><multihash>
//
// Decode will also detect and parse CidV0 strings. Strings
// starting with "Qm" are considered CidV0 and treated directly
// as B58-encoded multihashes.
func Decode(v string) (Cid, error) {
	if len(v) < 2 {
		return Undef, ErrCidTooShort
	}

	if len(v) == 46 && v[:2] == "Qm" {
		hash, err := mh.FromB58String(v)
		if err != nil {
			return Undef, err
		}

		return NewCidV0(hash), nil
	}

	_, data, err := mbase.Decode(v)
	if err != nil {
		return Undef, err
	}

	return Cast(data)
}

// Extract the encoding from a Cid.  If Decode on the same string did
// not return an error neither will this function.
func ExtractEncoding(v string) (mbase.Encoding, error) {
	if len(v) < 2 {
		return -1, ErrCidTooShort
	}

	if len(v) == 46 && v[:2] == "Qm" {
		return mbase.Base58BTC, nil
	}

	encoding := mbase.Encoding(v[0])

	// check encoding is valid
	_, err := mbase.NewEncoder(encoding)
	if err != nil {
		return -1, err
	}

	return encoding, nil
}

func uvError(read int) error {
	switch {
	case read == 0:
		return ErrVarintBuffSmall
	case read < 0:
		return ErrVarintTooBig
	default:
		return nil
	}
}

// Cast takes a Cid data slice, parses it and returns a Cid.
// For CidV1, the data buffer is in the form:
//
//     <version><codec-type><multihash>
//
// CidV0 are also supported. In particular, data buffers starting
// with length 34 bytes, which starts with bytes [18,32...] are considered
// binary multihashes.
//
// Please use decode when parsing a regular Cid string, as Cast does not
// expect multibase-encoded data. Cast accepts the output of Cid.Bytes().
func Cast(data []byte) (Cid, error) {
	if len(data) == 34 && data[0] == 18 && data[1] == 32 {
		h, err := mh.Cast(data)
		if err != nil {
			return Undef, err
		}

		return NewCidV0(h), nil
	}

	vers, n := binary.Uvarint(data)
	if err := uvError(n); err != nil {
		return Undef, err
	}

	if vers != 1 {
		return Undef, fmt.Errorf("expected 1 as the cid version number, got: %d", vers)
	}

	_, cn := binary.Uvarint(data[n:])
	if err := uvError(cn); err != nil {
		return Undef, err
	}

	rest := data[n+cn:]
	h, err := mh.Cast(rest)
	if err != nil {
		return Undef, err
	}

	return Cid{string(data[0 : n+cn+len(h)])}, nil
}

// UnmarshalBinary is equivalent to Cast(). It implements the
// encoding.BinaryUnmarshaler interface.
func (c *Cid) UnmarshalBinary(data []byte) error {
	casted, err := Cast(data)
	if err != nil {
		return err
	}
	c.str = casted.str
	return nil
}

// UnmarshalText is equivalent to Decode(). It implements the
// encoding.TextUnmarshaler interface.
func (c *Cid) UnmarshalText(text []byte) error {
	decodedCid, err := Decode(string(text))
	if err != nil {
		return err
	}
	c.str = decodedCid.str
	return nil
}

// Version returns the Cid version.
func (c Cid) Version() uint64 {
	if len(c.str) == 34 && c.str[0] == 18 && c.str[1] == 32 {
		return 0
	}
	return 1
}

// Type returns the multicodec-packed content type of a Cid.
func (c Cid) Type() uint64 {
	if c.Version() == 0 {
		return DagProtobuf
	}
	_, n := uvarint(c.str)
	codec, _ := uvarint(c.str[n:])
	return codec
}

// String returns the default string representation of a
// Cid. Currently, Base58 is used as the encoding for the
// multibase string.
func (c Cid) String() string {
	switch c.Version() {
	case 0:
		return c.Hash().B58String()
	case 1:
		mbstr, err := mbase.Encode(mbase.Base58BTC, c.Bytes())
		if err != nil {
			panic("should not error with hardcoded mbase: " + err.Error())
		}

		return mbstr
	default:
		panic("not possible to reach this point")
	}
}

// String returns the string representation of a Cid
// encoded is selected base
func (c Cid) StringOfBase(base mbase.Encoding) (string, error) {
	switch c.Version() {
	case 0:
		if base != mbase.Base58BTC {
			return "", ErrInvalidEncoding
		}
		return c.Hash().B58String(), nil
	case 1:
		return mbase.Encode(base, c.Bytes())
	default:
		panic("not possible to reach this point")
	}
}

// Encode return the string representation of a Cid in a given base
// when applicable.  Version 0 Cid's are always in Base58 as they do
// not take a multibase prefix.
func (c Cid) Encode(base mbase.Encoder) string {
	switch c.Version() {
	case 0:
		return c.Hash().B58String()
	case 1:
		return base.Encode(c.Bytes())
	default:
		panic("not possible to reach this point")
	}
}

// Hash returns the multihash contained by a Cid.
func (c Cid) Hash() mh.Multihash {
	bytes := c.Bytes()

	if c.Version() == 0 {
		return mh.Multihash(bytes)
	}

	// skip version length
	_, n1 := binary.Uvarint(bytes)
	// skip codec length
	_, n2 := binary.Uvarint(bytes[n1:])

	return mh.Multihash(bytes[n1+n2:])
}

// Bytes returns the byte representation of a Cid.
// The output of bytes can be parsed back into a Cid
// with Cast().
func (c Cid) Bytes() []byte {
	return []byte(c.str)
}

// MarshalBinary is equivalent to Bytes(). It implements the
// encoding.BinaryMarshaler interface.
func (c Cid) MarshalBinary() ([]byte, error) {
	return c.Bytes(), nil
}

// MarshalText is equivalent to String(). It implements the
// encoding.TextMarshaler interface.
func (c Cid) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

// Equals checks that two Cids are the same.
// In order for two Cids to be considered equal, the
// Version, the Codec and the Multihash must match.
func (c Cid) Equals(o Cid) bool {
	return c == o
}

// UnmarshalJSON parses the JSON representation of a Cid.
func (c *Cid) UnmarshalJSON(b []byte) error {
	if len(b) < 2 {
		return fmt.Errorf("invalid cid json blob")
	}
	obj := struct {
		CidTarget string `json:"/"`
	}{}
	objptr := &obj
	err := json.Unmarshal(b, &objptr)
	if err != nil {
		return err
	}
	if objptr == nil {
		*c = Cid{}
		return nil
	}

	if obj.CidTarget == "" {
		return fmt.Errorf("cid was incorrectly formatted")
	}

	out, err := Decode(obj.CidTarget)
	if err != nil {
		return err
	}

	*c = out

	return nil
}

// MarshalJSON procudes a JSON representation of a Cid, which looks as follows:
//
//    { "/": "<cid-string>" }
//
// Note that this formatting comes from the IPLD specification
// (https://github.com/ipld/specs/tree/master/ipld)
func (c Cid) MarshalJSON() ([]byte, error) {
	if !c.Defined() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("{\"/\":\"%s\"}", c.String())), nil
}

// KeyString returns the binary representation of the Cid as a string
func (c Cid) KeyString() string {
	return c.str
}

// Loggable returns a Loggable (as defined by
// https://godoc.org/github.com/ipfs/go-log).
func (c Cid) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"cid": c,
	}
}

// Prefix builds and returns a Prefix out of a Cid.
func (c Cid) Prefix() Prefix {
	dec, _ := mh.Decode(c.Hash()) // assuming we got a valid multiaddr, this will not error
	return Prefix{
		MhType:   dec.Code,
		MhLength: dec.Length,
		Version:  c.Version(),
		Codec:    c.Type(),
	}
}

// Prefix represents all the metadata of a Cid,
// that is, the Version, the Codec, the Multihash type
// and the Multihash length. It does not contains
// any actual content information.
// NOTE: The use -1 in MhLength to mean default length is deprecated,
//   use the V0Builder or V1Builder structures instead
type Prefix struct {
	Version  uint64
	Codec    uint64
	MhType   uint64
	MhLength int
}

// Sum uses the information in a prefix to perform a multihash.Sum()
// and return a newly constructed Cid with the resulting multihash.
func (p Prefix) Sum(data []byte) (Cid, error) {
	length := p.MhLength
	if p.MhType == mh.ID {
		length = -1
	}

	hash, err := mh.Sum(data, p.MhType, length)
	if err != nil {
		return Undef, err
	}

	switch p.Version {
	case 0:
		return NewCidV0(hash), nil
	case 1:
		return NewCidV1(p.Codec, hash), nil
	default:
		return Undef, fmt.Errorf("invalid cid version")
	}
}

// Bytes returns a byte representation of a Prefix. It looks like:
//
//     <version><codec><mh-type><mh-length>
func (p Prefix) Bytes() []byte {
	buf := make([]byte, 4*binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, p.Version)
	n += binary.PutUvarint(buf[n:], p.Codec)
	n += binary.PutUvarint(buf[n:], uint64(p.MhType))
	n += binary.PutUvarint(buf[n:], uint64(p.MhLength))
	return buf[:n]
}

// PrefixFromBytes parses a Prefix-byte representation onto a
// Prefix.
func PrefixFromBytes(buf []byte) (Prefix, error) {
	r := bytes.NewReader(buf)
	vers, err := binary.ReadUvarint(r)
	if err != nil {
		return Prefix{}, err
	}

	codec, err := binary.ReadUvarint(r)
	if err != nil {
		return Prefix{}, err
	}

	mhtype, err := binary.ReadUvarint(r)
	if err != nil {
		return Prefix{}, err
	}

	mhlen, err := binary.ReadUvarint(r)
	if err != nil {
		return Prefix{}, err
	}

	return Prefix{
		Version:  vers,
		Codec:    codec,
		MhType:   mhtype,
		MhLength: int(mhlen),
	}, nil
}

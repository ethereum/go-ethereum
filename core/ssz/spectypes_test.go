// Copyright 2026 The go-ethereum Authors
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

package ssz_test

// The types of the ssz_generic spec test format (tests/formats/ssz_generic/
// README.md), hand-written the way production users of core/ssz are expected
// to write theirs: explicit per-type methods built from the package
// primitives, no reflection, no code generation. They double as the
// ergonomics proof of the public API.

import (
	"encoding/hex"
	"fmt"
	"math"
	"strings"

	"github.com/ethereum/go-ethereum/core/ssz"
	"github.com/holiman/uint256"
	"gopkg.in/yaml.v3"
)

// specObject is what the harness drives: an SSZ object that can also load
// its expected value from a test case's value.yaml.
type specObject interface {
	ssz.Object
	fromYAML(node *yaml.Node) error
}

// yamlHex decodes a YAML "0x..." string into bytes.
func yamlHex(node *yaml.Node) ([]byte, error) {
	return hex.DecodeString(strings.TrimPrefix(node.Value, "0x"))
}

// yamlBig decodes a YAML decimal scalar (possibly quoted) into a uint256.
func yamlBig(node *yaml.Node) (*uint256.Int, error) {
	v := new(uint256.Int)
	if err := v.SetFromDecimal(node.Value); err != nil {
		return nil, fmt.Errorf("bad integer %q: %v", node.Value, err)
	}
	return v, nil
}

// errListTooLong is the encode-side counterpart of the decoder's limit
// check: a value holding more elements than its type allows must not
// serialize, or Encode would produce bytes that Decode rejects.
func errListTooLong(n int, limit uint64) error {
	return fmt.Errorf("list of %d elements exceeds limit %d: %w", n, limit, ssz.ErrTooBig)
}

// Basic kinds (bool, uintN), parameterized because the scalar and vector
// handlers take the element type from the case name.

type basicKind int

const (
	kindBool basicKind = iota
	kindUint8
	kindUint16
	kindUint32
	kindUint64
	kindUint128
	kindUint256
)

func parseBasicKind(s string) (basicKind, error) {
	switch s {
	case "bool":
		return kindBool, nil
	case "uint8":
		return kindUint8, nil
	case "uint16":
		return kindUint16, nil
	case "uint32":
		return kindUint32, nil
	case "uint64":
		return kindUint64, nil
	case "uint128":
		return kindUint128, nil
	case "uint256":
		return kindUint256, nil
	}
	return 0, fmt.Errorf("unknown basic type %q", s)
}

func (k basicKind) size() int {
	switch k {
	case kindBool, kindUint8:
		return 1
	case kindUint16:
		return 2
	case kindUint32:
		return 4
	case kindUint64:
		return 8
	case kindUint128:
		return 16
	default:
		return 32
	}
}

// append serializes one basic value held in a uint256 (bools are 0/1).
func (k basicKind) append(dst []byte, v *uint256.Int) []byte {
	switch k {
	case kindBool:
		return ssz.AppendBool(dst, v[0] != 0)
	case kindUint8:
		return ssz.AppendUint8(dst, uint8(v[0]))
	case kindUint16:
		return ssz.AppendUint16(dst, uint16(v[0]))
	case kindUint32:
		return ssz.AppendUint32(dst, uint32(v[0]))
	case kindUint64:
		return ssz.AppendUint64(dst, v[0])
	case kindUint128:
		return ssz.AppendUint128(dst, v)
	default:
		return ssz.AppendUint256(dst, v)
	}
}

// decode strictly decodes one basic value from exactly size() bytes.
func (k basicKind) decode(data []byte, v *uint256.Int) error {
	d := ssz.NewDecoder(data)
	switch k {
	case kindBool:
		b := d.Bool()
		v.Clear()
		if b {
			v.SetOne()
		}
	case kindUint8:
		v.SetUint64(uint64(d.Uint8()))
	case kindUint16:
		v.SetUint64(uint64(d.Uint16()))
	case kindUint32:
		v.SetUint64(uint64(d.Uint32()))
	case kindUint64:
		v.SetUint64(d.Uint64())
	case kindUint128:
		d.Uint128(v)
	default:
		d.Uint256(v)
	}
	return d.Close()
}

func (k basicKind) fromYAML(node *yaml.Node, v *uint256.Int) error {
	if k == kindBool {
		switch node.Value {
		case "true":
			v.SetOne()
		case "false":
			v.Clear()
		default:
			return fmt.Errorf("bad bool %q", node.Value)
		}
		return nil
	}
	b, err := yamlBig(node)
	if err != nil {
		return err
	}
	v.Set(b)
	return nil
}

// testBasic covers the boolean and uints handlers.
type testBasic struct {
	kind basicKind
	v    uint256.Int
}

func (t *testBasic) SizeSSZ() int { return t.kind.size() }
func (t *testBasic) MarshalSSZTo(dst []byte) ([]byte, error) {
	return t.kind.append(dst, &t.v), nil
}
func (t *testBasic) UnmarshalSSZ(data []byte) error {
	return t.kind.decode(data, &t.v)
}
func (t *testBasic) HashTreeRoot() [32]byte {
	buf, _ := t.MarshalSSZTo(nil)
	return ssz.Merkleize(ssz.Pack(buf), 1)
}
func (t *testBasic) fromYAML(node *yaml.Node) error {
	return t.kind.fromYAML(node, &t.v)
}

// testBasicVector covers basic_vector: Vector[elem, N].
type testBasicVector struct {
	kind   basicKind
	length int
	vals   []uint256.Int
}

func (t *testBasicVector) SizeSSZ() int { return t.kind.size() * t.length }
func (t *testBasicVector) MarshalSSZTo(dst []byte) ([]byte, error) {
	if len(t.vals) != t.length {
		return nil, fmt.Errorf("vector value has %d elements, type wants %d: %w", len(t.vals), t.length, ssz.ErrSize)
	}
	for i := range t.vals {
		dst = t.kind.append(dst, &t.vals[i])
	}
	return dst, nil
}
func (t *testBasicVector) UnmarshalSSZ(data []byte) error {
	if t.length == 0 {
		return fmt.Errorf("zero-length vector is an illegal type: %w", ssz.ErrSize)
	}
	size := t.kind.size()
	if len(data) != size*t.length {
		return fmt.Errorf("vector[%d] needs %d bytes, have %d: %w", t.length, size*t.length, len(data), ssz.ErrSize)
	}
	t.vals = make([]uint256.Int, t.length)
	for i := range t.vals {
		if err := t.kind.decode(data[i*size:(i+1)*size], &t.vals[i]); err != nil {
			return err
		}
	}
	return nil
}
func (t *testBasicVector) HashTreeRoot() [32]byte {
	buf, _ := t.MarshalSSZTo(nil)
	limit := uint64((t.kind.size()*t.length + 31) / 32)
	return ssz.Merkleize(ssz.Pack(buf), limit)
}
func (t *testBasicVector) fromYAML(node *yaml.Node) error {
	t.vals = make([]uint256.Int, len(node.Content))
	for i, c := range node.Content {
		if err := t.kind.fromYAML(c, &t.vals[i]); err != nil {
			return err
		}
	}
	return nil
}

// testBitvector covers bitvector: Bitvector[N]. Runtime form: packed bytes.
type testBitvector struct {
	nbits uint64
	data  []byte
}

func (t *testBitvector) SizeSSZ() int { return ssz.BitvectorSize(t.nbits) }
func (t *testBitvector) MarshalSSZTo(dst []byte) ([]byte, error) {
	return append(dst, t.data...), nil
}
func (t *testBitvector) UnmarshalSSZ(data []byte) error {
	if t.nbits == 0 {
		return fmt.Errorf("zero-length bitvector is an illegal type: %w", ssz.ErrSize)
	}
	if err := ssz.ValidateBitvector(data, t.nbits); err != nil {
		return err
	}
	t.data = append([]byte(nil), data...)
	return nil
}
func (t *testBitvector) HashTreeRoot() [32]byte {
	return ssz.Merkleize(ssz.PackBits(t.data), (t.nbits+255)/256)
}
func (t *testBitvector) fromYAML(node *yaml.Node) error {
	b, err := yamlHex(node)
	if err != nil {
		return err
	}
	t.data = b
	return nil
}

// testBitlist covers bitlist: Bitlist[limit]. Runtime form: packed data bits
// without the delimiter, plus the bit length.
type testBitlist struct {
	limit uint64
	bits  []byte
	nbits uint64
}

func (t *testBitlist) SizeSSZ() int { return ssz.BitlistSize(t.nbits) }
func (t *testBitlist) MarshalSSZTo(dst []byte) ([]byte, error) {
	if t.nbits > t.limit {
		return nil, errListTooLong(int(t.nbits), t.limit)
	}
	return ssz.AppendBitlist(dst, t.bits, t.nbits), nil
}
func (t *testBitlist) UnmarshalSSZ(data []byte) error {
	bits, nbits, err := ssz.DecodeBitlist(data, t.limit)
	if err != nil {
		return err
	}
	t.bits, t.nbits = bits, nbits
	return nil
}
func (t *testBitlist) HashTreeRoot() [32]byte {
	return ssz.MixInLength(ssz.Merkleize(ssz.PackBits(t.bits), (t.limit+255)/256), t.nbits)
}
func (t *testBitlist) fromYAML(node *yaml.Node) error {
	// The YAML form of a bitlist is its serialized form, delimiter included.
	ser, err := yamlHex(node)
	if err != nil {
		return err
	}
	return t.UnmarshalSSZ(ser)
}

// Field helpers shared by the containers below.

// List[uint16, limit]
func sizeUint16List(vals []uint16) int { return 2 * len(vals) }
func appendUint16List(dst []byte, vals []uint16, limit uint64) ([]byte, error) {
	if uint64(len(vals)) > limit {
		return nil, errListTooLong(len(vals), limit)
	}
	for _, v := range vals {
		dst = ssz.AppendUint16(dst, v)
	}
	return dst, nil
}
func decodeUint16List(data []byte, limit uint64) ([]uint16, error) {
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("uint16 list of %d bytes: %w", len(data), ssz.ErrSize)
	}
	if uint64(len(data)/2) > limit {
		return nil, errListTooLong(len(data)/2, limit)
	}
	vals := make([]uint16, len(data)/2)
	for i := range vals {
		d := ssz.NewDecoder(data[2*i : 2*i+2])
		vals[i] = d.Uint16()
		if err := d.Close(); err != nil {
			return nil, err
		}
	}
	return vals, nil
}
func htrUint16List(vals []uint16, limit uint64) [32]byte {
	buf, _ := appendUint16List(nil, vals, limit)
	return ssz.MixInLength(ssz.Merkleize(ssz.Pack(buf), (limit*2+31)/32), uint64(len(vals)))
}

// ByteList[limit]
func htrByteList(data []byte, limit uint64) [32]byte {
	return ssz.MixInLength(ssz.Merkleize(ssz.Pack(data), (limit+31)/32), uint64(len(data)))
}

// hexBlob is a []byte YAML-decoded from 0x-hex.
type hexBlob []byte

func (h *hexBlob) UnmarshalYAML(node *yaml.Node) error {
	b, err := yamlHex(node)
	if err != nil {
		return err
	}
	*h = b
	return nil
}

// yamlBitlist is a bitlist YAML-decoded from its serialized hex form. The
// limit is enforced by the containing type, not at YAML load.
type yamlBitlist struct {
	bits  []byte
	nbits uint64
}

func (y *yamlBitlist) UnmarshalYAML(node *yaml.Node) error {
	ser, err := yamlHex(node)
	if err != nil {
		return err
	}
	bits, nbits, err := ssz.DecodeBitlist(ser, math.MaxUint64)
	if err != nil {
		return err
	}
	y.bits, y.nbits = bits, nbits
	return nil
}

// The pre-defined Container structures of the test format.

// SingleFieldTestStruct { A: byte }
type singleFieldTestStruct struct {
	A uint8 `yaml:"A"`
}

func (s *singleFieldTestStruct) SizeSSZ() int { return 1 }
func (s *singleFieldTestStruct) MarshalSSZTo(dst []byte) ([]byte, error) {
	return ssz.AppendUint8(dst, s.A), nil
}
func (s *singleFieldTestStruct) UnmarshalSSZ(data []byte) error {
	d := ssz.NewDecoder(data)
	s.A = d.Uint8()
	return d.Close()
}
func (s *singleFieldTestStruct) HashTreeRoot() [32]byte {
	roots := [][32]byte{ssz.Merkleize(ssz.Pack([]byte{s.A}), 1)}
	return ssz.Merkleize(roots, 1)
}
func (s *singleFieldTestStruct) fromYAML(node *yaml.Node) error { return node.Decode(s) }

// SmallTestStruct { A, B: uint16 }
type smallTestStruct struct {
	A uint16 `yaml:"A"`
	B uint16 `yaml:"B"`
}

func (s *smallTestStruct) SizeSSZ() int { return 4 }
func (s *smallTestStruct) MarshalSSZTo(dst []byte) ([]byte, error) {
	dst = ssz.AppendUint16(dst, s.A)
	return ssz.AppendUint16(dst, s.B), nil
}
func (s *smallTestStruct) UnmarshalSSZ(data []byte) error {
	d := ssz.NewDecoder(data)
	s.A = d.Uint16()
	s.B = d.Uint16()
	return d.Close()
}
func (s *smallTestStruct) HashTreeRoot() [32]byte {
	roots := [][32]byte{
		ssz.Merkleize(ssz.Pack(ssz.AppendUint16(nil, s.A)), 1),
		ssz.Merkleize(ssz.Pack(ssz.AppendUint16(nil, s.B)), 1),
	}
	return ssz.Merkleize(roots, 2)
}
func (s *smallTestStruct) fromYAML(node *yaml.Node) error { return node.Decode(s) }

// FixedTestStruct { A: uint8, B: uint64, C: uint32 }
type fixedTestStruct struct {
	A uint8  `yaml:"A"`
	B uint64 `yaml:"B"`
	C uint32 `yaml:"C"`
}

func (s *fixedTestStruct) SizeSSZ() int { return 13 }
func (s *fixedTestStruct) MarshalSSZTo(dst []byte) ([]byte, error) {
	dst = ssz.AppendUint8(dst, s.A)
	dst = ssz.AppendUint64(dst, s.B)
	return ssz.AppendUint32(dst, s.C), nil
}
func (s *fixedTestStruct) UnmarshalSSZ(data []byte) error {
	d := ssz.NewDecoder(data)
	s.A = d.Uint8()
	s.B = d.Uint64()
	s.C = d.Uint32()
	return d.Close()
}
func (s *fixedTestStruct) HashTreeRoot() [32]byte {
	roots := [][32]byte{
		ssz.Merkleize(ssz.Pack([]byte{s.A}), 1),
		ssz.Merkleize(ssz.Pack(ssz.AppendUint64(nil, s.B)), 1),
		ssz.Merkleize(ssz.Pack(ssz.AppendUint32(nil, s.C)), 1),
	}
	return ssz.Merkleize(roots, 3)
}
func (s *fixedTestStruct) fromYAML(node *yaml.Node) error { return node.Decode(s) }

// VarTestStruct { A: uint16, B: List[uint16, 1024], C: uint8 }
type varTestStruct struct {
	A uint16   `yaml:"A"`
	B []uint16 `yaml:"B"`
	C uint8    `yaml:"C"`
}

const varTestStructBLimit = 1024

func (s *varTestStruct) SizeSSZ() int { return 2 + 4 + 1 + sizeUint16List(s.B) }
func (s *varTestStruct) MarshalSSZTo(dst []byte) ([]byte, error) {
	dst = ssz.AppendUint16(dst, s.A)
	dst = ssz.AppendOffset(dst, 7) // fixed part: 2 + 4 + 1
	dst = ssz.AppendUint8(dst, s.C)
	return appendUint16List(dst, s.B, varTestStructBLimit)
}
func (s *varTestStruct) UnmarshalSSZ(data []byte) error {
	d := ssz.NewDecoder(data)
	s.A = d.Uint16()
	d.Offset()
	s.C = d.Uint8()
	sections, err := d.Sections()
	if err != nil {
		return err
	}
	s.B, err = decodeUint16List(sections[0], varTestStructBLimit)
	return err
}
func (s *varTestStruct) HashTreeRoot() [32]byte {
	roots := [][32]byte{
		ssz.Merkleize(ssz.Pack(ssz.AppendUint16(nil, s.A)), 1),
		htrUint16List(s.B, varTestStructBLimit),
		ssz.Merkleize(ssz.Pack([]byte{s.C}), 1),
	}
	return ssz.Merkleize(roots, 3)
}
func (s *varTestStruct) fromYAML(node *yaml.Node) error { return node.Decode(s) }

//	ComplexTestStruct {
//	  A: uint16, B: List[uint16, 128], C: uint8, D: ByteList[256],
//	  E: VarTestStruct, F: Vector[FixedTestStruct, 4], G: Vector[VarTestStruct, 2],
//	}
type complexTestStruct struct {
	A uint16             `yaml:"A"`
	B []uint16           `yaml:"B"`
	C uint8              `yaml:"C"`
	D hexBlob            `yaml:"D"`
	E varTestStruct      `yaml:"E"`
	F [4]fixedTestStruct `yaml:"F"`
	G [2]varTestStruct   `yaml:"G"`
}

const (
	complexBLimit = 128
	complexDLimit = 256
	// A(2) + offB(4) + C(1) + offD(4) + offE(4) + F(4*13) + offG(4)
	complexFixedSize = 2 + 4 + 1 + 4 + 4 + 4*13 + 4
)

func (s *complexTestStruct) SizeSSZ() int {
	size := complexFixedSize + sizeUint16List(s.B) + len(s.D) + s.E.SizeSSZ()
	for i := range s.G {
		size += ssz.BytesPerLengthOffset + s.G[i].SizeSSZ()
	}
	return size
}
func (s *complexTestStruct) MarshalSSZTo(dst []byte) ([]byte, error) {
	if uint64(len(s.D)) > complexDLimit {
		return nil, errListTooLong(len(s.D), complexDLimit)
	}
	offset := complexFixedSize
	dst = ssz.AppendUint16(dst, s.A)
	dst = ssz.AppendOffset(dst, offset)
	offset += sizeUint16List(s.B)
	dst = ssz.AppendUint8(dst, s.C)
	dst = ssz.AppendOffset(dst, offset)
	offset += len(s.D)
	dst = ssz.AppendOffset(dst, offset)
	offset += s.E.SizeSSZ()
	var err error
	for i := range s.F {
		if dst, err = s.F[i].MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}
	dst = ssz.AppendOffset(dst, offset)
	// Variable part.
	if dst, err = appendUint16List(dst, s.B, complexBLimit); err != nil {
		return nil, err
	}
	dst = append(dst, s.D...)
	if dst, err = s.E.MarshalSSZTo(dst); err != nil {
		return nil, err
	}
	return ssz.MarshalVariableSlice(dst, []*varTestStruct{&s.G[0], &s.G[1]})
}
func (s *complexTestStruct) UnmarshalSSZ(data []byte) error {
	d := ssz.NewDecoder(data)
	s.A = d.Uint16()
	d.Offset() // B
	s.C = d.Uint8()
	d.Offset() // D
	d.Offset() // E
	for i := range s.F {
		var buf [13]byte
		d.Bytes(buf[:])
		if err := s.F[i].UnmarshalSSZ(buf[:]); err != nil {
			return err
		}
	}
	d.Offset() // G
	sections, err := d.Sections()
	if err != nil {
		return err
	}
	if s.B, err = decodeUint16List(sections[0], complexBLimit); err != nil {
		return err
	}
	if uint64(len(sections[1])) > complexDLimit {
		return errListTooLong(len(sections[1]), complexDLimit)
	}
	s.D = append(hexBlob(nil), sections[1]...)
	if err = s.E.UnmarshalSSZ(sections[2]); err != nil {
		return err
	}
	gs, err := ssz.UnmarshalVariableSlice[varTestStruct, *varTestStruct](sections[3], 2)
	if err != nil {
		return err
	}
	if len(gs) != 2 {
		return fmt.Errorf("vector G has %d elements, want 2: %w", len(gs), ssz.ErrSize)
	}
	copy(s.G[:], gs)
	return nil
}
func (s *complexTestStruct) HashTreeRoot() [32]byte {
	fRoots := make([][32]byte, 4)
	for i := range s.F {
		fRoots[i] = s.F[i].HashTreeRoot()
	}
	gRoots := make([][32]byte, 2)
	for i := range s.G {
		gRoots[i] = s.G[i].HashTreeRoot()
	}
	roots := [][32]byte{
		ssz.Merkleize(ssz.Pack(ssz.AppendUint16(nil, s.A)), 1),
		htrUint16List(s.B, complexBLimit),
		ssz.Merkleize(ssz.Pack([]byte{s.C}), 1),
		htrByteList(s.D, complexDLimit),
		s.E.HashTreeRoot(),
		ssz.Merkleize(fRoots, 4),
		ssz.Merkleize(gRoots, 2),
	}
	return ssz.Merkleize(roots, 7)
}
func (s *complexTestStruct) fromYAML(node *yaml.Node) error { return node.Decode(s) }

//	BitsStruct {
//	  A: Bitlist[5], B: Bitvector[2], C: Bitvector[1], D: Bitlist[6], E: Bitvector[8]
//	}
type bitsStruct struct {
	A yamlBitlist `yaml:"A"`
	B hexBlob     `yaml:"B"`
	C hexBlob     `yaml:"C"`
	D yamlBitlist `yaml:"D"`
	E hexBlob     `yaml:"E"`
}

const (
	bitsALimit = 5
	bitsDLimit = 6
	// offA(4) + B(1) + C(1) + offD(4) + E(1)
	bitsFixedSize = 4 + 1 + 1 + 4 + 1
)

func (s *bitsStruct) SizeSSZ() int {
	return bitsFixedSize + ssz.BitlistSize(s.A.nbits) + ssz.BitlistSize(s.D.nbits)
}
func (s *bitsStruct) MarshalSSZTo(dst []byte) ([]byte, error) {
	if s.A.nbits > bitsALimit {
		return nil, errListTooLong(int(s.A.nbits), bitsALimit)
	}
	if s.D.nbits > bitsDLimit {
		return nil, errListTooLong(int(s.D.nbits), bitsDLimit)
	}
	offset := bitsFixedSize
	dst = ssz.AppendOffset(dst, offset)
	offset += ssz.BitlistSize(s.A.nbits)
	dst = append(dst, s.B...)
	dst = append(dst, s.C...)
	dst = ssz.AppendOffset(dst, offset)
	dst = append(dst, s.E...)
	dst = ssz.AppendBitlist(dst, s.A.bits, s.A.nbits)
	return ssz.AppendBitlist(dst, s.D.bits, s.D.nbits), nil
}
func (s *bitsStruct) UnmarshalSSZ(data []byte) error {
	d := ssz.NewDecoder(data)
	d.Offset() // A
	var b, c, e [1]byte
	d.Bytes(b[:])
	d.Bytes(c[:])
	d.Offset() // D
	d.Bytes(e[:])
	sections, err := d.Sections()
	if err != nil {
		return err
	}
	if err := ssz.ValidateBitvector(b[:], 2); err != nil {
		return err
	}
	if err := ssz.ValidateBitvector(c[:], 1); err != nil {
		return err
	}
	if err := ssz.ValidateBitvector(e[:], 8); err != nil {
		return err
	}
	s.B, s.C, s.E = b[:], c[:], e[:]
	if s.A.bits, s.A.nbits, err = ssz.DecodeBitlist(sections[0], bitsALimit); err != nil {
		return err
	}
	s.D.bits, s.D.nbits, err = ssz.DecodeBitlist(sections[1], bitsDLimit)
	return err
}
func (s *bitsStruct) HashTreeRoot() [32]byte {
	roots := [][32]byte{
		ssz.MixInLength(ssz.Merkleize(ssz.PackBits(s.A.bits), 1), s.A.nbits),
		ssz.Merkleize(ssz.PackBits(s.B), 1),
		ssz.Merkleize(ssz.PackBits(s.C), 1),
		ssz.MixInLength(ssz.Merkleize(ssz.PackBits(s.D.bits), 1), s.D.nbits),
		ssz.Merkleize(ssz.PackBits(s.E), 1),
	}
	return ssz.Merkleize(roots, 5)
}
func (s *bitsStruct) fromYAML(node *yaml.Node) error { return node.Decode(s) }

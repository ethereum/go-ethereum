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

package ssz

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"testing"
)

// naiveContainerRoot is hash_tree_root of a ProgressiveContainer transcribed
// from EIP-7495 as amended by EIPs PR 12419: the field roots laid out at the
// set positions of the mask with a zero chunk at each cleared one, the
// progressive recursion over that list, and the packed mask hashed in on the
// right. It packs the mask itself, so the production word is checked too.
func naiveContainerRoot(mask []int, roots [][32]byte) [32]byte {
	chunks := make([][32]byte, len(mask))
	var word [32]byte
	k := 0
	for i, b := range mask {
		if b == 1 {
			chunks[i] = roots[k]
			word[i/8] |= 1 << (i % 8)
			k++
		}
	}
	root := naiveMerkleizeProgressive(chunks, 1)
	var buf [64]byte
	copy(buf[:32], root[:])
	copy(buf[32:], word[:])
	return sha256.Sum256(buf[:])
}

// mask builds an active_fields literal from runs: mask(1, 0, 20, 1, 1) is one
// 1, twenty 0s and one 1.
func mask(runs ...int) []int {
	var m []int
	for i := 0; i+1 < len(runs); i += 2 {
		m = append(m, slices.Repeat([]int{runs[i]}, runs[i+1])...)
	}
	return m
}

func mustPanic(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: did not panic", name)
		}
	}()
	f()
}

// TestActiveFieldsRules checks the illegal configurations of EIP-7495 and
// the accepted edges. The rejected cases are named after the type rejection
// fixtures of ethereum/ssz-specs (tests/fillers/ssz/test_illegal_types.py);
// layout_field_count, the fourth rule, is TestProgressiveContainerFieldCount.
func TestActiveFieldsRules(t *testing.T) {
	for _, c := range []struct {
		name string
		mask []int
		ok   bool
	}{
		{"empty_layout", nil, false},
		{"layout_too_wide", mask(0, 256, 1, 1), false},
		{"layout_ends_on_a_gap", []int{1, 0}, false},
		{"entry_not_a_bit", []int{1, 2, 1}, false},
		{"single_field", []int{1}, true},
		{"leading_gaps", []int{0, 0, 0, 0, 1}, true},
		{"widest_layout", mask(0, 255, 1, 1), true},
	} {
		af, err := NewActiveFields(c.mask...)
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok %v", c.name, err, c.ok)
			continue
		}
		if !c.ok {
			mustPanic(t, c.name, func() { MustActiveFields(c.mask...) })
			continue
		}
		nfields := 0
		for i, b := range c.mask {
			nfields += b
			if af.Active(i) != (b == 1) {
				t.Errorf("%s: Active(%d) = %v, want %v", c.name, i, af.Active(i), b == 1)
			}
		}
		if af.Width() != len(c.mask) || af.NumFields() != nfields {
			t.Errorf("%s: Width %d NumFields %d, want %d %d", c.name, af.Width(), af.NumFields(), len(c.mask), nfields)
		}
		if af.Active(-1) || af.Active(len(c.mask)) {
			t.Errorf("%s: positions outside the configuration are active", c.name)
		}
	}
}

// TestActiveFieldsPacking pins pack_bits(active_fields): bit i of the
// configuration is bit i%8 of byte i/8 of the mix-in word, and the mix-in is
// plain sha256 over root ‖ word.
func TestActiveFieldsPacking(t *testing.T) {
	for _, c := range []struct {
		mask      []int
		byteIndex int
		want      byte
	}{
		{[]int{1, 0, 1, 0, 1}, 0, 0b10101},
		{[]int{0, 0, 0, 0, 1}, 0, 0b10000},
		{mask(1, 1, 0, 20, 1, 1), 2, 0b100000},
		{mask(0, 255, 1, 1), 31, 0b10000000},
	} {
		af := MustActiveFields(c.mask...)
		if af.bits[c.byteIndex] != c.want {
			t.Errorf("%v: byte %d = %08b, want %08b", c.mask, c.byteIndex, af.bits[c.byteIndex], c.want)
		}
		for _, b := range af.bits[c.byteIndex+1:] {
			if b != 0 {
				t.Errorf("%v: bits set beyond the configuration", c.mask)
			}
		}
	}
	af := MustActiveFields(1, 0, 1, 0, 1)
	root := [32]byte{0xaa}
	var buf [64]byte
	copy(buf[:32], root[:])
	buf[32] = 0b10101
	if got, want := MixInActiveFields(root, af), sha256.Sum256(buf[:]); got != want {
		t.Errorf("MixInActiveFields = %x, want %x", got, want)
	}
}

// TestProgressiveContainerFieldCount is the fourth illegal configuration,
// layout_field_count: the number of roots must equal the number of set bits.
// Both directions panic, and so does the zero ActiveFields, which would
// otherwise return a plausible root for an empty field list.
func TestProgressiveContainerFieldCount(t *testing.T) {
	for _, c := range []struct {
		name  string
		af    ActiveFields
		roots int
	}{
		{"one_root_too_few", MustActiveFields(1, 0, 1), 1},
		{"one_root_too_many", MustActiveFields(1, 0, 1), 3},
		{"zero_value_no_roots", ActiveFields{}, 0},
		{"zero_value_one_root", ActiveFields{}, 1},
	} {
		mustPanic(t, c.name, func() { HashTreeRootProgressiveContainer(c.af, make([][32]byte, c.roots)) })
	}
}

// TestProgressiveContainerAgainstSpec compares the production root with the
// transcription of the EIP for the layouts of the spec vectors, the EIP-7807
// payload, the Gloas BeaconState, the widest legal layout and a layout whose
// last field opens a new subtree, with random field roots.
func TestProgressiveContainerAgainstSpec(t *testing.T) {
	rng := rand.New(rand.NewSource(7495))
	for _, m := range [][]int{
		{1},
		{0, 0, 0, 0, 1},
		{1, 0, 1, 0, 1},
		{1, 0, 1, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 1, 1},
		mask(1, 18),             // EIP-7807 ExecutionPayload
		mask(1, 46),             // Gloas BeaconState
		mask(1, 1, 0, 20, 1, 1), // last field in the fourth subtree
		mask(0, 255, 1, 1),      // widest legal layout
	} {
		af := MustActiveFields(m...)
		roots := randomChunks(rng, af.NumFields())
		if got, want := HashTreeRootProgressiveContainer(af, roots), naiveContainerRoot(m, roots); got != want {
			t.Errorf("mask %v: root %x, spec transcription %x", m, got, want)
		}
	}
}

// TestProgressiveContainerGaps is the point of EIPs PR 12419: a gap is a zero
// chunk at its position, not a position skipped. The same two roots under
// [1, 0, 1] and [1, 1] must differ, and the gapped root must differ from the
// compacted reading, merkleizing the two roots and mixing in the [1, 0, 1]
// word, which is what Nethermind and Prysm before its fix #17618 compute.
func TestProgressiveContainerGaps(t *testing.T) {
	roots := randomChunks(rand.New(rand.NewSource(12419)), 2)
	gapped := HashTreeRootProgressiveContainer(MustActiveFields(1, 0, 1), roots)
	dense := HashTreeRootProgressiveContainer(MustActiveFields(1, 1), roots)
	compacted := MixInActiveFields(MerkleizeProgressive(roots), MustActiveFields(1, 0, 1))
	if gapped == dense {
		t.Error("[1, 0, 1] and [1, 1] give the same root for the same field roots")
	}
	if gapped == compacted {
		t.Error("the gap was compacted away")
	}
	if want := naiveContainerRoot([]int{1, 0, 1}, roots); gapped != want {
		t.Errorf("gapped root %x, spec transcription %x", gapped, want)
	}
}

// Chunks of basic values, for the published roots below.
func chunkOf(data []byte) [32]byte  { return Pack(data)[0] }
func chunkUint8(v uint8) [32]byte   { return chunkOf(AppendUint8(nil, v)) }
func chunkUint16(v uint16) [32]byte { return chunkOf(AppendUint16(nil, v)) }
func chunkUint32(v uint32) [32]byte { return chunkOf(AppendUint32(nil, v)) }
func chunkUint64(v uint64) [32]byte { return chunkOf(AppendUint64(nil, v)) }

// sampleSquare is SampleSquare of the ssz-specs fixtures,
// ProgressiveContainer(active_fields=[1, 0, 1]) { side: uint16, color: uint8 }.
type sampleSquare struct {
	side  uint16
	color uint8
}

var afSampleSquare = MustActiveFields(1, 0, 1)

func (s *sampleSquare) SizeSSZ() int { return 3 }
func (s *sampleSquare) MarshalSSZTo(dst []byte) ([]byte, error) {
	dst = AppendUint16(dst, s.side)
	return AppendUint8(dst, s.color), nil
}
func (s *sampleSquare) UnmarshalSSZ(data []byte) error {
	d := NewDecoder(data)
	s.side = d.Uint16()
	s.color = d.Uint8()
	return d.Close()
}
func (s *sampleSquare) HashTreeRoot() [32]byte {
	return HashTreeRootProgressiveContainer(afSampleSquare, [][32]byte{chunkUint16(s.side), chunkUint8(s.color)})
}

// sampleBoundedListField is SampleBoundedListField of the ssz-specs fixtures,
// ProgressiveContainer(active_fields=[1, 0, 1]) { head: uint64, body: List[uint16, 4] }.
type sampleBoundedListField struct {
	head uint64
	body []uint16
}

const sampleBodyLimit = 4

func (s *sampleBoundedListField) SizeSSZ() int { return 8 + 4 + 2*len(s.body) }
func (s *sampleBoundedListField) MarshalSSZTo(dst []byte) ([]byte, error) {
	if len(s.body) > sampleBodyLimit {
		return nil, fmt.Errorf("body of %d elements: %w", len(s.body), ErrTooBig)
	}
	dst = AppendUint64(dst, s.head)
	dst = AppendOffset(dst, 12)
	for _, v := range s.body {
		dst = AppendUint16(dst, v)
	}
	return dst, nil
}
func (s *sampleBoundedListField) UnmarshalSSZ(data []byte) error {
	d := NewDecoder(data)
	s.head = d.Uint64()
	d.Offset()
	sections, err := d.Sections()
	if err != nil {
		return err
	}
	body := sections[0]
	if len(body)%2 != 0 {
		return fmt.Errorf("body of %d bytes: %w", len(body), ErrSize)
	}
	if len(body)/2 > sampleBodyLimit {
		return fmt.Errorf("body of %d elements: %w", len(body)/2, ErrTooBig)
	}
	s.body = make([]uint16, len(body)/2)
	for i := range s.body {
		s.body[i] = NewDecoder(body[2*i : 2*i+2]).Uint16()
	}
	return nil
}
func (s *sampleBoundedListField) HashTreeRoot() [32]byte {
	var ser []byte
	for _, v := range s.body {
		ser = AppendUint16(ser, v)
	}
	body := MixInLength(Merkleize(Pack(ser), (sampleBodyLimit*2+31)/32), uint64(len(s.body)))
	return HashTreeRootProgressiveContainer(afSampleSquare, [][32]byte{chunkUint64(s.head), body})
}

// TestProgressiveContainerKnownRoots checks container roots published by
// other implementations, so that correctness does not rest on this package's
// reading of the EIP: the ethereum/ssz-specs v0.1.0 fixtures
// (fixtures/ssz/ssz/test_progressive_containers), the vector shared by
// Erigon's cl/merkle_tree tests and Prysm PR #17618, dynamic-ssz's two-gap
// case (reflection/common_test.go) and two roots sigp/tree_hash computed from
// the EIP pseudocode (tests/tests.rs).
func TestProgressiveContainerKnownRoots(t *testing.T) {
	pc := func(af ActiveFields, roots ...[32]byte) [32]byte {
		return HashTreeRootProgressiveContainer(af, roots)
	}
	af101 := MustActiveFields(1, 0, 1)
	square := func(side uint16, color uint8) [32]byte {
		return (&sampleSquare{side, color}).HashTreeRoot()
	}
	for _, c := range []struct {
		name string
		root string
		got  [32]byte
	}{
		// ssz-specs v0.1.0 fixtures.
		{"SampleSquare [1,0,1] side 0x1234 color 0x56",
			"5ebd038215d6c6868befbe172ffb9442b2f5ade276bd96eb304c1da38deff823",
			square(0x1234, 0x56)},
		{"SampleCircle [0,1,1] radius 0x1234 color 0x56",
			"44dd01593fff4f0bea317b62a9e70d20f063e7413f331598d681d9e645fa8eae",
			pc(MustActiveFields(0, 1, 1), chunkUint16(0x1234), chunkUint8(0x56))},
		{"SampleOneField [1] a 0xbeef",
			"a88f083e786a9c55bf466e4954c4e25b3112cad5a5a2f85248d3a88d42b9fc58",
			pc(MustActiveFields(1), chunkUint16(0xbeef))},
		{"SampleLeadingGaps [0,0,1] c 0x11223344",
			"ee417c8b5e7a3051945e454ee09fc3e3b32a26352332a2d669a8912bcf4383d2",
			pc(MustActiveFields(0, 0, 1), chunkUint32(0x11223344))},
		{"SampleMultipleGaps [1,0,0,1,0,1] a 1 b 0x0203 c 0x04050607",
			"4c581934216751d2ec29ea5bbaf238a78ae8c240c594afa8662b449eb51cd455",
			pc(MustActiveFields(1, 0, 0, 1, 0, 1), chunkUint8(1), chunkUint16(0x0203), chunkUint32(0x04050607))},
		{"SampleWidestLayout [0]*255+[1] tail 0xab",
			"1bf935e375b4ebf1516fe0dae3e6d959168fd74f470b694b064e2d504491fba0",
			pc(MustActiveFields(mask(0, 255, 1, 1)...), chunkUint8(0xab))},
		{"SampleLevelBoundary [1]+[0]*20+[1] first 0x1234 last 0x56",
			"4ced50eb70f7547000227b026e875c87dfe4d4daa3be3f308f570268c5a55dc7",
			pc(MustActiveFields(mask(1, 1, 0, 20, 1, 1)...), chunkUint16(0x1234), chunkUint8(0x56))},
		{"SampleBoundedListField [1,0,1] head 7 body [1,2]",
			"abc61ac7614150aaab32d5a8ef1b71baa60a3d7f48a56e33599add004fbaa0ea",
			(&sampleBoundedListField{7, []uint16{1, 2}}).HashTreeRoot()},
		{"SampleBoundedListField [1,0,1] head 7 body []",
			"a1e6b73e491b250c41328bbff3f8bc31d3b44f3dd06cbed35ac370a2e386f69d",
			(&sampleBoundedListField{7, nil}).HashTreeRoot()},
		{"SampleProgressiveFields [1,1,1] head 7 numbers [1,2,3] flags [1,0,1]",
			"0c621f05791743c263be5a717c45dec81b4c175d19572e4cca0d5933f7044f0e",
			pc(MustActiveFields(1, 1, 1), chunkUint64(7),
				MixInLength(MerkleizeProgressive(Pack(AppendUint64(AppendUint64(AppendUint64(nil, 1), 2), 3))), 3),
				HashTreeRootProgressiveBitlist([]byte{0b101}, 3))},
		{"SampleOuterShape [1,0,1] head 1 inner SampleInnerShape [1,0,1] {x 0x0203, y 4}",
			"ccf454ac88b0e70221c4458d2df7f53ff547eacfd8d15b00a3077bd611b7f6a1",
			pc(af101, chunkUint8(1), pc(af101, chunkUint16(0x0203), chunkUint8(4)))},
		{"SampleSquareContainer, a Container { tag 0xff, shape SampleSquare }",
			"82a3af75922fff248c6e24745fc37f8946ce3e5af7b8bf4778f297ec48ea533c",
			Merkleize([][32]byte{chunkUint8(0xff), square(0x1234, 0x56)}, 2)},
		{"SampleSquareProgressiveList [Square(1,2), Square(3,4)]",
			"a892d5d4f0ef9b1f4e0c3f9fd9617f06d405561d56f676dcd9de228b2fbcbe59",
			HashTreeRootProgressiveRoots([][32]byte{square(1, 2), square(3, 4)})},
		// Erigon TestProgressiveContainerRootInactiveFieldVector and Prysm PR #17618.
		{"Erigon/Prysm [1,0,1] roots LE(1) LE(2)",
			"3a6584864e28437da67deac288c46c9b60cee55880b19b12cfe68a7d1d5bc491",
			pc(af101, chunkUint64(1), chunkUint64(2))},
		// dynamic-ssz progressive_container_3.
		{"dynamic-ssz [1,1,0,0,1,1] uint64 12345 uint32 67890 bool true uint16 999",
			"69cf970cadf7ef2055070f37237e7d9b1e9807beb9c245fc04fc81532e021b20",
			pc(MustActiveFields(1, 1, 0, 0, 1, 1), chunkUint64(12345), chunkUint32(67890), chunkUint8(1), chunkUint16(999))},
		// sigp/tree_hash tests.rs, computed from the EIP pseudocode.
		{"tree_hash [1] x uint8 125",
			"b6a2f148c33179dec1bdaa979a11776ff2d881fca93974b286443a8539dc0872",
			pc(MustActiveFields(1), chunkUint8(125))},
		{"tree_hash [1,0,0,0,0,0,0,0,0,1] a 0xaa b 0xbb",
			"346cfa442bcc5f2288316f185aa5b7e0a292d2e287862cc6f177ee1362862148",
			pc(MustActiveFields(1, 0, 0, 0, 0, 0, 0, 0, 0, 1), chunkUint8(0xaa), chunkUint8(0xbb))},
	} {
		if want := mustRoot(t, c.root); c.got != want {
			t.Errorf("%s: root %x, want %x", c.name, c.got, want)
		}
	}
}

// TestProgressiveContainerFixtureBytes checks the two fixture types with a
// serialization against their published bytes, round trip and root, and the
// decode failures the fixtures and the ssz-specs HEAD rejection cases carry:
// a trailing byte, a truncated field, and a first offset off by one in either
// direction. Serialization is that of a Container, so these exercise nothing
// new in the decoder; they pin that a progressive container type built from
// the primitives rejects what the reference rejects.
func TestProgressiveContainerFixtureBytes(t *testing.T) {
	for _, c := range []struct {
		obj, fresh Object
		serialized string
		root       string
	}{
		{&sampleSquare{0x1234, 0x56}, new(sampleSquare),
			"341256", "5ebd038215d6c6868befbe172ffb9442b2f5ade276bd96eb304c1da38deff823"},
		{&sampleBoundedListField{7, []uint16{1, 2}}, new(sampleBoundedListField),
			"07000000000000000c00000001000200", "abc61ac7614150aaab32d5a8ef1b71baa60a3d7f48a56e33599add004fbaa0ea"},
		{&sampleBoundedListField{7, nil}, new(sampleBoundedListField),
			"07000000000000000c000000", "a1e6b73e491b250c41328bbff3f8bc31d3b44f3dd06cbed35ac370a2e386f69d"},
	} {
		want, _ := hex.DecodeString(c.serialized)
		enc, err := Encode(c.obj)
		if err != nil || !bytes.Equal(enc, want) {
			t.Errorf("%T: encoded %x (err %v), want %x", c.obj, enc, err, want)
		}
		if got := c.obj.HashTreeRoot(); got != mustRoot(t, c.root) {
			t.Errorf("%T: root %x, want %s", c.obj, got, c.root)
		}
		if err := Decode(want, c.fresh); err != nil {
			t.Errorf("%T: decode: %v", c.obj, err)
		} else if got := c.fresh.HashTreeRoot(); got != mustRoot(t, c.root) {
			t.Errorf("%T: root after decode %x, want %s", c.obj, got, c.root)
		}
	}
	for _, c := range []struct {
		name string
		obj  Object
		data string
		want error
	}{
		{"decode_failure_trailing_byte", new(sampleSquare), "34125600", ErrTrailing},
		{"truncated_field", new(sampleSquare), "34", ErrSize},
		{"decode_failure_first_offset", new(sampleBoundedListField), "07000000000000000d000000", ErrOffset},
		{"first_offset_overlap", new(sampleBoundedListField), "07000000000000000b000000", ErrOffset},
	} {
		data, _ := hex.DecodeString(c.data)
		if err := Decode(data, c.obj); !errors.Is(err, c.want) {
			t.Errorf("%s: error %v, want %v", c.name, err, c.want)
		}
	}
}

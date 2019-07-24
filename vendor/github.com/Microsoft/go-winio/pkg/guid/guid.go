// Package guid provides a GUID type. The backing structure for a GUID is
// identical to that used by the golang.org/x/sys/windows GUID type.
// There are two main binary encodings used for a GUID, the big-endian encoding,
// and the Windows (mixed-endian) encoding. See here for details:
// https://en.wikipedia.org/wiki/Universally_unique_identifier#Encoding
package guid

import (
	"crypto/rand"
	"encoding"
	"encoding/binary"
	"fmt"
	"strconv"

	"golang.org/x/sys/windows"
)

// Variant specifies which GUID variant (or "type") of the GUID. It determines
// how the entirety of the rest of the GUID is interpreted.
type Variant uint8

// The variants specified by RFC 4122.
const (
	// VariantUnknown specifies a GUID variant which does not conform to one of
	// the variant encodings specified in RFC 4122.
	VariantUnknown Variant = iota
	VariantNCS
	VariantRFC4122
	VariantMicrosoft
	VariantFuture
)

// Version specifies how the bits in the GUID were generated. For instance, a
// version 4 GUID is randomly generated, and a version 5 is generated from the
// hash of an input string.
type Version uint8

var _ = (encoding.TextMarshaler)(GUID{})
var _ = (encoding.TextUnmarshaler)(&GUID{})

// GUID represents a GUID/UUID. It has the same structure as
// golang.org/x/sys/windows.GUID so that it can be used with functions expecting
// that type. It is defined as its own type so that stringification and
// marshaling can be supported. The representation matches that used by native
// Windows code.
type GUID windows.GUID

// NewV4 returns a new version 4 (pseudorandom) GUID, as defined by RFC 4122.
func NewV4() (GUID, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return GUID{}, err
	}

	b[6] = (b[6] & 0x0f) | 0x40 // Version 4 (randomly generated)
	b[8] = (b[8] & 0x3f) | 0x80 // RFC4122 variant

	return FromArray(b), nil
}

func fromArray(b [16]byte, order binary.ByteOrder) GUID {
	var g GUID
	g.Data1 = order.Uint32(b[0:4])
	g.Data2 = order.Uint16(b[4:6])
	g.Data3 = order.Uint16(b[6:8])
	copy(g.Data4[:], b[8:16])
	return g
}

func (g GUID) toArray(order binary.ByteOrder) [16]byte {
	b := [16]byte{}
	order.PutUint32(b[0:4], g.Data1)
	order.PutUint16(b[4:6], g.Data2)
	order.PutUint16(b[6:8], g.Data3)
	copy(b[8:16], g.Data4[:])
	return b
}

// FromArray constructs a GUID from a big-endian encoding array of 16 bytes.
func FromArray(b [16]byte) GUID {
	return fromArray(b, binary.BigEndian)
}

// ToArray returns an array of 16 bytes representing the GUID in big-endian
// encoding.
func (g GUID) ToArray() [16]byte {
	return g.toArray(binary.BigEndian)
}

// FromWindowsArray constructs a GUID from a Windows encoding array of bytes.
func FromWindowsArray(b [16]byte) GUID {
	return fromArray(b, binary.LittleEndian)
}

// ToWindowsArray returns an array of 16 bytes representing the GUID in Windows
// encoding.
func (g GUID) ToWindowsArray() [16]byte {
	return g.toArray(binary.LittleEndian)
}

func (g GUID) String() string {
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		g.Data1,
		g.Data2,
		g.Data3,
		g.Data4[:2],
		g.Data4[2:])
}

// FromString parses a string containing a GUID and returns the GUID. The only
// format currently supported is the `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`
// format.
func FromString(s string) (GUID, error) {
	if len(s) != 36 {
		return GUID{}, fmt.Errorf("invalid GUID %q", s)
	}
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return GUID{}, fmt.Errorf("invalid GUID %q", s)
	}

	var g GUID

	data1, err := strconv.ParseUint(s[0:8], 16, 32)
	if err != nil {
		return GUID{}, fmt.Errorf("invalid GUID %q", s)
	}
	g.Data1 = uint32(data1)

	data2, err := strconv.ParseUint(s[9:13], 16, 16)
	if err != nil {
		return GUID{}, fmt.Errorf("invalid GUID %q", s)
	}
	g.Data2 = uint16(data2)

	data3, err := strconv.ParseUint(s[14:18], 16, 16)
	if err != nil {
		return GUID{}, fmt.Errorf("invalid GUID %q", s)
	}
	g.Data3 = uint16(data3)

	for i, x := range []int{19, 21, 24, 26, 28, 30, 32, 34} {
		v, err := strconv.ParseUint(s[x:x+2], 16, 8)
		if err != nil {
			return GUID{}, fmt.Errorf("invalid GUID %q", s)
		}
		g.Data4[i] = uint8(v)
	}

	return g, nil
}

// Variant returns the GUID variant, as defined in RFC 4122.
func (g GUID) Variant() Variant {
	b := g.Data4[0]
	if b&0x80 == 0 {
		return VariantNCS
	} else if b&0xc0 == 0x80 {
		return VariantRFC4122
	} else if b&0xe0 == 0xc0 {
		return VariantMicrosoft
	} else if b&0xe0 == 0xe0 {
		return VariantFuture
	}
	return VariantUnknown
}

// Version returns the GUID version, as defined in RFC 4122.
func (g GUID) Version() Version {
	return Version((g.Data3 & 0xF000) >> 12)
}

// MarshalText returns the textual representation of the GUID.
func (g GUID) MarshalText() ([]byte, error) {
	return []byte(g.String()), nil
}

// UnmarshalText takes the textual representation of a GUID, and unmarhals it
// into this GUID.
func (g *GUID) UnmarshalText(text []byte) error {
	g2, err := FromString(string(text))
	if err != nil {
		return err
	}
	*g = g2
	return nil
}

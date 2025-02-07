// Copyright 2025 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

package rlp

import (
	"errors"
	"fmt"
	"io"
	"reflect"
)

// Fields mirror the RLP encoding of struct fields.
type Fields struct {
	Required []any
	Optional []any // equivalent to those tagged with `rlp:"optional"`
}

var _ interface {
	Encoder
	Decoder
} = (*Fields)(nil)

// EncodeRLP encodes the `f.Required` and `f.Optional` slices to `w`,
// concatenated as a single list, as if they were fields in a struct. The
// optional values are treated identically to those tagged with
// `rlp:"optional"`.
func (f *Fields) EncodeRLP(w io.Writer) error {
	includeOptional, err := f.optionalInclusionFlags()
	if err != nil {
		return err
	}

	b := NewEncoderBuffer(w)
	err = b.InList(func() error {
		for _, v := range f.Required {
			if err := Encode(b, v); err != nil {
				return err
			}
		}

		for i, v := range f.Optional {
			if !includeOptional[i] {
				return nil
			}
			if err := Encode(b, v); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return b.Flush()
}

var errUnsupportedOptionalFieldType = errors.New("unsupported optional field type")

// optionalInclusionFlags returns a slice of booleans, the same length as
// `f.Optional`, indicating whether or not the respective field MUST be written
// to a list. A field must be written if it or any later field value is non-nil;
// the returned slice is therefore monotonic non-increasing from true to false.
func (f *Fields) optionalInclusionFlags() ([]bool, error) {
	flags := make([]bool, len(f.Optional))
	var include bool
	for i := len(f.Optional) - 1; i >= 0; i-- {
		switch v := reflect.ValueOf(f.Optional[i]); v.Kind() {
		case reflect.Slice, reflect.Pointer:
			include = include || !v.IsNil()
		default:
			return nil, fmt.Errorf("%w: %T", errUnsupportedOptionalFieldType, f.Optional[i])
		}
		flags[i] = include
	}
	return flags, nil
}

// DecodeRLP implements the [Decoder] interface. All destination fields, be they
// required or optional, MUST be pointers and all optional fields MUST be
// provided in case they are present in the RLP being decoded.
//
// Typically, the arguments to this method mirror those passed to
// [Fields.EncodeRLP] except for being pointers. See the example.
func (f *Fields) DecodeRLP(s *Stream) error {
	return s.FromList(func() error {
		for _, v := range f.Required {
			if err := s.Decode(v); err != nil {
				return err
			}
		}

		for _, v := range f.Optional {
			if !s.MoreDataInList() {
				return nil
			}
			if err := s.Decode(v); err != nil {
				return err
			}
		}
		return nil
	})
}

// Nillable wraps `field` to mirror the behaviour of an `rlp:"nil"` tag; i.e. if
// a zero-sized RLP item is decoded into the returned Decoder then it is dropped
// and `*field` is set to nil, otherwise the RLP item is decoded directly into
// `field`. The return argument is intended for use with [Fields].
func Nillable[T any](field **T) Decoder {
	return &nillable[T]{field}
}

type nillable[T any] struct{ v **T }

func (n *nillable[T]) DecodeRLP(s *Stream) error {
	_, size, err := s.Kind()
	if err != nil {
		return err
	}
	if size > 0 {
		return s.Decode(n.v)
	}
	*n.v = nil
	_, err = s.Raw() // consume the item
	return err
}

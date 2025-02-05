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

// InList is a convenience wrapper, calling `fn` between calls to
// [EncoderBuffer.List] and [EncoderBuffer.ListEnd]. If `fn` returns an error,
// it is propagated directly.
func (b EncoderBuffer) InList(fn func() error) error {
	l := b.List()
	if err := fn(); err != nil {
		return err
	}
	b.ListEnd(l)
	return nil
}

// EncodeListToBuffer is equivalent to [Encode], writing the RLP encoding of
// each element to `b`, except that it wraps the writes inside a call to
// [EncoderBuffer.InList].
func EncodeListToBuffer[T any](b EncoderBuffer, vals []T) error {
	return b.InList(func() error {
		for _, v := range vals {
			if err := Encode(b, v); err != nil {
				return err
			}
		}
		return nil
	})
}

// FromList is a convenience wrapper, calling `fn` between calls to
// [Stream.List] and [Stream.ListEnd]. If `fn` returns an error, it is
// propagated directly.
func (s *Stream) FromList(fn func() error) error {
	if _, err := s.List(); err != nil {
		return err
	}
	if err := fn(); err != nil {
		return err
	}
	return s.ListEnd()
}

// DecodeList assumes that the next item in `s` is a list and decodes every item
// in said list to a `*T`.
//
// The returned slice is guaranteed to be non-nil, even if the list is empty.
// This is in keeping with other behaviour in this package and it is therefore
// the responsibility of callers to respect `rlp:"nil"` struct tags.
func DecodeList[T any](s *Stream) ([]*T, error) {
	vals := []*T{}
	err := s.FromList(func() error {
		for s.MoreDataInList() {
			var v T
			if err := s.Decode(&v); err != nil {
				return err
			}
			vals = append(vals, &v)
		}
		return nil
	})
	return vals, err
}

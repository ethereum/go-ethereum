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

import "errors"

// The spec models `Optional[T]` as `List[T, 1]`: 0 elements = absent,
// 1 element = present. We mirror that on the Go side by representing
// optional fields as length-0-or-1 slices and validating the length
// during a post-decode pass.

// errOptionalTooLong is returned when an optional list decoded with more
// than one element.
var errOptionalTooLong = errors.New("ssz: optional list has more than 1 element")

// checkOptional validates that an optional slice has length 0 or 1.
func checkOptional[T any](s []T) error {
	if len(s) > 1 {
		return errOptionalTooLong
	}
	return nil
}

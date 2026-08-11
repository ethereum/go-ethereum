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

package rpc

import "bytes"

// Helpers for finding the bounds of JSON values without parsing them, so a
// request is not walked by encoding/json once per layer. They require input
// encoding/json has already accepted, and hand out sub-slices of it which are
// still decoded afterwards. Do not use these on unchecked input.

func isJSONSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// skipJSONSpace returns the offset of the first byte at or after i that is not
// insignificant whitespace.
func skipJSONSpace(data []byte, i int) int {
	for i < len(data) && isJSONSpace(data[i]) {
		i++
	}
	return i
}

// scanJSONString returns the offset just past the string beginning at data[i],
// which must be its opening quote.
func scanJSONString(data []byte, i int) int {
	i++ // opening quote
	for i < len(data) {
		j := bytes.IndexByte(data[i:], '"')
		if j < 0 {
			return len(data)
		}
		// The quote ends the string unless an odd number of backslashes run up
		// to it, in which case it is escaped.
		k, n := i+j-1, 0
		for k >= i && data[k] == '\\' {
			n++
			k--
		}
		i += j + 1
		if n%2 == 0 {
			return i
		}
	}
	return len(data)
}

// scanJSONValue returns the offset just past the JSON value beginning at
// data[i], skipping any whitespace in front of it.
func scanJSONValue(data []byte, i int) int {
	i = skipJSONSpace(data, i)
	if i >= len(data) {
		return len(data)
	}
	switch data[i] {
	case '"':
		return scanJSONString(data, i)
	case '{', '[':
		depth := 0
		for i < len(data) {
			switch data[i] {
			case '"':
				i = scanJSONString(data, i)
				continue
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth == 0 {
					return i + 1
				}
			}
			i++
		}
		return len(data)
	default:
		// A number, true, false or null, ending at the next structural byte.
		for ; i < len(data); i++ {
			c := data[i]
			if c == ',' || c == '}' || c == ']' || isJSONSpace(c) {
				return i
			}
		}
		return len(data)
	}
}

// forEachJSONField calls fn with the key and raw value of every member of the
// JSON object in data. Nothing is called if data does not hold an object.
func forEachJSONField(data []byte, fn func(key, value []byte)) {
	i := skipJSONSpace(data, 0)
	if i >= len(data) || data[i] != '{' {
		return
	}
	i++
	for {
		i = skipJSONSpace(data, i)
		if i >= len(data) || data[i] == '}' {
			return
		}
		if data[i] == ',' {
			i++
			continue
		}
		if data[i] != '"' {
			return
		}
		keyStart := i
		keyEnd := scanJSONString(data, i)
		i = skipJSONSpace(data, keyEnd)
		if i >= len(data) || data[i] != ':' {
			return
		}
		valStart := skipJSONSpace(data, i+1)
		valEnd := scanJSONValue(data, valStart)
		i = valEnd
		if keyEnd-1 <= keyStart {
			return
		}
		// The key keeps any escapes it had, which is fine for the field names
		// this is used with.
		fn(data[keyStart+1:keyEnd-1], data[valStart:valEnd])
	}
}

// forEachJSONElement calls fn with the raw value of every element of the JSON
// array in data. Nothing is called if data does not hold an array.
func forEachJSONElement(data []byte, fn func(value []byte)) {
	i := skipJSONSpace(data, 0)
	if i >= len(data) || data[i] != '[' {
		return
	}
	i++
	for {
		i = skipJSONSpace(data, i)
		if i >= len(data) || data[i] == ']' {
			return
		}
		if data[i] == ',' {
			i++
			continue
		}
		start := i
		i = scanJSONValue(data, i)
		fn(data[start:i])
	}
}

// isJSONNull reports whether data holds the JSON null literal.
func isJSONNull(data []byte) bool {
	i := skipJSONSpace(data, 0)
	return bytes.Equal(bytes.TrimRight(data[i:], " \t\n\r"), []byte("null"))
}

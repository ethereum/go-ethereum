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

package engine

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// marshalHexBytesArray writes an array of hex-encoded byte slices to buf.
// A nil slice is written as "null" to match encoding/json semantics.
func marshalHexBytesArray(buf []byte, items []hexutil.Bytes) []byte {
	if items == nil {
		return append(buf, "null"...)
	}
	buf = append(buf, '[')
	for i, item := range items {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = writeHexBytes(buf, item)
	}
	buf = append(buf, ']')
	return buf
}

// writeHexBytes writes a hex-encoded byte slice as a JSON string ("0x...") to buf.
func writeHexBytes(buf []byte, data []byte) []byte {
	buf = append(buf, '"', '0', 'x')
	buf = slices.Grow(buf, len(data)*2+1)
	cur := len(buf)
	buf = buf[:cur+len(data)*2]
	hex.Encode(buf[cur:], data)
	buf = append(buf, '"')
	return buf
}

func decodeJSONObject(input []byte, fn func(key string, value json.RawMessage) error) error {
	dec := json.NewDecoder(bytes.NewReader(input))
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '{' {
		return fmt.Errorf("expected JSON object")
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := tok.(string)
		if !ok {
			return fmt.Errorf("expected JSON object key")
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return err
		}
		if err := fn(key, value); err != nil {
			return err
		}
	}
	tok, err = dec.Token()
	if err != nil {
		return err
	}
	delim, ok = tok.(json.Delim)
	if !ok || delim != '}' {
		return fmt.Errorf("expected end of JSON object")
	}
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing data")
		}
		return err
	}
	return nil
}

func decodeJSONArray(input []byte, fn func(value json.RawMessage) error) error {
	dec := json.NewDecoder(bytes.NewReader(input))
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '[' {
		return fmt.Errorf("expected JSON array")
	}
	for dec.More() {
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return err
		}
		if err := fn(value); err != nil {
			return err
		}
	}
	tok, err = dec.Token()
	if err != nil {
		return err
	}
	delim, ok = tok.(json.Delim)
	if !ok || delim != ']' {
		return fmt.Errorf("expected end of JSON array")
	}
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing data")
		}
		return err
	}
	return nil
}

func isJSONNull(input []byte) bool {
	return bytes.Equal(bytes.TrimSpace(input), []byte("null"))
}

func unmarshalHexBytesArray(input []byte) ([]hexutil.Bytes, error) {
	if isJSONNull(input) {
		return nil, nil
	}
	items := make([]hexutil.Bytes, 0)
	if err := decodeJSONArray(input, func(value json.RawMessage) error {
		var item hexutil.Bytes
		if err := item.UnmarshalJSON(value); err != nil {
			return err
		}
		items = append(items, item)
		return nil
	}); err != nil {
		return nil, err
	}
	return items, nil
}

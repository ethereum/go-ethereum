// Copyright 2018 The go-ethereum Authors
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

package mru

import (
	"encoding/binary"
	"time"
)

// TimestampProvider sets the time source of the mru package
var TimestampProvider timestampProvider = NewDefaultTimestampProvider()

// Encodes a point in time as a Unix epoch
type Timestamp struct {
	Time uint64 // Unix epoch timestamp, in seconds
}

// 8 bytes uint64 Time
const timestampLength = 8

// timestampProvider interface describes a source of timestamp information
type timestampProvider interface {
	Now() Timestamp // returns the current timestamp information
}

// binaryGet populates the timestamp structure from the given byte slice
func (t *Timestamp) binaryGet(data []byte) error {
	if len(data) != timestampLength {
		return NewError(ErrCorruptData, "timestamp data has the wrong size")
	}
	t.Time = binary.LittleEndian.Uint64(data[:8])
	return nil
}

// binaryPut Serializes a Timestamp to a byte slice
func (t *Timestamp) binaryPut(data []byte) error {
	if len(data) != timestampLength {
		return NewError(ErrCorruptData, "timestamp data has the wrong size")
	}
	binary.LittleEndian.PutUint64(data, t.Time)
	return nil
}

type DefaultTimestampProvider struct {
}

// NewDefaultTimestampProvider creates a system clock based timestamp provider
func NewDefaultTimestampProvider() *DefaultTimestampProvider {
	return &DefaultTimestampProvider{}
}

// Now returns the current time according to this provider
func (dtp *DefaultTimestampProvider) Now() Timestamp {
	return Timestamp{
		Time: uint64(time.Now().Unix()),
	}
}

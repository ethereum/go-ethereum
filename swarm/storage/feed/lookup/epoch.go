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

package lookup

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Epoch represents a time slot at a particular frequency level
type Epoch struct {
	Time  uint64 `json:"time"`  // Time stores the time at which the update or lookup takes place
	Level uint8  `json:"level"` // Level indicates the frequency level as the exponent of a power of 2
}

// EpochID is a unique identifier for an Epoch, based on its level and base time.
type EpochID [8]byte

// EpochLength stores the serialized binary length of an Epoch
const EpochLength = 8

// MaxTime contains the highest possible time value an Epoch can handle
const MaxTime uint64 = (1 << 56) - 1

// Base returns the base time of the Epoch
func (e *Epoch) Base() uint64 {
	return getBaseTime(e.Time, e.Level)
}

// ID Returns the unique identifier of this epoch
func (e *Epoch) ID() EpochID {
	base := e.Base()
	var id EpochID
	binary.LittleEndian.PutUint64(id[:], base)
	id[7] = e.Level
	return id
}

// MarshalBinary implements the encoding.BinaryMarshaller interface
func (e *Epoch) MarshalBinary() (data []byte, err error) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b[:], e.Time)
	b[7] = e.Level
	return b, nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaller interface
func (e *Epoch) UnmarshalBinary(data []byte) error {
	if len(data) != EpochLength {
		return errors.New("Invalid data unmarshalling Epoch")
	}
	b := make([]byte, 8)
	copy(b, data)
	e.Level = b[7]
	b[7] = 0
	e.Time = binary.LittleEndian.Uint64(b)
	return nil
}

// After returns true if this epoch occurs later or exactly at the other epoch.
func (e *Epoch) After(epoch Epoch) bool {
	if e.Time == epoch.Time {
		return e.Level < epoch.Level
	}
	return e.Time >= epoch.Time
}

// Equals compares two epochs and returns true if they refer to the same time period.
func (e *Epoch) Equals(epoch Epoch) bool {
	return e.Level == epoch.Level && e.Base() == epoch.Base()
}

// String implements the Stringer interface.
func (e *Epoch) String() string {
	return fmt.Sprintf("Epoch{Time:%d, Level:%d}", e.Time, e.Level)
}

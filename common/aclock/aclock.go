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

// Package aclock contains an adjustable clock implementation. The time can be
// adjusted by adding an offset.
package aclock

import (
	"errors"
	"time"
)

var offset time.Duration

// AddOffset adds offset d to the current offset. d cannot be negative, and an
// error will be returned if the resulting offset overflows time.Duration.
func AddOffset(d time.Duration) (time.Duration, error) {
	if d < 0 {
		return 0, errors.New("aclock: duration for offset cannot be negative")
	}
	if offset+d < offset {
		return 0, errors.New("aclock: offset overflow")
	}
	offset += d
	return offset, nil
}

// Now returns the current time with the offset applied.
func Now() time.Time {
	return time.Now().Add(offset)
}

// NowWithOffset returns the current time with the offset applied and the offset
// itself.
func NowWithOffset() (time.Time, time.Duration) {
	return time.Now().Add(offset), offset
}

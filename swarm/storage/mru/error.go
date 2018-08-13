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
	"fmt"
)

const (
	ErrInit = iota
	ErrNotFound
	ErrIO
	ErrUnauthorized
	ErrInvalidValue
	ErrDataOverflow
	ErrNothingToReturn
	ErrCorruptData
	ErrInvalidSignature
	ErrNotSynced
	ErrPeriodDepth
	ErrCnt
)

// Error is a the typed error object used for Mutable Resources
type Error struct {
	code int
	err  string
}

// Error implements the error interface
func (e *Error) Error() string {
	return e.err
}

// Code returns the error code
// Error codes are enumerated in the error.go file within the mru package
func (e *Error) Code() int {
	return e.code
}

// NewError creates a new Mutable Resource Error object with the specified code and custom error message
func NewError(code int, s string) error {
	if code < 0 || code >= ErrCnt {
		panic("no such error code!")
	}
	r := &Error{
		err: s,
	}
	switch code {
	case ErrNotFound, ErrIO, ErrUnauthorized, ErrInvalidValue, ErrDataOverflow, ErrNothingToReturn, ErrInvalidSignature, ErrNotSynced, ErrPeriodDepth, ErrCorruptData:
		r.code = code
	}
	return r
}

// NewErrorf is a convenience version of NewError that incorporates printf-style formatting
func NewErrorf(code int, format string, args ...interface{}) error {
	return NewError(code, fmt.Sprintf(format, args...))
}

// Copyright 2022 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"strings"
)

type nameFilter struct {
	fulls map[string]bool // path/to/contract.sol:Type
	files map[string]bool // path/to/contract.sol:*
	types map[string]bool // *:Type
}

func newNameFilter(patterns ...string) (*nameFilter, error) {
	f := &nameFilter{
		fulls: make(map[string]bool),
		files: make(map[string]bool),
		types: make(map[string]bool),
	}
	for _, pattern := range patterns {
		if err := f.add(pattern); err != nil {
			return nil, err
		}
	}
	return f, nil
}

func (f *nameFilter) add(pattern string) error {
	ft := strings.Split(pattern, ":")
	if len(ft) != 2 {
		// filenames and types must not include ':' symbol
		return fmt.Errorf("invalid pattern: %s", pattern)
	}

	file, typ := ft[0], ft[1]
	if file == "*" {
		f.types[typ] = true
		return nil
	} else if typ == "*" {
		f.files[file] = true
		return nil
	}
	f.fulls[pattern] = true
	return nil
}

func (f *nameFilter) Matches(name string) bool {
	ft := strings.Split(name, ":")
	if len(ft) != 2 {
		// If contract names are always of the fully-qualified form
		// <filePath>:<type>, then this case will never happen.
		return false
	}

	file, typ := ft[0], ft[1]
	// full paths > file paths > types
	return f.fulls[name] || f.files[file] || f.types[typ]
}

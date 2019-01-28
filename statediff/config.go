// Copyright 2015 The go-ethereum Authors
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

// Contains a batch of utility type declarations used by the tests. As the node
// operates on unique types, a lot of them are needed to check various features.

package statediff

import "fmt"

type Config struct {
	Mode StateDiffMode // Mode for storing diffs
	Path string        // Path for storing diffs
}

type StateDiffMode int

const (
	CSV StateDiffMode = iota
	IPLD
	LDB
	SQL
)

func (mode StateDiffMode) IsValid() bool {
	return mode >= IPLD && mode <= SQL
}

// String implements the stringer interface.
func (mode StateDiffMode) String() string {
	switch mode {
	case CSV:
		return "csv"
	case IPLD:
		return "ipfs"
	case LDB:
		return "ldb"
	case SQL:
		return "sql"
	default:
		return "unknown"
	}
}

func NewMode(mode string) (StateDiffMode, error) {
	stateDiffMode := StateDiffMode(0)
	err := stateDiffMode.UnmarshalText([]byte(mode))
	return stateDiffMode, err
}

func (mode StateDiffMode) MarshalText() ([]byte, error) {
	switch mode {
	case CSV:
		return []byte("ipfs"), nil
	case IPLD:
		return []byte("ipfs"), nil
	case LDB:
		return []byte("ldb"), nil
	case SQL:
		return []byte("sql"), nil
	default:
		return nil, fmt.Errorf("unknown state diff storage mode %d", mode)
	}
}

func (mode *StateDiffMode) UnmarshalText(text []byte) error {
	switch string(text) {
	case "csv":
		*mode = CSV
	case "ipfs":
		*mode = IPLD
	case "ldb":
		*mode = LDB
	case "sql":
		*mode = SQL
	default:
		return fmt.Errorf(`unknown state diff storage mode %q, want "ipfs", "ldb" or "sql"`, text)
	}
	return nil
}

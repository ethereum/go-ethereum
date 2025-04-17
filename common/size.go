// Copyright 2014 The go-ethereum Authors
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

package common

import (
	"fmt"
)

// StorageSize is a wrapper around a float value that supports user friendly
// formatting.
type StorageSize float64

// String implements the stringer interface.
func (s StorageSize) String() string {
	if s > Terabyte {
		return fmt.Sprintf("%.2f TiB", s/Terabyte)
	} else if s > Gigabyte {
		return fmt.Sprintf("%.2f GiB", s/Gigabyte)
	} else if s > Megabyte {
		return fmt.Sprintf("%.2f MiB", s/Megabyte)
	} else if s > Kilobyte {
		return fmt.Sprintf("%.2f KiB", s/Kilobyte)
	} else {
		return fmt.Sprintf("%.2f B", s)
	}
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (s StorageSize) TerminalString() string {
	if s > Terabyte {
		return fmt.Sprintf("%.2fTiB", s/Terabyte)
	} else if s > Gigabyte {
		return fmt.Sprintf("%.2fGiB", s/Gigabyte)
	} else if s > Megabyte {
		return fmt.Sprintf("%.2fMiB", s/Megabyte)
	} else if s > Kilobyte {
		return fmt.Sprintf("%.2fKiB", s/Kilobyte)
	} else {
		return fmt.Sprintf("%.2fB", s)
	}
}

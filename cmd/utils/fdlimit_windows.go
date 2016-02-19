// Copyright 2016 The go-ethereum Authors
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

package utils

// #include <stdio.h>
import "C"

// raiseFdLimit tries to maximize the file descriptor allowance of this process
// to the maximum hard-limit allowed by the OS.
func raiseFdLimit(max uint64) error {
	if max > 2048 {
		max = 2048 // windows limits
	}
	C._setmaxstdio(C.int(max))
	return nil
}

// getFdLimit retrieves the number of file descriptors allowed to be opened by this
// process.
func getFdLimit() (int, error) {
	return int(C._getmaxstdio()), nil
}

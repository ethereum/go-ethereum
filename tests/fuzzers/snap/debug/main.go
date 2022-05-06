// Copyright 2021 The go-ethereum Authors
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

package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/tests/fuzzers/snap"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: debug <file>\n")
		os.Exit(1)
	}
	crasher := os.Args[1]
	data, err := ioutil.ReadFile(crasher)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading crasher %v: %v", crasher, err)
		os.Exit(1)
	}
	snap.FuzzTrieNodes(data)
}

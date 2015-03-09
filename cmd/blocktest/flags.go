/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Gustav Simonsson <gustav.simonsson@gmail.com>
 */
package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	TestFile string
)

func Init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s <testfile>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	TestFile = flag.Arg(0)
}

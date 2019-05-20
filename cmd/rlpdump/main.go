// Copyright 2015 The go-ethereum Authors
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

// rlpdump is a pretty-printer for RLP data.
package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/rlp"
)

var (
	hexMode = flag.String("hex", "", "dump given hex data")
	noASCII = flag.Bool("noascii", false, "don't print ASCII strings readably")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "[-noascii] [-hex <data>] [filename]")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, `
Dumps RLP data from the given file in readable form.
If the filename is omitted, data is read from stdin.`)
	}
}

func main() {
	flag.Parse()

	var r io.Reader
	switch {
	case *hexMode != "":
		data, err := hex.DecodeString(*hexMode)
		if err != nil {
			die(err)
		}
		r = bytes.NewReader(data)

	case flag.NArg() == 0:
		r = os.Stdin

	case flag.NArg() == 1:
		fd, err := os.Open(flag.Arg(0))
		if err != nil {
			die(err)
		}
		defer fd.Close()
		r = fd

	default:
		fmt.Fprintln(os.Stderr, "Error: too many arguments")
		flag.Usage()
		os.Exit(2)
	}

	s := rlp.NewStream(r, 0)
	for {
		if err := dump(s, 0); err != nil {
			if err != io.EOF {
				die(err)
			}
			break
		}
		fmt.Println()
	}
}

func dump(s *rlp.Stream, depth int) error {
	kind, size, err := s.Kind()
	if err != nil {
		return err
	}
	switch kind {
	case rlp.Byte, rlp.String:
		str, err := s.Bytes()
		if err != nil {
			return err
		}
		if len(str) == 0 || !*noASCII && isASCII(str) {
			fmt.Printf("%s%q", ws(depth), str)
		} else {
			fmt.Printf("%s%x", ws(depth), str)
		}
	case rlp.List:
		s.List()
		defer s.ListEnd()
		if size == 0 {
			fmt.Print(ws(depth) + "[]")
		} else {
			fmt.Println(ws(depth) + "[")
			for i := 0; ; i++ {
				if i > 0 {
					fmt.Print(",\n")
				}
				if err := dump(s, depth+1); err == rlp.EOL {
					break
				} else if err != nil {
					return err
				}
			}
			fmt.Print(ws(depth) + "]")
		}
	}
	return nil
}

func isASCII(b []byte) bool {
	for _, c := range b {
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}

func ws(n int) string {
	return strings.Repeat("  ", n)
}

func die(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

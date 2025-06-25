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
	"bufio"
	"bytes"
	"container/list"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	hexMode     = flag.String("hex", "", "dump given hex data")
	reverseMode = flag.Bool("reverse", false, "convert ASCII to rlp")
	noASCII     = flag.Bool("noascii", false, "don't print ASCII strings readably")
	single      = flag.Bool("single", false, "print only the first element, discard the rest")
	showpos     = flag.Bool("pos", false, "display element byte posititions")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "[-noascii] [-hex <data>][-reverse] [filename]")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, `
Dumps RLP data from the given file in readable form.
If the filename is omitted, data is read from stdin.`)
	}
}

func main() {
	flag.Parse()

	var r *inStream
	switch {
	case *hexMode != "":
		data, err := hex.DecodeString(strings.TrimPrefix(*hexMode, "0x"))
		if err != nil {
			die(err)
		}
		r = newInStream(bytes.NewReader(data), int64(len(data)))

	case flag.NArg() == 0:
		r = newInStream(bufio.NewReader(os.Stdin), 0)

	case flag.NArg() == 1:
		fd, err := os.Open(flag.Arg(0))
		if err != nil {
			die(err)
		}
		defer fd.Close()
		var size int64
		finfo, err := fd.Stat()
		if err == nil {
			size = finfo.Size()
		}
		r = newInStream(bufio.NewReader(fd), size)

	default:
		fmt.Fprintln(os.Stderr, "Error: too many arguments")
		flag.Usage()
		os.Exit(2)
	}

	out := os.Stdout
	if *reverseMode {
		data, err := textToRlp(r)
		if err != nil {
			die(err)
		}
		fmt.Printf("%#x\n", data)
		return
	} else {
		err := rlpToText(r, out)
		if err != nil {
			die(err)
		}
	}
}

func rlpToText(in *inStream, out io.Writer) error {
	stream := rlp.NewStream(in, 0)
	for {
		if err := dump(in, stream, 0, out); err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		fmt.Fprintln(out)
		if *single {
			break
		}
	}
	return nil
}

func dump(in *inStream, s *rlp.Stream, depth int, out io.Writer) error {
	if *showpos {
		fmt.Fprintf(out, "%s: ", in.posLabel())
	}
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
			fmt.Fprintf(out, "%s%q", ws(depth), str)
		} else {
			fmt.Fprintf(out, "%s%x", ws(depth), str)
		}
	case rlp.List:
		s.List()
		defer s.ListEnd()
		if size == 0 {
			fmt.Fprint(out, ws(depth)+"[]")
		} else {
			fmt.Fprintln(out, ws(depth)+"[")
			for i := 0; ; i++ {
				if i > 0 {
					fmt.Fprint(out, ",\n")
				}
				if err := dump(in, s, depth+1, out); err == rlp.EOL {
					break
				} else if err != nil {
					return err
				}
			}
			fmt.Fprint(out, ws(depth)+"]")
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

// textToRlp converts text into RLP (best effort).
func textToRlp(r io.Reader) ([]byte, error) {
	// We're expecting the input to be well-formed, meaning that
	// - each element is on a separate line
	// - each line is either an (element OR a list start/end) + comma
	// - an element is either hex-encoded bytes OR a quoted string
	var (
		scanner = bufio.NewScanner(r)
		obj     []interface{}
		stack   = list.New()
	)
	for scanner.Scan() {
		t := strings.TrimSpace(scanner.Text())
		if len(t) == 0 {
			continue
		}
		switch t {
		case "[": // list start
			stack.PushFront(obj)
			obj = make([]interface{}, 0)
		case "]", "],": // list end
			parent := stack.Remove(stack.Front()).([]interface{})
			obj = append(parent, obj)
		case "[],": // empty list
			obj = append(obj, make([]interface{}, 0))
		default: // element
			data := []byte(t)[:len(t)-1] // cut off comma
			if data[0] == '"' {          // ascii string
				data = []byte(t)[1 : len(data)-1]
			} else { // hex data
				data = common.FromHex(string(data))
			}
			obj = append(obj, data)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	data, err := rlp.EncodeToBytes(obj[0])
	return data, err
}

type inStream struct {
	br      rlp.ByteReader
	pos     int
	columns int
}

func newInStream(br rlp.ByteReader, totalSize int64) *inStream {
	col := int(math.Ceil(math.Log10(float64(totalSize))))
	return &inStream{br: br, columns: col}
}

func (rc *inStream) Read(b []byte) (n int, err error) {
	n, err = rc.br.Read(b)
	rc.pos += n
	return n, err
}

func (rc *inStream) ReadByte() (byte, error) {
	b, err := rc.br.ReadByte()
	if err == nil {
		rc.pos++
	}
	return b, err
}

func (rc *inStream) posLabel() string {
	l := strconv.FormatInt(int64(rc.pos), 10)
	if len(l) < rc.columns {
		l = strings.Repeat(" ", rc.columns-len(l)) + l
	}
	return l
}

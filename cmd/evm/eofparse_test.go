// Copyright 2024 The go-ethereum Authors
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
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

func FuzzEofParsing(f *testing.F) {
	// Seed with corpus from execution-spec-tests
	for i := 0; ; i++ {
		fname := fmt.Sprintf("testdata/eof/eof_corpus_%d.txt", i)
		corpus, err := os.Open(fname)
		if err != nil {
			break
		}
		f.Logf("Reading seed data from %v", fname)
		scanner := bufio.NewScanner(corpus)
		scanner.Buffer(make([]byte, 1024), 10*1024*1024)
		for scanner.Scan() {
			s := scanner.Text()
			if len(s) >= 2 && strings.HasPrefix(s, "0x") {
				s = s[2:]
			}
			b, err := hex.DecodeString(s)
			if err != nil {
				panic(err) // rotten corpus
			}
			f.Add(b)
		}
		corpus.Close()
		if err := scanner.Err(); err != nil {
			panic(err) // rotten corpus
		}
	}
	// And do the fuzzing
	f.Fuzz(func(t *testing.T, data []byte) {
		var (
			jt = vm.NewEOFInstructionSetForTesting()
			c  vm.Container
		)
		cpy := common.CopyBytes(data)
		if err := c.UnmarshalBinary(data, true); err == nil {
			c.ValidateCode(&jt, true)
			if have := c.MarshalBinary(); !bytes.Equal(have, data) {
				t.Fatal("Unmarshal-> Marshal failure!")
			}
		}
		if err := c.UnmarshalBinary(data, false); err == nil {
			c.ValidateCode(&jt, false)
			if have := c.MarshalBinary(); !bytes.Equal(have, data) {
				t.Fatal("Unmarshal-> Marshal failure!")
			}
		}
		if !bytes.Equal(cpy, data) {
			panic("data modified during unmarshalling")
		}
	})
}

func TestEofParseInitcode(t *testing.T) {
	testEofParse(t, true, "testdata/eof/results.initcode.txt")
}

func TestEofParseRegular(t *testing.T) {
	testEofParse(t, false, "testdata/eof/results.regular.txt")
}

func testEofParse(t *testing.T, isInitCode bool, wantFile string) {
	var wantFn func() string
	var wantLoc = 0
	{ // Configure the want-reader
		wants, err := os.Open(wantFile)
		if err != nil {
			t.Fatal(err)
		}
		scanner := bufio.NewScanner(wants)
		scanner.Buffer(make([]byte, 1024), 10*1024*1024)
		wantFn = func() string {
			if scanner.Scan() {
				wantLoc++
				return scanner.Text()
			}
			return "end of file reached"
		}
	}

	for i := 0; ; i++ {
		fname := fmt.Sprintf("testdata/eof/eof_corpus_%d.txt", i)
		corpus, err := os.Open(fname)
		if err != nil {
			break
		}
		t.Logf("# Reading seed data from %v", fname)
		scanner := bufio.NewScanner(corpus)
		scanner.Buffer(make([]byte, 1024), 10*1024*1024)
		line := 1
		for scanner.Scan() {
			s := scanner.Text()
			if len(s) >= 2 && strings.HasPrefix(s, "0x") {
				s = s[2:]
			}
			b, err := hex.DecodeString(s)
			if err != nil {
				panic(err) // rotten corpus
			}
			have := "OK"
			if _, err := parse(b, isInitCode); err != nil {
				have = fmt.Sprintf("ERR: %v", err)
			}
			if false { // Change this to generate the want-output
				fmt.Printf("%v\n", have)
			} else {
				want := wantFn()
				if have != want {
					if len(want) > 100 {
						want = want[:100]
					}
					if len(b) > 100 {
						b = b[:100]
					}
					t.Errorf("%v:%d\n%v\ninput %x\nisInit: %v\nhave: %q\nwant: %q\n",
						fname, line, fmt.Sprintf("%v:%d", wantFile, wantLoc), b, isInitCode, have, want)
				}
			}
			line++
		}
		corpus.Close()
	}
}

func BenchmarkEofParse(b *testing.B) {
	corpus, err := os.Open("testdata/eof/eof_benches.txt")
	if err != nil {
		b.Fatal(err)
	}
	defer corpus.Close()
	scanner := bufio.NewScanner(corpus)
	scanner.Buffer(make([]byte, 1024), 10*1024*1024)
	line := 1
	for scanner.Scan() {
		s := scanner.Text()
		if len(s) >= 2 && strings.HasPrefix(s, "0x") {
			s = s[2:]
		}
		data, err := hex.DecodeString(s)
		if err != nil {
			b.Fatal(err) // rotten corpus
		}
		b.Run(fmt.Sprintf("test-%d", line), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				_, _ = parse(data, false)
			}
		})
		line++
	}
}

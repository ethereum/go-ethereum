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
		fname := fmt.Sprintf("testdata/eof_corpus_%d.txt", i)
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
			jt = vm.NewPragueEOFInstructionSetForTesting()
			c  vm.Container
		)
		cpy := common.CopyBytes(data)
		if err := c.UnmarshalBinary(data, true); err == nil {
			c.ValidateCode(&jt, true)
		}
		if err := c.UnmarshalBinary(data, false); err == nil {
			c.ValidateCode(&jt, false)
		}
		if !bytes.Equal(cpy, data) {
			panic("data modified during unmarshalling")
		}
	})
}

func TestEofParseInitcode(t *testing.T) {
	testEofParse(t, true, "testdata/results.initcode.txt")
}

func TestEofParseRegular(t *testing.T) {
	testEofParse(t, false, "testdata/results.regular.txt")
}

func testEofParse(t *testing.T, isInitCode bool, wantFile string) {
	var wantFn func() string

	{ // Configure the want-reader
		wants, err := os.Open(wantFile)
		if err != nil {
			t.Fatal(err)
		}
		scanner := bufio.NewScanner(wants)
		scanner.Buffer(make([]byte, 1024), 10*1024*1024)
		wantFn = func() string {
			if scanner.Scan() {
				return scanner.Text()
			}
			return "end of file reached"
		}
	}

	for i := 0; ; i++ {
		fname := fmt.Sprintf("testdata/eof_corpus_%d.txt", i)
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
			have := parse(b, isInitCode)
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
					t.Fatalf("%v:%d\ninput %x\nisInit: %v\nhave: %q\nwant: %q\n",
						fname, line, b, isInitCode, have, want)
				}
			}
			line++

		}
		corpus.Close()
	}
}

func parse(data []byte, isInitCode bool) string {
	var (
		jt  = vm.NewPragueEOFInstructionSetForTesting()
		c   vm.Container
		err = c.UnmarshalBinary(data, isInitCode)
	)
	if err == nil {
		if err = c.ValidateCode(&jt, isInitCode); err == nil {
			return "OK"
		}
		return fmt.Sprintf("ERR: %v", err)
	}
	return fmt.Sprintf("ERR: %v", err)
}

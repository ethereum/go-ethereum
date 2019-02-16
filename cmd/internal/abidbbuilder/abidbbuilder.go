// Copyright 2019 The go-ethereum Authors
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
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

var (
	inDir   = flag.String("i", "", "input directory to read")
	outFile = flag.String("o", "", "file to write to (overwrites if exists)")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "-i directory -o outputfile")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, `
This is a little helper-utility to collect the data from 
https://github.com/ethereum-lists/4bytes and massage it into a 
clef-digestable format. 

It parses the signatures from the given directory, and writes
them to the given outputfile as a json struct.

Afterwards, you can do 

   [cmd/clef]$ go-bindata resources

To generatee the bindata.go asset file.
`)
	}
}

func main() {
	flag.Parse()
	in := *inDir
	out := *outFile
	if in == "" {
		fmt.Fprintf(os.Stderr, "input directory not given\n")
		os.Exit(1)
	}
	if out == "" {
		fmt.Fprintf(os.Stderr, "output file not given\n")
		os.Exit(1)
	}
	data, err := readFiles(in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading data: %v\n", err)
		os.Exit(1)
	}
	err = dumpData(data, out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing data: %v\n", err)
		os.Exit(1)
	}
}

func dumpData(db map[string]string, outfile string) error {
	data, err := json.Marshal(db)
	if err != nil {
		return err
	}
	fmt.Printf("data size %d kB\n", len(data)/1000)
	return ioutil.WriteFile(outfile, data, 0644)

}
func readFiles(dir string) (map[string]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		log.Fatal(err)
	}
	files, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	db := make(map[string]string)
	for _, file := range files {
		// Only bother with signature files
		sig, err := hex.DecodeString(file.Name())
		if err != nil {
			continue
		}
		if len(sig) != 4 {
			fmt.Printf("Invalid sig, wrong length: %x", sig)
		}
		dat, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", dir, file.Name()))
		if err != nil {
			fmt.Printf("err reading file: %v\n", err)
			continue
		}
		selectors := strings.Split(string(dat), ";")
		if len(selectors) > 1 {
			fmt.Printf("sig `%s`\n", sig)
			for _, selector := range selectors {
				fmt.Printf(" - %v\n", selector)
			}
			fmt.Println(" -- ignoring this signature\n")
			continue
		}
		selector := strings.TrimSpace(selectors[0])
		// We do a basic sanity check here, not fully verifying the correctness of
		// arguments, e.g the parameter types. We assume that the 4byte db comes
		// from a somewhat trusted source
		want := crypto.Keccak256([]byte(selector))[:4]
		if !bytes.Equal(sig, want) {
			fmt.Printf("Erroneous selector: %s, have %x want %x", selector, sig, want)
			continue
		}
		db[fmt.Sprintf("%x", sig)] = selector
	}
	return db, nil
}

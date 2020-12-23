// Copyright 2017 The go-ethereum Authors
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

package testutil

import (
	"bytes"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"testing"
)

// TempFileWithContent is a helper function that creates a temp file that contains the following string content then closes the file handle
// it returns the complete file path
func TempFileWithContent(t *testing.T, content string) string {
	tempFile, err := ioutil.TempFile("", "swarm-temp-file")
	if err != nil {
		t.Fatal(err)
	}

	_, err = io.Copy(tempFile, strings.NewReader(content))
	if err != nil {
		os.RemoveAll(tempFile.Name())
		t.Fatal(err)
	}
	if err = tempFile.Close(); err != nil {
		t.Fatal(err)
	}
	return tempFile.Name()
}

// RandomBytes returns pseudo-random deterministic result
// because test fails must be reproducible
func RandomBytes(seed, length int) []byte {
	b := make([]byte, length)
	reader := rand.New(rand.NewSource(int64(seed)))
	for n := 0; n < length; {
		read, err := reader.Read(b[n:])
		if err != nil {
			panic(err)
		}
		n += read
	}
	return b
}

func RandomReader(seed, length int) *bytes.Reader {
	return bytes.NewReader(RandomBytes(seed, length))
}

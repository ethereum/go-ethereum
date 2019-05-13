// Copyright 2016 The go-ethereum Authors
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

package discv5

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func getnacl() (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		_, err := exec.LookPath("sel_ldr_x86_64")
		return "amd64p32", err
	case "i386":
		_, err := exec.LookPath("sel_ldr_i386")
		return "i386", err
	default:
		return "", errors.New("nacl is not supported on " + runtime.GOARCH)
	}
}

// runWithPlaygroundTime executes the caller
// in the NaCl sandbox with faketime enabled.
//
// This function must be called from a Test* function
// and the caller must skip the actual test when isHost is true.
func runWithPlaygroundTime(t *testing.T) (isHost bool) {
	if runtime.GOOS == "nacl" {
		return false
	}

	// Get the caller.
	callerPC, _, _, ok := runtime.Caller(1)
	if !ok {
		panic("can't get caller")
	}
	callerFunc := runtime.FuncForPC(callerPC)
	if callerFunc == nil {
		panic("can't get caller")
	}
	callerName := callerFunc.Name()[strings.LastIndexByte(callerFunc.Name(), '.')+1:]
	if !strings.HasPrefix(callerName, "Test") {
		panic("must be called from witin a Test* function")
	}
	testPattern := "^" + callerName + "$"

	// Unfortunately runtime.faketime (playground time mode) only works on NaCl. The NaCl
	// SDK must be installed and linked into PATH for this to work.
	arch, err := getnacl()
	if err != nil {
		t.Skip(err)
	}

	// Compile and run the calling test using NaCl.
	// The extra tag ensures that the TestMain function in sim_main_test.go is used.
	cmd := exec.Command("go", "test", "-v", "-tags", "faketime_simulation", "-timeout", "100h", "-run", testPattern, ".")
	cmd.Env = append([]string{"GOOS=nacl", "GOARCH=" + arch}, os.Environ()...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	go skipPlaygroundOutputHeaders(os.Stdout, stdout)
	go skipPlaygroundOutputHeaders(os.Stderr, stderr)
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}

	// Ensure that the test function doesn't run in the (non-NaCl) host process.
	return true
}

func skipPlaygroundOutputHeaders(out io.Writer, in io.Reader) {
	// Additional output can be printed without the headers
	// before the NaCl binary starts running (e.g. compiler error messages).
	bufin := bufio.NewReader(in)
	output, err := bufin.ReadBytes(0)
	output = bytes.TrimSuffix(output, []byte{0})
	if len(output) > 0 {
		out.Write(output)
	}
	if err != nil {
		return
	}
	bufin.UnreadByte()

	// Playback header: 0 0 P B <8-byte time> <4-byte data length>
	head := make([]byte, 4+8+4)
	for {
		if _, err := io.ReadFull(bufin, head); err != nil {
			if err != io.EOF {
				fmt.Fprintln(out, "read error:", err)
			}
			return
		}
		if !bytes.HasPrefix(head, []byte{0x00, 0x00, 'P', 'B'}) {
			fmt.Fprintf(out, "expected playback header, got %q\n", head)
			io.Copy(out, bufin)
			return
		}
		// Copy data until next header.
		size := binary.BigEndian.Uint32(head[12:])
		io.CopyN(out, bufin, int64(size))
	}
}

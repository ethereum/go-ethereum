// Copyright 2015 The go-ethereum Authors
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

package rpc

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Tests that continuously binding and closing an IPC socket doesn't fail.
func TestIPCBindTrashing(t *testing.T) {
	// Create an OS dependent IPC path to trash
	var endpoint = ""

	switch runtime.GOOS {
	case "windows":
		endpoint = `\\.\pipe\socket.ipc`

	default:
		path, _ := ioutil.TempDir("", "")
		defer os.RemoveAll(path)
		endpoint = filepath.Join(path, "socket.ipc")
	}
	// Bind and close the socket a number of times
	for i := 0; i < 100; i++ {
		listener, err := CreateIPCListener(endpoint)
		if err != nil {
			t.Fatalf("failed to create IPC socket: %v", err)
		}
		conn, err := NewIPCClient(endpoint)
		if err != nil {
			t.Fatalf("failed to connect IPC socket: %v", err)
		}
		conn.Close()
		listener.Close()
	}
}

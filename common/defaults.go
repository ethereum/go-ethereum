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

package common

import (
	"path/filepath"
	"runtime"
	"os"
)

const (
	DefaultIPCSocket    = "ipc.sock"          // Default (relative) name of the IPC RPC unix socket
	DefaultIPCNamedPipe = `\\.\pipe\ethereum` // Default name of the IPC RPC named pipe
	DefaultHTTPHost     = "localhost"         // Default host interface for the HTTP RPC server
	DefaultHTTPPort     = 8545                // Default TCP port for the HTTP RPC server
	DefaultWSHost       = "localhost"         // Default host interface for the websocket RPC server
	DefaultWSPort       = 8546                // Default TCP port for the websocket RPC server
)

// DefaultDataDir is the default data directory to use for the databases and other
// persistence requirements.
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := HomeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "Ethereum")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "Ethereum")
		} else {
			return filepath.Join(home, ".ethereum")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}

// DefaultIPCEndpoint returns the default location where to open the IPC endpoint.
//
// On Windows this is DefaultIPCNamedPipe.
// On OSX this is DefaultDataDir/DefaultIPCSocket.
// Else this is $ENV{XDG_RUNTIME_DIR}/ethereum/DefaultIPCSocket, if $ENV{XDG_RUNTIME_DIR} is not set,
// or the ethereum sub directory could not be created this is DefaultDataDir/DefaultIPCSocket.
func DefaultIPCEndpoint() string {
	if runtime.GOOS == "windows" {
		return DefaultIPCNamedPipe
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(DefaultDataDir(), DefaultIPCSocket)
	}
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		dir = filepath.Join(dir, "ethereum")
		// create "ethereum" subdirectory if it doesn't exist
		if err := os.MkdirAll(dir, 0700); err == nil {
			return filepath.Join(dir, DefaultIPCSocket)
		}
	}

	return filepath.Join(DefaultDataDir(), DefaultIPCSocket)
}

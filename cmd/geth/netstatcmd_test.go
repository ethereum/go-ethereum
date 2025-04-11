// Copyright 2023 The go-ethereum Authors
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
	"io"
	"testing"

	"github.com/urfave/cli/v2"
)

// Tests that the netstat command exists and has the correct flags
func TestNetstatCommand(t *testing.T) {
	app := cli.NewApp()
	// Use an io.Discard-like writer to avoid output during tests
	app.Writer = io.Discard
	app.Commands = []*cli.Command{netstatCommand}

	// Verify that the monitor flag is registered
	var monitorFlagFound bool
	for _, flag := range netstatCommand.Flags {
		if flag.Names()[0] == monitorFlag.Name {
			monitorFlagFound = true
			break
		}
	}
	if !monitorFlagFound {
		t.Error("monitor flag not registered")
	}

	// Test that the command help output works
	if err := app.Run([]string{"geth", "netstat", "--help"}); err != nil {
		t.Fatalf("netstat --help failed: %v", err)
	}
}

// Test that the connection direction detection works correctly
func TestConnectionDirection(t *testing.T) {
	inbound := connectionDirection(true)
	outbound := connectionDirection(false)

	if inbound != "Inbound (remote dialed us)" {
		t.Errorf("expected 'Inbound (remote dialed us)', got '%s'", inbound)
	}
	if outbound != "Outbound (we dialed remote)" {
		t.Errorf("expected 'Outbound (we dialed remote)', got '%s'", outbound)
	}
}

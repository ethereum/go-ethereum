// Copyright 2020 The go-ethereum Authors
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

package test

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("CI") != "" {
		os.Exit(0)
	}
	flag.Parse()
	if *remote == "" {
		fmt.Fprintf(os.Stderr, "Need -remote to run this test\n")
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestPing(t *testing.T) {
	PingTests(t)
}

func TestAmplification(t *testing.T) {
	AmplificationTests(t)
}

func TestFindnode(t *testing.T) {
	FindnodeTests(t)
}

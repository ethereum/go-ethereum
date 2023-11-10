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

// Package testlog provides a log handler for unit tests.
package testlog

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

type relay struct {
	t *testing.T
}

func (r *relay) Write(p []byte) (n int, err error) {
	r.t.Log(strings.TrimSpace(string(p)))
	return len(p), nil
}

// Logger returns a logger which logs to the unit test log of t.
func Logger(t *testing.T, level log.Lvl) log.Logger {
	l := log.New()
	l.SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(&relay{t}, log.TerminalFormat(false))))
	return l
}

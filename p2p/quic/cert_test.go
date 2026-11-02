// Copyright 2026 The go-ethereum Authors
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

package quic

import (
	"testing"

	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
)

// TestCertRotatorLogger ensures NewCertRotator stores the logger it is given.
// The rotation loop calls r.log.Error when certificate generation fails, so a
// nil logger there would panic the rotation goroutine.
func TestCertRotatorLogger(t *testing.T) {
	logger := testlog.Logger(t, log.LvlDebug)
	r, err := NewCertRotator(logger)
	if err != nil {
		t.Fatal(err)
	}
	if r.log == nil {
		t.Fatal("NewCertRotator does not store the provided logger")
	}
	if r.log != logger {
		t.Fatal("NewCertRotator stores a logger different from the one provided")
	}
}

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

package rpc

import "testing"

func TestWSGetConfigNoAuth(t *testing.T) {
	config, err := wsGetConfig("ws://example.com:1234", "")
	if err != nil {
		t.Logf("wsGetConfig failed: %s", err)
		t.Fail()
		return
	}
	if config.Location.User != nil {
		t.Log("User should have been stripped from the URL")
		t.Fail()
	}
	if config.Location.Hostname() != "example.com" ||
		config.Location.Port() != "1234" || config.Location.Scheme != "ws" {
		t.Logf("Unexpected URL: %s", config.Location)
		t.Fail()
	}
}

func TestWSGetConfigWithBasicAuth(t *testing.T) {
	config, err := wsGetConfig("wss://testuser:test-PASS_01@example.com:1234", "")
	if err != nil {
		t.Logf("wsGetConfig failed: %s", err)
		t.Fail()
		return
	}
	if config.Location.User != nil {
		t.Log("User should have been stripped from the URL")
		t.Fail()
	}
	if config.Header.Get("Authorization") != "Basic dGVzdHVzZXI6dGVzdC1QQVNTXzAx" {
		t.Log("Basic auth header is incorrect")
		t.Fail()
	}
}

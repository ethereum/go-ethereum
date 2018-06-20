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

package api

import (
	"testing"
)

func testStorage(t *testing.T, f func(*Storage, bool)) {
	testAPI(t, func(api *API, toEncrypt bool) {
		f(NewStorage(api), toEncrypt)
	})
}

func TestStoragePutGet(t *testing.T) {
	testStorage(t, func(api *Storage, toEncrypt bool) {
		content := "hello"
		exp := expResponse(content, "text/plain", 0)
		// exp := expResponse([]byte(content), "text/plain", 0)
		bzzkey, wait, err := api.Put(content, exp.MimeType, toEncrypt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wait()
		bzzhash := bzzkey.Hex()
		// to check put against the API#Get
		resp0 := testGet(t, api.api, bzzhash, "")
		checkResponse(t, resp0, exp)

		// check storage#Get
		resp, err := api.Get(bzzhash)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		checkResponse(t, &testResponse{nil, resp}, exp)
	})
}

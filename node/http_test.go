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

package node

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestGetAuthBody(t *testing.T) {
	dir, err := ioutil.TempDir("", "httpclient-test")
	if err != nil {
		t.Fatal("cannot create temporary directory:", err)
	}
	defer os.RemoveAll(dir)
	client := newHTTP(dir)

	text := "test"
	hash := crypto.Sha3Hash([]byte(text))
	if err := ioutil.WriteFile(path.Join(dir, "test.content"), []byte(text), os.ModePerm); err != nil {
		t.Fatal("could not write test file", err)
	}
	content, err := client.GetAuthBody("file:///test.content", hash)
	if err != nil {
		t.Errorf("no error expected, got %v", err)
	}
	if string(content) != text {
		t.Errorf("incorrect content. expected %v, got %v", text, string(content))
	}

	hash = common.Hash{}
	content, err = client.GetAuthBody("file:///test.content", hash)
	expected := "body hash mismatch 0000000000000000000000000000000000000000000000000000000000000000 != 9c22ff5f21f0b81b113e63f7db6da94fedef11b2119b4088b89664fb9a3cb658 (exp)"
	if err == nil {
		t.Errorf("expected error, got nothing")
	} else {
		if err.Error() != expected {
			t.Errorf("expected error '%s' got '%v'", expected, err)
		}
	}

}

type rterr error
type rt struct{ error }

// roundtripper with an error. this can prove the roundtrip for testing
// without having the need to write a Response
func (e rt) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	return &http.Response{}, e.error
}

func TestHasScheme(t *testing.T) {
	client := newHTTP("")
	if client.HasScheme("scheme") {
		t.Errorf("expected scheme not to be registered")
	}
	client.registerSchemes([]URLScheme{{"scheme", &rt{rterr(errors.New("rt"))}}})
	if !client.HasScheme("scheme") {
		t.Errorf("expected scheme to be registered")
	}
	body, err := client.GetBody("scheme://url.com")
	if _, ok := err.(rterr); !ok {
		t.Fatalf("failed to use registered scheme. Got error %v and body %v", body, err)
	}
}

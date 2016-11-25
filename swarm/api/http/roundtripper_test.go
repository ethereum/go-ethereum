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

package http

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"
)

const port = "3222"

func TestRoundTripper(t *testing.T) {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "text/plain")
			http.ServeContent(w, r, "", time.Unix(0, 0), strings.NewReader(r.RequestURI))
		} else {
			http.Error(w, "Method "+r.Method+" is not supported.", http.StatusMethodNotAllowed)
		}
	})
	go http.ListenAndServe(":"+port, serveMux)

	rt := &RoundTripper{Port: port}
	trans := &http.Transport{}
	trans.RegisterProtocol("bzz", rt)
	client := &http.Client{Transport: trans}
	resp, err := client.Get("bzz://test.com/path")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
		return
	}

	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
		return
	}
	if string(content) != "/HTTP/1.1:/test.com/path" {
		t.Errorf("incorrect response from http server: expected '%v', got '%v'", "/HTTP/1.1:/test.com/path", string(content))
	}

}

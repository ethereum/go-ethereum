// Copyright 2017 The go-ethereum Authors
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
	"bytes"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

func TestBzzrGetPath(t *testing.T) {

	var err error

	maxproxyattempts := 3

	testmanifest := []string{
		`{"entries":[{"path":"a/","hash":"674af7073604ebfc0282a4ab21e5ef1a3c22913866879ebc0816f8a89896b2ed","contentType":"application/bzz-manifest+json","status":0}]}`,
		`{"entries":[{"path":"a","hash":"011b4d03dd8c01f1049143cf9c4c817e4b167f1d1b83e5c6f0f10d89ba1e7bce","contentType":"","status":0},{"path":"b/","hash":"0a87b1c3e4bf013686cdf107ec58590f2004610ee58cc2240f26939f691215f5","contentType":"application/bzz-manifest+json","status":0}]}`,
		`{"entries":[{"path":"b","hash":"011b4d03dd8c01f1049143cf9c4c817e4b167f1d1b83e5c6f0f10d89ba1e7bce","contentType":"","status":0},{"path":"c","hash":"011b4d03dd8c01f1049143cf9c4c817e4b167f1d1b83e5c6f0f10d89ba1e7bce","contentType":"","status":0}]}`,
	}

	testrequests := make(map[string]int)
	testrequests["/"] = 0
	testrequests["/a"] = 1
	testrequests["/a/b"] = 2
	testrequests["/x"] = 0
	testrequests[""] = 0

	expectedfailrequests := []string{"", "/x"}

	reader := [3]*bytes.Reader{}

	key := [3]storage.Key{}

	dir, _ := ioutil.TempDir("", "bzz-storage-test")

	storeparams := &storage.StoreParams{
		ChunkDbPath:   dir,
		DbCapacity:    5000000,
		CacheCapacity: 5000,
		Radius:        0,
	}

	localStore, err := storage.NewLocalStore(storage.MakeHashFunc("SHA3"), storeparams)
	if err != nil {
		t.Fatal(err)
	}
	chunker := storage.NewTreeChunker(storage.NewChunkerParams())
	dpa := &storage.DPA{
		Chunker:    chunker,
		ChunkStore: localStore,
	}
	dpa.Start()
	defer dpa.Stop()

	wg := &sync.WaitGroup{}

	for i, mf := range testmanifest {
		reader[i] = bytes.NewReader([]byte(mf))
		key[i], err = dpa.Store(reader[i], int64(len(mf)), wg, nil)
		if err != nil {
			t.Fatal(err)
		}
		wg.Wait()
	}

	a := api.NewApi(dpa, nil)

	/// \todo iterate port numbers up if fail
	StartHttpServer(a, &Server{Addr: "127.0.0.1:8504", CorsString: ""})
	// how to wait for ListenAndServe to have initialized? This is pretty cruuuude
	// if we fix it we don't need maxproxyattempts anymore either
	time.Sleep(1000 * time.Millisecond)
	for i := 0; i <= maxproxyattempts; i++ {
		_, err := http.Get("http://127.0.0.1:8504/bzzr:/" + common.ToHex(key[0])[2:] + "/a")
		if i == maxproxyattempts {
			t.Fatalf("Failed to connect to proxy after %v attempts: %v", i, err)
		} else if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		break
	}

	for k, v := range testrequests {
		var resp *http.Response
		var respbody []byte

		url := "http://127.0.0.1:8504/bzzr:/"
		if k[:] != "" {
			url += common.ToHex(key[0])[2:] + "/" + k[1:] + "?content_type=text/plain"
		}
		resp, err = http.Get(url)
		defer resp.Body.Close()
		respbody, err = ioutil.ReadAll(resp.Body)

		if string(respbody) != testmanifest[v] {
			isexpectedfailrequest := false

			for _, r := range expectedfailrequests {
				if k[:] == r {
					isexpectedfailrequest = true
				}
			}
			if isexpectedfailrequest == false {
				t.Fatalf("Response body does not match, expected: %v, got %v", testmanifest[v], string(respbody))
			}
		}
	}

}

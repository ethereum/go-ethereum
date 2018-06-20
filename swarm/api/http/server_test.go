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
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/api"
	swarm "github.com/ethereum/go-ethereum/swarm/api/client"
	"github.com/ethereum/go-ethereum/swarm/multihash"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/testutil"
)

func init() {
	loglevel := flag.Int("loglevel", 2, "loglevel")
	flag.Parse()
	log.Root().SetHandler(log.CallerFileHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(os.Stderr, log.TerminalFormat(true)))))
}

func TestResourcePostMode(t *testing.T) {
	path := ""
	errstr := "resourcePostMode for '%s' should be raw %v frequency %d, was raw %v, frequency %d"
	r, f, err := resourcePostMode(path)
	if err != nil {
		t.Fatal(err)
	} else if r || f != 0 {
		t.Fatalf(errstr, path, false, 0, r, f)
	}

	path = "raw"
	r, f, err = resourcePostMode(path)
	if err != nil {
		t.Fatal(err)
	} else if !r || f != 0 {
		t.Fatalf(errstr, path, true, 0, r, f)
	}

	path = "13"
	r, f, err = resourcePostMode(path)
	if err != nil {
		t.Fatal(err)
	} else if r || f == 0 {
		t.Fatalf(errstr, path, false, 13, r, f)
	}

	path = "raw/13"
	r, f, err = resourcePostMode(path)
	if err != nil {
		t.Fatal(err)
	} else if !r || f == 0 {
		t.Fatalf(errstr, path, true, 13, r, f)
	}

	path = "foo/13"
	r, f, err = resourcePostMode(path)
	if err == nil {
		t.Fatal("resourcePostMode for 'foo/13' should fail, returned error nil")
	}
}

func serverFunc(api *api.API) testutil.TestServer {
	return NewServer(api)
}

// test the transparent resolving of multihash resource types with bzz:// scheme
//
// first upload data, and store the multihash to the resulting manifest in a resource update
// retrieving the update with the multihash should return the manifest pointing directly to the data
// and raw retrieve of that hash should return the data
func TestBzzResourceMultihash(t *testing.T) {

	srv := testutil.NewTestSwarmServer(t, serverFunc)
	defer srv.Close()

	// add the data our multihash aliased manifest will point to
	databytes := "bar"
	url := fmt.Sprintf("%s/bzz:/", srv.URL)
	resp, err := http.Post(url, "text/plain", bytes.NewReader([]byte(databytes)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("err %s", resp.Status)
	}
	b, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		t.Fatal(err)
	}
	s := common.FromHex(string(b))
	mh := multihash.ToMultihash(s)

	mhHex := hexutil.Encode(mh)
	log.Info("added data", "manifest", string(b), "data", common.ToHex(mh))

	// our mutable resource "name"
	keybytes := "foo.eth"

	// create the multihash update
	url = fmt.Sprintf("%s/bzz-resource:/%s/13", srv.URL, keybytes)
	resp, err = http.Post(url, "application/octet-stream", bytes.NewReader([]byte(mhHex)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("err %s", resp.Status)
	}
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	rsrcResp := &storage.Address{}
	err = json.Unmarshal(b, rsrcResp)
	if err != nil {
		t.Fatalf("data %s could not be unmarshaled: %v", b, err)
	}

	correctManifestAddrHex := "d689648fb9e00ddc7ebcf474112d5881c5bf7dbc6e394681b1d224b11b59b5e0"
	if rsrcResp.Hex() != correctManifestAddrHex {
		t.Fatalf("Response resource key mismatch, expected '%s', got '%s'", correctManifestAddrHex, rsrcResp)
	}

	// get bzz manifest transparent resource resolve
	url = fmt.Sprintf("%s/bzz:/%s", srv.URL, rsrcResp)
	resp, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("err %s", resp.Status)
	}
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, []byte(databytes)) {
		t.Fatalf("retrieved data mismatch, expected %x, got %x", databytes, b)
	}
}

// Test resource updates using the raw update methods
func TestBzzResource(t *testing.T) {
	srv := testutil.NewTestSwarmServer(t, serverFunc)
	defer srv.Close()

	// our mutable resource "name"
	keybytes := "foo.eth"

	// data of update 1
	databytes := make([]byte, 666)
	_, err := rand.Read(databytes)
	if err != nil {
		t.Fatal(err)
	}

	// creates resource and sets update 1
	url := fmt.Sprintf("%s/bzz-resource:/%s/raw/13", srv.URL, []byte(keybytes))
	resp, err := http.Post(url, "application/octet-stream", bytes.NewReader(databytes))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("err %s", resp.Status)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	rsrcResp := &storage.Address{}
	err = json.Unmarshal(b, rsrcResp)
	if err != nil {
		t.Fatalf("data %s could not be unmarshaled: %v", b, err)
	}

	correctManifestAddrHex := "d689648fb9e00ddc7ebcf474112d5881c5bf7dbc6e394681b1d224b11b59b5e0"
	if rsrcResp.Hex() != correctManifestAddrHex {
		t.Fatalf("Response resource key mismatch, expected '%s', got '%s'", correctManifestAddrHex, rsrcResp.Hex())
	}

	// get the manifest
	url = fmt.Sprintf("%s/bzz-raw:/%s", srv.URL, rsrcResp)
	resp, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("err %s", resp.Status)
	}
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	manifest := &api.Manifest{}
	err = json.Unmarshal(b, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Entries) != 1 {
		t.Fatalf("Manifest has %d entries", len(manifest.Entries))
	}

	correctRootKeyHex := "f667277e004e8486c7a3631fd226802430e84e9a81b6085d31f512a591ae0065"
	if manifest.Entries[0].Hash != correctRootKeyHex {
		t.Fatalf("Expected manifest path '%s', got '%s'", correctRootKeyHex, manifest.Entries[0].Hash)
	}

	// get bzz manifest transparent resource resolve
	url = fmt.Sprintf("%s/bzz:/%s", srv.URL, rsrcResp)
	resp, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("err %s", resp.Status)
	}
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	// get non-existent name, should fail
	url = fmt.Sprintf("%s/bzz-resource:/bar", srv.URL)
	resp, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// get latest update (1.1) through resource directly
	log.Info("get update latest = 1.1", "addr", correctManifestAddrHex)
	url = fmt.Sprintf("%s/bzz-resource:/%s", srv.URL, correctManifestAddrHex)
	resp, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("err %s", resp.Status)
	}
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(databytes, b) {
		t.Fatalf("Expected body '%x', got '%x'", databytes, b)
	}

	// update 2
	log.Info("update 2")
	url = fmt.Sprintf("%s/bzz-resource:/%s/raw", srv.URL, correctManifestAddrHex)
	data := []byte("foo")
	resp, err = http.Post(url, "application/octet-stream", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Update returned %s", resp.Status)
	}

	// get latest update (1.2) through resource directly
	log.Info("get update 1.2")
	url = fmt.Sprintf("%s/bzz-resource:/%s", srv.URL, correctManifestAddrHex)
	resp, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("err %s", resp.Status)
	}
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, b) {
		t.Fatalf("Expected body '%x', got '%x'", data, b)
	}

	// get latest update (1.2) with specified period
	log.Info("get update latest = 1.2")
	url = fmt.Sprintf("%s/bzz-resource:/%s/1", srv.URL, correctManifestAddrHex)
	resp, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("err %s", resp.Status)
	}
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, b) {
		t.Fatalf("Expected body '%x', got '%x'", data, b)
	}

	// get first update (1.1) with specified period and version
	log.Info("get first update 1.1")
	url = fmt.Sprintf("%s/bzz-resource:/%s/1/1", srv.URL, correctManifestAddrHex)
	resp, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("err %s", resp.Status)
	}
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(databytes, b) {
		t.Fatalf("Expected body '%x', got '%x'", databytes, b)
	}
}

func TestBzzGetPath(t *testing.T) {
	testBzzGetPath(false, t)
	testBzzGetPath(true, t)
}

func testBzzGetPath(encrypted bool, t *testing.T) {
	var err error

	testmanifest := []string{
		`{"entries":[{"path":"b","hash":"011b4d03dd8c01f1049143cf9c4c817e4b167f1d1b83e5c6f0f10d89ba1e7bce","contentType":"","status":0},{"path":"c","hash":"011b4d03dd8c01f1049143cf9c4c817e4b167f1d1b83e5c6f0f10d89ba1e7bce","contentType":"","status":0}]}`,
		`{"entries":[{"path":"a","hash":"011b4d03dd8c01f1049143cf9c4c817e4b167f1d1b83e5c6f0f10d89ba1e7bce","contentType":"","status":0},{"path":"b/","hash":"<key0>","contentType":"application/bzz-manifest+json","status":0}]}`,
		`{"entries":[{"path":"a/","hash":"<key1>","contentType":"application/bzz-manifest+json","status":0}]}`,
	}

	testrequests := make(map[string]int)
	testrequests["/"] = 2
	testrequests["/a/"] = 1
	testrequests["/a/b/"] = 0
	testrequests["/x"] = 0
	testrequests[""] = 0

	expectedfailrequests := []string{"", "/x"}

	reader := [3]*bytes.Reader{}

	addr := [3]storage.Address{}

	srv := testutil.NewTestSwarmServer(t, serverFunc)
	defer srv.Close()

	for i, mf := range testmanifest {
		reader[i] = bytes.NewReader([]byte(mf))
		var wait func()
		addr[i], wait, err = srv.FileStore.Store(reader[i], int64(len(mf)), encrypted)
		for j := i + 1; j < len(testmanifest); j++ {
			testmanifest[j] = strings.Replace(testmanifest[j], fmt.Sprintf("<key%v>", i), addr[i].Hex(), -1)
		}
		if err != nil {
			t.Fatal(err)
		}
		wait()
	}

	rootRef := addr[2].Hex()

	_, err = http.Get(srv.URL + "/bzz-raw:/" + rootRef + "/a")
	if err != nil {
		t.Fatalf("Failed to connect to proxy: %v", err)
	}

	for k, v := range testrequests {
		var resp *http.Response
		var respbody []byte

		url := srv.URL + "/bzz-raw:/"
		if k[:] != "" {
			url += rootRef + "/" + k[1:] + "?content_type=text/plain"
		}
		resp, err = http.Get(url)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()
		respbody, err = ioutil.ReadAll(resp.Body)

		if string(respbody) != testmanifest[v] {
			isexpectedfailrequest := false

			for _, r := range expectedfailrequests {
				if k[:] == r {
					isexpectedfailrequest = true
				}
			}
			if !isexpectedfailrequest {
				t.Fatalf("Response body does not match, expected: %v, got %v", testmanifest[v], string(respbody))
			}
		}
	}

	for k, v := range testrequests {
		var resp *http.Response
		var respbody []byte

		url := srv.URL + "/bzz-hash:/"
		if k[:] != "" {
			url += rootRef + "/" + k[1:]
		}
		resp, err = http.Get(url)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()
		respbody, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Read request body: %v", err)
		}

		if string(respbody) != addr[v].Hex() {
			isexpectedfailrequest := false

			for _, r := range expectedfailrequests {
				if k[:] == r {
					isexpectedfailrequest = true
				}
			}
			if !isexpectedfailrequest {
				t.Fatalf("Response body does not match, expected: %v, got %v", addr[v], string(respbody))
			}
		}
	}

	ref := addr[2].Hex()

	for _, c := range []struct {
		path string
		json string
		html string
	}{
		{
			path: "/",
			json: `{"common_prefixes":["a/"]}`,
			html: fmt.Sprintf("<!DOCTYPE html>\n<html>\n<head>\n  <meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\">\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n\t\t<link rel=\"shortcut icon\" type=\"image/x-icon\" href=\"data:image/png;base64,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\"/>\n\t<title>Swarm index of bzz:/%s/</title>\n</head>\n\n<body>\n  <h1>Swarm index of bzz:/%s/</h1>\n  <hr>\n  <table>\n    <thead>\n      <tr>\n\t<th>Path</th>\n\t<th>Type</th>\n\t<th>Size</th>\n      </tr>\n    </thead>\n\n    <tbody>\n      \n\t<tr>\n\t  <td><a href=\"a/\">a/</a></td>\n\t  <td>DIR</td>\n\t  <td>-</td>\n\t</tr>\n      \n\n      \n  </table>\n  <hr>\n</body>\n", ref, ref),
		},
		{
			path: "/a/",
			json: `{"common_prefixes":["a/b/"],"entries":[{"hash":"011b4d03dd8c01f1049143cf9c4c817e4b167f1d1b83e5c6f0f10d89ba1e7bce","path":"a/a","mod_time":"0001-01-01T00:00:00Z"}]}`,
			html: fmt.Sprintf("<!DOCTYPE html>\n<html>\n<head>\n  <meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\">\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n\t\t<link rel=\"shortcut icon\" type=\"image/x-icon\" href=\"data:image/png;base64,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\"/>\n\t<title>Swarm index of bzz:/%s/a/</title>\n</head>\n\n<body>\n  <h1>Swarm index of bzz:/%s/a/</h1>\n  <hr>\n  <table>\n    <thead>\n      <tr>\n\t<th>Path</th>\n\t<th>Type</th>\n\t<th>Size</th>\n      </tr>\n    </thead>\n\n    <tbody>\n      \n\t<tr>\n\t  <td><a href=\"b/\">b/</a></td>\n\t  <td>DIR</td>\n\t  <td>-</td>\n\t</tr>\n      \n\n      \n\t<tr>\n\t  <td><a href=\"a\">a</a></td>\n\t  <td></td>\n\t  <td>0</td>\n\t</tr>\n      \n  </table>\n  <hr>\n</body>\n", ref, ref),
		},
		{
			path: "/a/b/",
			json: `{"entries":[{"hash":"011b4d03dd8c01f1049143cf9c4c817e4b167f1d1b83e5c6f0f10d89ba1e7bce","path":"a/b/b","mod_time":"0001-01-01T00:00:00Z"},{"hash":"011b4d03dd8c01f1049143cf9c4c817e4b167f1d1b83e5c6f0f10d89ba1e7bce","path":"a/b/c","mod_time":"0001-01-01T00:00:00Z"}]}`,
			html: fmt.Sprintf("<!DOCTYPE html>\n<html>\n<head>\n  <meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\">\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n\t\t<link rel=\"shortcut icon\" type=\"image/x-icon\" href=\"data:image/png;base64,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\"/>\n\t<title>Swarm index of bzz:/%s/a/b/</title>\n</head>\n\n<body>\n  <h1>Swarm index of bzz:/%s/a/b/</h1>\n  <hr>\n  <table>\n    <thead>\n      <tr>\n\t<th>Path</th>\n\t<th>Type</th>\n\t<th>Size</th>\n      </tr>\n    </thead>\n\n    <tbody>\n      \n\n      \n\t<tr>\n\t  <td><a href=\"b\">b</a></td>\n\t  <td></td>\n\t  <td>0</td>\n\t</tr>\n      \n\t<tr>\n\t  <td><a href=\"c\">c</a></td>\n\t  <td></td>\n\t  <td>0</td>\n\t</tr>\n      \n  </table>\n  <hr>\n</body>\n", ref, ref),
		},
		{
			path: "/x",
		},
		{
			path: "",
		},
	} {
		k := c.path
		url := srv.URL + "/bzz-list:/"
		if k[:] != "" {
			url += rootRef + "/" + k[1:]
		}
		t.Run("json list "+c.path, func(t *testing.T) {
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("HTTP request: %v", err)
			}
			defer resp.Body.Close()
			respbody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Read response body: %v", err)
			}

			body := strings.TrimSpace(string(respbody))
			if body != c.json {
				isexpectedfailrequest := false

				for _, r := range expectedfailrequests {
					if k[:] == r {
						isexpectedfailrequest = true
					}
				}
				if !isexpectedfailrequest {
					t.Errorf("Response list body %q does not match, expected: %v, got %v", k, c.json, body)
				}
			}
		})
		t.Run("html list "+c.path, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Fatalf("New request: %v", err)
			}
			req.Header.Set("Accept", "text/html")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("HTTP request: %v", err)
			}
			defer resp.Body.Close()
			respbody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Read response body: %v", err)
			}

			if string(respbody) != c.html {
				isexpectedfailrequest := false

				for _, r := range expectedfailrequests {
					if k[:] == r {
						isexpectedfailrequest = true
					}
				}
				if !isexpectedfailrequest {
					t.Errorf("Response list body %q does not match, expected: %q, got %q", k, c.html, string(respbody))
				}
			}
		})
	}

	nonhashtests := []string{
		srv.URL + "/bzz:/name",
		srv.URL + "/bzz-immutable:/nonhash",
		srv.URL + "/bzz-raw:/nonhash",
		srv.URL + "/bzz-list:/nonhash",
		srv.URL + "/bzz-hash:/nonhash",
	}

	nonhashresponses := []string{
		"cannot resolve name: no DNS to resolve name: &#34;name&#34;",
		"cannot resolve nonhash: immutable address not a content hash: &#34;nonhash&#34;",
		"cannot resolve nonhash: no DNS to resolve name: &#34;nonhash&#34;",
		"cannot resolve nonhash: no DNS to resolve name: &#34;nonhash&#34;",
		"cannot resolve nonhash: no DNS to resolve name: &#34;nonhash&#34;",
	}

	for i, url := range nonhashtests {
		var resp *http.Response
		var respbody []byte

		resp, err = http.Get(url)

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()
		respbody, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("ReadAll failed: %v", err)
		}
		if !strings.Contains(string(respbody), nonhashresponses[i]) {
			t.Fatalf("Non-Hash response body does not match, expected: %v, got: %v", nonhashresponses[i], string(respbody))
		}
	}
}

// TestBzzRootRedirect tests that getting the root path of a manifest without
// a trailing slash gets redirected to include the trailing slash so that
// relative URLs work as expected.
func TestBzzRootRedirect(t *testing.T) {
	testBzzRootRedirect(false, t)
}
func TestBzzRootRedirectEncrypted(t *testing.T) {
	testBzzRootRedirect(true, t)
}

func testBzzRootRedirect(toEncrypt bool, t *testing.T) {
	srv := testutil.NewTestSwarmServer(t, serverFunc)
	defer srv.Close()

	// create a manifest with some data at the root path
	client := swarm.NewClient(srv.URL)
	data := []byte("data")
	file := &swarm.File{
		ReadCloser: ioutil.NopCloser(bytes.NewReader(data)),
		ManifestEntry: api.ManifestEntry{
			Path:        "",
			ContentType: "text/plain",
			Size:        int64(len(data)),
		},
	}
	hash, err := client.Upload(file, "", toEncrypt)
	if err != nil {
		t.Fatal(err)
	}

	// define a CheckRedirect hook which ensures there is only a single
	// redirect to the correct URL
	redirected := false
	httpClient := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if redirected {
				return errors.New("too many redirects")
			}
			redirected = true
			expectedPath := "/bzz:/" + hash + "/"
			if req.URL.Path != expectedPath {
				return fmt.Errorf("expected redirect to %q, got %q", expectedPath, req.URL.Path)
			}
			return nil
		},
	}

	// perform the GET request and assert the response
	res, err := httpClient.Get(srv.URL + "/bzz:/" + hash)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if !redirected {
		t.Fatal("expected GET /bzz:/<hash> to redirect to /bzz:/<hash>/ but it didn't")
	}
	gotData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotData, data) {
		t.Fatalf("expected response to equal %q, got %q", data, gotData)
	}
}

func TestMethodsNotAllowed(t *testing.T) {
	srv := testutil.NewTestSwarmServer(t, serverFunc)
	defer srv.Close()
	databytes := "bar"
	for _, c := range []struct {
		url  string
		code int
	}{
		{
			url:  fmt.Sprintf("%s/bzz-list:/", srv.URL),
			code: 405,
		}, {
			url:  fmt.Sprintf("%s/bzz-hash:/", srv.URL),
			code: 405,
		},
		{
			url:  fmt.Sprintf("%s/bzz-immutable:/", srv.URL),
			code: 405,
		},
	} {
		res, _ := http.Post(c.url, "text/plain", bytes.NewReader([]byte(databytes)))
		if res.StatusCode != c.code {
			t.Fatal("should have failed")
		}
	}

}

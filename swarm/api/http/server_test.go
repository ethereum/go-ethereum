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
	"archive/tar"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/api"
	swarm "github.com/ethereum/go-ethereum/swarm/api/client"
	"github.com/ethereum/go-ethereum/swarm/multihash"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/mru"
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
	return NewServer(api, "")
}

func newTestSigner() (*mru.GenericSigner, error) {
	privKey, err := crypto.HexToECDSA("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	if err != nil {
		return nil, err
	}
	return mru.NewGenericSigner(privKey), nil
}

// test the transparent resolving of multihash resource types with bzz:// scheme
//
// first upload data, and store the multihash to the resulting manifest in a resource update
// retrieving the update with the multihash should return the manifest pointing directly to the data
// and raw retrieve of that hash should return the data
func TestBzzResourceMultihash(t *testing.T) {

	signer, _ := newTestSigner()

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

	log.Info("added data", "manifest", string(b), "data", common.ToHex(mh))

	// our mutable resource "name"
	keybytes := "foo.eth"

	updateRequest, err := mru.NewCreateUpdateRequest(&mru.ResourceMetadata{
		Name:      keybytes,
		Frequency: 13,
		StartTime: srv.GetCurrentTime(),
		Owner:     signer.Address(),
	})
	if err != nil {
		t.Fatal(err)
	}
	updateRequest.SetData(mh, true)

	if err := updateRequest.Sign(signer); err != nil {
		t.Fatal(err)
	}
	log.Info("added data", "manifest", string(b), "data", common.ToHex(mh))

	body, err := updateRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	// create the multihash update
	url = fmt.Sprintf("%s/bzz-resource:/", srv.URL)
	resp, err = http.Post(url, "application/json", bytes.NewReader(body))
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

	correctManifestAddrHex := "6d3bc4664c97d8b821cb74bcae43f592494fb46d2d9cd31e69f3c7c802bbbd8e"
	if rsrcResp.Hex() != correctManifestAddrHex {
		t.Fatalf("Response resource key mismatch, expected '%s', got '%s'", correctManifestAddrHex, rsrcResp.Hex())
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
	signer, _ := newTestSigner()

	defer srv.Close()

	// our mutable resource "name"
	keybytes := "foo.eth"

	// data of update 1
	databytes := make([]byte, 666)
	_, err := rand.Read(databytes)
	if err != nil {
		t.Fatal(err)
	}

	updateRequest, err := mru.NewCreateUpdateRequest(&mru.ResourceMetadata{
		Name:      keybytes,
		Frequency: 13,
		StartTime: srv.GetCurrentTime(),
		Owner:     signer.Address(),
	})
	if err != nil {
		t.Fatal(err)
	}
	updateRequest.SetData(databytes, false)

	if err := updateRequest.Sign(signer); err != nil {
		t.Fatal(err)
	}

	body, err := updateRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	// creates resource and sets update 1
	url := fmt.Sprintf("%s/bzz-resource:/", srv.URL)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
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

	correctManifestAddrHex := "6d3bc4664c97d8b821cb74bcae43f592494fb46d2d9cd31e69f3c7c802bbbd8e"
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
	correctRootKeyHex := "68f7ba07ac8867a4c841a4d4320e3cdc549df23702dc7285fcb6acf65df48562"
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

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected get non-existent resource to fail with StatusNotFound (404), got %d", resp.StatusCode)
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

	// 1.- get metadata about this resource
	url = fmt.Sprintf("%s/bzz-resource:/%s/", srv.URL, correctManifestAddrHex)
	resp, err = http.Get(url + "meta")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Get resource metadata returned %s", resp.Status)
	}
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	updateRequest = &mru.Request{}
	if err = updateRequest.UnmarshalJSON(b); err != nil {
		t.Fatalf("Error decoding resource metadata: %s", err)
	}
	data := []byte("foo")
	updateRequest.SetData(data, false)
	if err = updateRequest.Sign(signer); err != nil {
		t.Fatal(err)
	}
	body, err = updateRequest.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	resp, err = http.Post(url, "application/json", bytes.NewReader(body))
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
		var wait func(context.Context) error
		ctx := context.TODO()
		addr[i], wait, err = srv.FileStore.Store(ctx, reader[i], int64(len(mf)), encrypted)
		for j := i + 1; j < len(testmanifest); j++ {
			testmanifest[j] = strings.Replace(testmanifest[j], fmt.Sprintf("<key%v>", i), addr[i].Hex(), -1)
		}
		if err != nil {
			t.Fatal(err)
		}
		err = wait(ctx)
		if err != nil {
			t.Fatal(err)
		}
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
		path          string
		json          string
		pageFragments []string
	}{
		{
			path: "/",
			json: `{"common_prefixes":["a/"]}`,
			pageFragments: []string{
				fmt.Sprintf("Swarm index of bzz:/%s/", ref),
				`<a class="normal-link" href="a/">a/</a>`,
			},
		},
		{
			path: "/a/",
			json: `{"common_prefixes":["a/b/"],"entries":[{"hash":"011b4d03dd8c01f1049143cf9c4c817e4b167f1d1b83e5c6f0f10d89ba1e7bce","path":"a/a","mod_time":"0001-01-01T00:00:00Z"}]}`,
			pageFragments: []string{
				fmt.Sprintf("Swarm index of bzz:/%s/a/", ref),
				`<a class="normal-link" href="b/">b/</a>`,
				`<a class="normal-link" href="a">a</a>`,
			},
		},
		{
			path: "/a/b/",
			json: `{"entries":[{"hash":"011b4d03dd8c01f1049143cf9c4c817e4b167f1d1b83e5c6f0f10d89ba1e7bce","path":"a/b/b","mod_time":"0001-01-01T00:00:00Z"},{"hash":"011b4d03dd8c01f1049143cf9c4c817e4b167f1d1b83e5c6f0f10d89ba1e7bce","path":"a/b/c","mod_time":"0001-01-01T00:00:00Z"}]}`,
			pageFragments: []string{
				fmt.Sprintf("Swarm index of bzz:/%s/a/b/", ref),
				`<a class="normal-link" href="b">b</a>`,
				`<a class="normal-link" href="c">c</a>`,
			},
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
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Read response body: %v", err)
			}

			body := string(b)

			for _, f := range c.pageFragments {
				if !strings.Contains(body, f) {
					isexpectedfailrequest := false

					for _, r := range expectedfailrequests {
						if k[:] == r {
							isexpectedfailrequest = true
						}
					}
					if !isexpectedfailrequest {
						t.Errorf("Response list body %q does not contain %q: body %q", k, f, body)
					}
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
		`cannot resolve name: no DNS to resolve name: "name"`,
		`cannot resolve nonhash: immutable address not a content hash: "nonhash"`,
		`cannot resolve nonhash: no DNS to resolve name: "nonhash"`,
		`cannot resolve nonhash: no DNS to resolve name: "nonhash"`,
		`cannot resolve nonhash: no DNS to resolve name: "nonhash"`,
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

func TestBzzTar(t *testing.T) {
	testBzzTar(false, t)
	testBzzTar(true, t)
}

func testBzzTar(encrypted bool, t *testing.T) {
	srv := testutil.NewTestSwarmServer(t, serverFunc)
	defer srv.Close()
	fileNames := []string{"tmp1.txt", "tmp2.lock", "tmp3.rtf"}
	fileContents := []string{"tmp1textfilevalue", "tmp2lockfilelocked", "tmp3isjustaplaintextfile"}

	buf := &bytes.Buffer{}
	tw := tar.NewWriter(buf)
	defer tw.Close()

	for i, v := range fileNames {
		size := int64(len(fileContents[i]))
		hdr := &tar.Header{
			Name:    v,
			Mode:    0644,
			Size:    size,
			ModTime: time.Now(),
			Xattrs: map[string]string{
				"user.swarm.content-type": "text/plain",
			},
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}

		// copy the file into the tar stream
		n, err := io.Copy(tw, bytes.NewBufferString(fileContents[i]))
		if err != nil {
			t.Fatal(err)
		} else if n != size {
			t.Fatal("size mismatch")
		}
	}

	//post tar stream
	url := srv.URL + "/bzz:/"
	if encrypted {
		url = url + "encrypt"
	}
	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-tar")
	client := &http.Client{}
	resp2, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("err %s", resp2.Status)
	}
	swarmHash, err := ioutil.ReadAll(resp2.Body)
	resp2.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	// now do a GET to get a tarball back
	req, err = http.NewRequest("GET", fmt.Sprintf(srv.URL+"/bzz:/%s", string(swarmHash)), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Accept", "application/x-tar")
	resp2, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()

	file, err := ioutil.TempFile("", "swarm-downloaded-tarball")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())
	_, err = io.Copy(file, resp2.Body)
	if err != nil {
		t.Fatalf("error getting tarball: %v", err)
	}
	file.Sync()
	file.Close()

	tarFileHandle, err := os.Open(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(tarFileHandle)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("error reading tar stream: %s", err)
		}
		bb := make([]byte, hdr.Size)
		_, err = tr.Read(bb)
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}
		passed := false
		for i, v := range fileNames {
			if v == hdr.Name {
				if string(bb) == fileContents[i] {
					passed = true
					break
				}
			}
		}
		if !passed {
			t.Fatalf("file %s did not pass content assertion", hdr.Name)
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
			t.Fatalf("should have failed. requested url: %s, expected code %d, got %d", c.url, c.code, res.StatusCode)
		}
	}

}

// HTTP convenience function
func httpDo(httpMethod string, url string, reqBody io.Reader, headers map[string]string, verbose bool, t *testing.T) (*http.Response, string) {
	// Build the Request
	req, err := http.NewRequest(httpMethod, url, reqBody)
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if verbose {
		t.Log(req.Method, req.URL, req.Header, req.Body)
	}

	// Send Request out
	httpClient := &http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	// Read the HTTP Body
	buffer, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body := string(buffer)

	return res, body
}

func TestGet(t *testing.T) {
	// Setup Swarm
	srv := testutil.NewTestSwarmServer(t, serverFunc)
	defer srv.Close()

	testCases := []struct {
		uri                string
		method             string
		headers            map[string]string
		expectedStatusCode int
		assertResponseBody string
		verbose            bool
	}{
		{
			// Accept: text/html GET / -> 200 HTML, Swarm Landing Page
			uri:                fmt.Sprintf("%s/", srv.URL),
			method:             "GET",
			headers:            map[string]string{"Accept": "text/html"},
			expectedStatusCode: 200,
			assertResponseBody: "<a href=\"/bzz:/theswarm.eth\">Swarm</a>: Serverless Hosting Incentivised peer-to-peer Storage and Content Distribution",
			verbose:            false,
		},
		{
			// Accept: application/json GET / -> 200 'Welcome to Swarm'
			uri:                fmt.Sprintf("%s/", srv.URL),
			method:             "GET",
			headers:            map[string]string{"Accept": "application/json"},
			expectedStatusCode: 200,
			assertResponseBody: "Welcome to Swarm!",
			verbose:            false,
		},
		{
			// GET /robots.txt -> 200
			uri:                fmt.Sprintf("%s/robots.txt", srv.URL),
			method:             "GET",
			headers:            map[string]string{"Accept": "text/html"},
			expectedStatusCode: 200,
			assertResponseBody: "User-agent: *\nDisallow: /",
			verbose:            false,
		},
		{
			// GET /path_that_doesnt exist -> 400
			uri:                fmt.Sprintf("%s/nonexistent_path", srv.URL),
			method:             "GET",
			headers:            map[string]string{},
			expectedStatusCode: 400,
			verbose:            false,
		},
		{
			// GET bzz-invalid:/ -> 400
			uri:                fmt.Sprintf("%s/bzz:asdf/", srv.URL),
			method:             "GET",
			headers:            map[string]string{},
			expectedStatusCode: 400,
			verbose:            false,
		},
		{
			// GET bzz-invalid:/ -> 400
			uri:                fmt.Sprintf("%s/tbz2/", srv.URL),
			method:             "GET",
			headers:            map[string]string{},
			expectedStatusCode: 400,
			verbose:            false,
		},
		{
			// GET bzz-invalid:/ -> 400
			uri:                fmt.Sprintf("%s/bzz-rack:/", srv.URL),
			method:             "GET",
			headers:            map[string]string{},
			expectedStatusCode: 400,
			verbose:            false,
		},
		{
			// GET bzz-invalid:/ -> 400
			uri:                fmt.Sprintf("%s/bzz-ls", srv.URL),
			method:             "GET",
			headers:            map[string]string{},
			expectedStatusCode: 400,
			verbose:            false,
		},
	}

	for _, testCase := range testCases {
		t.Run("GET "+testCase.uri, func(t *testing.T) {
			res, body := httpDo(testCase.method, testCase.uri, nil, testCase.headers, testCase.verbose, t)
			if res.StatusCode != testCase.expectedStatusCode {
				t.Fatalf("expected %s %s to return a %v but it didn't", testCase.method, testCase.uri, testCase.expectedStatusCode)
			}
			if testCase.assertResponseBody != "" && !strings.Contains(body, testCase.assertResponseBody) {
				t.Fatalf("expected %s %s to have %s within HTTP response body but it didn't", testCase.method, testCase.uri, testCase.assertResponseBody)
			}
		})
	}
}

func TestModify(t *testing.T) {
	// Setup Swarm and upload a test file to it
	srv := testutil.NewTestSwarmServer(t, serverFunc)
	defer srv.Close()

	swarmClient := swarm.NewClient(srv.URL)
	data := []byte("data")
	file := &swarm.File{
		ReadCloser: ioutil.NopCloser(bytes.NewReader(data)),
		ManifestEntry: api.ManifestEntry{
			Path:        "",
			ContentType: "text/plain",
			Size:        int64(len(data)),
		},
	}

	hash, err := swarmClient.Upload(file, "", false)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		uri                   string
		method                string
		headers               map[string]string
		requestBody           []byte
		expectedStatusCode    int
		assertResponseBody    string
		assertResponseHeaders map[string]string
		verbose               bool
	}{
		{
			// DELETE bzz:/hash -> 200 OK
			uri:                fmt.Sprintf("%s/bzz:/%s", srv.URL, hash),
			method:             "DELETE",
			headers:            map[string]string{},
			expectedStatusCode: 200,
			assertResponseBody: "8b634aea26eec353ac0ecbec20c94f44d6f8d11f38d4578a4c207a84c74ef731",
			verbose:            false,
		},
		{
			// PUT bzz:/hash -> 405 Method Not Allowed
			uri:                fmt.Sprintf("%s/bzz:/%s", srv.URL, hash),
			method:             "PUT",
			headers:            map[string]string{},
			expectedStatusCode: 405,
			verbose:            false,
		},
		{
			// PUT bzz-raw:/hash -> 405 Method Not Allowed
			uri:                fmt.Sprintf("%s/bzz-raw:/%s", srv.URL, hash),
			method:             "PUT",
			headers:            map[string]string{},
			expectedStatusCode: 405,
			verbose:            false,
		},
		{
			// PATCH bzz:/hash -> 405 Method Not Allowed
			uri:                fmt.Sprintf("%s/bzz:/%s", srv.URL, hash),
			method:             "PATCH",
			headers:            map[string]string{},
			expectedStatusCode: 405,
			verbose:            false,
		},
		{
			// POST bzz-raw:/ -> 200 OK
			uri:                   fmt.Sprintf("%s/bzz-raw:/", srv.URL),
			method:                "POST",
			headers:               map[string]string{},
			requestBody:           []byte("POSTdata"),
			expectedStatusCode:    200,
			assertResponseHeaders: map[string]string{"Content-Length": "64"},
			verbose:               false,
		},
		{
			// POST bzz-raw:/encrypt -> 200 OK
			uri:                   fmt.Sprintf("%s/bzz-raw:/encrypt", srv.URL),
			method:                "POST",
			headers:               map[string]string{},
			requestBody:           []byte("POSTdata"),
			expectedStatusCode:    200,
			assertResponseHeaders: map[string]string{"Content-Length": "128"},
			verbose:               false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.method+" "+testCase.uri, func(t *testing.T) {
			reqBody := bytes.NewReader(testCase.requestBody)
			res, body := httpDo(testCase.method, testCase.uri, reqBody, testCase.headers, testCase.verbose, t)

			if res.StatusCode != testCase.expectedStatusCode {
				t.Fatalf("expected %s %s to return a %v but it returned a %v instead", testCase.method, testCase.uri, testCase.expectedStatusCode, res.StatusCode)
			}
			if testCase.assertResponseBody != "" && !strings.Contains(body, testCase.assertResponseBody) {
				t.Log(body)
				t.Fatalf("expected %s %s to have %s within HTTP response body but it didn't", testCase.method, testCase.uri, testCase.assertResponseBody)
			}
			for key, value := range testCase.assertResponseHeaders {
				if res.Header.Get(key) != value {
					t.Logf("expected %s=%s in HTTP response header but got %s", key, value, res.Header.Get(key))
				}
			}
		})
	}
}

func TestMultiPartUpload(t *testing.T) {
	// POST /bzz:/ Content-Type: multipart/form-data
	verbose := false
	// Setup Swarm
	srv := testutil.NewTestSwarmServer(t, serverFunc)
	defer srv.Close()

	url := fmt.Sprintf("%s/bzz:/", srv.URL)

	buf := new(bytes.Buffer)
	form := multipart.NewWriter(buf)
	form.WriteField("name", "John Doe")
	file1, _ := form.CreateFormFile("cv", "cv.txt")
	file1.Write([]byte("John Doe's Credentials"))
	file2, _ := form.CreateFormFile("profile_picture", "profile.jpg")
	file2.Write([]byte("imaginethisisjpegdata"))
	form.Close()

	headers := map[string]string{
		"Content-Type":   form.FormDataContentType(),
		"Content-Length": strconv.Itoa(buf.Len()),
	}
	res, body := httpDo("POST", url, buf, headers, verbose, t)

	if res.StatusCode != 200 {
		t.Fatalf("expected POST multipart/form-data to return 200, but it returned %d", res.StatusCode)
	}
	if len(body) != 64 {
		t.Fatalf("expected POST multipart/form-data to return a 64 char manifest but the answer was %d chars long", len(body))
	}
}
